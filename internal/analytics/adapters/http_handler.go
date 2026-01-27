package adapters

import (
	"encoding/json"
	"net/http"

	"github.com/Keneke-Einar/delivertrack/internal/analytics/domain"
	"github.com/Keneke-Einar/delivertrack/internal/analytics/ports"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type       string                 `json:"type"`
		EntityID   int                    `json:"entity_id"`
		EntityType string                 `json:"entity_type"`
		Value      float64                `json:"value"`
		Metadata   map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	metric, err := h.service.RecordMetric(r.Context(), domain.MetricType(req.Type), req.EntityID, req.EntityType, req.Value, req.Metadata)
	if err != nil {
		http.Error(w, "Failed to record metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metric)
}

// GetDeliveryStats handles GET /analytics/delivery-stats
func (h *HTTPHandler) GetDeliveryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "last_30_days"
	}

	stats, err := h.service.GetDeliveryStats(r.Context(), period)
	if err != nil {
		http.Error(w, "Failed to get delivery stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
