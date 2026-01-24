package models

import (
	"time"
)

type Package struct {
	ID               string    `json:"id" db:"id"`
	TrackingNumber   string    `json:"tracking_number" db:"tracking_number"`
	SenderName       string    `json:"sender_name" db:"sender_name"`
	SenderAddress    string    `json:"sender_address" db:"sender_address"`
	RecipientName    string    `json:"recipient_name" db:"recipient_name"`
	RecipientAddress string    `json:"recipient_address" db:"recipient_address"`
	Status           string    `json:"status" db:"status"`
	CurrentLocation  string    `json:"current_location" db:"current_location"`
	Weight           float64   `json:"weight" db:"weight"`
	Description      string    `json:"description" db:"description"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type PackageStatus string

const (
	StatusPending        PackageStatus = "pending"
	StatusInTransit      PackageStatus = "in_transit"
	StatusOutForDelivery PackageStatus = "out_for_delivery"
	StatusDelivered      PackageStatus = "delivered"
	StatusFailed         PackageStatus = "failed"
	StatusCancelled      PackageStatus = "cancelled"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"`
	Name      string    `json:"name" db:"name"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Delivery struct {
	ID          string     `json:"id" db:"id"`
	PackageID   string     `json:"package_id" db:"package_id"`
	DriverID    string     `json:"driver_id" db:"driver_id"`
	Status      string     `json:"status" db:"status"`
	ScheduledAt time.Time  `json:"scheduled_at" db:"scheduled_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	Notes       string     `json:"notes" db:"notes"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type Location struct {
	PackageID string    `json:"package_id" bson:"package_id"`
	Latitude  float64   `json:"latitude" bson:"latitude"`
	Longitude float64   `json:"longitude" bson:"longitude"`
	Address   string    `json:"address" bson:"address"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}
