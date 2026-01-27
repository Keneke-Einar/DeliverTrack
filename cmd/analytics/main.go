package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	analyticsAdapters "github.com/Keneke-Einar/delivertrack/internal/analytics/adapters"
	analyticsApp "github.com/Keneke-Einar/delivertrack/internal/analytics/app"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"

	"github.com/Keneke-Einar/delivertrack/proto/analytics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var version = "dev"

func main() {
	cfg, err := config.Load("analytics")
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

	// Wire up dependencies using layered architecture

	// Auth layer
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)
	authHandler := authAdapters.NewHTTPHandler(authService, cfg.Auth.JWTExpiration)

	// Analytics layer
	analyticsRepo := analyticsAdapters.NewPostgresMetricRepository(db.DB)
	analyticsService := analyticsApp.NewAnalyticsService(analyticsRepo)
	analyticsHTTPHandler := analyticsAdapters.NewHTTPHandler(analyticsService)
	analyticsGRPCHandler := analyticsAdapters.NewGRPCHandler(analyticsService)

	// Setup HTTP router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/register", authHandler.Register)

	// Protected routes - analytics endpoints
	mux.HandleFunc("/metrics", authMiddleware(authService, analyticsHTTPHandler.RecordMetric))
	mux.HandleFunc("/stats/deliveries", authMiddleware(authService, analyticsHTTPHandler.GetDeliveryStats))

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Analytics HTTP service v%s starting on port %s", version, port)
		log.Printf("Endpoints: POST /login, POST /register")
		log.Printf("           POST /metrics, GET /stats/deliveries")

		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatal(err)
		}
	}()

	// Start gRPC server
	grpcPort := "50054"
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	analytics.RegisterAnalyticsServiceServer(grpcServer, analyticsGRPCHandler)
	reflection.Register(grpcServer) // Enable reflection for debugging

	log.Printf("Analytics gRPC service v%s starting on port %s", version, grpcPort)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"analytics"}`)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"analytics","version":"%s"}`, version)
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
