package adapters

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/pkg/mongodb"
)

func TestMongoDBLocationRepository_Integration(t *testing.T) {
	// Skip if no MongoDB URL is provided
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		t.Skip("MONGO_URL not set, skipping integration test")
	}

	// Connect to MongoDB
	mongoClient, err := mongodb.New(mongoURL)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(context.Background())

	// Create repository
	repo := NewMongoDBLocationRepository(mongoClient)

	ctx := context.Background()

	// Create a test location
	location, err := domain.NewLocation(1, 1, 40.7128, -74.0060)
	if err != nil {
		t.Fatalf("Failed to create location: %v", err)
	}

	accuracy := 10.5
	speed := 25.0
	heading := 90.0
	altitude := 100.0
	location.SetOptionalFields(&accuracy, &speed, &heading, &altitude)

	// Test Create
	err = repo.Create(ctx, location)
	if err != nil {
		t.Fatalf("Failed to create location: %v", err)
	}

	// Test GetLatestByDeliveryID
	latest, err := repo.GetLatestByDeliveryID(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get latest location: %v", err)
	}

	if latest.DeliveryID != 1 {
		t.Errorf("Expected delivery ID 1, got %d", latest.DeliveryID)
	}

	if latest.CourierID != 1 {
		t.Errorf("Expected courier ID 1, got %d", latest.CourierID)
	}

	if latest.Latitude != 40.7128 {
		t.Errorf("Expected latitude 40.7128, got %f", latest.Latitude)
	}

	if latest.Longitude != -74.0060 {
		t.Errorf("Expected longitude -74.0060, got %f", latest.Longitude)
	}

	// Test GetByDeliveryID
	locations, err := repo.GetByDeliveryID(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Failed to get locations by delivery ID: %v", err)
	}

	if len(locations) != 1 {
		t.Errorf("Expected 1 location, got %d", len(locations))
	}

	// Test GetLatestByCourierID
	courierLatest, err := repo.GetLatestByCourierID(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get latest courier location: %v", err)
	}

	if courierLatest.CourierID != 1 {
		t.Errorf("Expected courier ID 1, got %d", courierLatest.CourierID)
	}

	// Test GetByCourierID
	courierLocations, err := repo.GetByCourierID(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Failed to get locations by courier ID: %v", err)
	}

	if len(courierLocations) != 1 {
		t.Errorf("Expected 1 courier location, got %d", len(courierLocations))
	}

	// Test multiple locations
	location2, err := domain.NewLocation(1, 1, 40.7589, -73.9851)
	if err != nil {
		t.Fatalf("Failed to create second location: %v", err)
	}

	// Add a small delay to ensure different timestamps
	time.Sleep(1 * time.Millisecond)
	err = repo.Create(ctx, location2)
	if err != nil {
		t.Fatalf("Failed to create second location: %v", err)
	}

	// Latest should now be the second location
	latest2, err := repo.GetLatestByDeliveryID(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get latest location after second insert: %v", err)
	}

	if latest2.Latitude != 40.7589 {
		t.Errorf("Expected latest latitude 40.7589, got %f", latest2.Latitude)
	}

	// Get all locations
	allLocations, err := repo.GetByDeliveryID(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Failed to get all locations: %v", err)
	}

	if len(allLocations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(allLocations))
	}

	// Test limit
	limitedLocations, err := repo.GetByDeliveryID(ctx, 1, 1)
	if err != nil {
		t.Fatalf("Failed to get limited locations: %v", err)
	}

	if len(limitedLocations) != 1 {
		t.Errorf("Expected 1 limited location, got %d", len(limitedLocations))
	}

	// Test non-existent delivery
	nonExistent, err := repo.GetLatestByDeliveryID(ctx, 999)
	if err == nil {
		t.Error("Expected error for non-existent delivery, got none")
	}

	if nonExistent != nil {
		t.Errorf("Expected nil for non-existent delivery, got %v", nonExistent)
	}
}

func TestMongoDBLocationRepository_GetByDeliveryID_Limit(t *testing.T) {
	// Skip if no MongoDB URL is provided
	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		t.Skip("MONGO_URL not set, skipping integration test")
	}

	// Connect to MongoDB
	mongoClient, err := mongodb.New(mongoURL)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(context.Background())

	repo := NewMongoDBLocationRepository(mongoClient)
	ctx := context.Background()

	// Create multiple locations for the same delivery
	for i := 0; i < 5; i++ {
		location, err := domain.NewLocation(2, 1, 40.7128+float64(i), -74.0060+float64(i))
		if err != nil {
			t.Fatalf("Failed to create location %d: %v", i, err)
		}

		// Add delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
		err = repo.Create(ctx, location)
		if err != nil {
			t.Fatalf("Failed to create location %d: %v", i, err)
		}
	}

	// Test limit
	limited, err := repo.GetByDeliveryID(ctx, 2, 3)
	if err != nil {
		t.Fatalf("Failed to get limited locations: %v", err)
	}

	if len(limited) != 3 {
		t.Errorf("Expected 3 locations with limit, got %d", len(limited))
	}

	// Test unlimited (should return all)
	all, err := repo.GetByDeliveryID(ctx, 2, 100)
	if err != nil {
		t.Fatalf("Failed to get all locations: %v", err)
	}

	if len(all) != 5 {
		t.Errorf("Expected 5 locations without limit, got %d", len(all))
	}
}