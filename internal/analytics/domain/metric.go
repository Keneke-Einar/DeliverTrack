package domain

import (
	"errors"
	"time"
)

var (
	ErrAnalyticsNotFound = errors.New("analytics data not found")
	ErrInvalidMetric     = errors.New("invalid metric data")
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeDeliveryCreated   MetricType = "delivery_created"
	MetricTypeDeliveryCompleted MetricType = "delivery_completed"
	MetricTypeDeliveryCancelled MetricType = "delivery_cancelled"
	MetricTypeUserRegistered    MetricType = "user_registered"
	MetricTypeLoginAttempt      MetricType = "login_attempt"
)

// Metric represents an analytics metric
type Metric struct {
	ID         int
	Type       MetricType
	EntityID   int
	EntityType string
	Value      float64
	Metadata   map[string]interface{}
	Timestamp  time.Time
	CreatedAt  time.Time
}

// NewMetric creates a new metric with validation
func NewMetric(metricType MetricType, entityID int, entityType string, value float64) (*Metric, error) {
	if entityID <= 0 || entityType == "" {
		return nil, ErrInvalidMetric
	}

	return &Metric{
		Type:       metricType,
		EntityID:   entityID,
		EntityType: entityType,
		Value:      value,
		Metadata:   make(map[string]interface{}),
		Timestamp:  time.Now(),
		CreatedAt:  time.Now(),
	}, nil
}

// AddMetadata adds metadata to the metric
func (m *Metric) AddMetadata(key string, value interface{}) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
}

// DeliveryStats represents delivery statistics
type DeliveryStats struct {
	TotalDeliveries     int
	CompletedDeliveries int
	PendingDeliveries   int
	CancelledDeliveries int
	AverageDeliveryTime float64
	Period              string
}
