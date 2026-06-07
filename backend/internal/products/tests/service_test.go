package products_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/isu-zero/isu-zero/internal/navigation"
	"github.com/isu-zero/isu-zero/internal/products"
	"github.com/isu-zero/isu-zero/internal/pubsub"
	"github.com/nats-io/nats.go"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL not set — run tests via scripts/test.sh")
	}
	db, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func setupBus(t *testing.T) *pubsub.Bus {
	t.Helper()
	url := os.Getenv("NATS_URL")
	if url == "" {
		t.Fatal("NATS_URL not set — run tests via scripts/test.sh")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	bus, err := pubsub.New(nc)
	if err != nil {
		t.Fatalf("failed to create pub/sub bus: %v", err)
	}
	t.Cleanup(func() { nc.Close() })
	return bus
}

// ── Service tests ─────────────────────────────────────────────────────────────

func TestSearch_ReturnsMatchingProducts(t *testing.T) {
	db := setupDB(t)
	svc := products.NewService(db)

	results, err := svc.Search(context.Background(), "cola")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'cola', got none")
	}

	found := false
	for _, p := range results {
		if strings.Contains(strings.ToLower(p.Name), "cola") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected result containing 'cola', got: %+v", results)
	}
}

func TestSearch_ExcludesUnnamedProducts(t *testing.T) {
	db := setupDB(t)
	svc := products.NewService(db)

	// Search broadly — unnamed product should never appear
	results, err := svc.Search(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	for _, p := range results {
		if p.Name == "" {
			t.Errorf("Search returned an unnamed product: %+v", p)
		}
	}
}

func TestSearch_ReturnsEmptyForUnknownQuery(t *testing.T) {
	db := setupDB(t)
	svc := products.NewService(db)

	results, err := svc.Search(context.Background(), "xyznotaproduct")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for unknown query, got %d", len(results))
	}
}

func TestGetByID_ReturnsCorrectProduct(t *testing.T) {
	db := setupDB(t)
	svc := products.NewService(db)

	// Known ID from seed_test.sql
	id := "c0000000-0000-0000-0000-000000000001"
	product, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if product.ID != id {
		t.Errorf("expected ID %s, got %s", id, product.ID)
	}
	if product.Name != "Coca-Cola 500ml" {
		t.Errorf("expected name 'Coca-Cola 500ml', got '%s'", product.Name)
	}
}

func TestGetByID_ReturnsNavCoordinates(t *testing.T) {
	db := setupDB(t)
	svc := products.NewService(db)

	id := "c0000000-0000-0000-0000-000000000001"
	product, err := svc.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if product.NavX == 0 && product.NavY == 0 {
		t.Error("expected non-zero nav coordinates from waypoint join")
	}
}

func TestGetByID_ErrorsOnUnknownID(t *testing.T) {
	db := setupDB(t)
	svc := products.NewService(db)

	_, err := svc.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Error("expected error for unknown ID, got nil")
	}
}

// ── HTTP handler tests ────────────────────────────────────────────────────────

func TestSearchHandler_MissingQueryParam(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	navSvc := navigation.NewService(bus)
	svc := products.NewService(db)
	handler := products.NewHandler(svc, navSvc)

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearchHandler_ReturnsJSON(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	navSvc := navigation.NewService(bus)
	svc := products.NewService(db)
	handler := products.NewHandler(svc, navSvc)

	req := httptest.NewRequest(http.MethodGet, "/search?q=cola", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestNavigateHandler_UnknownProduct(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	navSvc := navigation.NewService(bus)
	svc := products.NewService(db)
	handler := products.NewHandler(svc, navSvc)

	req := httptest.NewRequest(http.MethodPost, "/00000000-0000-0000-0000-000000000000/navigate", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}