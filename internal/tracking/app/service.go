package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/websocket"
)

// TrackingService implements tracking use cases
type TrackingService struct {
	repo     ports.LocationRepository
	wsHub    *websocket.Hub
}

// NewTrackingService creates a new tracking service
func NewTrackingService(repo ports.LocationRepository) *TrackingService {
	return &TrackingService{
		repo:  repo,
		wsHub: websocket.NewHub(),
	}
}

// SetWebSocketHub sets the WebSocket hub for broadcasting location updates
func (s *TrackingService) SetWebSocketHub(hub *websocket.Hub) {
	s.wsHub = hub
}

// GetWebSocketHub returns the WebSocket hub
func (s *TrackingService) GetWebSocketHub() *websocket.Hub {
	return s.wsHub
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

	// Broadcast location update to WebSocket clients
	if s.wsHub != nil {
		go s.wsHub.BroadcastLocation(req.DeliveryID, location)
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

// CalculateETAToDestination calculates ETA from current location to destination
func (s *TrackingService) CalculateETAToDestination(ctx context.Context, req ports.CalculateETAToDestinationRequest) (*ports.CalculateETAResponse, error) {
	// Get current location
	currentLocation, err := s.repo.GetLatestByDeliveryID(ctx, req.DeliveryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current location: %w", err)
	}

	// Calculate distance using Haversine formula
	distanceKm := calculateHaversineDistance(
		currentLocation.Latitude, currentLocation.Longitude,
		req.DestLat, req.DestLng,
	)

	// Estimate average speed (can be improved with historical data)
	averageSpeed := 25.0 // km/h - typical urban delivery speed

	// Calculate ETA in hours, then convert to duration
	etaHours := distanceKm / averageSpeed
	eta := time.Duration(etaHours * float64(time.Hour))

	return &ports.CalculateETAResponse{
		ETA:          eta,
		DistanceKm:   distanceKm,
		AverageSpeed: averageSpeed,
	}, nil
}

// calculateHaversineDistance calculates the approximate distance between two points using simplified formula
func calculateHaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Simplified distance calculation (approximate km)
	// 1 degree latitude = 111 km
	// 1 degree longitude = 111 km * cos(latitude)
	latDiff := lat2 - lat1
	lngDiff := lng2 - lng1

	// Average latitude for longitude correction
	avgLat := (lat1 + lat2) / 2

	latKm := latDiff * 111.0
	lngKm := lngDiff * 111.0 * (3.141592653589793 * avgLat / 180) // cos approximation

	return (latKm*latKm + lngKm*lngKm)
}
