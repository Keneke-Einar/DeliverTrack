package adapters

import (
	"encoding/json"
	"net/http"

	"github.com/Keneke-Einar/delivertrack/internal/notification/domain"
	"github.com/Keneke-Einar/delivertrack/internal/notification/ports"
	httputil "github.com/Keneke-Einar/delivertrack/pkg/http"
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
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user and trace context
	traceCtx := httputil.ExtractTraceContext(r, "notification-service", "send_notification_http")

	var req struct {
		UserID    int    `json:"user_id"`
		Type      string `json:"type"`
		Subject   string `json:"subject"`
		Message   string `json:"message"`
		Recipient string `json:"recipient"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	notification, err := h.service.SendNotification(traceCtx, req.UserID, domain.NotificationType(req.Type), req.Subject, req.Message, req.Recipient)
	if err != nil {
		httputil.SendErrorResponse(w, "Failed to send notification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// GetNotifications handles GET /notifications
func (h *HTTPHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user and trace context
	traceCtx := httputil.ExtractTraceContext(r, "notification-service", "get_notifications_http")

	// Get user ID from context (set by auth middleware)
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		httputil.SendErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := userIDValue.(int)
	if !ok {
		httputil.SendErrorResponse(w, "Invalid user context", http.StatusInternalServerError)
		return
	}

	notifications, err := h.service.GetUserNotifications(traceCtx, userID, 10)
	if err != nil {
		httputil.SendErrorResponse(w, "Failed to get notifications", http.StatusInternalServerError)
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
		httputil.SendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user and trace context
	traceCtx := httputil.ExtractTraceContext(r, "notification-service", "mark_as_read_http")

	var req struct {
		NotificationID int `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.MarkAsRead(traceCtx, req.NotificationID)
	if err != nil {
		httputil.SendErrorResponse(w, "Failed to mark as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
