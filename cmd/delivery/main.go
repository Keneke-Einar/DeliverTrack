package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	deliveryAdapters "github.com/Keneke-Einar/delivertrack/internal/delivery/adapters"
	deliveryApp "github.com/Keneke-Einar/delivertrack/internal/delivery/app"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"

	"github.com/Keneke-Einar/delivertrack/proto/delivery"
	"github.com/Keneke-Einar/delivertrack/proto/notification"
	"github.com/Keneke-Einar/delivertrack/proto/analytics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var version = "dev"

func main() {
	cfg, err := config.Load("delivery")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	port := cfg.Service.Port
	databaseURL := cfg.Database.URL
	jwtSecret := cfg.Auth.JWTSecret

	// Initialize database
	db, err := postgres.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established")

	// Initialize gRPC clients for inter-service communication
	notificationConn, err := grpc.Dial(cfg.Services.Notification, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to notification service: %v", err)
	}
	defer notificationConn.Close()
	notificationClient := notification.NewNotificationServiceClient(notificationConn)

	analyticsConn, err := grpc.Dial(cfg.Services.Analytics, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to analytics service: %v", err)
	}
	defer analyticsConn.Close()
	analyticsClient := analytics.NewAnalyticsServiceClient(analyticsConn)

	log.Println("gRPC clients initialized")

	// Auth layer
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)
	authHandler := authAdapters.NewHTTPHandler(authService, cfg.Auth.JWTExpiration)

	// Delivery layer
	deliveryRepo := deliveryAdapters.NewPostgresDeliveryRepository(db.DB)
	deliveryService := deliveryApp.NewDeliveryService(deliveryRepo, notificationClient, analyticsClient)
	deliveryHTTPHandler := deliveryAdapters.NewHTTPHandler(deliveryService)
	deliveryGRPCHandler := deliveryAdapters.NewGRPCHandler(deliveryService)

	// Setup HTTP router with middleware
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/register", authHandler.Register)

	// Protected routes - delivery endpoints
	mux.HandleFunc("/deliveries", authMiddleware(authService, deliveryHTTPHandler.ListDeliveries))
	mux.HandleFunc("/deliveries/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
		if path == "" {
			// Handle POST /deliveries
			authMiddleware(authService, deliveryHTTPHandler.CreateDelivery)(w, r)
			return
		}

		// Check if path ends with /status
		if strings.HasSuffix(path, "/status") {
			// Handle PUT /deliveries/:id/status
			authMiddleware(authService, deliveryHTTPHandler.UpdateDeliveryStatus)(w, r)
		} else {
			// Handle GET /deliveries/:id
			authMiddleware(authService, deliveryHTTPHandler.GetDelivery)(w, r)
		}
	})

	// Wrap with CORS middleware
	httpHandler := corsMiddleware(mux)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Delivery HTTP service v%s starting on port %s", version, port)
		log.Printf("Endpoints: POST /login, POST /register")
		log.Printf("           POST /deliveries, GET /deliveries/:id, PUT /deliveries/:id/status, GET /deliveries?status=xxx")

		if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
			log.Fatal(err)
		}
	}()

	// Start gRPC server
	grpcPort := "50051"
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	delivery.RegisterDeliveryServiceServer(grpcServer, deliveryGRPCHandler)
	reflection.Register(grpcServer) // Enable reflection for debugging

	log.Printf("Delivery gRPC service v%s starting on port %s", version, grpcPort)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"delivery"}`)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"delivery","version":"%s"}`, version)
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
