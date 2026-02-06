package app

import (
	"context"
	"fmt"	
	"strconv"	
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/pkg/resilience"
	"github.com/Keneke-Einar/delivertrack/pkg/websocket"
	"github.com/Keneke-Einar/delivertrack/proto/delivery"
	"go.uber.org/zap"
)

// TrackingService implements tracking use cases
type TrackingService struct {
	repo           ports.LocationRepository
	wsHub          *websocket.Hub
	publisher      messaging.Publisher
	deliveryClient delivery.DeliveryServiceClient
	deliveryCB     *resilience.CircuitBreaker
	logger         *logger.Logger
}

// NewTrackingService creates a new tracking service
func NewTrackingService(repo ports.LocationRepository, publisher messaging.Publisher, deliveryClient delivery.DeliveryServiceClient, authService authPorts.AuthService, logger *logger.Logger) *TrackingService {
	return &TrackingService{
		repo:           repo,
		wsHub:          websocket.NewHub(authService),
		publisher:      publisher,
		deliveryClient: deliveryClient,
		deliveryCB:     resilience.NewCircuitBreaker("delivery", 3, 10*time.Second),
		logger:         logger,
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
	s.logger.InfoWithFields(ctx, "Recording location update",
		zap.Int("delivery_id", req.DeliveryID),
		zap.Int("courier_id", req.CourierID),
		zap.Float64("latitude", req.Latitude),
		zap.Float64("longitude", req.Longitude))

	// Create domain entity with validation
	location, err := domain.NewLocation(req.DeliveryID, req.CourierID, req.Latitude, req.Longitude)
	if err != nil {
		s.logger.ErrorWithFields(ctx, "Failed to create location domain entity",
			zap.Int("delivery_id", req.DeliveryID),
			zap.Error(err))
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

	// Update delivery ETA asynchronously
	go func() {
		// Get delivery details to calculate ETA
		deliveryReq := &delivery.GetDeliveryRequest{
			DeliveryId: fmt.Sprintf("%d", req.DeliveryID),
		}
		deliveryResp, err := s.deliveryClient.GetDelivery(context.Background(), deliveryReq)
		if err != nil {
			fmt.Printf("Failed to get delivery for ETA calculation: %v\n", err)
			return
		}

		// Send customer notification about location update
		if s.wsHub != nil && deliveryResp.Delivery.CustomerId != "" {
			customerID, err := strconv.Atoi(deliveryResp.Delivery.CustomerId)
			if err == nil {
				go s.wsHub.BroadcastCustomerNotification(customerID, "location_update", 
					fmt.Sprintf("Your delivery #%d location has been updated", req.DeliveryID),
					map[string]interface{}{
						"delivery_id": req.DeliveryID,
						"latitude":    location.Latitude,
						"longitude":   location.Longitude,
					})
			}
		}

		// Calculate ETA to delivery location
		destLat := deliveryResp.Delivery.DeliveryLocation.Latitude
		destLng := deliveryResp.Delivery.DeliveryLocation.Longitude
		
		// Simple ETA calculation (can be improved)
		distanceKm := calculateHaversineDistance(
			location.Latitude, location.Longitude,
			destLat, destLng,
		)
		averageSpeed := 25.0 // km/h
		etaHours := distanceKm / averageSpeed
		etaSeconds := int64(etaHours * 3600)
		// etaTimestamp := time.Now().Unix() + etaSeconds

		// For now, just log the ETA. In a real implementation, you might update the delivery record
		fmt.Printf("Calculated ETA for delivery %d: %d seconds (%f km at %f km/h)\n", 
			req.DeliveryID, etaSeconds, distanceKm, averageSpeed)
	}()

	// Send location update notification asynchronously via event publishing
	go func() {
		traceCtx := messaging.ExtractTraceContextFromContext(ctx, "tracking-service", "record_location")
		event := messaging.NewEventWithTrace("location.updated", "tracking-service", "record_location", map[string]interface{}{
			"delivery_id": fmt.Sprintf("%d", req.DeliveryID),
			"courier_id":  fmt.Sprintf("%d", req.CourierID),
			"latitude":    location.Latitude,
			"longitude":   location.Longitude,
			"accuracy":    location.Accuracy,
			"speed":       location.Speed,
			"heading":     location.Heading,
			"altitude":    location.Altitude,
		}, traceCtx)

		// Publish event asynchronously with retry
		err := resilience.Retry(ctx, resilience.DefaultRetryConfig(), func() error {
			return s.publisher.Publish(ctx, "tracking-events", "location.updated", event)
		})
		if err != nil {
			fmt.Printf("Failed to publish location update event: %v\n", err)
		}
	}()

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
