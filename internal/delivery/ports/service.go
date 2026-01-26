package ports

import (
	"context"

	"github.com/keneke/delivertrack/internal/delivery/domain"
)

// DeliveryService defines the interface for delivery business operations
type DeliveryService interface {
	// CreateDelivery creates a new delivery
	CreateDelivery(ctx context.Context, customerID int, courierID *int, pickupLocation, deliveryLocation, notes string, scheduledDate *string) (*domain.Delivery, error)

	// GetDelivery retrieves a delivery by ID
	GetDelivery(ctx context.Context, id int, role string, customerID, courierID *int) (*domain.Delivery, error)

	// ListDeliveries lists deliveries with optional filters
	ListDeliveries(ctx context.Context, status string, customerID int, role string, userCustomerID, userCourierID *int) ([]*domain.Delivery, error)

	// UpdateDeliveryStatus updates a delivery status
	UpdateDeliveryStatus(ctx context.Context, id int, status, notes string, role string, customerID, courierID *int) error
}
