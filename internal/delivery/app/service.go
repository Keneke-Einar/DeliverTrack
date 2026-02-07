package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/pkg/resilience"
	"github.com/Keneke-Einar/delivertrack/proto/delivery"
	"go.uber.org/zap"
)

// DeliveryService implements the delivery use cases
type DeliveryService struct {
	repo           ports.DeliveryRepository
	publisher      messaging.Publisher
	deliveryClient delivery.DeliveryServiceClient
	deliveryCB     *resilience.CircuitBreaker
	logger         *logger.Logger
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(repo ports.DeliveryRepository, publisher messaging.Publisher, deliveryClient delivery.DeliveryServiceClient, logger *logger.Logger) *DeliveryService {
	return &DeliveryService{
		repo:           repo,
		publisher:      publisher,
		deliveryClient: deliveryClient,
		deliveryCB:     resilience.NewCircuitBreaker("delivery", 3, 10*time.Second),
		logger:         logger,
	}
}

// CreateDelivery creates a new delivery
func (s *DeliveryService) CreateDelivery(ctx context.Context, req ports.CreateDeliveryRequest) (*domain.Delivery, error) {
	s.logger.InfoWithFields(ctx, "Creating new delivery",
		zap.Int("customer_id", req.CustomerID),
		zap.String("method", "CreateDelivery"))

	// Create domain entity with validation
	delivery, err := domain.NewDelivery(req.CustomerID, req.PickupLocation, req.DeliveryLocation)
	if err != nil {
		s.logger.ErrorWithFields(ctx, "Failed to create delivery domain entity",
			zap.Int("customer_id", req.CustomerID),
			zap.Error(err))
		return nil, err
	}

	// Set optional fields
	delivery.CourierID = req.CourierID
	delivery.Notes = req.Notes

	if req.ScheduledDate != nil && *req.ScheduledDate != "" {
		parsedDate, err := time.Parse(time.RFC3339, *req.ScheduledDate)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %w", err)
		}
		delivery.ScheduledDate = &parsedDate
	}

	// Persist to repository
	if err := s.repo.Create(ctx, delivery); err != nil {
		s.logger.ErrorWithFields(ctx, "Failed to persist delivery",
			zap.Int("customer_id", req.CustomerID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create delivery: %w", err)
	}

	s.logger.InfoWithFields(ctx, "Delivery created successfully",
		zap.Int("delivery_id", delivery.ID),
		zap.Int("customer_id", req.CustomerID),
		zap.String("status", string(delivery.Status)))

	// Publish delivery created event
	traceCtx := messaging.ExtractTraceContextFromContext(ctx, "delivery-service", "create_delivery")
	event := messaging.NewEventWithTrace("delivery.created", "delivery-service", "create_delivery", map[string]interface{}{
		"delivery_id":       fmt.Sprintf("%d", delivery.ID),
		"customer_id":       delivery.CustomerID,
		"courier_id":        delivery.CourierID,
		"pickup_location":   delivery.PickupLocation,
		"delivery_location": delivery.DeliveryLocation,
		"status":           delivery.Status,
		"scheduled_date":   delivery.ScheduledDate,
		"notes":            delivery.Notes,
	}, traceCtx)

	// Publish event asynchronously with retry
	go func() {
		err := resilience.Retry(ctx, resilience.DefaultRetryConfig(), func() error {
			return s.publisher.Publish(ctx, "delivery-events", "delivery.created", event)
		})
		if err != nil {
			fmt.Printf("Failed to publish delivery created event: %v\n", err)
		}
	}()

	return delivery, nil
}

// GetDelivery retrieves a delivery by ID with authorization
func (s *DeliveryService) GetDelivery(ctx context.Context, req ports.GetDeliveryRequest) (*domain.Delivery, error) {
	delivery, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if !delivery.CanBeModifiedBy(req.Role, req.UserCustomerID, req.UserCourierID) {
		return nil, domain.ErrUnauthorized
	}

	return delivery, nil
}

// ListDeliveries lists deliveries with optional filters and authorization
func (s *DeliveryService) ListDeliveries(ctx context.Context, req ports.ListDeliveriesRequest) ([]*domain.Delivery, error) {
	// Apply authorization filters
	filterCustomerID := req.CustomerID
	if req.Role == "customer" && req.UserCustomerID != nil {
		filterCustomerID = *req.UserCustomerID
	}

	var deliveries []*domain.Delivery
	var err error

	if req.Status != "" {
		deliveries, err = s.repo.GetByStatus(ctx, req.Status, filterCustomerID)
	} else {
		deliveries, err = s.repo.GetAll(ctx, filterCustomerID)
	}

	if err != nil {
		return nil, err
	}

	// Filter results based on authorization
	if req.Role == "courier" && req.UserCourierID != nil {
		filtered := make([]*domain.Delivery, 0)
		for _, d := range deliveries {
			if d.CourierID != nil && *d.CourierID == *req.UserCourierID {
				filtered = append(filtered, d)
			}
		}
		return filtered, nil
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates a delivery status with authorization
func (s *DeliveryService) UpdateDeliveryStatus(ctx context.Context, req ports.UpdateDeliveryStatusRequest) error {
	// Get delivery to check authorization
	delivery, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}

	// Check authorization
	if !delivery.CanBeModifiedBy(req.Role, req.UserCustomerID, req.UserCourierID) {
		return domain.ErrUnauthorized
	}

	// If a courier is updating status to "assigned", assign them to the delivery
	if req.Role == "courier" && req.UserCourierID != nil && req.Status == "assigned" && delivery.CourierID == nil {
		if err := delivery.AssignCourier(*req.UserCourierID); err != nil {
			return err
		}
		// Update the repository with courier assignment
		if err := s.repo.AssignCourier(ctx, req.ID, *req.UserCourierID); err != nil {
			return err
		}
	} else {
		// Validate and update status in domain entity
		if err := delivery.UpdateStatus(req.Status); err != nil {
			return err
		}
	}

	// Persist the status update
	if err := s.repo.UpdateStatus(ctx, req.ID, req.Status, req.Notes); err != nil {
		return err
	}

	// Publish delivery status changed event
	traceCtx := messaging.ExtractTraceContextFromContext(ctx, "delivery-service", "update_delivery_status")
	event := messaging.NewEventWithTrace("delivery.status_changed", "delivery-service", "update_delivery_status", map[string]interface{}{
		"delivery_id":     fmt.Sprintf("%d", req.ID),
		"customer_id":     delivery.CustomerID,
		"courier_id":      delivery.CourierID,
		"old_status":      delivery.Status,
		"new_status":      req.Status,
		"notes":          req.Notes,
		"updated_by_role": req.Role,
	}, traceCtx)

	// Publish event asynchronously with retry
	go func() {
		err := resilience.Retry(ctx, resilience.DefaultRetryConfig(), func() error {
			return s.publisher.Publish(ctx, "delivery-events", "delivery.status_changed", event)
		})
		if err != nil {
			fmt.Printf("Failed to publish delivery status changed event: %v\n", err)
		}
	}()

	return nil
}
