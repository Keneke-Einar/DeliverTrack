package domain

import (
	"errors"
	"fmt"
	"time"

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
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidUserData    = errors.New("invalid user data")
)

// User represents an authenticated user in the domain
type User struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string
	Role         string
	CustomerID   *int
	CourierID    *int
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser creates a new user with validation
func NewUser(username, email, password, role string, customerID, courierID *int) (*User, error) {
	// Validate required fields
	if username == "" || email == "" || password == "" || role == "" {
		return nil, ErrInvalidUserData
	}

	// Validate role
	if !IsValidRole(role) {
		return nil, ErrInvalidRole
	}

	// Hash password
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	return &User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		CustomerID:   customerID,
		CourierID:    courierID,
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// VerifyPassword checks if the provided password matches the user's hash
func (u *User) VerifyPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}

// IsActive checks if the user account is active
func (u *User) IsActive() bool {
	return u.Active
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role string) bool {
	return u.Role == role
}

// CanAccessResource checks if user can access a resource based on role and ownership
func (u *User) CanAccessResource(requiredRole string, resourceCustomerID, resourceCourierID *int) bool {
	// Admin can access everything
	if u.Role == RoleAdmin {
		return true
	}

	// Check role match
	if requiredRole != "" && u.Role != requiredRole {
		return false
	}

	// Check customer ownership
	if resourceCustomerID != nil && u.CustomerID != nil {
		return *u.CustomerID == *resourceCustomerID
	}

	// Check courier ownership
	if resourceCourierID != nil && u.CourierID != nil {
		return *u.CourierID == *resourceCourierID
	}

	return false
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// IsValidRole checks if a role is valid
func IsValidRole(role string) bool {
	return role == RoleCustomer || role == RoleCourier || role == RoleAdmin
}

// Claims represents JWT claims for authorization
type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CustomerID *int   `json:"customer_id,omitempty"`
	CourierID  *int   `json:"courier_id,omitempty"`
}

// ToPublicUser returns a user without sensitive information
func (u *User) ToPublicUser() *PublicUser {
	return &PublicUser{
		ID:         u.ID,
		Username:   u.Username,
		Email:      u.Email,
		Role:       u.Role,
		CustomerID: u.CustomerID,
		CourierID:  u.CourierID,
		Active:     u.Active,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

// PublicUser represents a user without sensitive fields
type PublicUser struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	CustomerID *int      `json:"customer_id,omitempty"`
	CourierID  *int      `json:"courier_id,omitempty"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
