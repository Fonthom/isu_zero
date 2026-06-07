package interactions

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/isu-zero/isu-zero/internal/navigation"
	"github.com/isu-zero/isu-zero/internal/pubsub"
)

type Interaction struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	QueryText       string `json:"query_text"`
	Outcome         string `json:"outcome"`
	DurationSeconds int    `json:"duration_seconds"`
	CreatedAt       string `json:"created_at"`
}

type Service struct {
	db  *pgxpool.Pool
	bus *pubsub.Bus
}

func NewService(db *pgxpool.Pool, bus *pubsub.Bus) *Service {
	return &Service{db: db, bus: bus}
}

func (s *Service) Log(ctx context.Context, productID, queryText, outcome string, duration int) error {
	var pid *string
	if productID != "" {
		pid = &productID
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO interactions (product_id, query_text, outcome, duration_seconds)
		VALUES ($1, $2, $3, $4)
	`, pid, queryText, outcome, duration)
	return err
}

func (s *Service) Recent(ctx context.Context, limit int) ([]Interaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, COALESCE(product_id::text, ''), query_text, outcome,
		       COALESCE(duration_seconds, 0), created_at
		FROM interactions
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Interaction
	for rows.Next() {
		var i Interaction
		var createdAt time.Time
		if err := rows.Scan(&i.ID, &i.ProductID, &i.QueryText, &i.Outcome, &i.DurationSeconds, &createdAt); err != nil {
			return nil, err
		}
		i.CreatedAt = createdAt.Format(time.RFC3339)
		results = append(results, i)
	}
	return results, nil
}

// StartSubscribers listens for nav.goal.completed events and logs them.
// Returns an error if the subscription cannot be established.
func (s *Service) StartSubscribers(ctx context.Context) error {
	return s.bus.Subscribe(
		ctx,
		pubsub.SubjectNavGoalCompleted,
		"interactions-nav-completed",
		func(data []byte) {
			var result navigation.NavResult
			if err := json.Unmarshal(data, &result); err != nil {
				log.Printf("failed to parse nav result: %v", err)
				return
			}
			outcome := "navigated"
			if !result.Success {
				outcome = "not_found"
			}
			if err := s.Log(ctx, result.ProductID, "", outcome, 0); err != nil {
				log.Printf("failed to log interaction: %v", err)
			}
		},
	)
}

// HTTP handler

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) http.Handler {
	h := &Handler{svc: svc}
	r := chi.NewRouter()
	r.Get("/recent", h.recent)
	return r
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	interactions, err := h.svc.Recent(r.Context(), 50)
	if err != nil {
		http.Error(w, "failed to fetch interactions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(interactions)
}