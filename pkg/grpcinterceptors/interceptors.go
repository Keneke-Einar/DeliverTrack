package grpcinterceptors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TraceContextKey is the key used to store trace context in gRPC metadata
const TraceContextKey = "trace-context"

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

// Helper functions
func getValueFromContext(ctx context.Context, key string) string {
	if val := ctx.Value(key); val != nil {
		return val.(string)
	}
	return ""
}

func generateSpanID() string {
	// Simple span ID generation - in production, use a proper ID generator
	return fmt.Sprintf("%d", time.Now().UnixNano())
}