package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
	"github.com/Keneke-Einar/delivertrack/internal/analytics/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
)

// AnalyticsService implements analytics use cases
type AnalyticsService struct {
	repo     ports.MetricRepository
	consumer messaging.Consumer
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo ports.MetricRepository, consumer messaging.Consumer) *AnalyticsService {
	return &AnalyticsService{
		repo:     repo,
		consumer: consumer,
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

// StartEventConsumption starts consuming delivery events
func (s *AnalyticsService) StartEventConsumption() error {
	return s.consumer.Consume("analytics-delivery-events", s.handleDeliveryEvent)
}

// handleDeliveryEvent processes incoming delivery events
func (s *AnalyticsService) handleDeliveryEvent(event messaging.Event) error {
	ctx := context.Background()

	switch event.Type {
	case "delivery.created":
		return s.handleDeliveryCreated(ctx, event)
	case "delivery.status_changed":
		return s.handleDeliveryStatusChanged(ctx, event)
	default:
		// Ignore unknown event types
		return nil
	}
}

// handleDeliveryCreated processes delivery creation events
func (s *AnalyticsService) handleDeliveryCreated(ctx context.Context, event messaging.Event) error {
	deliveryIDStr, ok := event.Data["delivery_id"].(string)
	if !ok {
		return fmt.Errorf("invalid delivery_id in event data")
	}

	deliveryID, err := strconv.Atoi(deliveryIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse delivery_id: %w", err)
	}

	customerIDStr, ok := event.Data["customer_id"].(string)
	if !ok {
		return fmt.Errorf("invalid customer_id in event data")
	}

	customerID, err := strconv.Atoi(customerIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse customer_id: %w", err)
	}

	// Record delivery creation metric
	_, err = s.RecordMetric(ctx, domain.MetricTypeDeliveryCreated, deliveryID, "delivery", 1.0, map[string]interface{}{
		"customer_id": customerID,
		"source":      event.Source,
	})
	if err != nil {
		return fmt.Errorf("failed to record delivery creation metric: %w", err)
	}

	// Record customer activity metric
	_, err = s.RecordMetric(ctx, domain.MetricTypeCustomerActivity, customerID, "customer", 1.0, map[string]interface{}{
		"activity_type": "delivery_created",
		"delivery_id":   deliveryID,
		"source":        event.Source,
	})
	if err != nil {
		return fmt.Errorf("failed to record customer activity metric: %w", err)
	}

	return nil
}

// handleDeliveryStatusChanged processes delivery status change events
func (s *AnalyticsService) handleDeliveryStatusChanged(ctx context.Context, event messaging.Event) error {
	deliveryIDStr, ok := event.Data["delivery_id"].(string)
	if !ok {
		return fmt.Errorf("invalid delivery_id in event data")
	}

	deliveryID, err := strconv.Atoi(deliveryIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse delivery_id: %w", err)
	}

	oldStatus, _ := event.Data["old_status"].(string)
	newStatus, ok := event.Data["new_status"].(string)
	if !ok {
		return fmt.Errorf("invalid new_status in event data")
	}

	customerIDStr, ok := event.Data["customer_id"].(string)
	if !ok {
		return fmt.Errorf("invalid customer_id in event data")
	}

	customerID, err := strconv.Atoi(customerIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse customer_id: %w", err)
	}

	courierIDStr, ok := event.Data["courier_id"].(string)
	if !ok {
		return fmt.Errorf("invalid courier_id in event data")
	}

	courierID, err := strconv.Atoi(courierIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse courier_id: %w", err)
	}

	// Record delivery status change metric
	_, err = s.RecordMetric(ctx, domain.MetricTypeDeliveryStatusChanged, deliveryID, "delivery", 1.0, map[string]interface{}{
		"old_status":  oldStatus,
		"new_status":  newStatus,
		"customer_id": customerID,
		"courier_id":  courierID,
		"source":      event.Source,
	})
	if err != nil {
		return fmt.Errorf("failed to record delivery status change metric: %w", err)
	}

	// Record delivery completion if status is completed
	if newStatus == "completed" {
		_, err = s.RecordMetric(ctx, domain.MetricTypeDeliveryCompleted, deliveryID, "delivery", 1.0, map[string]interface{}{
			"customer_id": customerID,
			"courier_id":  courierID,
			"source":      event.Source,
		})
		if err != nil {
			return fmt.Errorf("failed to record delivery completion metric: %w", err)
		}
	}

	return nil
}
