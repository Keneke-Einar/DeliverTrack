package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/pkg/mongodb"
)

// MongoDBLocationRepository implements LocationRepository using MongoDB
type MongoDBLocationRepository struct {
	mongoDB *mongodb.MongoDB
}

// NewMongoDBLocationRepository creates a new MongoDB location repository
func NewMongoDBLocationRepository(mongoDB *mongodb.MongoDB) *MongoDBLocationRepository {
	return &MongoDBLocationRepository{
		mongoDB: mongoDB,
	}
}

// Create stores a new location
func (r *MongoDBLocationRepository) Create(ctx context.Context, location *domain.Location) error {
	courierLocation := &mongodb.CourierLocation{
		CourierID:  int64(location.CourierID),
		DeliveryID: int64(location.DeliveryID),
		Location: mongodb.GeoJSON{
			Type: "Point",
			Coordinates: []interface{}{
				location.Longitude, // MongoDB uses [longitude, latitude]
				location.Latitude,
			},
		},
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	// Set optional fields if provided
	if location.Accuracy != nil {
		courierLocation.Accuracy = *location.Accuracy
	}
	if location.Speed != nil {
		courierLocation.Speed = *location.Speed
	}
	if location.Heading != nil {
		courierLocation.Heading = *location.Heading
	}
	if location.Altitude != nil {
		courierLocation.Altitude = *location.Altitude
	}

	return r.mongoDB.InsertCourierLocation(ctx, courierLocation)
}

// GetByDeliveryID retrieves locations for a delivery
func (r *MongoDBLocationRepository) GetByDeliveryID(ctx context.Context, deliveryID int, limit int) ([]*domain.Location, error) {
	// Note: MongoDB stores courier locations, not delivery locations directly
	// We need to get courier locations and filter by delivery context
	// This is a simplified implementation - in practice, we'd need to join with delivery data

	// For now, we'll get recent courier locations (assuming deliveryID maps to courierID for simplicity)
	// In a real implementation, we'd need to track which courier is assigned to which delivery
	courierLocations, err := r.mongoDB.GetLocationHistoryByDeliveryID(ctx, int64(deliveryID), time.Now().Add(-24*time.Hour), int64(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get location history for delivery: %w", err)
	}

	locations := make([]*domain.Location, len(courierLocations))
	for i, cl := range courierLocations {
		coords := cl.Location.Coordinates.([]interface{})
		longitude := coords[0].(float64)
		latitude := coords[1].(float64)

		location := &domain.Location{
			DeliveryID: deliveryID,
			CourierID:  int(cl.CourierID),
			Latitude:   latitude,
			Longitude:  longitude,
			Timestamp:  cl.Timestamp,
			CreatedAt:  cl.CreatedAt,
			Accuracy:   &cl.Accuracy,
			Speed:      &cl.Speed,
			Heading:    &cl.Heading,
			Altitude:   &cl.Altitude,
		}
		locations[i] = location
	}

	return locations, nil
}

// GetLatestByDeliveryID retrieves the latest location for a delivery
func (r *MongoDBLocationRepository) GetLatestByDeliveryID(ctx context.Context, deliveryID int) (*domain.Location, error) {
	// Similar to GetByDeliveryID, simplified implementation
	courierLocation, err := r.mongoDB.GetLatestLocationByDeliveryID(ctx, int64(deliveryID))
	if err != nil {
		return nil, fmt.Errorf("failed to get latest location for delivery: %w", err)
	}

	coords := courierLocation.Location.Coordinates.([]interface{})
	longitude := coords[0].(float64)
	latitude := coords[1].(float64)

	return &domain.Location{
		DeliveryID: deliveryID,
		CourierID:  int(courierLocation.CourierID),
		Latitude:   latitude,
		Longitude:  longitude,
		Timestamp:  courierLocation.Timestamp,
		CreatedAt:  courierLocation.CreatedAt,
		Accuracy:   &courierLocation.Accuracy,
		Speed:      &courierLocation.Speed,
		Heading:    &courierLocation.Heading,
		Altitude:   &courierLocation.Altitude,
	}, nil
}

// GetByCourierID retrieves locations for a courier
func (r *MongoDBLocationRepository) GetByCourierID(ctx context.Context, courierID int, limit int) ([]*domain.Location, error) {
	courierLocations, err := r.mongoDB.GetCourierLocationHistory(ctx, int64(courierID), time.Now().Add(-24*time.Hour), int64(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get courier location history: %w", err)
	}

	locations := make([]*domain.Location, len(courierLocations))
	for i, cl := range courierLocations {
		coords := cl.Location.Coordinates.([]interface{})
		longitude := coords[0].(float64)
		latitude := coords[1].(float64)

		location := &domain.Location{
			CourierID:  courierID,
			Latitude:   latitude,
			Longitude:  longitude,
			Timestamp:  cl.Timestamp,
			CreatedAt:  cl.CreatedAt,
			Accuracy:   &cl.Accuracy,
			Speed:      &cl.Speed,
			Heading:    &cl.Heading,
			Altitude:   &cl.Altitude,
		}
		locations[i] = location
	}

	return locations, nil
}

// GetLatestByCourierID retrieves the latest location for a courier
func (r *MongoDBLocationRepository) GetLatestByCourierID(ctx context.Context, courierID int) (*domain.Location, error) {
	courierLocation, err := r.mongoDB.GetLatestCourierLocation(ctx, int64(courierID))
	if err != nil {
		return nil, fmt.Errorf("failed to get latest courier location: %w", err)
	}

	coords := courierLocation.Location.Coordinates.([]interface{})
	longitude := coords[0].(float64)
	latitude := coords[1].(float64)

	return &domain.Location{
		CourierID:  int(courierLocation.CourierID),
		Latitude:   latitude,
		Longitude:  longitude,
		Timestamp:  courierLocation.Timestamp,
		CreatedAt:  courierLocation.CreatedAt,
		Accuracy:   &courierLocation.Accuracy,
		Speed:      &courierLocation.Speed,
		Heading:    &courierLocation.Heading,
		Altitude:   &courierLocation.Altitude,
	}, nil
}