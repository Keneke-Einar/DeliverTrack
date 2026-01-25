package delivery

import (
	"database/sql"
	"time"
)

type Delivery struct {
	ID               int            `json:"id"`
	CustomerID       int            `json:"customer_id"`
	CourierID        sql.NullInt64  `json:"courier_id"`
	Status           string         `json:"status"`
	PickupLocation   sql.NullString `json:"pickup_location"`
	DeliveryLocation sql.NullString `json:"delivery_location"`
	ScheduledDate    sql.NullTime   `json:"scheduled_date"`
	DeliveredDate    sql.NullTime   `json:"delivered_date"`
	Notes            sql.NullString `json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// Create inserts a new delivery record
func (d *Delivery) Create(db interface {
	QueryRow(string, ...interface{}) *sql.Row
},
) error {
	query := `
		INSERT INTO deliveries (customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(query, d.CustomerID, d.CourierID, d.Status, d.PickupLocation, d.DeliveryLocation, d.ScheduledDate, d.Notes).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

// GetByID retrieves a delivery by ID
func GetByID(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, id int,
) (*Delivery, error) {
	var d Delivery
	query := `SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, delivered_date, notes, created_at, updated_at FROM deliveries WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&d.ID, &d.CustomerID, &d.CourierID, &d.Status, &d.PickupLocation, &d.DeliveryLocation, &d.ScheduledDate, &d.DeliveredDate, &d.Notes, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
