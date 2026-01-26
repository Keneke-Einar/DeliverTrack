package adapters

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/keneke/delivertrack/internal/delivery/ports"
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

// CreateDeliveryRequest represents the request payload for creating a delivery
type CreateDeliveryRequest struct {
	CustomerID       int     `json:"customer_id"`
	CourierID        *int    `json:"courier_id,omitempty"`
	PickupLocation   string  `json:"pickup_location"`
	DeliveryLocation string  `json:"delivery_location"`
	ScheduledDate    *string `json:"scheduled_date,omitempty"`
	Notes            string  `json:"notes,omitempty"`
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

	var req CreateDeliveryRequest
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

	// Create delivery
	delivery, err := h.service.CreateDelivery(
		r.Context(),
		req.CustomerID,
		req.CourierID,
		req.PickupLocation,
		req.DeliveryLocation,
		req.Notes,
		req.ScheduledDate,
	)
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
	delivery, err := h.service.GetDelivery(r.Context(), id, userRole, customerID, courierID)
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
	deliveries, err := h.service.ListDeliveries(r.Context(), status, filterCustomerID, userRole, customerID, courierID)
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
	err = h.service.UpdateDeliveryStatus(r.Context(), id, req.Status, req.Notes, userRole, customerID, courierID)
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
