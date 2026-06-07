package navigation

import (
	"context"
	"encoding/json"
	"log"

	"github.com/isu-zero/isu-zero/internal/pubsub"
)

type NavGoal struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	NavX        float64 `json:"nav_x"`
	NavY        float64 `json:"nav_y"`
}

type NavResult struct {
	ProductID string `json:"product_id"`
	Success   bool   `json:"success"`
}

type PhotoCaptured struct {
	WaypointID string `json:"waypoint_id"`
	FilePath   string `json:"file_path"`
}

type Service struct {
	bus *pubsub.Bus
}

func NewService(bus *pubsub.Bus) *Service {
	return &Service{bus: bus}
}

func (s *Service) RequestNavigation(ctx context.Context, productID, productName string, x, y float64) error {
	goal := NavGoal{
		ProductID:   productID,
		ProductName: productName,
		NavX:        x,
		NavY:        y,
	}
	data, err := json.Marshal(goal)
	if err != nil {
		return err
	}
	log.Printf("publishing nav goal for product %s (%.2f, %.2f)", productName, x, y)
	return s.bus.Publish(ctx, pubsub.SubjectNavGoalRequested, data)
}

func (s *Service) PublishPhotoCaptured(ctx context.Context, waypointID, filePath string) error {
	event := PhotoCaptured{
		WaypointID: waypointID,
		FilePath:   filePath,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Printf("publishing photo captured for waypoint %s at %s", waypointID, filePath)
	return s.bus.Publish(ctx, pubsub.SubjectPhotoCaptured, data)
}