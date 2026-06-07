package products

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/isu-zero/isu-zero/internal/navigation"
)

type Product struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	PhotoID    string  `json:"photo_id"`
	WaypointID string  `json:"waypoint_id"`
	CropPath   string  `json:"crop_path"`
	CropX      int     `json:"crop_x"`
	CropY      int     `json:"crop_y"`
	CropWidth  int     `json:"crop_width"`
	CropHeight int     `json:"crop_height"`
	NavX       float64 `json:"nav_x"`
	NavY       float64 `json:"nav_y"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Search(ctx context.Context, query string) ([]Product, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			p.id, COALESCE(p.name, ''), p.photo_id, p.waypoint_id,
			p.crop_path, p.crop_x, p.crop_y, p.crop_width, p.crop_height,
			w.nav_x, w.nav_y
		FROM products p
		JOIN waypoints w ON w.id = p.waypoint_id
		WHERE p.name IS NOT NULL
		  AND (
		    to_tsvector('english', p.name) @@ plainto_tsquery('english', $1)
		    OR p.name ILIKE '%' || $1 || '%'
		  )
		ORDER BY p.name
		LIMIT 20
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.PhotoID, &p.WaypointID,
			&p.CropPath, &p.CropX, &p.CropY, &p.CropWidth, &p.CropHeight,
			&p.NavX, &p.NavY,
		); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Product, error) {
	var p Product
	err := s.db.QueryRow(ctx, `
		SELECT
			p.id, COALESCE(p.name, ''), p.photo_id, p.waypoint_id,
			p.crop_path, p.crop_x, p.crop_y, p.crop_width, p.crop_height,
			w.nav_x, w.nav_y
		FROM products p
		JOIN waypoints w ON w.id = p.waypoint_id
		WHERE p.id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.PhotoID, &p.WaypointID,
		&p.CropPath, &p.CropX, &p.CropY, &p.CropWidth, &p.CropHeight,
		&p.NavX, &p.NavY,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// HTTP handler

type Handler struct {
	svc    *Service
	navSvc *navigation.Service
}

func NewHandler(svc *Service, navSvc *navigation.Service) http.Handler {
	h := &Handler{svc: svc, navSvc: navSvc}
	r := chi.NewRouter()
	r.Get("/search", h.search)
	r.Post("/{id}/navigate", h.navigate)
	return r
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing query param q", http.StatusBadRequest)
		return
	}
	results, err := h.svc.Search(r.Context(), q)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) navigate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	if err := h.navSvc.RequestNavigation(r.Context(), product.ID, product.Name, product.NavX, product.NavY); err != nil {
		http.Error(w, "navigation request failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"outcome": "navigating"})
}