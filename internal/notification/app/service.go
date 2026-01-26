package app

import (
	"context"
	"fmt"

	"github.com/keneke/delivertrack/internal/notification/domain"
	"github.com/keneke/delivertrack/internal/notification/ports"
)

// NotificationService implements notification use cases
type NotificationService struct {
	repo ports.NotificationRepository
}

// NewNotificationService creates a new notification service
func NewNotificationService(repo ports.NotificationRepository) *NotificationService {
	return &NotificationService{
		repo: repo,
	}
}

// SendNotification sends a notification to a user
func (s *NotificationService) SendNotification(
	ctx context.Context,
	userID int,
	notifType domain.NotificationType,
	subject, message, recipient string,
) (*domain.Notification, error) {
	// Create domain entity with validation
	notification, err := domain.NewNotification(userID, notifType, subject, message, recipient)
	if err != nil {
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
