package geocoding

import (
	"encoding/json"
	"net/http"

	httputil "github.com/Keneke-Einar/delivertrack/pkg/http"
)

// HTTPHandler handles geocoding HTTP requests
type HTTPHandler struct {
	service GeocodingService
}

// NewHTTPHandler creates a new geocoding HTTP handler
func NewHTTPHandler(service GeocodingService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// ForwardGeocode handles POST /geocode/forward
func (h *HTTPHandler) ForwardGeocode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Address == "" {
		httputil.SendErrorResponse(w, "address is required", http.StatusBadRequest)
		return
	}

	ctx := httputil.ExtractTraceContext(r, "geocoding-service", "forward_geocode")
	result, err := h.service.ForwardGeocode(ctx, req.Address)
	if err != nil {
		httputil.SendErrorResponse(w, "Address not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ReverseGeocode handles POST /geocode/reverse
func (h *HTTPHandler) ReverseGeocode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := httputil.ExtractTraceContext(r, "geocoding-service", "reverse_geocode")
	result, err := h.service.ReverseGeocode(ctx, req.Latitude, req.Longitude)
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Autocomplete handles GET /geocode/autocomplete?q=query
func (h *HTTPHandler) Autocomplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		httputil.SendErrorResponse(w, "q parameter is required", http.StatusBadRequest)
		return
	}

	ctx := httputil.ExtractTraceContext(r, "geocoding-service", "autocomplete")
	results, err := h.service.Autocomplete(ctx, query)
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}
