package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Role constants
const (
	RoleCustomer = "customer"
	RoleCourier  = "courier"
	RoleAdmin    = "admin"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden: insufficient permissions")
)

// User represents an authenticated user
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CustomerID   *int      `json:"customer_id,omitempty"`
	CourierID    *int      `json:"courier_id,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Claims represents JWT claims
type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CustomerID *int   `json:"customer_id,omitempty"`
	CourierID  *int   `json:"courier_id,omitempty"`
	jwt.RegisteredClaims
}

// Config holds authentication configuration
type Config struct {
	JWTSecret     []byte
	TokenDuration time.Duration
}

// AuthService handles authentication operations
type AuthService struct {
	db     *sql.DB
	config Config
}

// NewAuthService creates a new authentication service
func NewAuthService(db *sql.DB, secret string, tokenDuration time.Duration) *AuthService {
	return &AuthService{
		db: db,
		config: Config{
			JWTSecret:     []byte(secret),
			TokenDuration: tokenDuration,
		},
	}
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword checks if the provided password matches the hash
func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// CreateUser creates a new user with hashed password
func (s *AuthService) CreateUser(username, email, password, role string, customerID, courierID *int) (*User, error) {
	// Validate role
	if !IsValidRole(role) {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Insert user
	query := `
		INSERT INTO users (username, email, password_hash, role, customer_id, courier_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at, active
	`

	user := &User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CustomerID:   customerID,
		CourierID:    courierID,
	}

	err = s.db.QueryRow(query, username, email, passwordHash, role, customerID, courierID).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Active)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *AuthService) GetUserByUsername(username string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, customer_id, courier_id, active, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	user := &User{}
	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CustomerID,
		&user.CourierID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(userID int) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, customer_id, courier_id, active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := s.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CustomerID,
		&user.CourierID,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Authenticate verifies credentials and returns a JWT token
func (s *AuthService) Authenticate(username, password string) (string, *User, error) {
	// Get user by username
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.Active {
		return "", nil, ErrUnauthorized
	}

	// Verify password
	if err := VerifyPassword(password, user.PasswordHash); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	// Generate JWT token
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

// GenerateToken creates a JWT token for the user
func (s *AuthService) GenerateToken(user *User) (string, error) {
	expirationTime := time.Now().Add(s.config.TokenDuration)

	claims := &Claims{
		UserID:     user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Role:       user.Role,
		CustomerID: user.CustomerID,
		CourierID:  user.CourierID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.config.JWTSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.config.JWTSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrUnauthorized
	}

	// Expected format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrInvalidToken
	}

	return parts[1], nil
}

// IsValidRole checks if the role is valid
func IsValidRole(role string) bool {
	return role == RoleCustomer || role == RoleCourier || role == RoleAdmin
}

// HasRole checks if the user has the specified role
func HasRole(claims *Claims, role string) bool {
	return claims.Role == role
}

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(claims *Claims, roles ...string) bool {
	for _, role := range roles {
		if claims.Role == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if the user is an admin
func IsAdmin(claims *Claims) bool {
	return claims.Role == RoleAdmin
}

// CanAccessDelivery checks if the user can access a specific delivery
// Customers can only access their own deliveries
// Couriers can access deliveries assigned to them
// Admins can access all deliveries
func CanAccessDelivery(claims *Claims, customerID, courierID *int) bool {
	if IsAdmin(claims) {
		return true
	}

	if claims.Role == RoleCustomer && claims.CustomerID != nil && customerID != nil {
		return *claims.CustomerID == *customerID
	}

	if claims.Role == RoleCourier && claims.CourierID != nil && courierID != nil {
		return *claims.CourierID == *courierID
	}

	return false
}

// Context keys
type contextKey string

const (
	ContextKeyUser   contextKey = "user"
	ContextKeyClaims contextKey = "claims"
)

// GetClaimsFromContext retrieves claims from request context
func GetClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(ContextKeyClaims).(*Claims)
	if !ok {
		return nil, ErrUnauthorized
	}
	return claims, nil
}

// AuthMiddleware is a middleware that validates JWT tokens
func (s *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from header
		tokenString, err := ExtractTokenFromHeader(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := s.ValidateToken(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, ErrExpiredToken) {
				status = http.StatusUnauthorized
			}
			http.Error(w, err.Error(), status)
			return
		}

		// Add claims to context
		ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is a middleware that checks if the user has the required role
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetClaimsFromContext(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if !HasAnyRole(claims, roles...) {
				http.Error(w, ErrForbidden.Error(), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin is a middleware that requires admin role
func RequireAdmin(next http.Handler) http.Handler {
	return RequireRole(RoleAdmin)(next)
}

// RequireCourier is a middleware that requires courier role
func RequireCourier(next http.Handler) http.Handler {
	return RequireRole(RoleCourier)(next)
}

// RequireCustomer is a middleware that requires customer role
func RequireCustomer(next http.Handler) http.Handler {
	return RequireRole(RoleCustomer)(next)
}
