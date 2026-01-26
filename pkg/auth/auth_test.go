package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock database for testing
type mockDB struct{}

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify the hash
	if err := VerifyPassword(password, hash); err != nil {
		t.Errorf("Password verification failed: %v", err)
	}

	// Verify wrong password fails
	if err := VerifyPassword("wrongpassword", hash); err == nil {
		t.Error("Wrong password should not verify")
	}
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{RoleCustomer, true},
		{RoleCourier, true},
		{RoleAdmin, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsValidRole(tt.role)
		if result != tt.valid {
			t.Errorf("IsValidRole(%q) = %v, want %v", tt.role, result, tt.valid)
		}
	}
}

func TestHasRole(t *testing.T) {
	claims := &Claims{
		UserID: 1,
		Role:   RoleCustomer,
	}

	if !HasRole(claims, RoleCustomer) {
		t.Error("HasRole should return true for customer role")
	}

	if HasRole(claims, RoleAdmin) {
		t.Error("HasRole should return false for admin role")
	}
}

func TestHasAnyRole(t *testing.T) {
	claims := &Claims{
		UserID: 1,
		Role:   RoleCourier,
	}

	if !HasAnyRole(claims, RoleCustomer, RoleCourier) {
		t.Error("HasAnyRole should return true")
	}

	if HasAnyRole(claims, RoleCustomer, RoleAdmin) {
		t.Error("HasAnyRole should return false")
	}
}

func TestIsAdmin(t *testing.T) {
	adminClaims := &Claims{UserID: 1, Role: RoleAdmin}
	if !IsAdmin(adminClaims) {
		t.Error("IsAdmin should return true for admin")
	}

	customerClaims := &Claims{UserID: 2, Role: RoleCustomer}
	if IsAdmin(customerClaims) {
		t.Error("IsAdmin should return false for customer")
	}
}

func TestCanAccessDelivery(t *testing.T) {
	customerID := 1
	courierID := 2

	tests := []struct {
		name        string
		claims      *Claims
		customerID  *int
		courierID   *int
		canAccess   bool
		description string
	}{
		{
			name:        "Admin can access any delivery",
			claims:      &Claims{UserID: 1, Role: RoleAdmin},
			customerID:  &customerID,
			courierID:   &courierID,
			canAccess:   true,
			description: "Admin should access all deliveries",
		},
		{
			name:        "Customer can access own delivery",
			claims:      &Claims{UserID: 2, Role: RoleCustomer, CustomerID: &customerID},
			customerID:  &customerID,
			courierID:   &courierID,
			canAccess:   true,
			description: "Customer should access their own delivery",
		},
		{
			name:        "Customer cannot access other's delivery",
			claims:      &Claims{UserID: 2, Role: RoleCustomer, CustomerID: func() *int { v := 99; return &v }()},
			customerID:  &customerID,
			courierID:   &courierID,
			canAccess:   false,
			description: "Customer should not access other's delivery",
		},
		{
			name:        "Courier can access assigned delivery",
			claims:      &Claims{UserID: 3, Role: RoleCourier, CourierID: &courierID},
			customerID:  &customerID,
			courierID:   &courierID,
			canAccess:   true,
			description: "Courier should access assigned delivery",
		},
		{
			name:        "Courier cannot access unassigned delivery",
			claims:      &Claims{UserID: 3, Role: RoleCourier, CourierID: func() *int { v := 99; return &v }()},
			customerID:  &customerID,
			courierID:   &courierID,
			canAccess:   false,
			description: "Courier should not access unassigned delivery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanAccessDelivery(tt.claims, tt.customerID, tt.courierID)
			if result != tt.canAccess {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.canAccess)
			}
		})
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key", 24*time.Hour)

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     RoleCustomer,
	}

	// Generate token
	tokenString, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if tokenString == "" {
		t.Error("Token should not be empty")
	}

	// Validate token
	claims, err := service.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %d, want %d", claims.UserID, user.ID)
	}

	if claims.Username != user.Username {
		t.Errorf("Username mismatch: got %s, want %s", claims.Username, user.Username)
	}

	if claims.Role != user.Role {
		t.Errorf("Role mismatch: got %s, want %s", claims.Role, user.Role)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key", -1*time.Hour) // Expired immediately

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     RoleCustomer,
	}

	tokenString, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = service.ValidateToken(tokenString)
	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got: %v", err)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key", 24*time.Hour)

	_, err := service.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantError bool
	}{
		{
			name:      "Valid Bearer token",
			header:    "Bearer abc123xyz",
			wantToken: "abc123xyz",
			wantError: false,
		},
		{
			name:      "Missing token",
			header:    "",
			wantToken: "",
			wantError: true,
		},
		{
			name:      "Invalid format",
			header:    "abc123xyz",
			wantToken: "",
			wantError: true,
		},
		{
			name:      "Wrong prefix",
			header:    "Token abc123xyz",
			wantToken: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			token, err := ExtractTokenFromHeader(req)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractTokenFromHeader() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if token != tt.wantToken {
				t.Errorf("ExtractTokenFromHeader() token = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key", 24*time.Hour)

	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     RoleCustomer,
	}

	token, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			authHeader:     "Bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing token",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with auth middleware
			wrappedHandler := service.AuthMiddleware(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key", 24*time.Hour)

	tests := []struct {
		name           string
		userRole       string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "Admin accessing admin endpoint",
			userRole:       RoleAdmin,
			requiredRoles:  []string{RoleAdmin},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Customer accessing customer endpoint",
			userRole:       RoleCustomer,
			requiredRoles:  []string{RoleCustomer},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Customer accessing admin endpoint",
			userRole:       RoleCustomer,
			requiredRoles:  []string{RoleAdmin},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Courier accessing any of multiple allowed roles",
			userRole:       RoleCourier,
			requiredRoles:  []string{RoleAdmin, RoleCourier},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user and token
			user := &User{
				ID:       1,
				Username: "testuser",
				Email:    "test@example.com",
				Role:     tt.userRole,
			}

			token, _ := service.GenerateToken(user)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middlewares
			wrappedHandler := service.AuthMiddleware(RequireRole(tt.requiredRoles...)(handler))

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			// Record response
			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	claims := &Claims{
		UserID:   1,
		Username: "testuser",
		Role:     RoleCustomer,
	}

	// Test with claims in context
	ctx := context.WithValue(context.Background(), ContextKeyClaims, claims)
	retrievedClaims, err := GetClaimsFromContext(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if retrievedClaims.UserID != claims.UserID {
		t.Errorf("UserID mismatch: got %d, want %d", retrievedClaims.UserID, claims.UserID)
	}

	// Test without claims in context
	emptyCtx := context.Background()
	_, err = GetClaimsFromContext(emptyCtx)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got: %v", err)
	}
}

func TestLoginHandler(t *testing.T) {
	service := NewAuthService(nil, "test-secret-key", 24*time.Hour)

	tests := []struct {
		name           string
		method         string
		body           LoginRequest
		expectedStatus int
	}{
		{
			name:   "Invalid method",
			method: http.MethodGet,
			body: LoginRequest{
				Username: "testuser",
				Password: "password",
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "Missing username",
			method: http.MethodPost,
			body: LoginRequest{
				Username: "",
				Password: "password",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Missing password",
			method: http.MethodPost,
			body: LoginRequest{
				Username: "testuser",
				Password: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(tt.method, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			service.LoginHandler(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
