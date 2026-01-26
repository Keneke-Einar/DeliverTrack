package ports

import (
	"context"

	"github.com/Keneke-Einar/delivertrack/internal/notification/domain"
)

// NotificationRepository defines the notification persistence operations
type NotificationRepository interface {
	// Create stores a new notification
	Create(ctx context.Context, notification *domain.Notification) error

	// GetByID retrieves a notification by ID
	GetByID(ctx context.Context, id int) (*domain.Notification, error)

	// GetByUserID retrieves all notifications for a user
	GetByUserID(ctx context.Context, userID int, limit int) ([]*domain.Notification, error)

	// Update updates a notification's status
	Update(ctx context.Context, notification *domain.Notification) error

	// Delete deletes a notification
	Delete(ctx context.Context, id int) error
}
