package adapters

import (
	"context"
	"database/sql"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/notification/domain"
)

// PostgresNotificationRepository implements the NotificationRepository interface using PostgreSQL
type PostgresNotificationRepository struct {
	db *sql.DB
}

// NewPostgresNotificationRepository creates a new PostgreSQL repository
func NewPostgresNotificationRepository(db *sql.DB) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{db: db}
}

// Create stores a new notification
func (r *PostgresNotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	query := `
		INSERT INTO notifications (user_id, delivery_id, type, status, subject, message, recipient, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var deliveryID sql.NullInt64
	if notification.DeliveryID != nil {
		deliveryID = sql.NullInt64{Int64: int64(*notification.DeliveryID), Valid: true}
	}

	var sentAt sql.NullTime
	if notification.SentAt != nil {
		sentAt = sql.NullTime{Time: *notification.SentAt, Valid: true}
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		notification.UserID,
		deliveryID,
		notification.Type,
		notification.Status,
		notification.Subject,
		notification.Message,
		notification.Recipient,
		sentAt,
		notification.CreatedAt,
		notification.UpdatedAt,
	).Scan(&notification.ID)

	return err
}

// GetByID retrieves a notification by ID
func (r *PostgresNotificationRepository) GetByID(ctx context.Context, id int) (*domain.Notification, error) {
	query := `
		SELECT id, user_id, delivery_id, type, status, subject, message, recipient, sent_at, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`

	var notification domain.Notification
	var deliveryID sql.NullInt64
	var sentAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID,
		&notification.UserID,
		&deliveryID,
		&notification.Type,
		&notification.Status,
		&notification.Subject,
		&notification.Message,
		&notification.Recipient,
		&sentAt,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if deliveryID.Valid {
		dID := int(deliveryID.Int64)
		notification.DeliveryID = &dID
	}

	if sentAt.Valid {
		notification.SentAt = &sentAt.Time
	}

	return &notification, nil
}

// GetByUserID retrieves notifications for a user
func (r *PostgresNotificationRepository) GetByUserID(ctx context.Context, userID int, limit int) ([]*domain.Notification, error) {
	query := `
		SELECT id, user_id, delivery_id, type, status, subject, message, recipient, sent_at, created_at, updated_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		var notification domain.Notification
		var deliveryID sql.NullInt64
		var sentAt sql.NullTime

		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&deliveryID,
			&notification.Type,
			&notification.Status,
			&notification.Subject,
			&notification.Message,
			&notification.Recipient,
			&sentAt,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if deliveryID.Valid {
			dID := int(deliveryID.Int64)
			notification.DeliveryID = &dID
		}

		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}

		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// UpdateStatus updates the status of a notification
func (r *PostgresNotificationRepository) UpdateStatus(ctx context.Context, id int, status domain.NotificationStatus) error {
	query := `
		UPDATE notifications
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

// Update updates a notification
func (r *PostgresNotificationRepository) Update(ctx context.Context, notification *domain.Notification) error {
	query := `
		UPDATE notifications
		SET user_id = $1, delivery_id = $2, type = $3, status = $4, subject = $5, message = $6, recipient = $7, sent_at = $8, updated_at = $9
		WHERE id = $10
	`

	var deliveryID sql.NullInt64
	if notification.DeliveryID != nil {
		deliveryID = sql.NullInt64{Int64: int64(*notification.DeliveryID), Valid: true}
	}

	var sentAt sql.NullTime
	if notification.SentAt != nil {
		sentAt = sql.NullTime{Time: *notification.SentAt, Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		notification.UserID,
		deliveryID,
		notification.Type,
		notification.Status,
		notification.Subject,
		notification.Message,
		notification.Recipient,
		sentAt,
		notification.UpdatedAt,
		notification.ID,
	)

	return err
}

// Delete deletes a notification
func (r *PostgresNotificationRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM notifications WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}