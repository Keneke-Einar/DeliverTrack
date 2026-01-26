package adapters

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
)

// HTTPHandler handles HTTP requests for tracking operations
type HTTPHandler struct {
	service ports.TrackingService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service ports.TrackingService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RecordLocation handles POST /locations
func (h *HTTPHandler) RecordLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ports.RecordLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DeliveryID == 0 || req.CourierID == 0 {
		sendErrorResponse(w, "delivery_id and courier_id are required", http.StatusBadRequest)
		return
	}

	// Get user context from auth middleware
	userRole, _ := r.Context().Value("role").(string)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Authorization: only couriers can record locations, and only their own
	if userRole != "courier" {
		sendErrorResponse(w, "Only couriers can record locations", http.StatusForbidden)
		return
	}

	if courierID == nil || *courierID != req.CourierID {
		sendErrorResponse(w, "Couriers can only record their own locations", http.StatusForbidden)
		return
	}

	// Record location
	location, err := h.service.RecordLocation(r.Context(), req)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(location)
}

// GetDeliveryTrack handles GET /deliveries/{id}/track
func (h *HTTPHandler) GetDeliveryTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract delivery ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "track" {
		sendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		sendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Get limit from query params
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Authorization: customers can only track their own deliveries, couriers can track assigned deliveries
	if userRole == "customer" && customerID != nil {
		// In a real implementation, we'd check if the delivery belongs to this customer
		// For now, we'll allow it
	} else if userRole == "courier" && courierID != nil {
		// In a real implementation, we'd check if the courier is assigned to this delivery
		// For now, we'll allow it
	}

	// Get delivery track
	locations, err := h.service.GetDeliveryTrack(r.Context(), ports.GetDeliveryTrackRequest{
		DeliveryID: deliveryID,
		Limit:      limit,
	})
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"delivery_id": deliveryID,
		"locations":   locations,
	})
}

// GetCurrentLocation handles GET /deliveries/{id}/location
func (h *HTTPHandler) GetCurrentLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract delivery ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "location" {
		sendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		sendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Authorization: customers can only track their own deliveries, couriers can track assigned deliveries
	if userRole == "customer" && customerID != nil {
		// In a real implementation, we'd check if the delivery belongs to this customer
	} else if userRole == "courier" && courierID != nil {
		// In a real implementation, we'd check if the courier is assigned to this delivery
	}

	// Get current location
	location, err := h.service.GetCurrentLocation(r.Context(), ports.GetCurrentLocationRequest{
		DeliveryID: deliveryID,
	})
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(location)
}

// GetCourierLocation handles GET /couriers/{id}/location
func (h *HTTPHandler) GetCourierLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract courier ID from path
	path := strings.TrimPrefix(r.URL.Path, "/couriers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "location" {
		sendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	courierID, err := strconv.Atoi(parts[0])
	if err != nil {
		sendErrorResponse(w, "Invalid courier ID", http.StatusBadRequest)
		return
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	userCourierID, _ := r.Context().Value("courier_id").(*int)

	// Authorization: couriers can only access their own location, admins can access any
	if userRole == "courier" && userCourierID != nil && *userCourierID != courierID {
		sendErrorResponse(w, "Couriers can only access their own location", http.StatusForbidden)
		return
	}

	// Get courier location
	location, err := h.service.GetCourierLocation(r.Context(), ports.GetCourierLocationRequest{
		CourierID: courierID,
	})
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(location)
}

// CalculateETA handles POST /deliveries/{id}/eta
func (h *HTTPHandler) CalculateETA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract delivery ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "eta" {
		sendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		sendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	var req struct {
		DestLat float64 `json:"dest_lat"`
		DestLng float64 `json:"dest_lng"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DestLat == 0 && req.DestLng == 0 {
		sendErrorResponse(w, "dest_lat and dest_lng are required", http.StatusBadRequest)
		return
	}

	// Get user context for authorization
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Authorization: customers can calculate ETA for their deliveries, couriers for assigned deliveries
	if userRole == "customer" && customerID != nil {
		// In a real implementation, we'd check if the delivery belongs to this customer
	} else if userRole == "courier" && courierID != nil {
		// In a real implementation, we'd check if the courier is assigned to this delivery
	}

	// Calculate ETA
	eta, err := h.service.CalculateETAToDestination(r.Context(), ports.CalculateETAToDestinationRequest{
		DeliveryID: deliveryID,
		DestLat:    req.DestLat,
		DestLng:    req.DestLng,
	})
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eta)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// sendErrorResponse sends a JSON error response
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}