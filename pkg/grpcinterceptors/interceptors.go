package grpcinterceptors

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Keneke-Einar/delivertrack/pkg/auth/domain"
	"github.com/Keneke-Einar/delivertrack/pkg/auth/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TraceContextKey is the key used to store trace context in gRPC metadata
const TraceContextKey = "trace-context"

// AuthorizationMetadataKey is the key used for authorization header in gRPC metadata
const AuthorizationMetadataKey = "authorization"

// UserClaimsContextKey is the key used to store user claims in gRPC context
const UserClaimsContextKey = "user-claims"

// UnaryServerInterceptor extracts trace context from incoming gRPC requests
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract trace context from metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if traceHeaders := md.Get(TraceContextKey); len(traceHeaders) > 0 {
				// Parse trace context from header (format: trace_id:span_id:parent_span_id)
				parts := strings.Split(traceHeaders[0], ":")
				if len(parts) >= 2 {
					traceID := parts[0]
					spanID := parts[1]
					var parentSpanID string
					if len(parts) >= 3 {
						parentSpanID = parts[2]
					}

					// Add to context
					ctx = context.WithValue(ctx, "trace_id", traceID)
					ctx = context.WithValue(ctx, "span_id", spanID)
					if parentSpanID != "" {
						ctx = context.WithValue(ctx, "parent_span_id", parentSpanID)
					}
				}
			}
		}

		return handler(ctx, req)
	}
}

// UnaryClientInterceptor injects trace context into outgoing gRPC requests
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Extract trace context from context
		traceID := getValueFromContext(ctx, "trace_id")
		spanID := getValueFromContext(ctx, "span_id")

		if traceID != "" && spanID != "" {
			// Create new span ID for this call
			newSpanID := generateSpanID()

			// Add trace context to metadata
			traceContext := traceID + ":" + newSpanID + ":" + spanID
			md := metadata.Pairs(TraceContextKey, traceContext)
			ctx = metadata.NewOutgoingContext(ctx, md)

			// Update context with new span
			ctx = context.WithValue(ctx, "span_id", newSpanID)
			ctx = context.WithValue(ctx, "parent_span_id", spanID)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamServerInterceptor extracts trace context from incoming gRPC streams
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()

		// Extract trace context from metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if traceHeaders := md.Get(TraceContextKey); len(traceHeaders) > 0 {
				parts := strings.Split(traceHeaders[0], ":")
				if len(parts) >= 2 {
					traceID := parts[0]
					spanID := parts[1]
					var parentSpanID string
					if len(parts) >= 3 {
						parentSpanID = parts[2]
					}

					ctx = context.WithValue(ctx, "trace_id", traceID)
					ctx = context.WithValue(ctx, "span_id", spanID)
					if parentSpanID != "" {
						ctx = context.WithValue(ctx, "parent_span_id", parentSpanID)
					}
				}
			}
		}

		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// StreamClientInterceptor injects trace context into outgoing gRPC streams
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Extract trace context from context
		traceID := getValueFromContext(ctx, "trace_id")
		spanID := getValueFromContext(ctx, "span_id")

		if traceID != "" && spanID != "" {
			// Create new span ID for this call
			newSpanID := generateSpanID()

			// Add trace context to metadata
			traceContext := traceID + ":" + newSpanID + ":" + spanID
			md := metadata.Pairs(TraceContextKey, traceContext)
			ctx = metadata.NewOutgoingContext(ctx, md)

			// Update context with new span
			ctx = context.WithValue(ctx, "span_id", newSpanID)
			ctx = context.WithValue(ctx, "parent_span_id", spanID)
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// wrappedServerStream wraps grpc.ServerStream to provide context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func generateSpanID() string {
	// Simple span ID generation - in production, use a proper ID generator
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// LoggingUnaryServerInterceptor logs gRPC requests with correlation IDs
func LoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		correlationID := getCorrelationID(ctx)

		log.Printf("[gRPC] %s -> %s (correlation_id: %s)", info.FullMethod, "START", correlationID)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		status := "SUCCESS"
		if err != nil {
			status = "ERROR"
		}

		log.Printf("[gRPC] %s -> %s (correlation_id: %s, duration: %v)", info.FullMethod, status, correlationID, duration)

		return resp, err
	}
}

// ErrorHandlingUnaryServerInterceptor converts errors to appropriate gRPC status codes
func ErrorHandlingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, convertErrorToGRPCStatus(err)
		}
		return resp, nil
	}
}

// ErrorHandlingStreamServerInterceptor converts errors to appropriate gRPC status codes for streams
func ErrorHandlingStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, stream)
		if err != nil {
			return convertErrorToGRPCStatus(err)
		}
		return nil
	}
}

// convertErrorToGRPCStatus converts domain errors to gRPC status codes
func convertErrorToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	// Check for domain-specific errors
	switch err {
	case domain.ErrInvalidCredentials, domain.ErrInvalidToken, domain.ErrExpiredToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case domain.ErrUnauthorized, domain.ErrForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case domain.ErrUserNotFound, domain.ErrInvalidUserData:
		return status.Error(codes.InvalidArgument, err.Error())
	case domain.ErrUserExists:
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		// Check if it's already a gRPC status error
		if st, ok := status.FromError(err); ok {
			return st.Err()
		}
		// Default to internal error
		return status.Error(codes.Internal, err.Error())
	}
}

// GetUserClaimsFromContext extracts user claims from gRPC context
func GetUserClaimsFromContext(ctx context.Context) (*domain.Claims, bool) {
	claims, ok := ctx.Value(UserClaimsContextKey).(*domain.Claims)
	return claims, ok
}

// LoggingStreamServerInterceptor logs gRPC streaming requests with correlation IDs
func LoggingStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		correlationID := getCorrelationID(stream.Context())

		log.Printf("[gRPC Stream] %s -> %s (correlation_id: %s)", info.FullMethod, "START", correlationID)

		err := handler(srv, stream)

		duration := time.Since(start)
		status := "SUCCESS"
		if err != nil {
			status = "ERROR"
		}

		log.Printf("[gRPC Stream] %s -> %s (correlation_id: %s, duration: %v)", info.FullMethod, status, correlationID, duration)

		return err
	}
}

// getCorrelationID extracts correlation ID from context (uses trace_id if available)
func getCorrelationID(ctx context.Context) string {
	if traceID := getValueFromContext(ctx, "trace_id"); traceID != "" {
		return traceID
	}
	// Fallback to generating a simple correlation ID
	return fmt.Sprintf("corr-%d", time.Now().UnixNano())
}

// AuthUnaryServerInterceptor validates JWT tokens and extracts user claims
func AuthUnaryServerInterceptor(authService ports.AuthService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication for certain methods if needed
		if shouldSkipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get(AuthorizationMetadataKey)
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token from "Bearer <token>" format
		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == authHeaders[0] {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate token
		claims, err := authService.ValidateToken(ctx, tokenString)
		if err != nil {
			if err == domain.ErrExpiredToken {
				return nil, status.Error(codes.Unauthenticated, "token expired")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add claims to context
		ctx = context.WithValue(ctx, UserClaimsContextKey, claims)

		return handler(ctx, req)
	}
}

// AuthStreamServerInterceptor validates JWT tokens for streaming calls
func AuthStreamServerInterceptor(authService ports.AuthService) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication for certain methods if needed
		if shouldSkipAuth(info.FullMethod) {
			return handler(srv, stream)
		}

		ctx := stream.Context()

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get(AuthorizationMetadataKey)
		if len(authHeaders) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token from "Bearer <token>" format
		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == authHeaders[0] {
			return status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate token
		claims, err := authService.ValidateToken(ctx, tokenString)
		if err != nil {
			if err == domain.ErrExpiredToken {
				return status.Error(codes.Unauthenticated, "token expired")
			}
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add claims to context
		ctx = context.WithValue(ctx, UserClaimsContextKey, claims)
		wrappedStream := &authWrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// authWrappedServerStream wraps grpc.ServerStream to provide authenticated context
type authWrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *authWrappedServerStream) Context() context.Context {
	return w.ctx
}

// shouldSkipAuth determines if authentication should be skipped for a method
func shouldSkipAuth(method string) bool {
	// Add methods that don't require authentication
	skipMethods := []string{
		// Add any public methods here, e.g., health checks, login, etc.
	}

	for _, skip := range skipMethods {
		if strings.Contains(method, skip) {
			return true
		}
	}
	return false
}

// Helper functions
func getValueFromContext(ctx context.Context, key string) string {
	if val := ctx.Value(key); val != nil {
		return val.(string)
	}
	return ""
}