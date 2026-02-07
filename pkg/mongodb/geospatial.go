package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertCourierLocation inserts a new courier location record
func (m *MongoDB) InsertCourierLocation(ctx context.Context, location *CourierLocation) error {
	location.CreatedAt = time.Now()
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}

	_, err := m.CourierLocationsCollection().InsertOne(ctx, location)
	if err != nil {
		return fmt.Errorf("failed to insert courier location: %w", err)
	}
	return nil
}

// GetLatestCourierLocation returns the most recent location for a courier
func (m *MongoDB) GetLatestCourierLocation(ctx context.Context, courierID int64) (*CourierLocation, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	
	var location CourierLocation
	err := m.CourierLocationsCollection().FindOne(
		ctx,
		bson.M{"courier_id": courierID},
		opts,
	).Decode(&location)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get latest courier location: %w", err)
	}
	
	return &location, nil
}

// GetLatestLocationByDeliveryID returns the most recent location for a delivery
func (m *MongoDB) GetLatestLocationByDeliveryID(ctx context.Context, deliveryID int64) (*CourierLocation, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	
	var location CourierLocation
	err := m.CourierLocationsCollection().FindOne(
		ctx,
		bson.M{"delivery_id": deliveryID},
		opts,
	).Decode(&location)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get latest location for delivery: %w", err)
	}
	
	return &location, nil
}

// GetCourierLocationHistory returns location history for a courier within a time range
func (m *MongoDB) GetCourierLocationHistory(ctx context.Context, courierID int64, since time.Time, limit int64) ([]CourierLocation, error) {
	filter := bson.M{
		"courier_id": courierID,
		"timestamp":  bson.M{"$gte": since},
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(limit)
	
	cursor, err := m.CourierLocationsCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get courier location history: %w", err)
	}
	defer cursor.Close(ctx)
	
	var locations []CourierLocation
	if err := cursor.All(ctx, &locations); err != nil {
		return nil, fmt.Errorf("failed to decode courier locations: %w", err)
	}
	
	return locations, nil
}

// GetLocationHistoryByDeliveryID returns location history for a delivery within a time range
func (m *MongoDB) GetLocationHistoryByDeliveryID(ctx context.Context, deliveryID int64, since time.Time, limit int64) ([]CourierLocation, error) {
	filter := bson.M{
		"delivery_id": deliveryID,
		"timestamp":   bson.M{"$gte": since},
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(limit)
	
	cursor, err := m.CourierLocationsCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get location history for delivery: %w", err)
	}
	defer cursor.Close(ctx)
	
	var locations []CourierLocation
	if err := cursor.All(ctx, &locations); err != nil {
		return nil, fmt.Errorf("failed to decode locations for delivery: %w", err)
	}
	
	return locations, nil
}

// FindCouriersNearPoint finds couriers within a specified radius (in meters) of a point
func (m *MongoDB) FindCouriersNearPoint(ctx context.Context, longitude, latitude float64, radiusMeters float64, limit int64) ([]CourierLocation, error) {
	// Use $geoNear aggregation for finding nearby couriers
	pipeline := []bson.M{
		{
			"$geoNear": bson.M{
				"near": bson.M{
					"type":        "Point",
					"coordinates": []float64{longitude, latitude},
				},
				"distanceField": "distance",
				"maxDistance":   radiusMeters,
				"spherical":     true,
			},
		},
		{
			"$sort": bson.M{"timestamp": -1},
		},
		{
			"$group": bson.M{
				"_id": "$courier_id",
				"location": bson.M{"$first": "$$ROOT"},
			},
		},
		{
			"$replaceRoot": bson.M{"newRoot": "$location"},
		},
		{
			"$limit": limit,
		},
	}
	
	cursor, err := m.CourierLocationsCollection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to find couriers near point: %w", err)
	}
	defer cursor.Close(ctx)
	
	var locations []CourierLocation
	if err := cursor.All(ctx, &locations); err != nil {
		return nil, fmt.Errorf("failed to decode nearby couriers: %w", err)
	}
	
	return locations, nil
}

// InsertDeliveryZone inserts a new delivery zone
func (m *MongoDB) InsertDeliveryZone(ctx context.Context, zone *DeliveryZone) error {
	now := time.Now()
	zone.CreatedAt = now
	zone.UpdatedAt = now
	
	_, err := m.DeliveryZonesCollection().InsertOne(ctx, zone)
	if err != nil {
		return fmt.Errorf("failed to insert delivery zone: %w", err)
	}
	return nil
}

// GetDeliveryZone returns a delivery zone by name
func (m *MongoDB) GetDeliveryZone(ctx context.Context, name string) (*DeliveryZone, error) {
	var zone DeliveryZone
	err := m.DeliveryZonesCollection().FindOne(
		ctx,
		bson.M{"name": name},
	).Decode(&zone)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery zone: %w", err)
	}
	
	return &zone, nil
}

// GetActiveDeliveryZones returns all active delivery zones
func (m *MongoDB) GetActiveDeliveryZones(ctx context.Context) ([]DeliveryZone, error) {
	cursor, err := m.DeliveryZonesCollection().Find(
		ctx,
		bson.M{"active": true},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get active delivery zones: %w", err)
	}
	defer cursor.Close(ctx)
	
	var zones []DeliveryZone
	if err := cursor.All(ctx, &zones); err != nil {
		return nil, fmt.Errorf("failed to decode delivery zones: %w", err)
	}
	
	return zones, nil
}

// FindZonesContainingPoint finds all delivery zones that contain a given point
func (m *MongoDB) FindZonesContainingPoint(ctx context.Context, longitude, latitude float64) ([]DeliveryZone, error) {
	filter := bson.M{
		"active": true,
		"geometry": bson.M{
			"$geoIntersects": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{longitude, latitude},
				},
			},
		},
	}
	
	cursor, err := m.DeliveryZonesCollection().Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find zones containing point: %w", err)
	}
	defer cursor.Close(ctx)
	
	var zones []DeliveryZone
	if err := cursor.All(ctx, &zones); err != nil {
		return nil, fmt.Errorf("failed to decode zones: %w", err)
	}
	
	return zones, nil
}

// IsPointInZone checks if a point is within a specific delivery zone
func (m *MongoDB) IsPointInZone(ctx context.Context, zoneName string, longitude, latitude float64) (bool, error) {
	filter := bson.M{
		"name":   zoneName,
		"active": true,
		"geometry": bson.M{
			"$geoIntersects": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{longitude, latitude},
				},
			},
		},
	}
	
	count, err := m.DeliveryZonesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check if point is in zone: %w", err)
	}
	
	return count > 0, nil
}

// UpdateDeliveryZone updates a delivery zone
func (m *MongoDB) UpdateDeliveryZone(ctx context.Context, name string, update bson.M) error {
	update["updated_at"] = time.Now()
	
	result, err := m.DeliveryZonesCollection().UpdateOne(
		ctx,
		bson.M{"name": name},
		bson.M{"$set": update},
	)
	
	if err != nil {
		return fmt.Errorf("failed to update delivery zone: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("delivery zone not found: %s", name)
	}
	
	return nil
}

// DeleteOldCourierLocations removes location records older than the specified duration
func (m *MongoDB) DeleteOldCourierLocations(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)
	
	result, err := m.CourierLocationsCollection().DeleteMany(
		ctx,
		bson.M{"timestamp": bson.M{"$lt": cutoffTime}},
	)
	
	if err != nil {
		return 0, fmt.Errorf("failed to delete old courier locations: %w", err)
	}
	
	return result.DeletedCount, nil
}

// GetCourierLocationCount returns the total number of location records for a courier
func (m *MongoDB) GetCourierLocationCount(ctx context.Context, courierID int64) (int64, error) {
	count, err := m.CourierLocationsCollection().CountDocuments(
		ctx,
		bson.M{"courier_id": courierID},
	)
	
	if err != nil {
		return 0, fmt.Errorf("failed to count courier locations: %w", err)
	}
	
	return count, nil
}
