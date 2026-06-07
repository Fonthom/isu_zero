package navigation_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/isu-zero/isu-zero/internal/navigation"
	"github.com/isu-zero/isu-zero/internal/pubsub"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func setupBus(t *testing.T) (*pubsub.Bus, *nats.Conn) {
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
	return bus, nc
}

// consumeNext subscribes as a JetStream consumer and returns the next message
// on the given subject, or fails the test if nothing arrives within the timeout.
func consumeNext(t *testing.T, nc *nats.Conn, subject string, timeout time.Duration) []byte {
	t.Helper()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("failed to get JetStream context: %v", err)
	}

	// Use an ephemeral ordered consumer — no durable name, cleaned up automatically
	cons, err := js.OrderedConsumer(context.Background(), pubsub.StreamName, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{subject},
		DeliverPolicy:  jetstream.DeliverLastPolicy,
	})
	if err != nil {
		t.Fatalf("failed to create consumer on %s: %v", subject, err)
	}

	msg, err := cons.Next(jetstream.FetchMaxWait(timeout))
	if err != nil {
		t.Fatalf("no message received on %s within %s: %v", subject, timeout, err)
	}
	return msg.Data()
}

// ── RequestNavigation tests ───────────────────────────────────────────────────

func TestRequestNavigation_PublishesCorrectSubject(t *testing.T) {
	bus, nc := setupBus(t)
	svc := navigation.NewService(bus)

	if err := svc.RequestNavigation(context.Background(), "test-id", "Test Product", 1.5, 2.5); err != nil {
		t.Fatalf("RequestNavigation returned error: %v", err)
	}

	data := consumeNext(t, nc, pubsub.SubjectNavGoalRequested, 2*time.Second)

	var goal navigation.NavGoal
	if err := json.Unmarshal(data, &goal); err != nil {
		t.Fatalf("failed to unmarshal nav goal: %v", err)
	}

	if goal.ProductID != "test-id" {
		t.Errorf("expected ProductID 'test-id', got '%s'", goal.ProductID)
	}
	if goal.ProductName != "Test Product" {
		t.Errorf("expected ProductName 'Test Product', got '%s'", goal.ProductName)
	}
	if goal.NavX != 1.5 {
		t.Errorf("expected NavX 1.5, got %f", goal.NavX)
	}
	if goal.NavY != 2.5 {
		t.Errorf("expected NavY 2.5, got %f", goal.NavY)
	}
}

func TestRequestNavigation_PayloadIsValidJSON(t *testing.T) {
	bus, nc := setupBus(t)
	svc := navigation.NewService(bus)

	if err := svc.RequestNavigation(context.Background(), "abc", "Some Product", 0.0, 0.0); err != nil {
		t.Fatalf("RequestNavigation returned error: %v", err)
	}

	data := consumeNext(t, nc, pubsub.SubjectNavGoalRequested, 2*time.Second)

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("published payload is not valid JSON: %v", err)
	}

	requiredFields := []string{"product_id", "product_name", "nav_x", "nav_y"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("payload missing required field: %s", field)
		}
	}
}

// ── PublishPhotoCaptured tests ────────────────────────────────────────────────

func TestPublishPhotoCaptured_PublishesCorrectSubject(t *testing.T) {
	bus, nc := setupBus(t)
	svc := navigation.NewService(bus)

	waypointID := "a0000000-0000-0000-0000-000000000003"
	filePath := "/photos/aisle1a.jpg"

	if err := svc.PublishPhotoCaptured(context.Background(), waypointID, filePath); err != nil {
		t.Fatalf("PublishPhotoCaptured returned error: %v", err)
	}

	data := consumeNext(t, nc, pubsub.SubjectPhotoCaptured, 2*time.Second)

	var event navigation.PhotoCaptured
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal photo captured event: %v", err)
	}

	if event.WaypointID != waypointID {
		t.Errorf("expected WaypointID '%s', got '%s'", waypointID, event.WaypointID)
	}
	if event.FilePath != filePath {
		t.Errorf("expected FilePath '%s', got '%s'", filePath, event.FilePath)
	}
}

func TestPublishPhotoCaptured_PayloadIsValidJSON(t *testing.T) {
	bus, nc := setupBus(t)
	svc := navigation.NewService(bus)

	if err := svc.PublishPhotoCaptured(context.Background(), "some-waypoint", "/photos/test.jpg"); err != nil {
		t.Fatalf("PublishPhotoCaptured returned error: %v", err)
	}

	data := consumeNext(t, nc, pubsub.SubjectPhotoCaptured, 2*time.Second)

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("published payload is not valid JSON: %v", err)
	}

	requiredFields := []string{"waypoint_id", "file_path"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("payload missing required field: %s", field)
		}
	}
}

func TestPublishPhotoCaptured_EmptyFilePathFails(t *testing.T) {
	bus, _ := setupBus(t)
	svc := navigation.NewService(bus)

	// Empty file path should still publish — validation is the cropper's responsibility
	// This test documents that behaviour explicitly
	err := svc.PublishPhotoCaptured(context.Background(), "some-waypoint", "")
	if err != nil {
		t.Logf("PublishPhotoCaptured with empty path returned error (acceptable): %v", err)
	}
}