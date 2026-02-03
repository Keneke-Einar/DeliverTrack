package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/proto/common"
	"github.com/Keneke-Einar/delivertrack/proto/delivery"
	"github.com/Keneke-Einar/delivertrack/proto/notification"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
)

// createTestLogger creates a test logger for unit tests
func createTestLogger(t *testing.T) *logger.Logger {
	zapLogger := zaptest.NewLogger(t)
	return &logger.Logger{Logger: zapLogger}
}

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

// MockPublisher is a mock implementation of messaging.Publisher for testing
type MockPublisher struct {
	publishedEvents []messaging.Event
	publishErr      error
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		publishedEvents: make([]messaging.Event, 0),
	}
}

func (m *MockPublisher) Publish(ctx context.Context, exchange, routingKey string, event messaging.Event) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishedEvents = append(m.publishedEvents, event)
	return nil
}

func (m *MockPublisher) Close() error {
	return nil
}

func (m *MockPublisher) SetPublishError(err error) {
	m.publishErr = err
}

// MockDeliveryClient is a mock implementation of DeliveryServiceClient for testing
type MockDeliveryClient struct{}

func (m *MockDeliveryClient) CreateDelivery(ctx context.Context, in *delivery.CreateDeliveryRequest, opts ...grpc.CallOption) (*delivery.CreateDeliveryResponse, error) {
	return &delivery.CreateDeliveryResponse{
		DeliveryId:     "mock-delivery-id",
		TrackingNumber: "mock-tracking-number",
		CreatedAt:      1234567890,
	}, nil
}

func (m *MockDeliveryClient) GetDelivery(ctx context.Context, in *delivery.GetDeliveryRequest, opts ...grpc.CallOption) (*delivery.GetDeliveryResponse, error) {
	return &delivery.GetDeliveryResponse{
		Delivery: &delivery.Delivery{
			DeliveryId:       "mock-delivery-id",
			CustomerId:       "1",
			DriverId:         "1",
			TrackingNumber:   "mock-tracking-number",
			PickupLocation:   &common.Location{Latitude: 40.7128, Longitude: -74.0060},
			DeliveryLocation: &common.Location{Latitude: 40.7589, Longitude: -73.9851},
			Status:           delivery.DeliveryStatus_DELIVERY_STATUS_IN_TRANSIT,
			CreatedAt:        1234567890,
			UpdatedAt:        1234567890,
		},
	}, nil
}

func (m *MockDeliveryClient) UpdateDeliveryStatus(ctx context.Context, in *delivery.UpdateDeliveryStatusRequest, opts ...grpc.CallOption) (*delivery.UpdateDeliveryStatusResponse, error) {
	return &delivery.UpdateDeliveryStatusResponse{
		Success:   true,
		UpdatedAt: 1234567890,
	}, nil
}

func (m *MockDeliveryClient) AssignDriver(ctx context.Context, in *delivery.AssignDriverRequest, opts ...grpc.CallOption) (*delivery.AssignDriverResponse, error) {
	return &delivery.AssignDriverResponse{
		Success:    true,
		AssignedAt: 1234567890,
	}, nil
}

func (m *MockDeliveryClient) ListDeliveries(ctx context.Context, in *delivery.ListDeliveriesRequest, opts ...grpc.CallOption) (*delivery.ListDeliveriesResponse, error) {
	return &delivery.ListDeliveriesResponse{}, nil
}

func (m *MockDeliveryClient) CancelDelivery(ctx context.Context, in *delivery.CancelDeliveryRequest, opts ...grpc.CallOption) (*delivery.CancelDeliveryResponse, error) {
	return &delivery.CancelDeliveryResponse{
		Success:     true,
		CancelledAt: 1234567890,
	}, nil
}

func (m *MockDeliveryClient) GetDriverDeliveries(ctx context.Context, in *delivery.GetDriverDeliveriesRequest, opts ...grpc.CallOption) (*delivery.GetDriverDeliveriesResponse, error) {
	return &delivery.GetDriverDeliveriesResponse{}, nil
}

func (m *MockDeliveryClient) OptimizeRoute(ctx context.Context, in *delivery.OptimizeRouteRequest, opts ...grpc.CallOption) (*delivery.OptimizeRouteResponse, error) {
	return &delivery.OptimizeRouteResponse{}, nil
}

func (m *MockDeliveryClient) ConfirmDelivery(ctx context.Context, in *delivery.ConfirmDeliveryRequest, opts ...grpc.CallOption) (*delivery.ConfirmDeliveryResponse, error) {
	return &delivery.ConfirmDeliveryResponse{
		Success:     true,
		ConfirmedAt: 1234567890,
	}, nil
}

// MockNotificationClient is a mock implementation of NotificationServiceClient for testing
type MockNotificationClient struct{}

func (m *MockNotificationClient) SendNotification(ctx context.Context, in *notification.SendNotificationRequest, opts ...grpc.CallOption) (*notification.SendNotificationResponse, error) {
	return &notification.SendNotificationResponse{
		NotificationId: "mock-notification-id",
		Status:         notification.NotificationStatus_NOTIFICATION_STATUS_SENT,
		SentAt:         1234567890,
	}, nil
}

func (m *MockNotificationClient) SendBulkNotifications(ctx context.Context, in *notification.SendBulkNotificationsRequest, opts ...grpc.CallOption) (*notification.SendBulkNotificationsResponse, error) {
	return &notification.SendBulkNotificationsResponse{
		SuccessCount: 1,
		FailedCount:  0,
	}, nil
}

func (m *MockNotificationClient) SendDeliveryUpdate(ctx context.Context, in *notification.SendDeliveryUpdateRequest, opts ...grpc.CallOption) (*notification.SendDeliveryUpdateResponse, error) {
	return &notification.SendDeliveryUpdateResponse{
		NotificationId: "mock-delivery-notification-id",
		Success:        true,
		SentAt:         1234567890,
	}, nil
}

func (m *MockNotificationClient) GetNotificationHistory(ctx context.Context, in *notification.GetNotificationHistoryRequest, opts ...grpc.CallOption) (*notification.GetNotificationHistoryResponse, error) {
	return &notification.GetNotificationHistoryResponse{}, nil
}

func (m *MockNotificationClient) UpdatePreferences(ctx context.Context, in *notification.UpdatePreferencesRequest, opts ...grpc.CallOption) (*notification.UpdatePreferencesResponse, error) {
	return &notification.UpdatePreferencesResponse{Success: true}, nil
}

func (m *MockNotificationClient) GetPreferences(ctx context.Context, in *notification.GetPreferencesRequest, opts ...grpc.CallOption) (*notification.GetPreferencesResponse, error) {
	return &notification.GetPreferencesResponse{}, nil
}

func (m *MockNotificationClient) Subscribe(ctx context.Context, in *notification.SubscribeRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[notification.Notification], error) {
	return nil, nil
}

func (m *MockNotificationClient) MarkAsRead(ctx context.Context, in *notification.MarkAsReadRequest, opts ...grpc.CallOption) (*notification.MarkAsReadResponse, error) {
	return &notification.MarkAsReadResponse{Success: true}, nil
}

func TestTrackingService_RecordLocation(t *testing.T) {
	repo := NewMockLocationRepository()
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewTrackingService(repo, mockPublisher, mockDeliveryClient, testLogger)

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
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewTrackingService(repo, mockPublisher, mockDeliveryClient, testLogger)

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
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewTrackingService(repo, mockPublisher, mockDeliveryClient, testLogger)

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
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewTrackingService(repo, mockPublisher, mockDeliveryClient, testLogger)

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
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewTrackingService(repo, mockPublisher, mockDeliveryClient, testLogger)

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
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewTrackingService(repo, mockPublisher, mockDeliveryClient, testLogger)

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
