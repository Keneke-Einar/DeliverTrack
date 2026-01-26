package domain

import (
	"testing"
	"time"
)

// TestNewDelivery tests the NewDelivery constructor function
func TestNewDelivery(t *testing.T) {
	tests := []struct {
		name             string
		customerID       int
		pickupLocation   string
		deliveryLocation string
		expectError      bool
	}{
		{
			name:             "valid delivery",
			customerID:       1,
			pickupLocation:   "123 Main St",
			deliveryLocation: "456 Oak Ave",
			expectError:      false,
		},
		{
			name:             "invalid customer ID",
			customerID:       0,
			pickupLocation:   "123 Main St",
			deliveryLocation: "456 Oak Ave",
			expectError:      true,
		},
		{
			name:             "negative customer ID",
			customerID:       -1,
			pickupLocation:   "123 Main St",
			deliveryLocation: "456 Oak Ave",
			expectError:      true,
		},
		{
			name:             "empty pickup location",
			customerID:       1,
			pickupLocation:   "",
			deliveryLocation: "456 Oak Ave",
			expectError:      true,
		},
		{
			name:             "empty delivery location",
			customerID:       1,
			pickupLocation:   "123 Main St",
			deliveryLocation: "",
			expectError:      true,
		},
		{
			name:             "both locations empty",
			customerID:       1,
			pickupLocation:   "",
			deliveryLocation: "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delivery, err := NewDelivery(tt.customerID, tt.pickupLocation, tt.deliveryLocation)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if delivery != nil {
					t.Error("expected nil delivery on error")
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

			if delivery.CustomerID != tt.customerID {
				t.Errorf("expected customer ID %d, got %d", tt.customerID, delivery.CustomerID)
			}

			if delivery.PickupLocation != tt.pickupLocation {
				t.Errorf("expected pickup location %s, got %s", tt.pickupLocation, delivery.PickupLocation)
			}

			if delivery.DeliveryLocation != tt.deliveryLocation {
				t.Errorf("expected delivery location %s, got %s", tt.deliveryLocation, delivery.DeliveryLocation)
			}

			if delivery.Status != StatusPending {
				t.Errorf("expected status %s, got %s", StatusPending, delivery.Status)
			}

			if delivery.CourierID != nil {
				t.Error("expected courier ID to be nil")
			}

			if delivery.ScheduledDate != nil {
				t.Error("expected scheduled date to be nil")
			}

			if delivery.DeliveredDate != nil {
				t.Error("expected delivered date to be nil")
			}

			if delivery.Notes != "" {
				t.Errorf("expected empty notes, got %s", delivery.Notes)
			}

			if delivery.CreatedAt.IsZero() {
				t.Error("expected created at to be set")
			}

			if delivery.UpdatedAt.IsZero() {
				t.Error("expected updated at to be set")
			}
		})
	}
}

// TestDelivery_UpdateStatus tests the UpdateStatus method of Delivery
func TestDelivery_UpdateStatus(t *testing.T) {
	delivery := &Delivery{
		ID:         1,
		CustomerID: 1,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tests := []struct {
		name        string
		newStatus   string
		expectError bool
		expectedErr error
	}{
		{
			name:        "valid status change to assigned",
			newStatus:   StatusAssigned,
			expectError: false,
		},
		{
			name:        "valid status change to in_transit",
			newStatus:   StatusInTransit,
			expectError: false,
		},
		{
			name:        "valid status change to delivered",
			newStatus:   StatusDelivered,
			expectError: false,
		},
		{
			name:        "valid status change to cancelled",
			newStatus:   StatusCancelled,
			expectError: false,
		},
		{
			name:        "invalid status",
			newStatus:   "invalid_status",
			expectError: true,
			expectedErr: ErrInvalidStatus,
		},
		{
			name:        "empty status",
			newStatus:   "",
			expectError: true,
			expectedErr: ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalUpdatedAt := delivery.UpdatedAt
			originalDeliveredDate := delivery.DeliveredDate

			err := delivery.UpdateStatus(tt.newStatus)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.expectedErr != nil && err != tt.expectedErr {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if delivery.Status != tt.newStatus {
				t.Errorf("expected status %s, got %s", tt.newStatus, delivery.Status)
			}

			if !delivery.UpdatedAt.After(originalUpdatedAt) {
				t.Error("expected updated at to be updated")
			}

			if tt.newStatus == StatusDelivered {
				if delivery.DeliveredDate == nil {
					t.Error("expected delivered date to be set for delivered status")
				}
			} else {
				if delivery.DeliveredDate != originalDeliveredDate {
					t.Error("expected delivered date to remain unchanged for non-delivered status")
				}
			}
		})
	}
}

func TestDelivery_AssignCourier(t *testing.T) {
	delivery := &Delivery{
		ID:         1,
		CustomerID: 1,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tests := []struct {
		name        string
		courierID   int
		expectError bool
	}{
		{
			name:        "valid courier assignment",
			courierID:   2,
			expectError: false,
		},
		{
			name:        "invalid courier ID zero",
			courierID:   0,
			expectError: true,
		},
		{
			name:        "invalid courier ID negative",
			courierID:   -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalUpdatedAt := delivery.UpdatedAt
			originalCourierID := delivery.CourierID
			originalStatus := delivery.Status

			err := delivery.AssignCourier(tt.courierID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if delivery.CourierID != originalCourierID {
					t.Error("expected courier ID to remain unchanged on error")
				}
				if delivery.Status != originalStatus {
					t.Error("expected status to remain unchanged on error")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if delivery.CourierID == nil || *delivery.CourierID != tt.courierID {
				t.Errorf("expected courier ID %d, got %v", tt.courierID, delivery.CourierID)
			}

			if delivery.Status != StatusAssigned {
				t.Errorf("expected status %s, got %s", StatusAssigned, delivery.Status)
			}

			if !delivery.UpdatedAt.After(originalUpdatedAt) {
				t.Error("expected updated at to be updated")
			}
		})
	}
}

func TestDelivery_CanBeModifiedBy(t *testing.T) {
	delivery := &Delivery{
		ID:         1,
		CustomerID: 1,
		CourierID:  func() *int { i := 2; return &i }(),
		Status:     StatusPending,
	}

	tests := []struct {
		name       string
		role       string
		customerID *int
		courierID  *int
		expected   bool
	}{
		{
			name:     "admin can modify",
			role:     "admin",
			expected: true,
		},
		{
			name:       "customer can modify own delivery",
			role:       "customer",
			customerID: func() *int { i := 1; return &i }(),
			expected:   true,
		},
		{
			name:       "customer cannot modify other delivery",
			role:       "customer",
			customerID: func() *int { i := 999; return &i }(),
			expected:   false,
		},
		{
			name:      "customer with nil ID cannot modify",
			role:      "customer",
			customerID: nil,
			expected:  false,
		},
		{
			name:      "courier can modify assigned delivery",
			role:      "courier",
			courierID: func() *int { i := 2; return &i }(),
			expected:  true,
		},
		{
			name:      "courier cannot modify unassigned delivery",
			role:      "courier",
			courierID: func() *int { i := 999; return &i }(),
			expected:  false,
		},
		{
			name:     "courier with nil ID cannot modify",
			role:     "courier",
			courierID: nil,
			expected: false,
		},
		{
			name:     "unknown role cannot modify",
			role:     "unknown",
			expected: false,
		},
		{
			name:     "empty role cannot modify",
			role:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := delivery.CanBeModifiedBy(tt.role, tt.customerID, tt.courierID)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsValidStatus(t *testing.T) {
	validStatuses := []string{
		StatusPending,
		StatusAssigned,
		StatusInTransit,
		StatusDelivered,
		StatusCancelled,
	}

	invalidStatuses := []string{
		"",
		"invalid",
		"pending ", // with space
		"PENDING",  // uppercase
		"delivered2",
		"123",
	}

	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			// We can't test the private function directly, so we'll test it through UpdateStatus
			delivery := &Delivery{Status: StatusPending}
			err := delivery.UpdateStatus(status)
			if err != nil {
				t.Errorf("expected status %s to be valid, but got error: %v", status, err)
			}
		})
	}

	for _, status := range invalidStatuses {
		t.Run("invalid_"+status, func(t *testing.T) {
			delivery := &Delivery{Status: StatusPending}
			err := delivery.UpdateStatus(status)
			if err == nil {
				t.Errorf("expected status %s to be invalid, but no error was returned", status)
			}
		})
	}
}