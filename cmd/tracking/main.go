package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	trackingAdapters "github.com/Keneke-Einar/delivertrack/internal/tracking/adapters"
	trackingApp "github.com/Keneke-Einar/delivertrack/internal/tracking/app"

	authAdapters "github.com/Keneke-Einar/delivertrack/pkg/auth/adapters"
	authApp "github.com/Keneke-Einar/delivertrack/pkg/auth/app"
	authPorts "github.com/Keneke-Einar/delivertrack/pkg/auth/ports"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"github.com/Keneke-Einar/delivertrack/pkg/mongodb"
	"github.com/Keneke-Einar/delivertrack/pkg/postgres"
	"github.com/Keneke-Einar/delivertrack/pkg/websocket"
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

	// Initialize PostgreSQL for auth
	db, err := postgres.New(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	log.Println("PostgreSQL connection established")

	// Initialize MongoDB for geospatial data
	mongoClient, err := mongodb.New(mongoURL)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Close(context.Background())

	log.Println("MongoDB connection established")

	// Wire up dependencies using layered architecture

	// Auth layer
	userRepo := authAdapters.NewPostgresUserRepository(db.DB)
	tokenService := authAdapters.NewJWTTokenService(jwtSecret, cfg.Auth.JWTExpiration)
	authService := authApp.NewAuthService(userRepo, tokenService)
	authHandler := authAdapters.NewHTTPHandler(authService, cfg.Auth.JWTExpiration)

	// Tracking layer
	trackingRepo := trackingAdapters.NewMongoDBLocationRepository(mongoClient)
	trackingService := trackingApp.NewTrackingService(trackingRepo)
	trackingHandler := trackingAdapters.NewHTTPHandler(trackingService)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	trackingService.SetWebSocketHub(wsHub)

	// Start WebSocket hub in background
	go wsHub.Run()

	// Setup router with middleware
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/login", authHandler.Login)
	mux.HandleFunc("/register", authHandler.Register)

	// Protected routes - tracking endpoints
	mux.HandleFunc("/locations", authMiddleware(authService, trackingHandler.RecordLocation))

	// Delivery tracking routes
	mux.HandleFunc("/deliveries/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/deliveries/")
		parts := strings.Split(path, "/")

		if len(parts) >= 2 {
			switch parts[1] {
			case "track":
				// GET /deliveries/{id}/track
				authMiddleware(authService, trackingHandler.GetDeliveryTrack)(w, r)
			case "location":
				// GET /deliveries/{id}/location
				authMiddleware(authService, trackingHandler.GetCurrentLocation)(w, r)
			case "eta":
				// POST /deliveries/{id}/eta
				authMiddleware(authService, trackingHandler.CalculateETA)(w, r)
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
			authMiddleware(authService, trackingHandler.GetCourierLocation)(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// WebSocket routes
	mux.HandleFunc("/ws/deliveries/", wsHub.HandleWebSocket)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	log.Printf("Tracking service v%s starting on port %s", version, port)
	log.Printf("Endpoints: POST /login, POST /register")
	log.Printf("           POST /locations, GET /deliveries/{id}/track, GET /deliveries/{id}/location, GET /couriers/{id}/location")

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
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
