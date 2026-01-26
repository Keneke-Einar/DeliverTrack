package app

import (
	"context"
	"fmt"

	"github.com/keneke/delivertrack/internal/tracking/domain"
	"github.com/keneke/delivertrack/internal/tracking/ports"
)

// TrackingService implements tracking use cases
type TrackingService struct {
	repo ports.LocationRepository
}

// NewTrackingService creates a new tracking service
func NewTrackingService(repo ports.LocationRepository) *TrackingService {
	return &TrackingService{
		repo: repo,
	}
}

// RecordLocation records a new location point
func (s *TrackingService) RecordLocation(
	ctx context.Context,
	deliveryID, courierID int,
	latitude, longitude float64,
	accuracy, speed, heading, altitude *float64,
) (*domain.Location, error) {
	// Create domain entity with validation
	location, err := domain.NewLocation(deliveryID, courierID, latitude, longitude)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	location.SetOptionalFields(accuracy, speed, heading, altitude)

	// Persist to repository
	if err := s.repo.Create(ctx, location); err != nil {
		return nil, fmt.Errorf("failed to record location: %w", err)
	}

	return location, nil
}

// GetDeliveryTrack retrieves the tracking history for a delivery
func (s *TrackingService) GetDeliveryTrack(ctx context.Context, deliveryID int, limit int) ([]*domain.Location, error) {
	if limit <= 0 {
		limit = 100 // default limit
	}

	return s.repo.GetByDeliveryID(ctx, deliveryID, limit)
}

// GetCurrentLocation retrieves the current location for a delivery
func (s *TrackingService) GetCurrentLocation(ctx context.Context, deliveryID int) (*domain.Location, error) {
	return s.repo.GetLatestByDeliveryID(ctx, deliveryID)
}

// GetCourierLocation retrieves the current location for a courier
func (s *TrackingService) GetCourierLocation(ctx context.Context, courierID int) (*domain.Location, error) {
	return s.repo.GetLatestByCourierID(ctx, courierID)
}
