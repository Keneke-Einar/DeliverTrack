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
	"go.uber.org/zap"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"
	"github.com/Keneke-Einar/delivertrack/pkg/geocoding"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/grpcinterceptors"
	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"github.com/Keneke-Einar/delivertrack/pkg/messaging"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"

	"github.com/Keneke-Einar/delivertrack/proto/delivery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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

	// Initialize logger
	lg, err := logger.NewLogger(cfg.Logging, "delivery")
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

	// Initialize gRPC clients for inter-service communication
	deliveryConn, err := grpc.NewClient(cfg.Services.Delivery,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpcinterceptors.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(grpcinterceptors.StreamClientInterceptor()),
	)
	if err != nil {
		lg.Fatal("Failed to connect to delivery service", zap.Error(err))
	}
	defer deliveryConn.Close()
	deliveryClient := delivery.NewDeliveryServiceClient(deliveryConn)

	lg.Info("gRPC clients initialized")

	// Auth layer
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)
	authHandler := authAdapters.NewHTTPHandler(authService, cfg.Auth.JWTExpiration)

	// Delivery layer
	deliveryRepo := deliveryAdapters.NewPostgresDeliveryRepository(db.DB)

	// Initialize geocoding service
	geocodingSvc := geocoding.NewHTTPGeocodingService(lg)

	// Initialize RabbitMQ publisher for event publishing
	rabbitMQURL := cfg.RabbitMQ.URL
	publisher, err := messaging.NewRabbitMQPublisher(rabbitMQURL, lg)
	if err != nil {
		lg.Fatal("Failed to create RabbitMQ publisher", zap.Error(err))
	}
	defer publisher.Close()

	deliveryService := deliveryApp.NewDeliveryService(deliveryRepo, publisher, deliveryClient, geocodingSvc, lg)
	deliveryHTTPHandler := deliveryAdapters.NewHTTPHandler(deliveryService)
	geocodingHTTPHandler := geocoding.NewHTTPHandler(geocodingSvc)
	deliveryGRPCHandler := deliveryAdapters.NewGRPCHandler(deliveryService)

	// Setup HTTP router with middleware
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/register", authHandler.Register)

	// Geocoding routes (public - no auth required)
	mux.HandleFunc("/geocode/forward", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		geocodingHTTPHandler.ForwardGeocode(w, r)
	})
	mux.HandleFunc("/geocode/reverse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		geocodingHTTPHandler.ReverseGeocode(w, r)
	})
	mux.HandleFunc("/geocode/autocomplete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		geocodingHTTPHandler.Autocomplete(w, r)
	})

	// Catch-all root handler (must be last)
	mux.HandleFunc("/", rootHandler)

	// Protected routes - delivery endpoints
	// Routes use bare paths (gateway strips /api/delivery prefix before proxying)
	mux.HandleFunc("/deliveries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware(authService, deliveryHTTPHandler.ListDeliveries)(w, r)
		} else if r.Method == http.MethodPost {
			authMiddleware(authService, deliveryHTTPHandler.CreateDelivery)(w, r)
		} else {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/deliveries/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
		if path == "" {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
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
		lg.Info("Delivery HTTP service starting",
			zap.String("version", version),
			zap.String("port", port))
		lg.Info("HTTP endpoints available",
			zap.Strings("endpoints", []string{
				"POST /login", "POST /register",
				"POST /deliveries", "GET /deliveries/:id",
				"PUT /deliveries/:id/status", "GET /deliveries?status=xxx",
				"POST /geocode/forward", "POST /geocode/reverse", "GET /geocode/autocomplete",
			}))

		if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
			lg.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server
	grpcPort := "50051"
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
	delivery.RegisterDeliveryServiceServer(grpcServer, deliveryGRPCHandler)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer) // Enable reflection for debugging

	lg.Info("Delivery gRPC service starting",
		zap.String("version", version),
		zap.String("port", grpcPort))

	if err := grpcServer.Serve(lis); err != nil {
		lg.Fatal("Failed to serve gRPC", zap.Error(err))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"delivery"}`)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
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
