package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
)

// DeliveryService implements the delivery use cases
type DeliveryService struct {
	repo ports.DeliveryRepository
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(repo ports.DeliveryRepository) *DeliveryService {
	return &DeliveryService{
		repo: repo,
	}
}

// CreateDelivery creates a new delivery
func (s *DeliveryService) CreateDelivery(
	ctx context.Context,
	customerID int,
	courierID *int,
	pickupLocation, deliveryLocation, notes string,
	scheduledDate *string,
) (*domain.Delivery, error) {
	// Create domain entity with validation
	delivery, err := domain.NewDelivery(customerID, pickupLocation, deliveryLocation)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	delivery.CourierID = courierID
	delivery.Notes = notes

	if scheduledDate != nil && *scheduledDate != "" {
		parsedDate, err := time.Parse(time.RFC3339, *scheduledDate)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %w", err)
		}
		delivery.ScheduledDate = &parsedDate
	}

	// Persist to repository
	if err := s.repo.Create(ctx, delivery); err != nil {
		return nil, fmt.Errorf("failed to create delivery: %w", err)
	}

	return delivery, nil
}

// GetDelivery retrieves a delivery by ID with authorization
func (s *DeliveryService) GetDelivery(
	ctx context.Context,
	id int,
	role string,
	customerID, courierID *int,
) (*domain.Delivery, error) {
	delivery, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if !delivery.CanBeModifiedBy(role, customerID, courierID) {
		return nil, domain.ErrUnauthorized
	}

	return delivery, nil
}

// ListDeliveries lists deliveries with optional filters and authorization
func (s *DeliveryService) ListDeliveries(
	ctx context.Context,
	status string,
	customerID int,
	role string,
	userCustomerID, userCourierID *int,
) ([]*domain.Delivery, error) {
	// Apply authorization filters
	filterCustomerID := customerID
	if role == "customer" && userCustomerID != nil {
		filterCustomerID = *userCustomerID
	}

	var deliveries []*domain.Delivery
	var err error

	if status != "" {
		deliveries, err = s.repo.GetByStatus(ctx, status, filterCustomerID)
	} else {
		deliveries, err = s.repo.GetAll(ctx, filterCustomerID)
	}

	if err != nil {
		return nil, err
	}

	// Filter results based on authorization
	if role == "courier" && userCourierID != nil {
		filtered := make([]*domain.Delivery, 0)
		for _, d := range deliveries {
			if d.CourierID != nil && *d.CourierID == *userCourierID {
				filtered = append(filtered, d)
			}
		}
		return filtered, nil
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates a delivery status with authorization
func (s *DeliveryService) UpdateDeliveryStatus(
	ctx context.Context,
	id int,
	status, notes string,
	role string,
	customerID, courierID *int,
) error {
	// Get delivery to check authorization
	delivery, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check authorization
	if !delivery.CanBeModifiedBy(role, customerID, courierID) {
		return domain.ErrUnauthorized
	}

	// Validate and update status in domain entity
	if err := delivery.UpdateStatus(status); err != nil {
		return err
	}

	// Persist the update
	return s.repo.UpdateStatus(ctx, id, status, notes)
}
