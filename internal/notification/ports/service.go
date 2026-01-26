package ports

import (
	"context"

	"github.com/keneke/delivertrack/internal/notification/domain"
)

// NotificationService defines the notification use cases
type NotificationService interface {
	// SendNotification sends a notification to a user
	SendNotification(ctx context.Context, userID int, notifType domain.NotificationType, subject, message, recipient string) (*domain.Notification, error)

	// GetNotificationByID retrieves a notification by ID
	GetNotificationByID(ctx context.Context, id int) (*domain.Notification, error)

	// GetUserNotifications retrieves all notifications for a user
	GetUserNotifications(ctx context.Context, userID int, limit int) ([]*domain.Notification, error)

	// MarkAsRead marks a notification as read
	MarkAsRead(ctx context.Context, id int) error
}
