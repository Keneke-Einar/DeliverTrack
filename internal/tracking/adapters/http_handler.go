package adapters

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Keneke-Einar/delivertrack/internal/tracking/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/grpcinterceptors"
	httputil "github.com/Keneke-Einar/delivertrack/pkg/http"
	"google.golang.org/grpc/metadata"
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
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ports.RecordLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DeliveryID == 0 || req.CourierID == 0 {
		httputil.SendErrorResponse(w, "delivery_id and courier_id are required", http.StatusBadRequest)
		return
	}

	// Get user context from auth middleware
	userCtx := httputil.ExtractUserContext(r)

	// Authorization: only couriers can record locations, and only their own
	if userCtx.Role != "courier" {
		httputil.SendErrorResponse(w, "Only couriers can record locations", http.StatusForbidden)
		return
	}

	if userCtx.CourierID == nil || *userCtx.CourierID != req.CourierID {
		httputil.SendErrorResponse(w, "Couriers can only record their own locations", http.StatusForbidden)
		return
	}

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "tracking-service", "record_location_http")

	// Record location
	location, err := h.service.RecordLocation(ctx, req)
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(location)
}

// GetDeliveryTrack handles GET /deliveries/{id}/track
func (h *HTTPHandler) GetDeliveryTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract delivery ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "track" {
		httputil.SendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		httputil.SendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
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
	userCtx := httputil.ExtractUserContext(r)

	// Authorization: customers can only track their own deliveries, couriers can track assigned deliveries
	if userCtx.Role == "customer" && userCtx.CustomerID != nil {
		// In a real implementation, we'd check if the delivery belongs to this customer
		// For now, we'll allow it
	} else if userCtx.Role == "courier" && userCtx.CourierID != nil {
		// In a real implementation, we'd check if the courier is assigned to this delivery
		// For now, we'll allow it
	}

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "tracking-service", "get_delivery_track_http")

	// Get delivery track
	locations, err := h.service.GetDeliveryTrack(ctx, ports.GetDeliveryTrackRequest{
		DeliveryID: deliveryID,
		Limit:      limit,
	})
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
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
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract delivery ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "location" {
		httputil.SendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		httputil.SendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Get user context
	userCtx := httputil.ExtractUserContext(r)

	// Authorization: customers can only track their own deliveries, couriers can track assigned deliveries
	if userCtx.Role == "customer" && userCtx.CustomerID != nil {
		// In a real implementation, we'd check if the delivery belongs to this customer
	} else if userCtx.Role == "courier" && userCtx.CourierID != nil {
		// In a real implementation, we'd check if the courier is assigned to this delivery
	}

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "tracking-service", "get_current_location_http")

	// Add authorization metadata for gRPC calls
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		md := metadata.Pairs(grpcinterceptors.AuthorizationMetadataKey, authHeader)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Get current location
	location, err := h.service.GetCurrentLocation(ctx, ports.GetCurrentLocationRequest{
		DeliveryID: deliveryID,
	})
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(location)
}

// GetCourierLocation handles GET /couriers/{id}/location
func (h *HTTPHandler) GetCourierLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract courier ID from path
	path := strings.TrimPrefix(r.URL.Path, "/couriers/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "location" {
		httputil.SendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	courierID, err := strconv.Atoi(parts[0])
	if err != nil {
		httputil.SendErrorResponse(w, "Invalid courier ID", http.StatusBadRequest)
		return
	}

	// Get user context
	userCtx := httputil.ExtractUserContext(r)

	// Authorization: couriers can only access their own location, admins can access any
	if userCtx.Role == "courier" && userCtx.CourierID != nil && *userCtx.CourierID != courierID {
		httputil.SendErrorResponse(w, "Couriers can only access their own location", http.StatusForbidden)
		return
	}

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "tracking-service", "get_courier_location_http")

	// Get courier location
	location, err := h.service.GetCourierLocation(ctx, ports.GetCourierLocationRequest{
		CourierID: courierID,
	})
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(location)
}

// CalculateETA handles POST /deliveries/{id}/eta
func (h *HTTPHandler) CalculateETA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user and trace context
	userCtx := httputil.ExtractUserContext(r)
	traceCtx := httputil.ExtractTraceContext(r, "tracking-service", "calculate_eta_http")

	// Extract delivery ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "eta" {
		httputil.SendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deliveryID, err := strconv.Atoi(parts[0])
	if err != nil {
		httputil.SendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	var req struct {
		DestLat float64 `json:"dest_lat"`
		DestLng float64 `json:"dest_lng"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DestLat == 0 && req.DestLng == 0 {
		httputil.SendErrorResponse(w, "dest_lat and dest_lng are required", http.StatusBadRequest)
		return
	}

	// Authorization: customers can calculate ETA for their deliveries, couriers for assigned deliveries
	if userCtx.Role == "customer" && userCtx.CustomerID != nil {
		// In a real implementation, we'd check if the delivery belongs to this customer
	} else if userCtx.Role == "courier" && userCtx.CourierID != nil {
		// In a real implementation, we'd check if the courier is assigned to this delivery
	}

	// Calculate ETA
	eta, err := h.service.CalculateETAToDestination(traceCtx, ports.CalculateETAToDestinationRequest{
		DeliveryID: deliveryID,
		DestLat:    req.DestLat,
		DestLng:    req.DestLng,
	})
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eta)
}
