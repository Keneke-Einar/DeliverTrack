package adapters

import (
	"fmt"
	"time"

	"github.com/delivertrack/auth/domain"
	"github.com/golang-jwt/jwt/v5"
)

// JWTTokenService implements the TokenService interface using JWT
type JWTTokenService struct {
	secret        []byte
	tokenDuration time.Duration
}

// NewJWTTokenService creates a new JWT token service
func NewJWTTokenService(secret string, tokenDuration time.Duration) *JWTTokenService {
	return &JWTTokenService{
		secret:        []byte(secret),
		tokenDuration: tokenDuration,
	}
}

// JWTClaims extends jwt.RegisteredClaims with custom fields
type JWTClaims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CustomerID *int   `json:"customer_id,omitempty"`
	CourierID  *int   `json:"courier_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
func (s *JWTTokenService) GenerateToken(user *domain.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(s.tokenDuration)

	claims := JWTClaims{
		UserID:     user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Role:       user.Role,
		CustomerID: user.CustomerID,
		CourierID:  user.CourierID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTTokenService) ValidateToken(tokenString string) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, domain.ErrExpiredToken
	}

	return &domain.Claims{
		UserID:     claims.UserID,
		Username:   claims.Username,
		Email:      claims.Email,
		Role:       claims.Role,
		CustomerID: claims.CustomerID,
		CourierID:  claims.CourierID,
	}, nil
}
