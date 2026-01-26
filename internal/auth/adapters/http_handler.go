package adapters

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/delivertrack/auth/domain"
	"github.com/delivertrack/auth/ports"
)

// HTTPHandler handles HTTP requests for authentication operations
type HTTPHandler struct {
	authService   ports.AuthService
	tokenDuration time.Duration
}

// NewHTTPHandler creates a new HTTP handler for authentication
func NewHTTPHandler(authService ports.AuthService, tokenDuration time.Duration) *HTTPHandler {
	return &HTTPHandler{
		authService:   authService,
		tokenDuration: tokenDuration,
	}
}

// LoginRequest represents a login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string             `json:"token"`
	User      *domain.PublicUser `json:"user"`
	ExpiresIn int64              `json:"expires_in"` // seconds
}

// RegisterRequest represents a registration request payload
type RegisterRequest struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	CustomerID *int   `json:"customer_id,omitempty"`
	CourierID  *int   `json:"courier_id,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Login handles POST /login
func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		sendErrorResponse(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	token, user, err := h.authService.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		sendErrorResponse(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Send response
	response := LoginResponse{
		Token:     token,
		User:      user.ToPublicUser(),
		ExpiresIn: int64(h.tokenDuration.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Register handles POST /register
func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		sendErrorResponse(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Create user
	user, err := h.authService.Register(
		r.Context(),
		req.Username,
		req.Email,
		req.Password,
		req.Role,
		req.CustomerID,
		req.CourierID,
	)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == domain.ErrUserExists {
			statusCode = http.StatusConflict
		} else if err == domain.ErrInvalidRole || err == domain.ErrInvalidUserData {
			statusCode = http.StatusBadRequest
		}
		sendErrorResponse(w, err.Error(), statusCode)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user.ToPublicUser())
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
