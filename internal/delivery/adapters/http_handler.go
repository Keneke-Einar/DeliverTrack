package adapters

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Keneke-Einar/delivertrack/internal/delivery/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
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

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// CreateDelivery handles POST /deliveries
func (h *HTTPHandler) CreateDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ports.CreateDeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CustomerID == 0 || req.PickupLocation == "" || req.DeliveryLocation == "" {
		sendErrorResponse(w, "customer_id, pickup_location, and delivery_location are required", http.StatusBadRequest)
		return
	}

	// Get user context from auth middleware
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)

	// Authorization: customers can only create their own deliveries
	if userRole == "customer" && customerID != nil && *customerID != req.CustomerID {
		sendErrorResponse(w, "Customers can only create their own deliveries", http.StatusForbidden)
		return
	}

	// Create trace context for request tracing
	ctx := messaging.ContextWithTraceContext(r.Context(), &messaging.TraceContext{
		TraceID:    generateTraceID(),
		SpanID:     generateSpanID(),
		ServiceName: "delivery-service",
		Operation:  "create_delivery_http",
	})

	// Create delivery
	delivery, err := h.service.CreateDelivery(ctx, req)
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(delivery)
}

// GetDelivery handles GET /deliveries/:id
func (h *HTTPHandler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	id, err := strconv.Atoi(path)
	if err != nil {
		sendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Get delivery
	delivery, err := h.service.GetDelivery(r.Context(), ports.GetDeliveryRequest{
		ID: id,
		AuthContext: ports.AuthContext{
			Role:           userRole,
			UserCustomerID: customerID,
			UserCourierID:  courierID,
		},
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "unauthorized access" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "delivery not found" {
			statusCode = http.StatusNotFound
		}
		sendErrorResponse(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(delivery)
}

// ListDeliveries handles GET /deliveries
func (h *HTTPHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	status := r.URL.Query().Get("status")
	customerIDParam := r.URL.Query().Get("customer_id")

	var filterCustomerID int
	if customerIDParam != "" {
		var err error
		filterCustomerID, err = strconv.Atoi(customerIDParam)
		if err != nil {
			sendErrorResponse(w, "Invalid customer_id", http.StatusBadRequest)
			return
		}
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// List deliveries
	deliveries, err := h.service.ListDeliveries(r.Context(), ports.ListDeliveriesRequest{
		Status:     status,
		CustomerID: filterCustomerID,
		AuthContext: ports.AuthContext{
			Role:           userRole,
			UserCustomerID: customerID,
			UserCourierID:  courierID,
		},
	})
	if err != nil {
		sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deliveries)
}

// UpdateDeliveryStatus handles PUT /deliveries/:id/status
func (h *HTTPHandler) UpdateDeliveryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
	path = strings.TrimSuffix(path, "/status")
	id, err := strconv.Atoi(path)
	if err != nil {
		sendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		sendErrorResponse(w, "status is required", http.StatusBadRequest)
		return
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Update status
	err = h.service.UpdateDeliveryStatus(r.Context(), ports.UpdateDeliveryStatusRequest{
		ID:     id,
		Status: req.Status,
		Notes:  req.Notes,
		AuthContext: ports.AuthContext{
			Role:           userRole,
			UserCustomerID: customerID,
			UserCourierID:  courierID,
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
		sendErrorResponse(w, err.Error(), statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Status updated successfully"})
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

// Helper functions for trace ID generation
func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
