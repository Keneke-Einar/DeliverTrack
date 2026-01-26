package ports

import (
	"context"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
)

// AnalyticsService defines the analytics use cases
type AnalyticsService interface {
	// RecordMetric records a new analytics metric
	RecordMetric(ctx context.Context, metricType domain.MetricType, entityID int, entityType string, value float64, metadata map[string]interface{}) (*domain.Metric, error)

	// GetDeliveryStats retrieves delivery statistics for a period
	GetDeliveryStats(ctx context.Context, period string) (*domain.DeliveryStats, error)

	// GetMetricsByType retrieves metrics by type
	GetMetricsByType(ctx context.Context, metricType domain.MetricType, limit int) ([]*domain.Metric, error)
}
