package app

import (
	"context"
	"fmt"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
	"github.com/Keneke-Einar/delivertrack/internal/analytics/ports"
)

// AnalyticsService implements analytics use cases
type AnalyticsService struct {
	repo ports.MetricRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo ports.MetricRepository) *AnalyticsService {
	return &AnalyticsService{
		repo: repo,
	}
}

// RecordMetric records a new analytics metric
func (s *AnalyticsService) RecordMetric(
	ctx context.Context,
	metricType domain.MetricType,
	entityID int,
	entityType string,
	value float64,
	metadata map[string]interface{},
) (*domain.Metric, error) {
	// Create domain entity with validation
	metric, err := domain.NewMetric(metricType, entityID, entityType, value)
	if err != nil {
		return nil, err
	}

	// Add metadata if provided
	for key, val := range metadata {
		metric.AddMetadata(key, val)
	}

	// Persist to repository
	if err := s.repo.Create(ctx, metric); err != nil {
		return nil, fmt.Errorf("failed to record metric: %w", err)
	}

	return metric, nil
}

// GetDeliveryStats retrieves delivery statistics for a period
func (s *AnalyticsService) GetDeliveryStats(ctx context.Context, period string) (*domain.DeliveryStats, error) {
	if period == "" {
		period = "day" // default period
	}

	return s.repo.GetDeliveryStats(ctx, period)
}

// GetMetricsByType retrieves metrics by type
func (s *AnalyticsService) GetMetricsByType(ctx context.Context, metricType domain.MetricType, limit int) ([]*domain.Metric, error) {
	if limit <= 0 {
		limit = 100 // default limit
	}

	return s.repo.GetByType(ctx, metricType, limit)
}
