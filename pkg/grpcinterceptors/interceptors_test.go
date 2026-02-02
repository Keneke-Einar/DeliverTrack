package grpcinterceptors_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Keneke-Einar/delivertrack/pkg/auth/domain"
	"github.com/Keneke-Einar/delivertrack/pkg/grpcinterceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Mock auth service for testing
type mockAuthService struct {
	validateTokenFunc func(ctx context.Context, tokenString string) (*domain.Claims, error)
}

func (m *mockAuthService) Register(ctx context.Context, username, email, password, role string, customerID, courierID *int) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockAuthService) Authenticate(ctx context.Context, username, password string) (token string, user *domain.User, err error) {
	return "", nil, errors.New("not implemented")
}

func (m *mockAuthService) ValidateToken(ctx context.Context, tokenString string) (*domain.Claims, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, tokenString)
	}
	return nil, domain.ErrInvalidToken
}

func (m *mockAuthService) GetUser(ctx context.Context, id int) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func TestAuthUnaryServerInterceptor_ValidToken(t *testing.T) {
	mockAuth := &mockAuthService{
		validateTokenFunc: func(ctx context.Context, tokenString string) (*domain.Claims, error) {
			if tokenString == "valid-token" {
				return &domain.Claims{
					UserID:   1,
					Username: "testuser",
					Role:     domain.RoleCustomer,
				}, nil
			}
			return nil, domain.ErrInvalidToken
		},
	}

	interceptor := grpcinterceptors.AuthUnaryServerInterceptor(mockAuth)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		claims, ok := grpcinterceptors.GetUserClaimsFromContext(ctx)
		if !ok {
			t.Error("Expected user claims in context")
			return nil, errors.New("no claims")
		}
		if claims.UserID != 1 || claims.Username != "testuser" {
			t.Errorf("Unexpected claims: %+v", claims)
		}
		return "success", nil
	}

	md := metadata.Pairs("authorization", "Bearer valid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	result, err := interceptor(ctx, "test-req", &grpc.UnaryServerInfo{FullMethod: "/test.Service/Test"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return handler(ctx, req)
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got: %v", result)
	}
}

func TestAuthUnaryServerInterceptor_InvalidToken(t *testing.T) {
	mockAuth := &mockAuthService{
		validateTokenFunc: func(ctx context.Context, tokenString string) (*domain.Claims, error) {
			return nil, domain.ErrInvalidToken
		},
	}

	interceptor := grpcinterceptors.AuthUnaryServerInterceptor(mockAuth)

	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, "test-req", &grpc.UnaryServerInfo{FullMethod: "/test.Service/Test"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "should not reach here", nil
	})

	if err == nil {
		t.Error("Expected authentication error")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated error, got: %v", err)
	}
}

func TestAuthUnaryServerInterceptor_MissingAuthHeader(t *testing.T) {
	mockAuth := &mockAuthService{}

	interceptor := grpcinterceptors.AuthUnaryServerInterceptor(mockAuth)

	ctx := context.Background()

	_, err := interceptor(ctx, "test-req", &grpc.UnaryServerInfo{FullMethod: "/test.Service/Test"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return "should not reach here", nil
	})

	if err == nil {
		t.Error("Expected authentication error")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated error, got: %v", err)
	}
}

func TestErrorHandlingUnaryServerInterceptor(t *testing.T) {
	interceptor := grpcinterceptors.ErrorHandlingUnaryServerInterceptor()

	tests := []struct {
		name           string
		handlerError   error
		expectedCode   codes.Code
		expectedMsgContains string
	}{
		{
			name:           "domain invalid token",
			handlerError:   domain.ErrInvalidToken,
			expectedCode:   codes.Unauthenticated,
			expectedMsgContains: "invalid token",
		},
		{
			name:           "domain unauthorized",
			handlerError:   domain.ErrUnauthorized,
			expectedCode:   codes.PermissionDenied,
			expectedMsgContains: "unauthorized",
		},
		{
			name:           "generic error",
			handlerError:   errors.New("some internal error"),
			expectedCode:   codes.Internal,
			expectedMsgContains: "some internal error",
		},
		{
			name:           "grpc status error preserved",
			handlerError:   status.Error(codes.NotFound, "not found"),
			expectedCode:   codes.NotFound,
			expectedMsgContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interceptor(context.Background(), "test-req", &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, tt.handlerError
			})

			if err == nil {
				t.Error("Expected error")
				return
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Errorf("Expected gRPC status error, got: %v", err)
				return
			}

			if st.Code() != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, st.Code())
			}

			if !strings.Contains(st.Message(), tt.expectedMsgContains) {
				t.Errorf("Expected message to contain %q, got %q", tt.expectedMsgContains, st.Message())
			}
		})
	}
}

func TestGetUserClaimsFromContext(t *testing.T) {
	claims := &domain.Claims{
		UserID:   123,
		Username: "testuser",
		Role:     domain.RoleAdmin,
	}

	ctx := context.WithValue(context.Background(), grpcinterceptors.UserClaimsContextKey, claims)

	retrievedClaims, ok := grpcinterceptors.GetUserClaimsFromContext(ctx)
	if !ok {
		t.Error("Expected to find claims in context")
	}

	if retrievedClaims.UserID != claims.UserID ||
		retrievedClaims.Username != claims.Username ||
		retrievedClaims.Role != claims.Role {
		t.Errorf("Retrieved claims don't match: got %+v, want %+v", retrievedClaims, claims)
	}
}

func TestGetUserClaimsFromContext_NoClaims(t *testing.T) {
	ctx := context.Background()

	_, ok := grpcinterceptors.GetUserClaimsFromContext(ctx)
	if ok {
		t.Error("Expected no claims in context")
	}
}