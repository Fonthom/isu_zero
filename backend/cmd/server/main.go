package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/isu-zero/isu-zero/internal/interactions"
	"github.com/isu-zero/isu-zero/internal/navigation"
	"github.com/isu-zero/isu-zero/internal/products"
	"github.com/isu-zero/isu-zero/internal/pubsub"
)

func main() {
	ctx := context.Background()

	// Database
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("database unreachable: %v", err)
	}

	// NATS
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatalf("unable to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Pub/sub bus
	bus, err := pubsub.New(nc)
	if err != nil {
		log.Fatalf("unable to initialise pub/sub bus: %v", err)
	}

	// Services
	productSvc := products.NewService(db)
	navSvc := navigation.NewService(bus)
	interactionSvc := interactions.NewService(db, bus)

	// Start NATS subscribers
	if err := interactionSvc.StartSubscribers(ctx); err != nil {
		log.Fatalf("unable to start interaction subscribers: %v", err)
	}

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	// API routes
	r.Mount("/api/products", products.NewHandler(productSvc, navSvc))
	r.Mount("/api/interactions", interactions.NewHandler(interactionSvc))

	// Serve crop images from the shared photo volume
	r.Handle("/photos/*", http.StripPrefix("/photos/", http.FileServer(http.Dir("/photos"))))

	// Serve the UI — must be last so it doesn't swallow API routes
	uiDir := os.Getenv("UI_DIR")
	if uiDir == "" {
		uiDir = "./ui"
	}
	r.Handle("/*", http.FileServer(http.Dir(uiDir)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("ISU-Zero backend running on :%s\n", port)
	fmt.Printf("UI served from: %s\n", uiDir)
	log.Fatal(http.ListenAndServe(":"+port, r))
}