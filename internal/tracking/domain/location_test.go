package domain

import (
	"testing"
	"time"
)

func TestNewLocation(t *testing.T) {
	tests := []struct {
		name        string
		deliveryID  int
		courierID   int
		latitude    float64
		longitude   float64
		expectError bool
	}{
		{
			name:        "valid location",
			deliveryID:  1,
			courierID:   1,
			latitude:    40.7128,
			longitude:   -74.0060,
			expectError: false,
		},
		{
			name:        "invalid delivery ID",
			deliveryID:  0,
			courierID:   1,
			latitude:    40.7128,
			longitude:   -74.0060,
			expectError: true,
		},
		{
			name:        "invalid courier ID",
			deliveryID:  1,
			courierID:   0,
			latitude:    40.7128,
			longitude:   -74.0060,
			expectError: true,
		},
		{
			name:        "latitude too low",
			deliveryID:  1,
			courierID:   1,
			latitude:    -91.0,
			longitude:   -74.0060,
			expectError: true,
		},
		{
			name:        "latitude too high",
			deliveryID:  1,
			courierID:   1,
			latitude:    91.0,
			longitude:   -74.0060,
			expectError: true,
		},
		{
			name:        "longitude too low",
			deliveryID:  1,
			courierID:   1,
			latitude:    40.7128,
			longitude:   -181.0,
			expectError: true,
		},
		{
			name:        "longitude too high",
			deliveryID:  1,
			courierID:   1,
			latitude:    40.7128,
			longitude:   181.0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, err := NewLocation(tt.deliveryID, tt.courierID, tt.latitude, tt.longitude)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if location == nil {
					t.Errorf("expected location but got nil")
				}
				if location.DeliveryID != tt.deliveryID {
					t.Errorf("expected delivery ID %d, got %d", tt.deliveryID, location.DeliveryID)
				}
				if location.CourierID != tt.courierID {
					t.Errorf("expected courier ID %d, got %d", tt.courierID, location.CourierID)
				}
				if location.Latitude != tt.latitude {
					t.Errorf("expected latitude %f, got %f", tt.latitude, location.Latitude)
				}
				if location.Longitude != tt.longitude {
					t.Errorf("expected longitude %f, got %f", tt.longitude, location.Longitude)
				}
				if location.Timestamp.IsZero() {
					t.Errorf("expected timestamp to be set")
				}
				if location.CreatedAt.IsZero() {
					t.Errorf("expected created_at to be set")
				}
			}
		})
	}
}

func TestSetOptionalFields(t *testing.T) {
	location, err := NewLocation(1, 1, 40.7128, -74.0060)
	if err != nil {
		t.Fatalf("failed to create location: %v", err)
	}

	accuracy := 10.5
	speed := 25.0
	heading := 90.0
	altitude := 100.0

	location.SetOptionalFields(&accuracy, &speed, &heading, &altitude)

	if location.Accuracy == nil || *location.Accuracy != accuracy {
		t.Errorf("expected accuracy %f, got %v", accuracy, location.Accuracy)
	}
	if location.Speed == nil || *location.Speed != speed {
		t.Errorf("expected speed %f, got %v", speed, location.Speed)
	}
	if location.Heading == nil || *location.Heading != heading {
		t.Errorf("expected heading %f, got %v", heading, location.Heading)
	}
	if location.Altitude == nil || *location.Altitude != altitude {
		t.Errorf("expected altitude %f, got %v", altitude, location.Altitude)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name     string
		location *Location
		expected bool
	}{
		{
			name: "valid location",
			location: &Location{
				DeliveryID: 1,
				CourierID:  1,
				Latitude:   40.7128,
				Longitude:  -74.0060,
				Timestamp:  time.Now(),
				CreatedAt:  time.Now(),
			},
			expected: true,
		},
		{
			name: "invalid delivery ID",
			location: &Location{
				DeliveryID: 0,
				CourierID:  1,
				Latitude:   40.7128,
				Longitude:  -74.0060,
			},
			expected: false,
		},
		{
			name: "invalid courier ID",
			location: &Location{
				DeliveryID: 1,
				CourierID:  0,
				Latitude:   40.7128,
				Longitude:  -74.0060,
			},
			expected: false,
		},
		{
			name: "invalid latitude low",
			location: &Location{
				DeliveryID: 1,
				CourierID:  1,
				Latitude:   -91.0,
				Longitude:  -74.0060,
			},
			expected: false,
		},
		{
			name: "invalid latitude high",
			location: &Location{
				DeliveryID: 1,
				CourierID:  1,
				Latitude:   91.0,
				Longitude:  -74.0060,
			},
			expected: false,
		},
		{
			name: "invalid longitude low",
			location: &Location{
				DeliveryID: 1,
				CourierID:  1,
				Latitude:   40.7128,
				Longitude:  -181.0,
			},
			expected: false,
		},
		{
			name: "invalid longitude high",
			location: &Location{
				DeliveryID: 1,
				CourierID:  1,
				Latitude:   40.7128,
				Longitude:  181.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.location.IsValid()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}