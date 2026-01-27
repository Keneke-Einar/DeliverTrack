package adapters

import (
	"context"
	"strconv"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
	"github.com/Keneke-Einar/delivertrack/internal/analytics/ports"
	analyticsProto "github.com/Keneke-Einar/delivertrack/proto/analytics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler handles gRPC requests for analytics operations
type GRPCHandler struct {
	analyticsProto.UnimplementedAnalyticsServiceServer
	service ports.AnalyticsService
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(service ports.AnalyticsService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// RecordEvent implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) RecordEvent(ctx context.Context, req *analyticsProto.RecordEventRequest) (*analyticsProto.RecordEventResponse, error) {
	entityID, err := strconv.Atoi(req.EntityId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid entity_id: %v", err)
	}

	// Convert properties map[string]string to map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range req.Properties {
		metadata[k] = v
	}

	metric, err := h.service.RecordMetric(ctx, domain.MetricType(req.EventType), entityID, req.EntityType, 1.0, metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record event: %v", err)
	}

	return &analyticsProto.RecordEventResponse{
		Success:    true,
		RecordedAt: metric.Timestamp.Unix(),
	}, nil
}

// GetDeliveryMetrics implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GetDeliveryMetrics(ctx context.Context, req *analyticsProto.GetDeliveryMetricsRequest) (*analyticsProto.GetDeliveryMetricsResponse, error) {
	period := "last_30_days" // default
	if req.TimeRange != nil {
		period = "custom" // or construct from time_range
	}

	stats, err := h.service.GetDeliveryStats(ctx, period)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get delivery metrics: %v", err)
	}

	return &analyticsProto.GetDeliveryMetricsResponse{
		Metrics: &analyticsProto.DeliveryMetrics{
			TotalDeliveries:     int32(stats.TotalDeliveries),
			SuccessfulDeliveries: int32(stats.CompletedDeliveries),
			FailedDeliveries:    int32(stats.CancelledDeliveries),
			CancelledDeliveries: int32(stats.CancelledDeliveries),
			SuccessRate:         float64(stats.CompletedDeliveries) / float64(stats.TotalDeliveries) * 100,
			AverageDeliveryTime: float64(stats.AverageDeliveryTime),
			OnTimeDeliveries:    0,
			OnTimeRate:          0,
		},
	}, nil
}

// BatchRecordEvents implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) BatchRecordEvents(ctx context.Context, req *analyticsProto.BatchRecordEventsRequest) (*analyticsProto.BatchRecordEventsResponse, error) {
	// TODO: Implement batch recording
	return nil, status.Errorf(codes.Unimplemented, "method BatchRecordEvents not implemented")
}

// GetDriverPerformance implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GetDriverPerformance(ctx context.Context, req *analyticsProto.GetDriverPerformanceRequest) (*analyticsProto.GetDriverPerformanceResponse, error) {
	// TODO: Implement when service supports driver performance
	return nil, status.Errorf(codes.Unimplemented, "method GetDriverPerformance not implemented")
}

// GetCustomerAnalytics implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GetCustomerAnalytics(ctx context.Context, req *analyticsProto.GetCustomerAnalyticsRequest) (*analyticsProto.GetCustomerAnalyticsResponse, error) {
	// TODO: Implement when service supports customer analytics
	return nil, status.Errorf(codes.Unimplemented, "method GetCustomerAnalytics not implemented")
}

// GetSystemMetrics implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GetSystemMetrics(ctx context.Context, req *analyticsProto.GetSystemMetricsRequest) (*analyticsProto.GetSystemMetricsResponse, error) {
	// TODO: Implement when service supports system metrics
	return nil, status.Errorf(codes.Unimplemented, "method GetSystemMetrics not implemented")
}

// GenerateReport implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GenerateReport(ctx context.Context, req *analyticsProto.GenerateReportRequest) (*analyticsProto.GenerateReportResponse, error) {
	// TODO: Implement when service supports report generation
	return nil, status.Errorf(codes.Unimplemented, "method GenerateReport not implemented")
}

// GetDashboard implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GetDashboard(ctx context.Context, req *analyticsProto.GetDashboardRequest) (*analyticsProto.GetDashboardResponse, error) {
	// TODO: Implement when service supports dashboard
	return nil, status.Errorf(codes.Unimplemented, "method GetDashboard not implemented")
}

// GetRouteEfficiency implements analytics.AnalyticsServiceServer
func (h *GRPCHandler) GetRouteEfficiency(ctx context.Context, req *analyticsProto.GetRouteEfficiencyRequest) (*analyticsProto.GetRouteEfficiencyResponse, error) {
	// TODO: Implement when service supports route efficiency
	return nil, status.Errorf(codes.Unimplemented, "method GetRouteEfficiency not implemented")
}
