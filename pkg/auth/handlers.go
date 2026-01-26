package auth

import (
	"encoding/json"
	"net/http"
)

// LoginRequest represents a login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	User      *User  `json:"user"`
	ExpiresIn int64  `json:"expires_in"` // seconds
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

// LoginHandler handles user login
func (s *AuthService) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	token, user, err := s.Authenticate(req.Username, req.Password)
	if err != nil {
		sendErrorResponse(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Send response
	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresIn: int64(s.config.TokenDuration.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterHandler handles user registration
func (s *AuthService) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	// Validate role
	if !IsValidRole(req.Role) {
		sendErrorResponse(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Create user
	user, err := s.CreateUser(req.Username, req.Email, req.Password, req.Role, req.CustomerID, req.CourierID)
	if err != nil {
		sendErrorResponse(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate token
	token, err := s.GenerateToken(user)
	if err != nil {
		sendErrorResponse(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Send response
	response := LoginResponse{
		Token:     token,
		User:      user,
		ExpiresIn: int64(s.config.TokenDuration.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// MeHandler returns the current user's information
func (s *AuthService) MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := GetClaimsFromContext(r.Context())
	if err != nil {
		sendErrorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.GetUserByID(claims.UserID)
	if err != nil {
		sendErrorResponse(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// sendErrorResponse sends a JSON error response
func sendErrorResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
