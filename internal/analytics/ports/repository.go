package ports

import (
	"context"

	"github.com/keneke/delivertrack/internal/analytics/domain"
)

// MetricRepository defines the analytics metric persistence operations
type MetricRepository interface {
	// Create stores a new metric
	Create(ctx context.Context, metric *domain.Metric) error

	// GetByID retrieves a metric by ID
	GetByID(ctx context.Context, id int) (*domain.Metric, error)

	// GetByType retrieves metrics by type
	GetByType(ctx context.Context, metricType domain.MetricType, limit int) ([]*domain.Metric, error)

	// GetByEntityID retrieves metrics for a specific entity
	GetByEntityID(ctx context.Context, entityID int, entityType string, limit int) ([]*domain.Metric, error)

	// GetDeliveryStats retrieves aggregated delivery statistics
	GetDeliveryStats(ctx context.Context, period string) (*domain.DeliveryStats, error)
}
