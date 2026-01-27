package adapters

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
)

// PostgresMetricRepository implements the MetricRepository interface using PostgreSQL
type PostgresMetricRepository struct {
	db *sql.DB
}

// NewPostgresMetricRepository creates a new PostgreSQL repository
func NewPostgresMetricRepository(db *sql.DB) *PostgresMetricRepository {
	return &PostgresMetricRepository{db: db}
}

// Create stores a new metric
func (r *PostgresMetricRepository) Create(ctx context.Context, metric *domain.Metric) error {
	metadataJSON, err := json.Marshal(metric.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO metrics (type, entity_id, entity_type, value, metadata, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err = r.db.QueryRowContext(
		ctx,
		query,
		metric.Type,
		metric.EntityID,
		metric.EntityType,
		metric.Value,
		metadataJSON,
		metric.Timestamp,
		metric.CreatedAt,
	).Scan(&metric.ID)

	return err
}

// GetByType retrieves metrics by type
func (r *PostgresMetricRepository) GetByType(ctx context.Context, metricType domain.MetricType, limit int) ([]*domain.Metric, error) {
	query := `
		SELECT id, type, entity_id, entity_type, value, metadata, timestamp, created_at
		FROM metrics
		WHERE type = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, metricType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*domain.Metric
	for rows.Next() {
		var metric domain.Metric
		var metadataJSON []byte

		err := rows.Scan(
			&metric.ID,
			&metric.Type,
			&metric.EntityID,
			&metric.EntityType,
			&metric.Value,
			&metadataJSON,
			&metric.Timestamp,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			err = json.Unmarshal(metadataJSON, &metric.Metadata)
			if err != nil {
				return nil, err
			}
		}

		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

// GetByID retrieves a metric by ID
func (r *PostgresMetricRepository) GetByID(ctx context.Context, id int) (*domain.Metric, error) {
	query := `
		SELECT id, type, entity_id, entity_type, value, metadata, timestamp, created_at
		FROM metrics
		WHERE id = $1
	`

	var metric domain.Metric
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&metric.ID,
		&metric.Type,
		&metric.EntityID,
		&metric.EntityType,
		&metric.Value,
		&metadataJSON,
		&metric.Timestamp,
		&metric.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		err = json.Unmarshal(metadataJSON, &metric.Metadata)
		if err != nil {
			return nil, err
		}
	}

	return &metric, nil
}

// GetByEntityID retrieves metrics for a specific entity
func (r *PostgresMetricRepository) GetByEntityID(ctx context.Context, entityID int, entityType string, limit int) ([]*domain.Metric, error) {
	query := `
		SELECT id, type, entity_id, entity_type, value, metadata, timestamp, created_at
		FROM metrics
		WHERE entity_id = $1 AND entity_type = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, entityID, entityType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*domain.Metric
	for rows.Next() {
		var metric domain.Metric
		var metadataJSON []byte

		err := rows.Scan(
			&metric.ID,
			&metric.Type,
			&metric.EntityID,
			&metric.EntityType,
			&metric.Value,
			&metadataJSON,
			&metric.Timestamp,
			&metric.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			err = json.Unmarshal(metadataJSON, &metric.Metadata)
			if err != nil {
				return nil, err
			}
		}

		metrics = append(metrics, &metric)
	}

	return metrics, nil
}

// GetDeliveryStats retrieves delivery statistics
func (r *PostgresMetricRepository) GetDeliveryStats(ctx context.Context, period string) (*domain.DeliveryStats, error) {
	// Simple implementation - count metrics
	query := `
		SELECT
			COUNT(*) as total_deliveries,
			COUNT(CASE WHEN type = 'delivery_completed' THEN 1 END) as completed_deliveries,
			COUNT(CASE WHEN type = 'delivery_cancelled' THEN 1 END) as cancelled_deliveries
		FROM metrics
		WHERE type IN ('delivery_created', 'delivery_completed', 'delivery_cancelled')
	`

	var stats domain.DeliveryStats
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalDeliveries,
		&stats.CompletedDeliveries,
		&stats.CancelledDeliveries,
	)
	if err != nil {
		return nil, err
	}

	stats.PendingDeliveries = stats.TotalDeliveries - stats.CompletedDeliveries - stats.CancelledDeliveries
	stats.AverageDeliveryTime = 0.0 // TODO: calculate from data
	stats.Period = period

	return &stats, nil
}