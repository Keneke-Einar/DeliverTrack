package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	trackingAdapters "github.com/Keneke-Einar/delivertrack/internal/tracking/adapters"
	trackingApp "github.com/Keneke-Einar/delivertrack/internal/tracking/app"
	"go.uber.org/zap"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/grpcinterceptors"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/pkg/mongodb"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"
	"github.com/Keneke-Einar/delivertrack/pkg/websocket"

	"github.com/Keneke-Einar/delivertrack/proto/delivery"
	"github.com/Keneke-Einar/delivertrack/proto/tracking"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var version = "dev"

func main() {
	cfg, err := config.Load("tracking")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	port := cfg.Service.Port
	databaseURL := cfg.Database.URL
	mongoURL := cfg.MongoDB.URL
	jwtSecret := cfg.Auth.JWTSecret

	// Initialize logger
	lg, err := logger.NewLogger(cfg.Logging, "tracking")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize PostgreSQL for auth
	db, err := postgres.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	lg.Info("PostgreSQL connection established")

	// Initialize MongoDB for geospatial data
	mongoClient, err := mongodb.New(mongoURL)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(context.Background())

	lg.Info("MongoDB connection established")

	// Initialize gRPC clients for inter-service communication
	deliveryConn, err := grpc.NewClient(cfg.Services.Delivery,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpcinterceptors.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(grpcinterceptors.StreamClientInterceptor()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to delivery service: %v", err)
	}
	defer deliveryConn.Close()
	deliveryClient := delivery.NewDeliveryServiceClient(deliveryConn)

	lg.Info("gRPC clients initialized")

	// Auth layer
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)
	authHandler := authAdapters.NewHTTPHandler(authService, cfg.Auth.JWTExpiration)

	// Tracking layer
	trackingRepo := trackingAdapters.NewMongoDBLocationRepository(mongoClient)

	// Initialize RabbitMQ publisher for event publishing
	rabbitMQURL := cfg.RabbitMQ.URL
	publisher, err := messaging.NewRabbitMQPublisher(rabbitMQURL, lg)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()

	trackingService := trackingApp.NewTrackingService(trackingRepo, publisher, deliveryClient, authService, lg)
	trackingHTTPHandler := trackingAdapters.NewHTTPHandler(trackingService)
	trackingGRPCHandler := trackingAdapters.NewGRPCHandler(trackingService)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(authService)
	trackingService.SetWebSocketHub(wsHub)

	// Start WebSocket hub in background
	go wsHub.Run()

	// Setup HTTP router with middleware
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/register", authHandler.Register)

	// Protected routes - tracking endpoints
	mux.HandleFunc("/locations", authMiddleware(authService, trackingHTTPHandler.RecordLocation))

	// Delivery tracking routes
	mux.HandleFunc("/deliveries/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
		parts := strings.Split(path, "/")

		if len(parts) >= 2 {
			switch parts[1] {
			case "track":
				// GET /deliveries/{id}/track
				authMiddleware(authService, trackingHTTPHandler.GetDeliveryTrack)(w, r)
			case "location":
				// GET /deliveries/{id}/location
				authMiddleware(authService, trackingHTTPHandler.GetCurrentLocation)(w, r)
			case "eta":
				// POST /deliveries/{id}/eta
				authMiddleware(authService, trackingHTTPHandler.CalculateETA)(w, r)
			default:
				http.NotFound(w, r)
			}
		} else {
			http.NotFound(w, r)
		}
	})

	// Courier location routes
	mux.HandleFunc("/couriers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/couriers/")
		parts := strings.Split(path, "/")

		if len(parts) >= 2 && parts[1] == "location" {
			// GET /couriers/{id}/location
			authMiddleware(authService, trackingHTTPHandler.GetCourierLocation)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// WebSocket routes
	mux.HandleFunc("/ws/deliveries/", wsHub.HandleWebSocket)
	mux.HandleFunc("/ws/notifications", wsHub.HandleCustomerWebSocket)

	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		connectionCount := wsHub.GetConnectionCount()
		fmt.Fprintf(w, `{"websocket_connections": %d}`, connectionCount)
	})

	// Wrap with CORS middleware
	httpHandler := corsMiddleware(mux)

	// Start HTTP server in a goroutine
	go func() {
		lg.Info("Tracking HTTP service starting",
			zap.String("version", version),
			zap.String("port", port))
		lg.Info("HTTP endpoints available",
			zap.Strings("endpoints", []string{
				"POST /login", "POST /register",
				"POST /locations", "GET /deliveries/{id}/track",
				"GET /deliveries/{id}/location", "GET /couriers/{id}/location",
				"GET /metrics", "WS /ws/deliveries/{id}/track", "WS /ws/notifications"}))

		if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
			lg.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server
	grpcPort := "50052"
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		lg.Fatal("Failed to listen on gRPC port",
			zap.String("port", grpcPort), zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcinterceptors.ErrorHandlingUnaryServerInterceptor(),
			grpcinterceptors.LoggingUnaryServerInterceptor(lg),
			grpcinterceptors.AuthUnaryServerInterceptor(authService),
			grpcinterceptors.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcinterceptors.ErrorHandlingStreamServerInterceptor(),
			grpcinterceptors.LoggingStreamServerInterceptor(lg),
			grpcinterceptors.AuthStreamServerInterceptor(authService),
			grpcinterceptors.StreamServerInterceptor(),
		),
	)
	tracking.RegisterTrackingServiceServer(grpcServer, trackingGRPCHandler)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer) // Enable reflection for debugging

	lg.Info("Tracking gRPC service starting",
		zap.String("version", version),
		zap.String("port", grpcPort))

	if err := grpcServer.Serve(lis); err != nil {
		lg.Fatal("Failed to serve gRPC", zap.Error(err))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"tracking"}`)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"tracking","version":"%s"}`, version)
}

// authMiddleware validates JWT token and adds user info to context
func authMiddleware(authService authPorts.AuthService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Check Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"unauthorized","message":"Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, `{"error":"unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "role", claims.Role)
		ctx = context.WithValue(ctx, "customer_id", claims.CustomerID)
		ctx = context.WithValue(ctx, "courier_id", claims.CourierID)
		ctx = context.WithValue(ctx, "authorization", authHeader)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
