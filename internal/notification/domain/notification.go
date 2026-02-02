package domain

import (
	"errors"
	"time"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidNotification  = errors.New("invalid notification data")
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail         NotificationType = "email"
	NotificationTypeSMS           NotificationType = "sms"
	NotificationTypePush          NotificationType = "push"
	NotificationTypeDeliveryUpdate NotificationType = "delivery_update"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

// Notification represents a notification entity
type Notification struct {
	ID         int
	UserID     int
	DeliveryID *int
	Type       NotificationType
	Status     NotificationStatus
	Subject    string
	Message    string
	Recipient  string
	SentAt     *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewNotification creates a new notification with validation
func NewNotification(userID int, notifType NotificationType, subject, message, recipient string) (*Notification, error) {
	if userID <= 0 {
		return nil, ErrInvalidNotification
	}

	if subject == "" || message == "" || recipient == "" {
		return nil, ErrInvalidNotification
	}

	if notifType != NotificationTypeEmail && notifType != NotificationTypeSMS && notifType != NotificationTypePush {
		return nil, ErrInvalidNotification
	}

	return &Notification{
		UserID:    userID,
		Type:      notifType,
		Status:    NotificationStatusPending,
		Subject:   subject,
		Message:   message,
		Recipient: recipient,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// MarkAsSent marks the notification as sent
func (n *Notification) MarkAsSent() {
	n.Status = NotificationStatusSent
	now := time.Now()
	n.SentAt = &now
	n.UpdatedAt = now
}

// MarkAsFailed marks the notification as failed
func (n *Notification) MarkAsFailed() {
	n.Status = NotificationStatusFailed
	n.UpdatedAt = time.Now()
}
