package app

import (
	"context"
	"fmt"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
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
func (s *TrackingService) RecordLocation(ctx context.Context, req ports.RecordLocationRequest) (*domain.Location, error) {
	// Create domain entity with validation
	location, err := domain.NewLocation(req.DeliveryID, req.CourierID, req.Latitude, req.Longitude)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	location.SetOptionalFields(req.Accuracy, req.Speed, req.Heading, req.Altitude)

	// Persist to repository
	if err := s.repo.Create(ctx, location); err != nil {
		return nil, fmt.Errorf("failed to record location: %w", err)
	}

	return location, nil
}

// GetDeliveryTrack retrieves the tracking history for a delivery
func (s *TrackingService) GetDeliveryTrack(ctx context.Context, req ports.GetDeliveryTrackRequest) ([]*domain.Location, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // default limit
	}

	return s.repo.GetByDeliveryID(ctx, req.DeliveryID, limit)
}

// GetCurrentLocation retrieves the current location for a delivery
func (s *TrackingService) GetCurrentLocation(ctx context.Context, req ports.GetCurrentLocationRequest) (*domain.Location, error) {
	return s.repo.GetLatestByDeliveryID(ctx, req.DeliveryID)
}

// GetCourierLocation retrieves the current location for a courier
func (s *TrackingService) GetCourierLocation(ctx context.Context, req ports.GetCourierLocationRequest) (*domain.Location, error) {
	return s.repo.GetLatestByCourierID(ctx, req.CourierID)
}
