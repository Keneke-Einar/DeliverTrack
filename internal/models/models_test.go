package models

import (
	"testing"
	"time"
)

func TestPackageStatusConstants(t *testing.T) {
	tests := []struct {
		status   PackageStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusInTransit, "in_transit"},
		{StatusOutForDelivery, "out_for_delivery"},
		{StatusDelivered, "delivered"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

func TestPackageModel(t *testing.T) {
	pkg := Package{
		ID:             "test-id",
		TrackingNumber: "TRK123",
		SenderName:     "John Doe",
		RecipientName:  "Jane Smith",
		Status:         string(StatusPending),
		Weight:         2.5,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if pkg.TrackingNumber != "TRK123" {
		t.Errorf("Expected tracking number TRK123, got %s", pkg.TrackingNumber)
	}

	if pkg.Status != "pending" {
		t.Errorf("Expected status pending, got %s", pkg.Status)
	}

	if pkg.Weight != 2.5 {
		t.Errorf("Expected weight 2.5, got %f", pkg.Weight)
	}
}

func TestLocationModel(t *testing.T) {
	loc := Location{
		PackageID: "test-pkg-id",
		Latitude:  40.7357,
		Longitude: -74.1724,
		Address:   "Newark, NJ",
		Timestamp: time.Now(),
	}

	if loc.Latitude != 40.7357 {
		t.Errorf("Expected latitude 40.7357, got %f", loc.Latitude)
	}

	if loc.Longitude != -74.1724 {
		t.Errorf("Expected longitude -74.1724, got %f", loc.Longitude)
	}
}
