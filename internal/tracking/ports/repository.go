package ports

import (
	"context"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
)

// LocationRepository defines the interface for location data persistence
type LocationRepository interface {
	// Create stores a new location
	Create(ctx context.Context, location *domain.Location) error

	// GetByDeliveryID retrieves locations for a delivery
	GetByDeliveryID(ctx context.Context, deliveryID int, limit int) ([]*domain.Location, error)

	// GetLatestByDeliveryID retrieves the latest location for a delivery
	GetLatestByDeliveryID(ctx context.Context, deliveryID int) (*domain.Location, error)

	// GetByCourierID retrieves locations for a courier
	GetByCourierID(ctx context.Context, courierID int, limit int) ([]*domain.Location, error)

	// GetLatestByCourierID retrieves the latest location for a courier
	GetLatestByCourierID(ctx context.Context, courierID int) (*domain.Location, error)
}
