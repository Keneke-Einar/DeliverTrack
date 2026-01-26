package adapters

import (
	"context"
	"database/sql"

	"github.com/keneke/delivertrack/internal/delivery/domain"
)

// PostgresDeliveryRepository implements the DeliveryRepository interface using PostgreSQL
type PostgresDeliveryRepository struct {
	db *sql.DB
}

// NewPostgresDeliveryRepository creates a new PostgreSQL repository
func NewPostgresDeliveryRepository(db *sql.DB) *PostgresDeliveryRepository {
	return &PostgresDeliveryRepository{db: db}
}

// Create stores a new delivery
func (r *PostgresDeliveryRepository) Create(ctx context.Context, delivery *domain.Delivery) error {
	query := `
		INSERT INTO deliveries (customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	var courierID sql.NullInt64
	if delivery.CourierID != nil {
		courierID = sql.NullInt64{Int64: int64(*delivery.CourierID), Valid: true}
	}

	var scheduledDate sql.NullTime
	if delivery.ScheduledDate != nil {
		scheduledDate = sql.NullTime{Time: *delivery.ScheduledDate, Valid: true}
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		delivery.CustomerID,
		courierID,
		delivery.Status,
		delivery.PickupLocation,
		delivery.DeliveryLocation,
		scheduledDate,
		delivery.Notes,
	).Scan(&delivery.ID, &delivery.CreatedAt, &delivery.UpdatedAt)

	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a delivery by its ID
func (r *PostgresDeliveryRepository) GetByID(ctx context.Context, id int) (*domain.Delivery, error) {
	query := `
		SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, 
		       scheduled_date, delivered_date, notes, created_at, updated_at 
		FROM deliveries 
		WHERE id = $1
	`

	var d domain.Delivery
	var courierID sql.NullInt64
	var pickupLocation, deliveryLocation, notes sql.NullString
	var scheduledDate, deliveredDate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.CustomerID,
		&courierID,
		&d.Status,
		&pickupLocation,
		&deliveryLocation,
		&scheduledDate,
		&deliveredDate,
		&notes,
		&d.CreatedAt,
		&d.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrDeliveryNotFound
	}
	if err != nil {
		return nil, err
	}

	// Convert SQL types to domain types
	if courierID.Valid {
		cid := int(courierID.Int64)
		d.CourierID = &cid
	}
	if pickupLocation.Valid {
		d.PickupLocation = pickupLocation.String
	}
	if deliveryLocation.Valid {
		d.DeliveryLocation = deliveryLocation.String
	}
	if scheduledDate.Valid {
		d.ScheduledDate = &scheduledDate.Time
	}
	if deliveredDate.Valid {
		d.DeliveredDate = &deliveredDate.Time
	}
	if notes.Valid {
		d.Notes = notes.String
	}

	return &d, nil
}

// GetByStatus retrieves deliveries by status with optional customer filter
func (r *PostgresDeliveryRepository) GetByStatus(ctx context.Context, status string, customerID int) ([]*domain.Delivery, error) {
	var query string
	var rows *sql.Rows
	var err error

	if customerID > 0 {
		query = `
			SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, 
			       scheduled_date, delivered_date, notes, created_at, updated_at 
			FROM deliveries 
			WHERE status = $1 AND customer_id = $2 
			ORDER BY created_at DESC
		`
		rows, err = r.db.QueryContext(ctx, query, status, customerID)
	} else {
		query = `
			SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, 
			       scheduled_date, delivered_date, notes, created_at, updated_at 
			FROM deliveries 
			WHERE status = $1 
			ORDER BY created_at DESC
		`
		rows, err = r.db.QueryContext(ctx, query, status)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanDeliveries(rows)
}

// GetAll retrieves all deliveries with optional customer filter
func (r *PostgresDeliveryRepository) GetAll(ctx context.Context, customerID int) ([]*domain.Delivery, error) {
	var query string
	var rows *sql.Rows
	var err error

	if customerID > 0 {
		query = `
			SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, 
			       scheduled_date, delivered_date, notes, created_at, updated_at 
			FROM deliveries 
			WHERE customer_id = $1 
			ORDER BY created_at DESC
		`
		rows, err = r.db.QueryContext(ctx, query, customerID)
	} else {
		query = `
			SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, 
			       scheduled_date, delivered_date, notes, created_at, updated_at 
			FROM deliveries 
			ORDER BY created_at DESC
		`
		rows, err = r.db.QueryContext(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanDeliveries(rows)
}

// UpdateStatus updates the status of a delivery
func (r *PostgresDeliveryRepository) UpdateStatus(ctx context.Context, id int, status, notes string) error {
	query := `
		UPDATE deliveries 
		SET status = $1, notes = COALESCE(NULLIF($2, ''), notes), updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id
	`

	var returnedID int
	err := r.db.QueryRowContext(ctx, query, status, notes, id).Scan(&returnedID)
	if err == sql.ErrNoRows {
		return domain.ErrDeliveryNotFound
	}
	return err
}

// Update updates a delivery
func (r *PostgresDeliveryRepository) Update(ctx context.Context, delivery *domain.Delivery) error {
	query := `
		UPDATE deliveries 
		SET customer_id = $1, courier_id = $2, status = $3, 
		    pickup_location = $4, delivery_location = $5, 
		    scheduled_date = $6, delivered_date = $7, notes = $8, 
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $9
		RETURNING updated_at
	`

	var courierID sql.NullInt64
	if delivery.CourierID != nil {
		courierID = sql.NullInt64{Int64: int64(*delivery.CourierID), Valid: true}
	}

	var scheduledDate sql.NullTime
	if delivery.ScheduledDate != nil {
		scheduledDate = sql.NullTime{Time: *delivery.ScheduledDate, Valid: true}
	}

	var deliveredDate sql.NullTime
	if delivery.DeliveredDate != nil {
		deliveredDate = sql.NullTime{Time: *delivery.DeliveredDate, Valid: true}
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		delivery.CustomerID,
		courierID,
		delivery.Status,
		delivery.PickupLocation,
		delivery.DeliveryLocation,
		scheduledDate,
		deliveredDate,
		delivery.Notes,
		delivery.ID,
	).Scan(&delivery.UpdatedAt)

	if err == sql.ErrNoRows {
		return domain.ErrDeliveryNotFound
	}
	return err
}

// scanDeliveries is a helper to scan multiple delivery rows
func (r *PostgresDeliveryRepository) scanDeliveries(rows *sql.Rows) ([]*domain.Delivery, error) {
	var deliveries []*domain.Delivery

	for rows.Next() {
		var d domain.Delivery
		var courierID sql.NullInt64
		var pickupLocation, deliveryLocation, notes sql.NullString
		var scheduledDate, deliveredDate sql.NullTime

		err := rows.Scan(
			&d.ID,
			&d.CustomerID,
			&courierID,
			&d.Status,
			&pickupLocation,
			&deliveryLocation,
			&scheduledDate,
			&deliveredDate,
			&notes,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convert SQL types to domain types
		if courierID.Valid {
			cid := int(courierID.Int64)
			d.CourierID = &cid
		}
		if pickupLocation.Valid {
			d.PickupLocation = pickupLocation.String
		}
		if deliveryLocation.Valid {
			d.DeliveryLocation = deliveryLocation.String
		}
		if scheduledDate.Valid {
			d.ScheduledDate = &scheduledDate.Time
		}
		if deliveredDate.Valid {
			d.DeliveredDate = &deliveredDate.Time
		}
		if notes.Valid {
			d.Notes = notes.String
		}

		deliveries = append(deliveries, &d)
	}

	return deliveries, rows.Err()
}
