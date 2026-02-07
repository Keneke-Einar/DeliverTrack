package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/domain"
	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/proto/analytics"
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

// MockDeliveryRepository is a mock implementation of DeliveryRepository for testing
type MockDeliveryRepository struct {
	deliveries map[int]*domain.Delivery
	nextID     int
	createErr  error
	getByIDErr error
	updateErr  error
}

func NewMockDeliveryRepository() *MockDeliveryRepository {
	return &MockDeliveryRepository{
		deliveries: make(map[int]*domain.Delivery),
		nextID:     1,
	}
}

func (m *MockDeliveryRepository) Create(ctx context.Context, delivery *domain.Delivery) error {
	if m.createErr != nil {
		return m.createErr
	}
	delivery.ID = m.nextID
	m.deliveries[m.nextID] = delivery
	m.nextID++
	return nil
}

func (m *MockDeliveryRepository) GetByID(ctx context.Context, id int) (*domain.Delivery, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	delivery, exists := m.deliveries[id]
	if !exists {
		return nil, domain.ErrDeliveryNotFound
	}
	return delivery, nil
}

func (m *MockDeliveryRepository) GetByStatus(ctx context.Context, status string, customerID int) ([]*domain.Delivery, error) {
	var deliveries []*domain.Delivery
	for _, d := range m.deliveries {
		if d.Status == status && (customerID == 0 || d.CustomerID == customerID) {
			deliveries = append(deliveries, d)
		}
	}
	return deliveries, nil
}

func (m *MockDeliveryRepository) GetAll(ctx context.Context, customerID int) ([]*domain.Delivery, error) {
	var deliveries []*domain.Delivery
	for _, d := range m.deliveries {
		if customerID == 0 || d.CustomerID == customerID {
			deliveries = append(deliveries, d)
		}
	}
	return deliveries, nil
}

func (m *MockDeliveryRepository) UpdateStatus(ctx context.Context, id int, status, notes string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	delivery, exists := m.deliveries[id]
	if !exists {
		return domain.ErrDeliveryNotFound
	}
	delivery.Status = status
	delivery.Notes = notes
	delivery.UpdatedAt = time.Now()
	return nil
}

func (m *MockDeliveryRepository) AssignCourier(ctx context.Context, deliveryID, courierID int) error {
	delivery, exists := m.deliveries[deliveryID]
	if !exists {
		return domain.ErrDeliveryNotFound
	}
	delivery.CourierID = &courierID
	delivery.Status = "assigned"
	delivery.UpdatedAt = time.Now()
	return nil
}

func (m *MockDeliveryRepository) Update(ctx context.Context, delivery *domain.Delivery) error {
	m.deliveries[delivery.ID] = delivery
	return nil
}

func (m *MockDeliveryRepository) SetCreateError(err error) {
	m.createErr = err
}

func (m *MockDeliveryRepository) SetGetByIDError(err error) {
	m.getByIDErr = err
}

func (m *MockDeliveryRepository) SetUpdateError(err error) {
	m.updateErr = err
}

func (m *MockDeliveryRepository) AddDelivery(delivery *domain.Delivery) {
	m.deliveries[delivery.ID] = delivery
	if delivery.ID >= m.nextID {
		m.nextID = delivery.ID + 1
	}
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

func (m *MockPublisher) GetPublishedEvents() []messaging.Event {
	return m.publishedEvents
}

// MockNotificationClient is a mock implementation of NotificationServiceClient for testing
type MockNotificationClient struct{}

func (m *MockNotificationClient) SendNotification(ctx context.Context, in *notification.SendNotificationRequest, opts ...grpc.CallOption) (*notification.SendNotificationResponse, error) {
	return &notification.SendNotificationResponse{
		NotificationId: "mock-notification-id",
		Status:         notification.NotificationStatus_NOTIFICATION_STATUS_SENT,
		SentAt:         time.Now().Unix(),
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
		SentAt:         time.Now().Unix(),
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

// MockAnalyticsClient is a mock implementation of AnalyticsServiceClient for testing
type MockAnalyticsClient struct{}

func (m *MockAnalyticsClient) RecordEvent(ctx context.Context, in *analytics.RecordEventRequest, opts ...grpc.CallOption) (*analytics.RecordEventResponse, error) {
	return &analytics.RecordEventResponse{
		Success:    true,
		RecordedAt: time.Now().Unix(),
	}, nil
}

func (m *MockAnalyticsClient) BatchRecordEvents(ctx context.Context, in *analytics.BatchRecordEventsRequest, opts ...grpc.CallOption) (*analytics.BatchRecordEventsResponse, error) {
	return &analytics.BatchRecordEventsResponse{
		SuccessCount: 1,
		FailedCount:  0,
	}, nil
}

func (m *MockAnalyticsClient) GetDeliveryMetrics(ctx context.Context, in *analytics.GetDeliveryMetricsRequest, opts ...grpc.CallOption) (*analytics.GetDeliveryMetricsResponse, error) {
	return &analytics.GetDeliveryMetricsResponse{}, nil
}

func (m *MockAnalyticsClient) GetDriverPerformance(ctx context.Context, in *analytics.GetDriverPerformanceRequest, opts ...grpc.CallOption) (*analytics.GetDriverPerformanceResponse, error) {
	return &analytics.GetDriverPerformanceResponse{}, nil
}

func (m *MockAnalyticsClient) GetCustomerAnalytics(ctx context.Context, in *analytics.GetCustomerAnalyticsRequest, opts ...grpc.CallOption) (*analytics.GetCustomerAnalyticsResponse, error) {
	return &analytics.GetCustomerAnalyticsResponse{}, nil
}

func (m *MockAnalyticsClient) GetSystemMetrics(ctx context.Context, in *analytics.GetSystemMetricsRequest, opts ...grpc.CallOption) (*analytics.GetSystemMetricsResponse, error) {
	return &analytics.GetSystemMetricsResponse{}, nil
}

func (m *MockAnalyticsClient) GenerateReport(ctx context.Context, in *analytics.GenerateReportRequest, opts ...grpc.CallOption) (*analytics.GenerateReportResponse, error) {
	return &analytics.GenerateReportResponse{}, nil
}

func (m *MockAnalyticsClient) GetDashboard(ctx context.Context, in *analytics.GetDashboardRequest, opts ...grpc.CallOption) (*analytics.GetDashboardResponse, error) {
	return &analytics.GetDashboardResponse{}, nil
}

func (m *MockAnalyticsClient) GetRouteEfficiency(ctx context.Context, in *analytics.GetRouteEfficiencyRequest, opts ...grpc.CallOption) (*analytics.GetRouteEfficiencyResponse, error) {
	return &analytics.GetRouteEfficiencyResponse{}, nil
}

// MockDeliveryClient is a mock implementation of DeliveryServiceClient for testing
type MockDeliveryClient struct{}

func (m *MockDeliveryClient) CreateDelivery(ctx context.Context, in *delivery.CreateDeliveryRequest, opts ...grpc.CallOption) (*delivery.CreateDeliveryResponse, error) {
	return &delivery.CreateDeliveryResponse{}, nil
}

func (m *MockDeliveryClient) GetDelivery(ctx context.Context, in *delivery.GetDeliveryRequest, opts ...grpc.CallOption) (*delivery.GetDeliveryResponse, error) {
	return &delivery.GetDeliveryResponse{}, nil
}

func (m *MockDeliveryClient) UpdateDeliveryStatus(ctx context.Context, in *delivery.UpdateDeliveryStatusRequest, opts ...grpc.CallOption) (*delivery.UpdateDeliveryStatusResponse, error) {
	return &delivery.UpdateDeliveryStatusResponse{}, nil
}

func (m *MockDeliveryClient) AssignDriver(ctx context.Context, in *delivery.AssignDriverRequest, opts ...grpc.CallOption) (*delivery.AssignDriverResponse, error) {
	return &delivery.AssignDriverResponse{}, nil
}

func (m *MockDeliveryClient) ListDeliveries(ctx context.Context, in *delivery.ListDeliveriesRequest, opts ...grpc.CallOption) (*delivery.ListDeliveriesResponse, error) {
	return &delivery.ListDeliveriesResponse{}, nil
}

func (m *MockDeliveryClient) CancelDelivery(ctx context.Context, in *delivery.CancelDeliveryRequest, opts ...grpc.CallOption) (*delivery.CancelDeliveryResponse, error) {
	return &delivery.CancelDeliveryResponse{}, nil
}

func (m *MockDeliveryClient) GetDriverDeliveries(ctx context.Context, in *delivery.GetDriverDeliveriesRequest, opts ...grpc.CallOption) (*delivery.GetDriverDeliveriesResponse, error) {
	return &delivery.GetDriverDeliveriesResponse{}, nil
}

func (m *MockDeliveryClient) OptimizeRoute(ctx context.Context, in *delivery.OptimizeRouteRequest, opts ...grpc.CallOption) (*delivery.OptimizeRouteResponse, error) {
	return &delivery.OptimizeRouteResponse{}, nil
}

func (m *MockDeliveryClient) ConfirmDelivery(ctx context.Context, in *delivery.ConfirmDeliveryRequest, opts ...grpc.CallOption) (*delivery.ConfirmDeliveryResponse, error) {
	return &delivery.ConfirmDeliveryResponse{}, nil
}

func TestDeliveryService_CreateDelivery(t *testing.T) {
	tests := []struct {
		name           string
		customerID     int
		courierID      *int
		pickupLoc      string
		deliveryLoc    string
		notes          string
		scheduledDate  *string
		mockCreateErr  error
		expectError    bool
		expectedStatus string
	}{
		{
			name:           "successful creation",
			customerID:     1,
			courierID:      nil,
			pickupLoc:      "123 Main St",
			deliveryLoc:    "456 Oak Ave",
			notes:          "Handle with care",
			scheduledDate:  nil,
			mockCreateErr:  nil,
			expectError:    false,
			expectedStatus: domain.StatusPending,
		},
		{
			name:           "creation with courier",
			customerID:     1,
			courierID:      func() *int { i := 2; return &i }(),
			pickupLoc:      "123 Main St",
			deliveryLoc:    "456 Oak Ave",
			notes:          "",
			scheduledDate:  nil,
			mockCreateErr:  nil,
			expectError:    false,
			expectedStatus: domain.StatusPending,
		},
		{
			name:           "creation with scheduled date",
			customerID:     1,
			courierID:      nil,
			pickupLoc:      "123 Main St",
			deliveryLoc:    "456 Oak Ave",
			notes:          "",
			scheduledDate:  func() *string { s := "2024-01-01T10:00:00Z"; return &s }(),
			mockCreateErr:  nil,
			expectError:    false,
			expectedStatus: domain.StatusPending,
		},
		{
			name:          "invalid customer ID",
			customerID:    0,
			courierID:     nil,
			pickupLoc:     "123 Main St",
			deliveryLoc:   "456 Oak Ave",
			notes:         "",
			scheduledDate: nil,
			mockCreateErr: nil,
			expectError:   true,
		},
		{
			name:          "empty pickup location",
			customerID:    1,
			courierID:     nil,
			pickupLoc:     "",
			deliveryLoc:   "456 Oak Ave",
			notes:         "",
			scheduledDate: nil,
			mockCreateErr: nil,
			expectError:   true,
		},
		{
			name:          "repository error",
			customerID:    1,
			courierID:     nil,
			pickupLoc:     "123 Main St",
			deliveryLoc:   "456 Oak Ave",
			notes:         "",
			scheduledDate: nil,
			mockCreateErr: errors.New("database error"),
			expectError:   true,
		},
		{
			name:          "invalid scheduled date format",
			customerID:    1,
			courierID:     nil,
			pickupLoc:     "123 Main St",
			deliveryLoc:   "456 Oak Ave",
			notes:         "",
			scheduledDate: func() *string { s := "invalid-date"; return &s }(),
			mockCreateErr: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := NewMockDeliveryRepository()
			mockRepo.SetCreateError(tt.mockCreateErr)
			mockPublisher := NewMockPublisher()
			mockDeliveryClient := &MockDeliveryClient{}
			testLogger := createTestLogger(t)
			service := NewDeliveryService(mockRepo, mockPublisher, mockDeliveryClient, testLogger)

			delivery, err := service.CreateDelivery(context.Background(), ports.CreateDeliveryRequest{
				CustomerID:       tt.customerID,
				CourierID:        tt.courierID,
				PickupLocation:   tt.pickupLoc,
				DeliveryLocation: tt.deliveryLoc,
				Notes:            tt.notes,
				ScheduledDate:    tt.scheduledDate,
			})

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if delivery == nil {
				t.Error("expected delivery but got nil")
				return
			}

			if delivery.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, delivery.Status)
			}

			if delivery.CustomerID != tt.customerID {
				t.Errorf("expected customer ID %d, got %d", tt.customerID, delivery.CustomerID)
			}

			if delivery.PickupLocation != tt.pickupLoc {
				t.Errorf("expected pickup location %s, got %s", tt.pickupLoc, delivery.PickupLocation)
			}

			if delivery.DeliveryLocation != tt.deliveryLoc {
				t.Errorf("expected delivery location %s, got %s", tt.deliveryLoc, delivery.DeliveryLocation)
			}

			if tt.courierID != nil {
				if delivery.CourierID == nil || *delivery.CourierID != *tt.courierID {
					t.Errorf("expected courier ID %d, got %v", *tt.courierID, delivery.CourierID)
				}
			}

			if delivery.Notes != tt.notes {
				t.Errorf("expected notes %s, got %s", tt.notes, delivery.Notes)
			}
		})
	}
}

func TestDeliveryService_GetDelivery(t *testing.T) {
	mockRepo := NewMockDeliveryRepository()
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewDeliveryService(mockRepo, mockPublisher, mockDeliveryClient, testLogger)

	// Create a test delivery
	delivery := &domain.Delivery{
		ID:               1,
		CustomerID:       1,
		CourierID:        func() *int { i := 2; return &i }(),
		Status:           domain.StatusPending,
		PickupLocation:   "123 Main St",
		DeliveryLocation: "456 Oak Ave",
		Notes:            "Test delivery",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	mockRepo.AddDelivery(delivery)

	tests := []struct {
		name        string
		id          int
		role        string
		customerID  *int
		courierID   *int
		mockGetErr  error
		expectError bool
		expectedErr error
	}{
		{
			name:        "admin access",
			id:          1,
			role:        "admin",
			customerID:  nil,
			courierID:   nil,
			mockGetErr:  nil,
			expectError: false,
		},
		{
			name:        "customer access own delivery",
			id:          1,
			role:        "customer",
			customerID:  func() *int { i := 1; return &i }(),
			courierID:   nil,
			mockGetErr:  nil,
			expectError: false,
		},
		{
			name:        "courier access assigned delivery",
			id:          1,
			role:        "courier",
			customerID:  nil,
			courierID:   func() *int { i := 2; return &i }(),
			mockGetErr:  nil,
			expectError: false,
		},
		{
			name:        "customer access other delivery",
			id:          1,
			role:        "customer",
			customerID:  func() *int { i := 999; return &i }(),
			courierID:   nil,
			mockGetErr:  nil,
			expectError: true,
			expectedErr: domain.ErrUnauthorized,
		},
		{
			name:        "courier access unassigned delivery",
			id:          1,
			role:        "courier",
			customerID:  nil,
			courierID:   func() *int { i := 999; return &i }(),
			mockGetErr:  nil,
			expectError: true,
			expectedErr: domain.ErrUnauthorized,
		},
		{
			name:        "delivery not found",
			id:          999,
			role:        "admin",
			customerID:  nil,
			courierID:   nil,
			mockGetErr:  nil,
			expectError: true,
			expectedErr: domain.ErrDeliveryNotFound,
		},
		{
			name:        "repository error",
			id:          1,
			role:        "admin",
			customerID:  nil,
			courierID:   nil,
			mockGetErr:  errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.SetGetByIDError(tt.mockGetErr)

			result, err := service.GetDelivery(context.Background(), ports.GetDeliveryRequest{
				ID: tt.id,
				AuthContext: ports.AuthContext{
					Role:           tt.role,
					UserCustomerID: tt.customerID,
					UserCourierID:  tt.courierID,
				},
			})

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("expected delivery but got nil")
				return
			}

			if result.ID != tt.id {
				t.Errorf("expected ID %d, got %d", tt.id, result.ID)
			}
		})
	}
}

func TestDeliveryService_ListDeliveries(t *testing.T) {
	mockRepo := NewMockDeliveryRepository()
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewDeliveryService(mockRepo, mockPublisher, mockDeliveryClient, testLogger)

	// Create test deliveries
	deliveries := []*domain.Delivery{
		{
			ID:               1,
			CustomerID:       1,
			CourierID:        func() *int { i := 2; return &i }(),
			Status:           domain.StatusPending,
			PickupLocation:   "123 Main St",
			DeliveryLocation: "456 Oak Ave",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               2,
			CustomerID:       1,
			CourierID:        nil,
			Status:           domain.StatusAssigned,
			PickupLocation:   "789 Pine St",
			DeliveryLocation: "321 Elm St",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               3,
			CustomerID:       3,
			CourierID:        func() *int { i := 2; return &i }(),
			Status:           domain.StatusDelivered,
			PickupLocation:   "111 Oak St",
			DeliveryLocation: "222 Maple Ave",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	for _, d := range deliveries {
		mockRepo.AddDelivery(d)
	}

	tests := []struct {
		name           string
		status         string
		customerID     int
		role           string
		userCustomerID *int
		userCourierID  *int
		expectedCount  int
	}{
		{
			name:           "admin list all",
			status:         "",
			customerID:     0,
			role:           "admin",
			userCustomerID: nil,
			userCourierID:  nil,
			expectedCount:  3,
		},
		{
			name:           "admin list by status",
			status:         domain.StatusPending,
			customerID:     0,
			role:           "admin",
			userCustomerID: nil,
			userCourierID:  nil,
			expectedCount:  1,
		},
		{
			name:           "customer list own deliveries",
			status:         "",
			customerID:     0,
			role:           "customer",
			userCustomerID: func() *int { i := 1; return &i }(),
			userCourierID:  nil,
			expectedCount:  2,
		},
		{
			name:           "courier list assigned deliveries",
			status:         "",
			customerID:     0,
			role:           "courier",
			userCustomerID: nil,
			userCourierID:  func() *int { i := 2; return &i }(),
			expectedCount:  2,
		},
		{
			name:           "courier list by status",
			status:         domain.StatusDelivered,
			customerID:     0,
			role:           "courier",
			userCustomerID: nil,
			userCourierID:  func() *int { i := 2; return &i }(),
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ListDeliveries(context.Background(), ports.ListDeliveriesRequest{
				Status:     tt.status,
				CustomerID: tt.customerID,
				AuthContext: ports.AuthContext{
					Role:           tt.role,
					UserCustomerID: tt.userCustomerID,
					UserCourierID:  tt.userCourierID,
				},
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d deliveries, got %d", tt.expectedCount, len(result))
			}
		})
	}
}

func TestDeliveryService_UpdateDeliveryStatus(t *testing.T) {
	mockRepo := NewMockDeliveryRepository()
	mockPublisher := NewMockPublisher()
	mockDeliveryClient := &MockDeliveryClient{}
	testLogger := createTestLogger(t)
	service := NewDeliveryService(mockRepo, mockPublisher, mockDeliveryClient, testLogger)

	// Create a test delivery
	delivery := &domain.Delivery{
		ID:               1,
		CustomerID:       1,
		CourierID:        func() *int { i := 2; return &i }(),
		Status:           domain.StatusPending,
		PickupLocation:   "123 Main St",
		DeliveryLocation: "456 Oak Ave",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	mockRepo.AddDelivery(delivery)

	tests := []struct {
		name          string
		id            int
		status        string
		notes         string
		role          string
		customerID    *int
		courierID     *int
		mockUpdateErr error
		expectError   bool
		expectedErr   error
	}{
		{
			name:          "admin update status",
			id:            1,
			status:        domain.StatusAssigned,
			notes:         "Assigned to courier",
			role:          "admin",
			customerID:    nil,
			courierID:     nil,
			mockUpdateErr: nil,
			expectError:   false,
		},
		{
			name:          "customer update own delivery",
			id:            1,
			status:        domain.StatusCancelled,
			notes:         "Cancel delivery",
			role:          "customer",
			customerID:    func() *int { i := 1; return &i }(),
			courierID:     nil,
			mockUpdateErr: nil,
			expectError:   false,
		},
		{
			name:          "courier update assigned delivery",
			id:            1,
			status:        domain.StatusInTransit,
			notes:         "Picked up",
			role:          "courier",
			customerID:    nil,
			courierID:     func() *int { i := 2; return &i }(),
			mockUpdateErr: nil,
			expectError:   false,
		},
		{
			name:          "customer update other delivery",
			id:            1,
			status:        domain.StatusCancelled,
			notes:         "",
			role:          "customer",
			customerID:    func() *int { i := 999; return &i }(),
			courierID:     nil,
			mockUpdateErr: nil,
			expectError:   true,
			expectedErr:   domain.ErrUnauthorized,
		},
		{
			name:          "invalid status",
			id:            1,
			status:        "invalid_status",
			notes:         "",
			role:          "admin",
			customerID:    nil,
			courierID:     nil,
			mockUpdateErr: nil,
			expectError:   true,
			expectedErr:   domain.ErrInvalidStatus,
		},
		{
			name:          "delivery not found",
			id:            999,
			status:        domain.StatusAssigned,
			notes:         "",
			role:          "admin",
			customerID:    nil,
			courierID:     nil,
			mockUpdateErr: nil,
			expectError:   true,
			expectedErr:   domain.ErrDeliveryNotFound,
		},
		{
			name:          "repository error",
			id:            1,
			status:        domain.StatusAssigned,
			notes:         "",
			role:          "admin",
			customerID:    nil,
			courierID:     nil,
			mockUpdateErr: errors.New("database error"),
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.SetUpdateError(tt.mockUpdateErr)

			err := service.UpdateDeliveryStatus(context.Background(), ports.UpdateDeliveryStatusRequest{
				ID:     tt.id,
				Status: tt.status,
				Notes:  tt.notes,
				AuthContext: ports.AuthContext{
					Role:           tt.role,
					UserCustomerID: tt.customerID,
					UserCourierID:  tt.courierID,
				},
			})

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the status was updated in the mock
			updated, _ := mockRepo.GetByID(context.Background(), tt.id)
			if updated.Status != tt.status {
				t.Errorf("expected status %s, got %s", tt.status, updated.Status)
			}
			if updated.Notes != tt.notes {
				t.Errorf("expected notes %s, got %s", tt.notes, updated.Notes)
			}
		})
	}
}
