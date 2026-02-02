package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
	"github.com/Keneke-Einar/delivertrack/proto/analytics"
	"github.com/Keneke-Einar/delivertrack/proto/notification"
)

// DeliveryService implements the delivery use cases
type DeliveryService struct {
	repo               ports.DeliveryRepository
	notificationClient notification.NotificationServiceClient
	analyticsClient    analytics.AnalyticsServiceClient
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(repo ports.DeliveryRepository, notificationClient notification.NotificationServiceClient, analyticsClient analytics.AnalyticsServiceClient) *DeliveryService {
	return &DeliveryService{
		repo:               repo,
		notificationClient: notificationClient,
		analyticsClient:    analyticsClient,
	}
}

// CreateDelivery creates a new delivery
func (s *DeliveryService) CreateDelivery(ctx context.Context, req ports.CreateDeliveryRequest) (*domain.Delivery, error) {
	// Create domain entity with validation
	delivery, err := domain.NewDelivery(req.CustomerID, req.PickupLocation, req.DeliveryLocation)
	if err != nil {
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
		return nil, fmt.Errorf("failed to create delivery: %w", err)
	}

	// Record analytics event asynchronously
	go func() {
		analyticsReq := &analytics.RecordEventRequest{
			EventId:    fmt.Sprintf("delivery_created_%d_%d", delivery.ID, time.Now().Unix()),
			EventType:  "delivery.created",
			EntityType: "delivery",
			EntityId:   fmt.Sprintf("%d", delivery.ID),
			Properties: map[string]string{
				"customer_id":       fmt.Sprintf("%d", delivery.CustomerID),
				"courier_id":        fmt.Sprintf("%d", delivery.CourierID),
				"pickup_location":   delivery.PickupLocation,
				"delivery_location": delivery.DeliveryLocation,
			},
			Timestamp: time.Now().Unix(),
		}
		if _, err := s.analyticsClient.RecordEvent(context.Background(), analyticsReq); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to record analytics event: %v\n", err)
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

	// Validate and update status in domain entity
	if err := delivery.UpdateStatus(req.Status); err != nil {
		return err
	}

	// Persist the update
	if err := s.repo.UpdateStatus(ctx, req.ID, req.Status, req.Notes); err != nil {
		return err
	}

	// Send notification asynchronously
	go func() {
		notificationReq := &notification.SendDeliveryUpdateRequest{
			DeliveryId:     fmt.Sprintf("%d", req.ID),
			CustomerId:     fmt.Sprintf("%d", delivery.CustomerID),
			TrackingNumber: fmt.Sprintf("%d", req.ID), // Using ID as tracking number for now
			Status:         req.Status,
			Message:        fmt.Sprintf("Delivery status updated to %s", req.Status),
		}
		if _, err := s.notificationClient.SendDeliveryUpdate(context.Background(), notificationReq); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to send delivery update notification: %v\n", err)
		}
	}()

	// Record analytics event asynchronously
	go func() {
		analyticsReq := &analytics.RecordEventRequest{
			EventId:    fmt.Sprintf("delivery_status_update_%d_%d", req.ID, time.Now().Unix()),
			EventType:  "delivery.status_changed",
			EntityType: "delivery",
			EntityId:   fmt.Sprintf("%d", req.ID),
			Properties: map[string]string{
				"old_status":  delivery.Status,
				"new_status":  req.Status,
				"customer_id": fmt.Sprintf("%d", delivery.CustomerID),
				"courier_id":  fmt.Sprintf("%d", delivery.CourierID),
			},
			Timestamp: time.Now().Unix(),
		}
		if _, err := s.analyticsClient.RecordEvent(context.Background(), analyticsReq); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to record analytics event: %v\n", err)
		}
	}()

	return nil
}
