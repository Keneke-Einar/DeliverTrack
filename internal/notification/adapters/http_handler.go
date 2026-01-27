package adapters

import (
	"encoding/json"
	"net/http"

	"github.com/Keneke-Einar/delivertrack/internal/notification/domain"
	"github.com/Keneke-Einar/delivertrack/internal/notification/ports"
)

// HTTPHandler handles HTTP requests for notification operations
type HTTPHandler struct {
	service ports.NotificationService
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service ports.NotificationService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// SendNotification handles POST /notifications/send
func (h *HTTPHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID    int    `json:"user_id"`
		Type      string `json:"type"`
		Subject   string `json:"subject"`
		Message   string `json:"message"`
		Recipient string `json:"recipient"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	notification, err := h.service.SendNotification(r.Context(), req.UserID, domain.NotificationType(req.Type), req.Subject, req.Message, req.Recipient)
	if err != nil {
		http.Error(w, "Failed to send notification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// GetNotifications handles GET /notifications
func (h *HTTPHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	userID := 0 // parse, but for now assume

	notifications, err := h.service.GetUserNotifications(r.Context(), userID, 10)
	if err != nil {
		http.Error(w, "Failed to get notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// GetUserNotifications handles GET /notifications/user
func (h *HTTPHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	h.GetNotifications(w, r)
}

// MarkAsRead handles POST /notifications/mark-read
func (h *HTTPHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NotificationID int `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.MarkAsRead(r.Context(), req.NotificationID)
	if err != nil {
		http.Error(w, "Failed to mark as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
