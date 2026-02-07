package ports

import (
	"context"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
)

// AuthContext holds common authorization fields
type AuthContext struct {
	Role          string `json:"role"`
	UserCustomerID *int  `json:"user_customer_id,omitempty"`
	UserCourierID  *int  `json:"user_courier_id,omitempty"`
}

// CreateDeliveryRequest for creating a new delivery
type CreateDeliveryRequest struct {
	CustomerID       int     `json:"customer_id"`
	CourierID        *int    `json:"courier_id,omitempty"`
	PickupLocation   string  `json:"pickup_location"`   // Can be coordinates "(lng,lat)" or address
	DeliveryLocation string  `json:"delivery_location"` // Can be coordinates "(lng,lat)" or address
	Notes            string  `json:"notes,omitempty"`
	ScheduledDate    *string `json:"scheduled_date,omitempty"`
}

// GetDeliveryRequest for retrieving a delivery
type GetDeliveryRequest struct {
	ID int `json:"id"`
	AuthContext // Embedded for auth
}

// ListDeliveriesRequest for listing deliveries
type ListDeliveriesRequest struct {
	Status     string `json:"status,omitempty"`
	CustomerID int    `json:"customer_id"`
	AuthContext // Embedded for auth
}

// UpdateDeliveryStatusRequest for updating status
type UpdateDeliveryStatusRequest struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Notes  string `json:"notes,omitempty"`
	AuthContext // Embedded for auth
}

// DeliveryService defines the interface for delivery business operations
type DeliveryService interface {
	// CreateDelivery creates a new delivery
	CreateDelivery(ctx context.Context, req CreateDeliveryRequest) (*domain.Delivery, error)

	// GetDelivery retrieves a delivery by ID
	GetDelivery(ctx context.Context, req GetDeliveryRequest) (*domain.Delivery, error)

	// ListDeliveries lists deliveries with optional filters
	ListDeliveries(ctx context.Context, req ListDeliveriesRequest) ([]*domain.Delivery, error)

	// UpdateDeliveryStatus updates a delivery status
	UpdateDeliveryStatus(ctx context.Context, req UpdateDeliveryStatusRequest) error
}
