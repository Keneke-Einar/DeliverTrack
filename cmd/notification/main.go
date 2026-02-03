package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	notificationAdapters "github.com/Keneke-Einar/delivertrack/internal/notification/adapters"
	notificationApp "github.com/Keneke-Einar/delivertrack/internal/notification/app"
	"go.uber.org/zap"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/grpcinterceptors"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"

	"github.com/Keneke-Einar/delivertrack/proto/notification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var version = "dev"

func main() {
	cfg, err := config.Load("notification")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	port := cfg.Service.Port
	databaseURL := cfg.Database.URL
	jwtSecret := cfg.Auth.JWTSecret

	// Initialize logger
	lg, err := logger.NewLogger(cfg.Logging, "notification")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize database
	db, err := postgres.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	lg.Info("Database connection established")

	// Wire up dependencies using layered architecture

	// Auth layer
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)
	authHandler := authAdapters.NewHTTPHandler(authService, cfg.Auth.JWTExpiration)

	// Notification layer
	notificationRepo := notificationAdapters.NewPostgresNotificationRepository(db.DB)

	// Initialize RabbitMQ consumer for event handling
	rabbitMQURL := cfg.RabbitMQ.URL
	consumer, err := messaging.NewRabbitMQConsumer(rabbitMQURL, lg)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}
	defer consumer.Close()

	notificationService := notificationApp.NewNotificationService(notificationRepo, consumer, lg)
	notificationHTTPHandler := notificationAdapters.NewHTTPHandler(notificationService)
	notificationGRPCHandler := notificationAdapters.NewGRPCHandler(notificationService)

	// Start event consumption
	if err := notificationService.StartEventConsumption(); err != nil {
		log.Fatalf("Failed to start event consumption: %v", err)
	}
	lg.Info("Started consuming delivery and location events")

	// Setup HTTP router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/register", authHandler.Register)

	// Protected routes - notification endpoints
	mux.HandleFunc("/notifications", authMiddleware(authService, notificationHTTPHandler.GetUserNotifications))
	mux.HandleFunc("/notifications/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/notifications/")
		if path == "" {
			// POST /notifications
			authMiddleware(authService, notificationHTTPHandler.SendNotification)(w, r)
		} else {
			// PUT /notifications/{id}/read
			if strings.HasSuffix(path, "/read") {
				authMiddleware(authService, notificationHTTPHandler.MarkAsRead)(w, r)
			} else {
				http.NotFound(w, r)
			}
		}
	})

	// Start HTTP server in a goroutine
	go func() {
		lg.Info("Notification HTTP service starting",
			zap.String("version", version),
			zap.String("port", port))
		lg.Info("HTTP endpoints available",
			zap.Strings("endpoints", []string{
				"POST /login", "POST /register",
				"POST /notifications", "GET /notifications", "PUT /notifications/{id}/read"}))

		if err := http.ListenAndServe(":"+port, mux); err != nil {
			lg.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server
	grpcPort := "50053"
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
	notification.RegisterNotificationServiceServer(grpcServer, notificationGRPCHandler)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer) // Enable reflection for debugging

	lg.Info("Notification gRPC service starting",
		zap.String("version", version),
		zap.String("port", grpcPort))

	if err := grpcServer.Serve(lis); err != nil {
		lg.Fatal("Failed to serve gRPC", zap.Error(err))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"notification"}`)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"notification","version":"%s"}`, version)
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

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
