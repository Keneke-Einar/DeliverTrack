package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Keneke-Einar/DeliverTrack/internal/messaging"
	"github.com/Keneke-Einar/DeliverTrack/internal/models"
	"github.com/Keneke-Einar/DeliverTrack/internal/websocket"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler struct {
	db            *sql.DB
	mongoClient   *mongo.Client
	redisClient   *redis.Client
	messageBroker *messaging.MessageBroker
	wsHub         *websocket.Hub
}

func NewHandler(db *sql.DB, mongoClient *mongo.Client, redisClient *redis.Client, mb *messaging.MessageBroker, hub *websocket.Hub) *Handler {
	return &Handler{
		db:            db,
		mongoClient:   mongoClient,
		redisClient:   redisClient,
		messageBroker: mb,
		wsHub:         hub,
	}
}

func (h *Handler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var pkg models.Package
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO packages (tracking_number, sender_name, sender_address, recipient_name, recipient_address, status, weight, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err := h.db.QueryRow(
		query,
		pkg.TrackingNumber,
		pkg.SenderName,
		pkg.SenderAddress,
		pkg.RecipientName,
		pkg.RecipientAddress,
		models.StatusPending,
		pkg.Weight,
		pkg.Description,
	).Scan(&pkg.ID, &pkg.CreatedAt, &pkg.UpdatedAt)

	if err != nil {
		log.Printf("Error creating package: %v", err)
		http.Error(w, "Failed to create package", http.StatusInternalServerError)
		return
	}

	pkg.Status = string(models.StatusPending)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pkg)
}

func (h *Handler) GetPackage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackingNumber := vars["tracking_number"]

	// Try to get from Redis cache first
	ctx := context.Background()
	cachedData, err := h.redisClient.Get(ctx, "package:"+trackingNumber).Result()
	if err == nil {
		var pkg models.Package
		if err := json.Unmarshal([]byte(cachedData), &pkg); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			json.NewEncoder(w).Encode(pkg)
			return
		}
	}

	// If not in cache, get from database
	query := `
		SELECT id, tracking_number, sender_name, sender_address, recipient_name, recipient_address,
		       status, current_location, weight, description, created_at, updated_at
		FROM packages
		WHERE tracking_number = $1
	`

	var pkg models.Package
	err = h.db.QueryRow(query, trackingNumber).Scan(
		&pkg.ID,
		&pkg.TrackingNumber,
		&pkg.SenderName,
		&pkg.SenderAddress,
		&pkg.RecipientName,
		&pkg.RecipientAddress,
		&pkg.Status,
		&pkg.CurrentLocation,
		&pkg.Weight,
		&pkg.Description,
		&pkg.CreatedAt,
		&pkg.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error retrieving package: %v", err)
		http.Error(w, "Failed to retrieve package", http.StatusInternalServerError)
		return
	}

	// Cache the result
	if jsonData, err := json.Marshal(pkg); err == nil {
		h.redisClient.Set(ctx, "package:"+trackingNumber, jsonData, 5*time.Minute)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(pkg)
}

func (h *Handler) UpdatePackageStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackingNumber := vars["tracking_number"]

	var update struct {
		Status   string  `json:"status"`
		Location string  `json:"location"`
		Latitude float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update package in database
	query := `
		UPDATE packages
		SET status = $1, current_location = $2, updated_at = CURRENT_TIMESTAMP
		WHERE tracking_number = $3
		RETURNING id
	`

	var packageID string
	err := h.db.QueryRow(query, update.Status, update.Location, trackingNumber).Scan(&packageID)
	if err == sql.ErrNoRows {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error updating package: %v", err)
		http.Error(w, "Failed to update package", http.StatusInternalServerError)
		return
	}

	// Store location in MongoDB
	if update.Latitude != 0 && update.Longitude != 0 {
		collection := h.mongoClient.Database("delivertrack").Collection("locations")
		location := models.Location{
			PackageID: packageID,
			Latitude:  update.Latitude,
			Longitude: update.Longitude,
			Address:   update.Location,
			Timestamp: time.Now(),
		}
		_, err := collection.InsertOne(context.Background(), location)
		if err != nil {
			log.Printf("Error storing location: %v", err)
		}
	}

	// Invalidate cache
	ctx := context.Background()
	h.redisClient.Del(ctx, "package:"+trackingNumber)

	// Publish to RabbitMQ
	event := messaging.PackageEvent{
		PackageID:      packageID,
		TrackingNumber: trackingNumber,
		Status:         update.Status,
		Location:       update.Location,
		Latitude:       update.Latitude,
		Longitude:      update.Longitude,
		Timestamp:      time.Now().Format(time.RFC3339),
	}
	if err := h.messageBroker.PublishPackageUpdate(event); err != nil {
		log.Printf("Error publishing event: %v", err)
	}

	// Broadcast via WebSocket
	h.wsHub.BroadcastPackageUpdate(trackingNumber, map[string]interface{}{
		"status":    update.Status,
		"location":  update.Location,
		"latitude":  update.Latitude,
		"longitude": update.Longitude,
		"timestamp": event.Timestamp,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Package status updated successfully",
		"status":  update.Status,
	})
}

func (h *Handler) GetPackageLocations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	trackingNumber := vars["tracking_number"]

	// First, get the package ID
	var packageID string
	err := h.db.QueryRow("SELECT id FROM packages WHERE tracking_number = $1", trackingNumber).Scan(&packageID)
	if err == sql.ErrNoRows {
		http.Error(w, "Package not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve package", http.StatusInternalServerError)
		return
	}

	// Get locations from MongoDB
	collection := h.mongoClient.Database("delivertrack").Collection("locations")
	ctx := context.Background()

	cursor, err := collection.Find(ctx, bson.M{"package_id": packageID})
	if err != nil {
		log.Printf("Error querying locations: %v", err)
		http.Error(w, "Failed to retrieve locations", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var locations []models.Location
	if err := cursor.All(ctx, &locations); err != nil {
		log.Printf("Error decoding locations: %v", err)
		http.Error(w, "Failed to decode locations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locations)
}

func (h *Handler) ListPackages(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, tracking_number, sender_name, recipient_name, status, current_location, created_at, updated_at
		FROM packages
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := h.db.Query(query)
	if err != nil {
		log.Printf("Error listing packages: %v", err)
		http.Error(w, "Failed to list packages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var packages []models.Package
	for rows.Next() {
		var pkg models.Package
		err := rows.Scan(
			&pkg.ID,
			&pkg.TrackingNumber,
			&pkg.SenderName,
			&pkg.RecipientName,
			&pkg.Status,
			&pkg.CurrentLocation,
			&pkg.CreatedAt,
			&pkg.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning package: %v", err)
			continue
		}
		packages = append(packages, pkg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packages)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Check database
	if err := h.db.Ping(); err != nil {
		health["database"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["database"] = "healthy"
	}

	// Check Redis
	ctx := context.Background()
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		health["redis"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["redis"] = "healthy"
	}

	// Check MongoDB
	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		health["mongodb"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["mongodb"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// Count packages by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM packages
		GROUP BY status
	`
	rows, err := h.db.Query(statusQuery)
	if err == nil {
		defer rows.Close()
		statusCounts := make(map[string]int)
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err == nil {
				statusCounts[status] = count
			}
		}
		stats["status_counts"] = statusCounts
	}

	// Total packages
	var totalPackages int
	h.db.QueryRow("SELECT COUNT(*) FROM packages").Scan(&totalPackages)
	stats["total_packages"] = totalPackages

	// Packages created today
	var todayPackages int
	h.db.QueryRow("SELECT COUNT(*) FROM packages WHERE DATE(created_at) = CURRENT_DATE").Scan(&todayPackages)
	stats["packages_today"] = todayPackages

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func CORSMiddleware(next http.Handler) http.Handler {
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

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed in %v", time.Since(start))
	})
}
