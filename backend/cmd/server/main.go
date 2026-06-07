package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	// NATS
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatalf("unable to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Services
	bus := pubsub.New(nc)
	productSvc := products.NewService(db)
	navSvc := navigation.NewService(bus)
	interactionSvc := interactions.NewService(db, bus)

	// Start NATS subscribers in background
	interactionSvc.StartSubscribers(ctx)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Mount("/api/products", products.NewHandler(productSvc, navSvc))
	r.Mount("/api/interactions", interactions.NewHandler(interactionSvc))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("ISU-Zero backend running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}