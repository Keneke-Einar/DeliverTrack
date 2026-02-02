package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SendErrorResponse sends a standardized JSON error response
func SendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}

// UserContext represents extracted user information from request context
type UserContext struct {
	Role       string
	CustomerID *int
	CourierID  *int
}

// ExtractUserContext extracts user information from request context
func ExtractUserContext(r *http.Request) UserContext {
	userRole, _ := r.Context().Value("role").(string)
	customerID, _ := r.Context().Value("customer_id").(*int)
	courierID, _ := r.Context().Value("courier_id").(*int)

	return UserContext{
		Role:       userRole,
		CustomerID: customerID,
		CourierID:  courierID,
	}
}

// ExtractTraceContext extracts trace context from HTTP headers and creates a context
func ExtractTraceContext(r *http.Request, serviceName, operation string) context.Context {
	// Extract trace context from headers
	traceID := r.Header.Get("X-Trace-ID")
	spanID := r.Header.Get("X-Span-ID")
	parentSpanID := r.Header.Get("X-Parent-Span-ID")

	// Generate new IDs if not present
	if traceID == "" {
		traceID = messaging.GenerateTraceID()
	}
	if spanID == "" {
		spanID = messaging.GenerateSpanID()
	}

	// Create trace context
	traceCtx := &messaging.TraceContext{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		ServiceName:  serviceName,
		Operation:    operation,
	}

	// Add to context
	return messaging.ContextWithTraceContext(r.Context(), traceCtx)
}