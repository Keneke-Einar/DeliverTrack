package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Keneke-Einar/delivertrack/internal/notification/domain"
	"github.com/Keneke-Einar/delivertrack/internal/notification/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"go.uber.org/zap"
)

// NotificationService implements notification use cases
type NotificationService struct {
	repo     ports.NotificationRepository
	consumer messaging.Consumer
	logger   *logger.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(repo ports.NotificationRepository, consumer messaging.Consumer, logger *logger.Logger) *NotificationService {
	return &NotificationService{
		repo:     repo,
		consumer: consumer,
		logger:   logger,
	}
}

// SendNotification sends a notification to a user
func (s *NotificationService) SendNotification(
	ctx context.Context,
	userID int,
	notifType domain.NotificationType,
	subject, message, recipient string,
) (*domain.Notification, error) {
	s.logger.InfoWithFields(ctx, "Sending notification",
		zap.Int("user_id", userID),
		zap.String("type", string(notifType)),
		zap.String("recipient", recipient))

	// Create domain entity with validation
	notification, err := domain.NewNotification(userID, notifType, subject, message, recipient)
	if err != nil {
		s.logger.ErrorWithFields(ctx, "Failed to create notification domain entity",
			zap.Int("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	// Persist to repository
	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	// TODO: Integrate with actual notification service (email, SMS, push)
	// For now, just mark as sent
	notification.MarkAsSent()
	if err := s.repo.Update(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to update notification status: %w", err)
	}

	return notification, nil
}

// GetNotificationByID retrieves a notification by ID
func (s *NotificationService) GetNotificationByID(ctx context.Context, id int) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// GetUserNotifications retrieves all notifications for a user
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID int, limit int) ([]*domain.Notification, error) {
	if limit <= 0 {
		limit = 50 // default limit
	}

	return s.repo.GetByUserID(ctx, userID, limit)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id int) error {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	notification.MarkAsSent()
	return s.repo.Update(ctx, notification)
}

// StartEventConsumption starts consuming delivery and location events
func (s *NotificationService) StartEventConsumption() error {
	return s.consumer.Consume("notification-events", s.handleEvent)
}

// handleEvent processes incoming events
func (s *NotificationService) handleEvent(event messaging.Event) error {
	ctx := context.Background()

	switch event.Type {
	case "delivery.created":
		return s.handleDeliveryCreated(ctx, event)
	case "delivery.status_changed":
		return s.handleDeliveryStatusChanged(ctx, event)
	case "location.updated":
		return s.handleLocationUpdated(ctx, event)
	default:
		// Ignore unknown event types
		return nil
	}
}

// handleDeliveryCreated processes delivery creation events
func (s *NotificationService) handleDeliveryCreated(ctx context.Context, event messaging.Event) error {
	customerIDStr, ok := event.Data["customer_id"].(string)
	if !ok {
		return fmt.Errorf("invalid customer_id in event data")
	}

	customerID, err := strconv.Atoi(customerIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse customer_id: %w", err)
	}

	deliveryIDStr, ok := event.Data["delivery_id"].(string)
	if !ok {
		return fmt.Errorf("invalid delivery_id in event data")
	}

	// Send notification to customer about delivery creation
	_, err = s.SendNotification(
		ctx,
		customerID,
		domain.NotificationTypeDeliveryUpdate,
		"Delivery Created",
		fmt.Sprintf("Your delivery %s has been created and is being processed.", deliveryIDStr),
		fmt.Sprintf("customer_%d", customerID),
	)
	if err != nil {
		return fmt.Errorf("failed to send delivery created notification: %w", err)
	}

	return nil
}

// handleDeliveryStatusChanged processes delivery status change events
func (s *NotificationService) handleDeliveryStatusChanged(ctx context.Context, event messaging.Event) error {
	customerIDStr, ok := event.Data["customer_id"].(string)
	if !ok {
		return fmt.Errorf("invalid customer_id in event data")
	}

	customerID, err := strconv.Atoi(customerIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse customer_id: %w", err)
	}

	deliveryIDStr, ok := event.Data["delivery_id"].(string)
	if !ok {
		return fmt.Errorf("invalid delivery_id in event data")
	}

	newStatus, ok := event.Data["new_status"].(string)
	if !ok {
		return fmt.Errorf("invalid new_status in event data")
	}

	// Send notification to customer about status change
	_, err = s.SendNotification(
		ctx,
		customerID,
		domain.NotificationTypeDeliveryUpdate,
		"Delivery Status Update",
		fmt.Sprintf("Your delivery %s status has been updated to: %s", deliveryIDStr, newStatus),
		fmt.Sprintf("customer_%d", customerID),
	)
	if err != nil {
		return fmt.Errorf("failed to send delivery status notification: %w", err)
	}

	return nil
}

// handleLocationUpdated processes location update events
func (s *NotificationService) handleLocationUpdated(ctx context.Context, event messaging.Event) error {
	// For location updates, we could send notifications to customers
	// For now, we'll skip this to avoid spam, but the infrastructure is in place
	// TODO: Implement location-based notifications (e.g., "Courier is nearby")

	return nil
}
