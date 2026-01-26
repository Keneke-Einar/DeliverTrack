package app

import (
	"context"
	"fmt"

	"github.com/Keneke-Einar/delivertrack/pkg/auth/domain"
	"github.com/Keneke-Einar/delivertrack/pkg/auth/ports"
)

// AuthService implements authentication use cases
type AuthService struct {
	userRepo     ports.UserRepository
	tokenService ports.TokenService
}

// NewAuthService creates a new authentication service
func NewAuthService(userRepo ports.UserRepository, tokenService ports.TokenService) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}

// Register creates a new user account
func (s *AuthService) Register(
	ctx context.Context,
	username, email, password, role string,
	customerID, courierID *int,
) (*domain.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return nil, domain.ErrUserExists
	}

	existingUser, err = s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, domain.ErrUserExists
	}

	// Create new user (with validation)
	user, err := domain.NewUser(username, email, password, role, customerID, courierID)
	if err != nil {
		return nil, err
	}

	// Persist user
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Authenticate validates credentials and returns a token and user
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (string, *domain.User, error) {
	// Retrieve user
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive() {
		return "", nil, domain.ErrUnauthorized
	}

	// Verify password
	if err := user.VerifyPassword(password); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	// Generate token
	token, err := s.tokenService.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	return s.tokenService.ValidateToken(tokenString)
}

// GetUser retrieves a user by ID
func (s *AuthService) GetUser(ctx context.Context, id int) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
