package interactions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/isu-zero/isu-zero/internal/interactions"
	"github.com/isu-zero/isu-zero/internal/navigation"
	"github.com/isu-zero/isu-zero/internal/pubsub"
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

func TestLog_InsertsInteraction(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	productID := "c0000000-0000-0000-0000-000000000002"
	err := svc.Log(context.Background(), productID, "orange juice", "navigated", 30)
	if err != nil {
		t.Fatalf("Log returned error: %v", err)
	}

	// Verify it was inserted by fetching recent interactions
	recent, err := svc.Recent(context.Background(), 10)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}

	found := false
	for _, i := range recent {
		if i.QueryText == "orange juice" && i.Outcome == "navigated" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find logged interaction in recent results")
	}
}

func TestLog_NullProductID(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	// product_id is nullable — logging a not_found with no product should succeed
	err := svc.Log(context.Background(), "", "batteries", "not_found", 0)
	if err != nil {
		t.Fatalf("Log with empty product ID returned error: %v", err)
	}
}

func TestRecent_ReturnsInDescendingOrder(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	results, err := svc.Recent(context.Background(), 10)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}
	if len(results) < 2 {
		t.Skip("not enough interactions to test ordering")
	}

	for i := 1; i < len(results); i++ {
		prev, err := time.Parse(time.RFC3339, results[i-1].CreatedAt)
		if err != nil {
			t.Fatalf("failed to parse timestamp: %v", err)
		}
		curr, err := time.Parse(time.RFC3339, results[i].CreatedAt)
		if err != nil {
			t.Fatalf("failed to parse timestamp: %v", err)
		}
		if prev.Before(curr) {
			t.Errorf("interactions not in descending order at index %d", i)
		}
	}
}

func TestRecent_RespectsLimit(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	results, err := svc.Recent(context.Background(), 2)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestRecent_HandlesNullProductID(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	// seed_test.sql has a not_found interaction with NULL product_id
	// this should not crash
	results, err := svc.Recent(context.Background(), 50)
	if err != nil {
		t.Fatalf("Recent crashed on null product_id: %v", err)
	}

	found := false
	for _, i := range results {
		if i.QueryText == "batteries" && i.Outcome == "not_found" {
			found = true
			if i.ProductID != "" {
				t.Errorf("expected empty product ID for not_found interaction, got %s", i.ProductID)
			}
			break
		}
	}
	if !found {
		t.Error("expected to find 'batteries' not_found interaction from seed data")
	}
}

// ── Subscriber tests ──────────────────────────────────────────────────────────

func TestStartSubscribers_LogsNavCompleted(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	if err := svc.StartSubscribers(context.Background()); err != nil {
		t.Fatalf("StartSubscribers returned error: %v", err)
	}

	// Publish a nav.goal.completed event
	navBus := setupBus(t)
	result := navigation.NavResult{
		ProductID: "c0000000-0000-0000-0000-000000000004",
		Success:   true,
	}
	data, _ := json.Marshal(result)
	if err := navBus.Publish(context.Background(), pubsub.SubjectNavGoalCompleted, data); err != nil {
		t.Fatalf("failed to publish nav result: %v", err)
	}

	// Give the subscriber a moment to process
	time.Sleep(300 * time.Millisecond)

	recent, err := svc.Recent(context.Background(), 50)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}

	found := false
	for _, i := range recent {
		if i.ProductID == "c0000000-0000-0000-0000-000000000004" && i.Outcome == "navigated" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected interaction logged by subscriber after nav.goal.completed event")
	}
}

func TestStartSubscribers_LogsFailedNavigation(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)

	if err := svc.StartSubscribers(context.Background()); err != nil {
		t.Fatalf("StartSubscribers returned error: %v", err)
	}

	navBus := setupBus(t)
	result := navigation.NavResult{
		ProductID: "c0000000-0000-0000-0000-000000000003",
		Success:   false,
	}
	data, _ := json.Marshal(result)
	if err := navBus.Publish(context.Background(), pubsub.SubjectNavGoalCompleted, data); err != nil {
		t.Fatalf("failed to publish nav result: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	recent, err := svc.Recent(context.Background(), 50)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}

	found := false
	for _, i := range recent {
		if i.ProductID == "c0000000-0000-0000-0000-000000000003" && i.Outcome == "not_found" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected not_found interaction logged after failed navigation event")
	}
}

// ── HTTP handler tests ────────────────────────────────────────────────────────

func TestRecentHandler_ReturnsJSON(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)
	handler := interactions.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/recent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var results []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestRecentHandler_ResultsAreOrdered(t *testing.T) {
	db := setupDB(t)
	bus := setupBus(t)
	svc := interactions.NewService(db, bus)
	handler := interactions.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/recent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var results []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(results) < 2 {
		t.Skip("not enough data to verify ordering")
	}

	for i := 1; i < len(results); i++ {
		prev, _ := time.Parse(time.RFC3339, results[i-1]["created_at"].(string))
		curr, _ := time.Parse(time.RFC3339, results[i]["created_at"].(string))
		if prev.Before(curr) {
			t.Errorf("handler results not in descending order at index %d", i)
		}
	}
}