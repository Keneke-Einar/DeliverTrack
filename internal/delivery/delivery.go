package delivery

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Delivery struct {
	ID               int            `json:"id"`
	CustomerID       int            `json:"customer_id"`
	CourierID        sql.NullInt64  `json:"courier_id,omitempty"`
	Status           string         `json:"status"`
	PickupLocation   sql.NullString `json:"pickup_location,omitempty"`
	DeliveryLocation sql.NullString `json:"delivery_location,omitempty"`
	ScheduledDate    sql.NullTime   `json:"scheduled_date,omitempty"`
	DeliveredDate    sql.NullTime   `json:"delivered_date,omitempty"`
	Notes            sql.NullString `json:"notes,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
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

// Service handles delivery operations
type Service struct {
	db *sql.DB
}

// NewService creates a new delivery service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Create inserts a new delivery record
func (d *Delivery) Create(db interface {
	QueryRow(string, ...interface{}) *sql.Row
},
) error {
	query := `
		INSERT INTO deliveries (customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return db.QueryRow(query, d.CustomerID, d.CourierID, d.Status, d.PickupLocation, d.DeliveryLocation, d.ScheduledDate, d.Notes).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

// GetByID retrieves a delivery by ID
func GetByID(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, id int,
) (*Delivery, error) {
	var d Delivery
	query := `SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, delivered_date, notes, created_at, updated_at FROM deliveries WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&d.ID, &d.CustomerID, &d.CourierID, &d.Status, &d.PickupLocation, &d.DeliveryLocation, &d.ScheduledDate, &d.DeliveredDate, &d.Notes, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateStatus updates the status of a delivery
func UpdateStatus(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}, id int, status, notes string,
) error {
	query := `
		UPDATE deliveries 
		SET status = $1, notes = COALESCE(NULLIF($2, ''), notes), updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING id
	`
	var returnedID int
	err := db.QueryRow(query, status, notes, id).Scan(&returnedID)
	if err == sql.ErrNoRows {
		return errors.New("delivery not found")
	}
	return err
}

// GetByStatus retrieves deliveries by status with optional customer filter
func GetByStatus(db interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}, status string, customerID int,
) ([]*Delivery, error) {
	var query string
	var rows *sql.Rows
	var err error

	if customerID > 0 {
		query = `SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, delivered_date, notes, created_at, updated_at 
		         FROM deliveries WHERE status = $1 AND customer_id = $2 ORDER BY created_at DESC`
		rows, err = db.Query(query, status, customerID)
	} else {
		query = `SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, delivered_date, notes, created_at, updated_at 
		         FROM deliveries WHERE status = $1 ORDER BY created_at DESC`
		rows, err = db.Query(query, status)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		var d Delivery
		err := rows.Scan(&d.ID, &d.CustomerID, &d.CourierID, &d.Status, &d.PickupLocation, &d.DeliveryLocation, &d.ScheduledDate, &d.DeliveredDate, &d.Notes, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &d)
	}

	return deliveries, rows.Err()
}

// GetAll retrieves all deliveries with optional customer filter
func GetAll(db interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}, customerID int,
) ([]*Delivery, error) {
	var query string
	var rows *sql.Rows
	var err error

	if customerID > 0 {
		query = `SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, delivered_date, notes, created_at, updated_at 
		         FROM deliveries WHERE customer_id = $1 ORDER BY created_at DESC`
		rows, err = db.Query(query, customerID)
	} else {
		query = `SELECT id, customer_id, courier_id, status, pickup_location, delivery_location, scheduled_date, delivered_date, notes, created_at, updated_at 
		         FROM deliveries ORDER BY created_at DESC`
		rows, err = db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		var d Delivery
		err := rows.Scan(&d.ID, &d.CustomerID, &d.CourierID, &d.Status, &d.PickupLocation, &d.DeliveryLocation, &d.ScheduledDate, &d.DeliveredDate, &d.Notes, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &d)
	}

	return deliveries, rows.Err()
}

// HTTP Handlers

// CreateDeliveryHandler handles POST /deliveries
func (s *Service) CreateDeliveryHandler(w http.ResponseWriter, r *http.Request) {
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
		sendErrorResponse(w, "You can only create deliveries for yourself", http.StatusForbidden)
		return
	}

	// Create delivery object
	delivery := &Delivery{
		CustomerID: req.CustomerID,
		Status:     "pending",
	}

	if req.CourierID != nil {
		delivery.CourierID = sql.NullInt64{Int64: int64(*req.CourierID), Valid: true}
	}

	delivery.PickupLocation = sql.NullString{String: req.PickupLocation, Valid: true}
	delivery.DeliveryLocation = sql.NullString{String: req.DeliveryLocation, Valid: true}

	if req.ScheduledDate != nil && *req.ScheduledDate != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ScheduledDate)
		if err != nil {
			sendErrorResponse(w, "Invalid scheduled_date format. Use RFC3339", http.StatusBadRequest)
			return
		}
		delivery.ScheduledDate = sql.NullTime{Time: parsedTime, Valid: true}
	}

	if req.Notes != "" {
		delivery.Notes = sql.NullString{String: req.Notes, Valid: true}
	}

	// Insert into database
	if err := delivery.Create(s.db); err != nil {
		sendErrorResponse(w, fmt.Sprintf("Failed to create delivery: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"delivery": delivery,
		"message":  "Delivery created successfully",
	})
}

// GetDeliveryHandler handles GET /deliveries/:id
func (s *Service) GetDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		sendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(pathParts[len(pathParts)-1])
	if err != nil {
		sendErrorResponse(w, "Invalid delivery ID", http.StatusBadRequest)
		return
	}

	// Get delivery from database
	delivery, err := GetByID(s.db, id)
	if err == sql.ErrNoRows {
		sendErrorResponse(w, "Delivery not found", http.StatusNotFound)
		return
	}
	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("Failed to retrieve delivery: %v", err), http.StatusInternalServerError)
		return
	}

	// Authorization: customers can only view their own deliveries
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)

	if userRole == "customer" && (customerID == nil || delivery.CustomerID != *customerID) {
		sendErrorResponse(w, "You can only view your own deliveries", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(delivery)
}

// UpdateDeliveryStatusHandler handles PUT /deliveries/:id/status
func (s *Service) UpdateDeliveryStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		sendErrorResponse(w, "Invalid path", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(pathParts[len(pathParts)-2])
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
		sendErrorResponse(w, "Status is required", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := []string{"pending", "picked_up", "in_transit", "delivered", "cancelled"}
	isValid := false
	for _, vs := range validStatuses {
		if req.Status == vs {
			isValid = true
			break
		}
	}
	if !isValid {
		sendErrorResponse(w, "Invalid status. Must be one of: pending, picked_up, in_transit, delivered, cancelled", http.StatusBadRequest)
		return
	}

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	courierID, _ := r.Context().Value("courier_id").(*int)

	// Authorization: only couriers and admins can update status
	if userRole == "customer" {
		sendErrorResponse(w, "Customers cannot update delivery status", http.StatusForbidden)
		return
	}

	// Verify courier is assigned to this delivery (for courier role)
	if userRole == "courier" && courierID != nil {
		delivery, err := GetByID(s.db, id)
		if err != nil {
			sendErrorResponse(w, "Delivery not found", http.StatusNotFound)
			return
		}
		if !delivery.CourierID.Valid || int(delivery.CourierID.Int64) != *courierID {
			sendErrorResponse(w, "You can only update deliveries assigned to you", http.StatusForbidden)
			return
		}
	}

	// Update status
	if err := UpdateStatus(s.db, id, req.Status, req.Notes); err != nil {
		if err.Error() == "delivery not found" {
			sendErrorResponse(w, "Delivery not found", http.StatusNotFound)
		} else {
			sendErrorResponse(w, fmt.Sprintf("Failed to update status: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Delivery status updated successfully",
	})
}

// ListDeliveriesHandler handles GET /deliveries (with optional ?status=xxx query)
func (s *Service) ListDeliveriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	status := r.URL.Query().Get("status")

	// Get user context
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)

	var deliveries []*Delivery
	var err error

	// Determine which deliveries to fetch based on role
	filterCustomerID := 0
	if userRole == "customer" && customerID != nil {
		filterCustomerID = *customerID
	}

	if status != "" {
		deliveries, err = GetByStatus(s.db, status, filterCustomerID)
	} else {
		deliveries, err = GetAll(s.db, filterCustomerID)
	}

	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("Failed to retrieve deliveries: %v", err), http.StatusInternalServerError)
		return
	}

	if deliveries == nil {
		deliveries = []*Delivery{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(deliveries),
		"deliveries": deliveries,
	})
}

// Helper function to send error responses
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "error",
		"message": message,
	})
}
