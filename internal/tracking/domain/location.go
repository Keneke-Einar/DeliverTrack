package domain

import (
	"errors"
	"time"
)

var (
	ErrLocationNotFound    = errors.New("location not found")
	ErrInvalidLocation     = errors.New("invalid location data")
	ErrUnauthorized        = errors.New("unauthorized access")
)

// Location represents a tracking location point
type Location struct {
	ID          int
	DeliveryID  int
	CourierID   int
	Latitude    float64
	Longitude   float64
	Accuracy    *float64
	Speed       *float64
	Heading     *float64
	Altitude    *float64
	Timestamp   time.Time
	CreatedAt   time.Time
}

// NewLocation creates a new location with validation
func NewLocation(deliveryID, courierID int, latitude, longitude float64) (*Location, error) {
	if deliveryID <= 0 || courierID <= 0 {
		return nil, ErrInvalidLocation
	}

	if latitude < -90 || latitude > 90 {
		return nil, ErrInvalidLocation
	}

	if longitude < -180 || longitude > 180 {
		return nil, ErrInvalidLocation
	}

	return &Location{
		DeliveryID: deliveryID,
		CourierID:  courierID,
		Latitude:   latitude,
		Longitude:  longitude,
		Timestamp:  time.Now(),
		CreatedAt:  time.Now(),
	}, nil
}

// SetOptionalFields sets optional location fields
func (l *Location) SetOptionalFields(accuracy, speed, heading, altitude *float64) {
	l.Accuracy = accuracy
	l.Speed = speed
	l.Heading = heading
	l.Altitude = altitude
}

// IsValid checks if the location data is valid
func (l *Location) IsValid() bool {
	return l.DeliveryID > 0 &&
		l.CourierID > 0 &&
		l.Latitude >= -90 && l.Latitude <= 90 &&
		l.Longitude >= -180 && l.Longitude <= 180
}
