package ports

import (
	"context"

	"github.com/delivertrack/auth/domain"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Register creates a new user account
	Register(ctx context.Context, username, email, password, role string, customerID, courierID *int) (*domain.User, error)

	// Authenticate validates credentials and returns a token and user
	Authenticate(ctx context.Context, username, password string) (token string, user *domain.User, err error)

	// ValidateToken validates a JWT token and returns the claims
	ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error)

	// GetUser retrieves a user by ID
	GetUser(ctx context.Context, id int) (*domain.User, error)
}

// TokenService defines the interface for JWT token operations
type TokenService interface {
	// GenerateToken creates a new JWT token for a user
	GenerateToken(user *domain.User) (string, error)

	// ValidateToken validates a JWT token and returns the claims
	ValidateToken(tokenString string) (*domain.Claims, error)
}
