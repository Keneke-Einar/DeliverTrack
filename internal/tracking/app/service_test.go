package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
)

// MockLocationRepository is a mock implementation of LocationRepository for testing
type MockLocationRepository struct {
	locations map[int][]*domain.Location
	nextID    int
}

func NewMockLocationRepository() *MockLocationRepository {
	return &MockLocationRepository{
		locations: make(map[int][]*domain.Location),
		nextID:    1,
	}
}

func (m *MockLocationRepository) Create(ctx context.Context, location *domain.Location) error {
	location.ID = m.nextID
	m.nextID++

	// Store by delivery ID
	if m.locations[location.DeliveryID] == nil {
		m.locations[location.DeliveryID] = make([]*domain.Location, 0)
	}
	m.locations[location.DeliveryID] = append(m.locations[location.DeliveryID], location)

	return nil
}

func (m *MockLocationRepository) GetByDeliveryID(ctx context.Context, deliveryID int, limit int) ([]*domain.Location, error) {
	locations := m.locations[deliveryID]

	// Return most recent locations up to limit
	if len(locations) <= limit {
		return locations, nil
	}

	return locations[len(locations)-limit:], nil
}

func (m *MockLocationRepository) GetLatestByDeliveryID(ctx context.Context, deliveryID int) (*domain.Location, error) {
	locations := m.locations[deliveryID]
	if len(locations) == 0 {
		return nil, errors.New("location not found")
	}

	return locations[len(locations)-1], nil
}

func (m *MockLocationRepository) GetByCourierID(ctx context.Context, courierID int, limit int) ([]*domain.Location, error) {
	// For simplicity, return locations where courier ID matches
	var result []*domain.Location
	for _, locations := range m.locations {
		for _, loc := range locations {
			if loc.CourierID == courierID {
				result = append(result, loc)
			}
		}
	}

	// Return most recent up to limit
	if len(result) <= limit {
		return result, nil
	}

	return result[len(result)-limit:], nil
}

func (m *MockLocationRepository) GetLatestByCourierID(ctx context.Context, courierID int) (*domain.Location, error) {
	var latest *domain.Location
	for _, locations := range m.locations {
		for _, loc := range locations {
			if loc.CourierID == courierID {
				if latest == nil || loc.Timestamp.After(latest.Timestamp) {
					latest = loc
				}
			}
		}
	}

	if latest == nil {
		return nil, errors.New("location not found")
	}

	return latest, nil
}

func TestTrackingService_RecordLocation(t *testing.T) {
	repo := NewMockLocationRepository()
	service := NewTrackingService(repo)

	req := ports.RecordLocationRequest{
		DeliveryID: 1,
		CourierID:  1,
		Latitude:   40.7128,
		Longitude:  -74.0060,
		Accuracy:   &[]float64{10.5}[0],
		Speed:      &[]float64{25.0}[0],
		Heading:    &[]float64{90.0}[0],
		Altitude:   &[]float64{100.0}[0],
	}

	location, err := service.RecordLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if location == nil {
		t.Fatal("expected location, got nil")
	}

	if location.DeliveryID != req.DeliveryID {
		t.Errorf("expected delivery ID %d, got %d", req.DeliveryID, location.DeliveryID)
	}

	if location.CourierID != req.CourierID {
		t.Errorf("expected courier ID %d, got %d", req.CourierID, location.CourierID)
	}

	if location.Latitude != req.Latitude {
		t.Errorf("expected latitude %f, got %f", req.Latitude, location.Latitude)
	}

	if location.Longitude != req.Longitude {
		t.Errorf("expected longitude %f, got %f", req.Longitude, location.Longitude)
	}

	if location.Accuracy == nil || *location.Accuracy != *req.Accuracy {
		t.Errorf("expected accuracy %f, got %v", *req.Accuracy, location.Accuracy)
	}

	if location.Speed == nil || *location.Speed != *req.Speed {
		t.Errorf("expected speed %f, got %v", *req.Speed, location.Speed)
	}

	if location.Heading == nil || *location.Heading != *req.Heading {
		t.Errorf("expected heading %f, got %v", *req.Heading, location.Heading)
	}

	if location.Altitude == nil || *location.Altitude != *req.Altitude {
		t.Errorf("expected altitude %f, got %v", *req.Altitude, location.Altitude)
	}
}

func TestTrackingService_RecordLocation_InvalidData(t *testing.T) {
	repo := NewMockLocationRepository()
	service := NewTrackingService(repo)

	req := ports.RecordLocationRequest{
		DeliveryID: 0, // Invalid
		CourierID:  1,
		Latitude:   40.7128,
		Longitude:  -74.0060,
	}

	_, err := service.RecordLocation(context.Background(), req)
	if err == nil {
		t.Error("expected error for invalid delivery ID, got none")
	}
}

func TestTrackingService_GetDeliveryTrack(t *testing.T) {
	repo := NewMockLocationRepository()
	service := NewTrackingService(repo)

	// Add some test locations
	ctx := context.Background()
	for i := 1; i <= 5; i++ {
		req := ports.RecordLocationRequest{
			DeliveryID: 1,
			CourierID:  1,
			Latitude:   40.7128 + float64(i),
			Longitude:  -74.0060 + float64(i),
		}
		_, err := service.RecordLocation(ctx, req)
		if err != nil {
			t.Fatalf("failed to record location: %v", err)
		}
	}

	// Get track with limit
	trackReq := ports.GetDeliveryTrackRequest{
		DeliveryID: 1,
		Limit:      3,
	}

	locations, err := service.GetDeliveryTrack(ctx, trackReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(locations) != 3 {
		t.Errorf("expected 3 locations, got %d", len(locations))
	}

	// Verify they're the most recent (should be in chronological order)
	// The mock returns the last 3 locations: latitudes 43.7128, 44.7128, 45.7128
	expectedLats := []float64{43.7128, 44.7128, 45.7128}
	for i, loc := range locations {
		if loc.Latitude != expectedLats[i] {
			t.Errorf("expected latitude %f at index %d, got %f", expectedLats[i], i, loc.Latitude)
		}
	}
}

func TestTrackingService_GetCurrentLocation(t *testing.T) {
	repo := NewMockLocationRepository()
	service := NewTrackingService(repo)

	ctx := context.Background()

	// Add a location
	req := ports.RecordLocationRequest{
		DeliveryID: 1,
		CourierID:  1,
		Latitude:   40.7128,
		Longitude:  -74.0060,
	}
	_, err := service.RecordLocation(ctx, req)
	if err != nil {
		t.Fatalf("failed to record location: %v", err)
	}

	// Get current location
	currentReq := ports.GetCurrentLocationRequest{
		DeliveryID: 1,
	}

	location, err := service.GetCurrentLocation(ctx, currentReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if location == nil {
		t.Fatal("expected location, got nil")
	}

	if location.Latitude != 40.7128 {
		t.Errorf("expected latitude 40.7128, got %f", location.Latitude)
	}
}

func TestTrackingService_GetCourierLocation(t *testing.T) {
	repo := NewMockLocationRepository()
	service := NewTrackingService(repo)

	ctx := context.Background()

	// Add locations for different deliveries but same courier
	for i := 1; i <= 3; i++ {
		req := ports.RecordLocationRequest{
			DeliveryID: i,
			CourierID:  1,
			Latitude:   40.7128 + float64(i),
			Longitude:  -74.0060 + float64(i),
		}
		_, err := service.RecordLocation(ctx, req)
		if err != nil {
			t.Fatalf("failed to record location: %v", err)
		}
	}

	// Get courier location
	courierReq := ports.GetCourierLocationRequest{
		CourierID: 1,
	}

	location, err := service.GetCourierLocation(ctx, courierReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if location == nil {
		t.Fatal("expected location, got nil")
	}

	// Should be the latest (highest latitude)
	if location.Latitude != 40.7128+3 {
		t.Errorf("expected latest latitude 43.7128, got %f", location.Latitude)
	}
}

func TestTrackingService_CalculateETAToDestination(t *testing.T) {
	repo := NewMockLocationRepository()
	service := NewTrackingService(repo)

	ctx := context.Background()

	// Add current location
	req := ports.RecordLocationRequest{
		DeliveryID: 1,
		CourierID:  1,
		Latitude:   40.7128, // NYC
		Longitude:  -74.0060,
	}
	_, err := service.RecordLocation(ctx, req)
	if err != nil {
		t.Fatalf("failed to record location: %v", err)
	}

	// Calculate ETA to destination
	etaReq := ports.CalculateETAToDestinationRequest{
		DeliveryID: 1,
		DestLat:    40.7589, // Times Square (about 5km away)
		DestLng:    -73.9851,
	}

	eta, err := service.CalculateETAToDestination(ctx, etaReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eta == nil {
		t.Fatal("expected ETA response, got nil")
	}

	if eta.DistanceKm <= 0 {
		t.Errorf("expected positive distance, got %f", eta.DistanceKm)
	}

	if eta.AverageSpeed != 25.0 {
		t.Errorf("expected average speed 25.0, got %f", eta.AverageSpeed)
	}

	if eta.ETA <= 0 {
		t.Errorf("expected positive ETA duration, got %v", eta.ETA)
	}

	// ETA should be distance / speed
	expectedETAMinutes := (eta.DistanceKm / eta.AverageSpeed) * 60
	actualETAMinutes := eta.ETA.Minutes()

	if actualETAMinutes < expectedETAMinutes*0.9 || actualETAMinutes > expectedETAMinutes*1.1 {
		t.Errorf("ETA calculation seems incorrect. Expected ~%f minutes, got %f minutes",
			expectedETAMinutes, actualETAMinutes)
	}
}
