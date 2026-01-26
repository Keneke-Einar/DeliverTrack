package ports

import (
	"context"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
)

// TrackingService defines the interface for tracking business operations
type TrackingService interface {
	// RecordLocation records a new location point
	RecordLocation(ctx context.Context, deliveryID, courierID int, latitude, longitude float64, accuracy, speed, heading, altitude *float64) (*domain.Location, error)

	// GetDeliveryTrack retrieves the tracking history for a delivery
	GetDeliveryTrack(ctx context.Context, deliveryID int, limit int) ([]*domain.Location, error)

	// GetCurrentLocation retrieves the current location for a delivery
	GetCurrentLocation(ctx context.Context, deliveryID int) (*domain.Location, error)

	// GetCourierLocation retrieves the current location for a courier
	GetCourierLocation(ctx context.Context, courierID int) (*domain.Location, error)
}
