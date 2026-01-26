package ports

import (
	"context"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
)

// RecordLocationRequest for recording a location
type RecordLocationRequest struct {
	DeliveryID int      `json:"delivery_id"`
	CourierID  int      `json:"courier_id"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Accuracy   *float64 `json:"accuracy,omitempty"`
	Speed      *float64 `json:"speed,omitempty"`
	Heading    *float64 `json:"heading,omitempty"`
	Altitude   *float64 `json:"altitude,omitempty"`
}

// GetDeliveryTrackRequest for retrieving delivery track
type GetDeliveryTrackRequest struct {
	DeliveryID int `json:"delivery_id"`
	Limit      int `json:"limit,omitempty"`
}

// GetCurrentLocationRequest for retrieving current location
type GetCurrentLocationRequest struct {
	DeliveryID int `json:"delivery_id"`
}

// GetCourierLocationRequest for retrieving courier location
type GetCourierLocationRequest struct {
	CourierID int `json:"courier_id"`
}

// CalculateETAToDestinationRequest for calculating ETA to destination
type CalculateETAToDestinationRequest struct {
	DeliveryID  int     `json:"delivery_id"`
	DestLat     float64 `json:"dest_lat"`
	DestLng     float64 `json:"dest_lng"`
}

// CalculateETAResponse for ETA calculation response
type CalculateETAResponse struct {
	ETA         time.Duration `json:"eta"`
	DistanceKm  float64       `json:"distance_km"`
	AverageSpeed float64      `json:"average_speed_kmh"`
}

// TrackingService defines the interface for tracking business operations
type TrackingService interface {
	// RecordLocation records a new location point
	RecordLocation(ctx context.Context, req RecordLocationRequest) (*domain.Location, error)

	// GetDeliveryTrack retrieves the tracking history for a delivery
	GetDeliveryTrack(ctx context.Context, req GetDeliveryTrackRequest) ([]*domain.Location, error)

	// GetCurrentLocation retrieves the current location for a delivery
	GetCurrentLocation(ctx context.Context, req GetCurrentLocationRequest) (*domain.Location, error)

	// GetCourierLocation retrieves the current location for a courier
	GetCourierLocation(ctx context.Context, req GetCourierLocationRequest) (*domain.Location, error)

	// CalculateETAToDestination calculates ETA from current location to destination
	CalculateETAToDestination(ctx context.Context, req CalculateETAToDestinationRequest) (*CalculateETAResponse, error)
}
