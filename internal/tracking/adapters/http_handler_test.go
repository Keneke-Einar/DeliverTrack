package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/domain"
	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
)

// MockTrackingService is a mock implementation of TrackingService for testing
type MockTrackingService struct {
	recordLocationFunc         func(ctx context.Context, req ports.RecordLocationRequest) (*domain.Location, error)
	getDeliveryTrackFunc       func(ctx context.Context, req ports.GetDeliveryTrackRequest) ([]*domain.Location, error)
	getCurrentLocationFunc     func(ctx context.Context, req ports.GetCurrentLocationRequest) (*domain.Location, error)
	getCourierLocationFunc     func(ctx context.Context, req ports.GetCourierLocationRequest) (*domain.Location, error)
	calculateETAFunc           func(ctx context.Context, req ports.CalculateETAToDestinationRequest) (*ports.CalculateETAResponse, error)
}

func (m *MockTrackingService) RecordLocation(ctx context.Context, req ports.RecordLocationRequest) (*domain.Location, error) {
	if m.recordLocationFunc != nil {
		return m.recordLocationFunc(ctx, req)
	}
	return &domain.Location{}, nil
}

func (m *MockTrackingService) GetDeliveryTrack(ctx context.Context, req ports.GetDeliveryTrackRequest) ([]*domain.Location, error) {
	if m.getDeliveryTrackFunc != nil {
		return m.getDeliveryTrackFunc(ctx, req)
	}
	return []*domain.Location{}, nil
}

func (m *MockTrackingService) GetCurrentLocation(ctx context.Context, req ports.GetCurrentLocationRequest) (*domain.Location, error) {
	if m.getCurrentLocationFunc != nil {
		return m.getCurrentLocationFunc(ctx, req)
	}
	return &domain.Location{}, nil
}

func (m *MockTrackingService) GetCourierLocation(ctx context.Context, req ports.GetCourierLocationRequest) (*domain.Location, error) {
	if m.getCourierLocationFunc != nil {
		return m.getCourierLocationFunc(ctx, req)
	}
	return &domain.Location{}, nil
}

func (m *MockTrackingService) CalculateETAToDestination(ctx context.Context, req ports.CalculateETAToDestinationRequest) (*ports.CalculateETAResponse, error) {
	if m.calculateETAFunc != nil {
		return m.calculateETAFunc(ctx, req)
	}
	return &ports.CalculateETAResponse{}, nil
}

func TestHTTPHandler_RecordLocation(t *testing.T) {
	mockService := &MockTrackingService{
		recordLocationFunc: func(ctx context.Context, req ports.RecordLocationRequest) (*domain.Location, error) {
			return &domain.Location{
				ID:         1,
				DeliveryID: req.DeliveryID,
				CourierID:  req.CourierID,
				Latitude:   req.Latitude,
				Longitude:  req.Longitude,
				Timestamp:  time.Now(),
				CreatedAt:  time.Now(),
			}, nil
		},
	}

	handler := NewHTTPHandler(mockService)

	reqBody := ports.RecordLocationRequest{
		DeliveryID: 1,
		CourierID:  1,
		Latitude:   40.7128,
		Longitude:  -74.0060,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	ctx := context.WithValue(req.Context(), "role", "courier")
	ctx = context.WithValue(ctx, "courier_id", &[]int{1}[0])
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RecordLocation(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response domain.Location
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.DeliveryID != reqBody.DeliveryID {
		t.Errorf("expected delivery ID %d, got %d", reqBody.DeliveryID, response.DeliveryID)
	}
}

func TestHTTPHandler_RecordLocation_Unauthorized(t *testing.T) {
	mockService := &MockTrackingService{}
	handler := NewHTTPHandler(mockService)

	reqBody := ports.RecordLocationRequest{
		DeliveryID: 1,
		CourierID:  1,
		Latitude:   40.7128,
		Longitude:  -74.0060,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add customer role (not authorized)
	ctx := context.WithValue(req.Context(), "role", "customer")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RecordLocation(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestHTTPHandler_RecordLocation_WrongCourier(t *testing.T) {
	mockService := &MockTrackingService{}
	handler := NewHTTPHandler(mockService)

	reqBody := ports.RecordLocationRequest{
		DeliveryID: 1,
		CourierID:  2, // Different courier ID
		Latitude:   40.7128,
		Longitude:  -74.0060,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context for courier ID 1
	ctx := context.WithValue(req.Context(), "role", "courier")
	ctx = context.WithValue(ctx, "courier_id", &[]int{1}[0])
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RecordLocation(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestHTTPHandler_GetDeliveryTrack(t *testing.T) {
	mockService := &MockTrackingService{
		getDeliveryTrackFunc: func(ctx context.Context, req ports.GetDeliveryTrackRequest) ([]*domain.Location, error) {
			return []*domain.Location{
				{
					ID:         1,
					DeliveryID: req.DeliveryID,
					CourierID:  1,
					Latitude:   40.7128,
					Longitude:  -74.0060,
					Timestamp:  time.Now(),
					CreatedAt:  time.Now(),
				},
			}, nil
		},
	}

	handler := NewHTTPHandler(mockService)

	req := httptest.NewRequest("GET", "/deliveries/1/track", nil)

	// Add auth context
	ctx := context.WithValue(req.Context(), "role", "customer")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetDeliveryTrack(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response struct {
		DeliveryID int                    `json:"delivery_id"`
		Locations  []*domain.Location     `json:"locations"`
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.DeliveryID != 1 {
		t.Errorf("expected delivery ID 1, got %d", response.DeliveryID)
	}

	if len(response.Locations) != 1 {
		t.Errorf("expected 1 location, got %d", len(response.Locations))
	}
}

func TestHTTPHandler_GetCurrentLocation(t *testing.T) {
	mockService := &MockTrackingService{
		getCurrentLocationFunc: func(ctx context.Context, req ports.GetCurrentLocationRequest) (*domain.Location, error) {
			return &domain.Location{
				ID:         1,
				DeliveryID: req.DeliveryID,
				CourierID:  1,
				Latitude:   40.7128,
				Longitude:  -74.0060,
				Timestamp:  time.Now(),
				CreatedAt:  time.Now(),
			}, nil
		},
	}

	handler := NewHTTPHandler(mockService)

	req := httptest.NewRequest("GET", "/deliveries/1/location", nil)

	// Add auth context
	ctx := context.WithValue(req.Context(), "role", "customer")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetCurrentLocation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response domain.Location
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.DeliveryID != 1 {
		t.Errorf("expected delivery ID 1, got %d", response.DeliveryID)
	}
}

func TestHTTPHandler_GetCourierLocation(t *testing.T) {
	mockService := &MockTrackingService{
		getCourierLocationFunc: func(ctx context.Context, req ports.GetCourierLocationRequest) (*domain.Location, error) {
			return &domain.Location{
				ID:        1,
				CourierID: req.CourierID,
				Latitude:  40.7128,
				Longitude: -74.0060,
				Timestamp: time.Now(),
				CreatedAt: time.Now(),
			}, nil
		},
	}

	handler := NewHTTPHandler(mockService)

	req := httptest.NewRequest("GET", "/couriers/1/location", nil)

	// Add auth context
	ctx := context.WithValue(req.Context(), "role", "courier")
	ctx = context.WithValue(ctx, "courier_id", &[]int{1}[0])
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetCourierLocation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response domain.Location
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.CourierID != 1 {
		t.Errorf("expected courier ID 1, got %d", response.CourierID)
	}
}

func TestHTTPHandler_CalculateETA(t *testing.T) {
	mockService := &MockTrackingService{
		calculateETAFunc: func(ctx context.Context, req ports.CalculateETAToDestinationRequest) (*ports.CalculateETAResponse, error) {
			return &ports.CalculateETAResponse{
				ETA:          time.Hour,
				DistanceKm:   25.0,
				AverageSpeed: 25.0,
			}, nil
		},
	}

	handler := NewHTTPHandler(mockService)

	reqBody := struct {
		DestLat float64 `json:"dest_lat"`
		DestLng float64 `json:"dest_lng"`
	}{
		DestLat: 40.7589,
		DestLng: -73.9851,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/deliveries/1/eta", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	ctx := context.WithValue(req.Context(), "role", "customer")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CalculateETA(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response ports.CalculateETAResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.DistanceKm != 25.0 {
		t.Errorf("expected distance 25.0, got %f", response.DistanceKm)
	}

	if response.AverageSpeed != 25.0 {
		t.Errorf("expected speed 25.0, got %f", response.AverageSpeed)
	}
}

func TestHTTPHandler_InvalidJSON(t *testing.T) {
	mockService := &MockTrackingService{}
	handler := NewHTTPHandler(mockService)

	req := httptest.NewRequest("POST", "/locations", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	// Add auth context
	ctx := context.WithValue(req.Context(), "role", "courier")
	ctx = context.WithValue(ctx, "courier_id", &[]int{1}[0])
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.RecordLocation(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHTTPHandler_MethodNotAllowed(t *testing.T) {
	mockService := &MockTrackingService{}
	handler := NewHTTPHandler(mockService)

	req := httptest.NewRequest("GET", "/locations", nil)

	w := httptest.NewRecorder()
	handler.RecordLocation(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}