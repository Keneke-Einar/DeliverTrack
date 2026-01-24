package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Keneke-Einar/DeliverTrack/internal/config"
	"github.com/Keneke-Einar/DeliverTrack/internal/database"
	"github.com/Keneke-Einar/DeliverTrack/internal/handlers"
	"github.com/Keneke-Einar/DeliverTrack/internal/messaging"
	"github.com/Keneke-Einar/DeliverTrack/internal/websocket"
	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Println("Starting DeliverTrack Server...")

	// Initialize PostgreSQL
	pg, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pg.Close()

	// Initialize database schema
	if err := database.InitializeSchema(pg); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Initialize MongoDB
	mongoClient, err := database.NewMongoDB(cfg.MongoDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(nil)

	// Initialize Redis
	redisClient := database.NewRedis(cfg.RedisURL, cfg.RedisPassword)
	if redisClient == nil {
		log.Println("Warning: Redis connection failed, continuing without cache")
	}

	// Initialize RabbitMQ
	messageBroker, err := messaging.NewMessageBroker(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer messageBroker.Close()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Start consuming messages from RabbitMQ
	err = messageBroker.ConsumePackageUpdates(func(event messaging.PackageEvent) error {
		log.Printf("Received package update: %s - %s", event.TrackingNumber, event.Status)
		// Broadcast to WebSocket clients
		wsHub.BroadcastPackageUpdate(event.TrackingNumber, map[string]interface{}{
			"status":    event.Status,
			"location":  event.Location,
			"latitude":  event.Latitude,
			"longitude": event.Longitude,
			"timestamp": event.Timestamp,
		})
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to start message consumer: %v", err)
	}

	// Initialize handlers
	handler := handlers.NewHandler(pg, mongoClient, redisClient, messageBroker, wsHub)

	// Setup router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", handler.HealthCheck).Methods("GET")
	api.HandleFunc("/stats", handler.GetStats).Methods("GET")
	api.HandleFunc("/packages", handler.ListPackages).Methods("GET")
	api.HandleFunc("/packages", handler.CreatePackage).Methods("POST")
	api.HandleFunc("/packages/{tracking_number}", handler.GetPackage).Methods("GET")
	api.HandleFunc("/packages/{tracking_number}/status", handler.UpdatePackageStatus).Methods("PUT")
	api.HandleFunc("/packages/{tracking_number}/locations", handler.GetPackageLocations).Methods("GET")

	// WebSocket route
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(wsHub, w, r)
	})

	// Apply middlewares
	router.Use(handlers.CORSMiddleware)
	router.Use(handlers.LoggingMiddleware)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server is running on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
