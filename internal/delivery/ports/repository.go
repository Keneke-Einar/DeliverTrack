package ports

import (
	"context"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
)

// DeliveryRepository defines the interface for delivery data persistence
type DeliveryRepository interface {
	// Create stores a new delivery
	Create(ctx context.Context, delivery *domain.Delivery) error

	// GetByID retrieves a delivery by its ID
	GetByID(ctx context.Context, id int) (*domain.Delivery, error)

	// GetByStatus retrieves deliveries by status with optional customer filter
	GetByStatus(ctx context.Context, status string, customerID int) ([]*domain.Delivery, error)

	// GetAll retrieves all deliveries with optional customer filter
	GetAll(ctx context.Context, customerID int) ([]*domain.Delivery, error)

	// UpdateStatus updates the status of a delivery
	UpdateStatus(ctx context.Context, id int, status, notes string) error

	// AssignCourier assigns a courier to a delivery
	AssignCourier(ctx context.Context, deliveryID, courierID int) error

	// Update updates a delivery
	Update(ctx context.Context, delivery *domain.Delivery) error
}
