package common

import (
	"database/sql"
	"time"
)

type Courier struct {
	ID              int            `json:"id"`
	Name            string         `json:"name"`
	VehicleType     string         `json:"vehicle_type"`
	CurrentLocation sql.NullString `json:"current_location"`
	Phone           string         `json:"phone"`
	Status          string         `json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type Customer struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Address   string         `json:"address"`
	Contact   string         `json:"contact"`
	Email     sql.NullString `json:"email"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CreateCourier inserts a new courier
func CreateCourier(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, c *Courier) error {
	query := `
		INSERT INTO couriers (name, vehicle_type, phone, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(query, c.Name, c.VehicleType, c.Phone, c.Status).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// CreateCustomer inserts a new customer
func CreateCustomer(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, c *Customer) error {
	query := `
		INSERT INTO customers (name, address, contact, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(query, c.Name, c.Address, c.Contact, c.Email).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetCourierByID retrieves a courier by ID
func GetCourierByID(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, id int) (*Courier, error) {
	var c Courier
	query := `SELECT id, name, vehicle_type, current_location, phone, status, created_at, updated_at FROM couriers WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&c.ID, &c.Name, &c.VehicleType, &c.CurrentLocation, &c.Phone, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCustomerByID retrieves a customer by ID
func GetCustomerByID(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, id int) (*Customer, error) {
	var c Customer
	query := `SELECT id, name, address, contact, email, created_at, updated_at FROM customers WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&c.ID, &c.Name, &c.Address, &c.Contact, &c.Email, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
