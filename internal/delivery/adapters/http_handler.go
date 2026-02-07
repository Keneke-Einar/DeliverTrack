package adapters

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
	httputil "github.com/Keneke-Einar/delivertrack/pkg/http"
)

// HTTPHandler handles HTTP requests for delivery operations
type HTTPHandler struct {
	service ports.DeliveryService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service ports.DeliveryService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// UpdateStatusRequest represents the request payload for updating delivery status
type UpdateStatusRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes,omitempty"`
}

// CreateDelivery handles POST /deliveries
func (h *HTTPHandler) CreateDelivery(w http.ResponseWriter, r *http.Request) {

	var req ports.CreateDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CustomerID == 0 || req.PickupLocation == "" || req.DeliveryLocation == "" {
		httputil.SendErrorResponse(w, "customer_id, pickup_location, and delivery_location are required", http.StatusBadRequest)
		return
	}

	// Get user context from auth middleware
	userCtx := httputil.ExtractUserContext(r)

	// Authorization: customers can only create their own deliveries
	if userCtx.Role == "customer" && userCtx.CustomerID != nil && *userCtx.CustomerID != req.CustomerID {
		httputil.SendErrorResponse(w, "Customers can only create their own deliveries", http.StatusForbidden)
		return
	}

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "delivery-service", "create_delivery_http")

	// Create delivery
	delivery, err := h.service.CreateDelivery(ctx, req)
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(delivery)
}

// GetDelivery handles GET /deliveries/:id
func (h *HTTPHandler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	id, err := strconv.Atoi(path)
	if err != nil {
		httputil.SendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Get user context
	userCtx := httputil.ExtractUserContext(r)

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "delivery-service", "get_delivery_http")

	// Get delivery
	delivery, err := h.service.GetDelivery(ctx, ports.GetDeliveryRequest{
		ID: id,
		AuthContext: ports.AuthContext{
			Role:           userCtx.Role,
			UserCustomerID: userCtx.CustomerID,
			UserCourierID:  userCtx.CourierID,
		},
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "unauthorized access" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "delivery not found" {
			statusCode = http.StatusNotFound
		}
		httputil.SendErrorResponse(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(delivery)
}

// ListDeliveries handles GET /deliveries
func (h *HTTPHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {

	// Get query parameters
	status := r.URL.Query().Get("status")
	customerIDParam := r.URL.Query().Get("customer_id")

	var filterCustomerID int
	if customerIDParam != "" {
		var err error
		filterCustomerID, err = strconv.Atoi(customerIDParam)
		if err != nil {
			httputil.SendErrorResponse(w, "Invalid customer_id", http.StatusBadRequest)
			return
		}
	}

	// Get user context
	userCtx := httputil.ExtractUserContext(r)

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "delivery-service", "list_deliveries_http")

	// List deliveries
	deliveries, err := h.service.ListDeliveries(ctx, ports.ListDeliveriesRequest{
		Status:     status,
		CustomerID: filterCustomerID,
		AuthContext: ports.AuthContext{
			Role:           userCtx.Role,
			UserCustomerID: userCtx.CustomerID,
			UserCourierID:  userCtx.CourierID,
		},
	})
	if err != nil {
		httputil.SendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deliveries)
}

// UpdateDeliveryStatus handles PUT /deliveries/:id/status
func (h *HTTPHandler) UpdateDeliveryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	path = strings.TrimSuffix(path, "/status")
	id, err := strconv.Atoi(path)
	if err != nil {
		httputil.SendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		httputil.SendErrorResponse(w, "status is required", http.StatusBadRequest)
		return
	}

	// Get user context
	userCtx := httputil.ExtractUserContext(r)

	// Create trace context for request tracing
	ctx := httputil.ExtractTraceContext(r, "delivery-service", "update_delivery_status_http")

	// Update status
	err = h.service.UpdateDeliveryStatus(ctx, ports.UpdateDeliveryStatusRequest{
		ID:     id,
		Status: req.Status,
		Notes:  req.Notes,
		AuthContext: ports.AuthContext{
			Role:           userCtx.Role,
			UserCustomerID: userCtx.CustomerID,
			UserCourierID:  userCtx.CourierID,
		},
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "unauthorized access" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "delivery not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "invalid delivery status" {
			statusCode = http.StatusBadRequest
		}
		httputil.SendErrorResponse(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Status updated successfully"})
}

