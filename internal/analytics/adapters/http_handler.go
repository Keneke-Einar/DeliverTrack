package adapters

import (
	"encoding/json"
	"net/http"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
	"github.com/Keneke-Einar/delivertrack/internal/analytics/ports"
	httputil "github.com/Keneke-Einar/delivertrack/pkg/http"
)

// HTTPHandler handles HTTP requests for analytics operations
type HTTPHandler struct {
	service ports.AnalyticsService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service ports.AnalyticsService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RecordMetric handles POST /analytics/metrics
func (h *HTTPHandler) RecordMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract trace context
	traceCtx := httputil.ExtractTraceContext(r, "analytics-service", "record_metric_http")

	var req struct {
		Type       string                 `json:"type"`
		EntityID   int                    `json:"entity_id"`
		EntityType string                 `json:"entity_type"`
		Value      float64                `json:"value"`
		Metadata   map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	metric, err := h.service.RecordMetric(traceCtx, domain.MetricType(req.Type), req.EntityID, req.EntityType, req.Value, req.Metadata)
	if err != nil {
		httputil.SendErrorResponse(w, "Failed to record metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metric)
}

// GetDeliveryStats handles GET /analytics/delivery-stats
func (h *HTTPHandler) GetDeliveryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract trace context
	traceCtx := httputil.ExtractTraceContext(r, "analytics-service", "get_delivery_stats_http")

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "last_30_days"
	}

	stats, err := h.service.GetDeliveryStats(traceCtx, period)
	if err != nil {
		httputil.SendErrorResponse(w, "Failed to get delivery stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
