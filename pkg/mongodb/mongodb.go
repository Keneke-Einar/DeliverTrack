package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB represents a MongoDB connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// CourierLocation represents a courier's location with GeoJSON format
type CourierLocation struct {
	CourierID  int64     `bson:"courier_id" json:"courier_id"`
	DeliveryID int64     `bson:"delivery_id" json:"delivery_id"`
	Location   GeoJSON   `bson:"location" json:"location"`
	Timestamp  time.Time `bson:"timestamp" json:"timestamp"`
	Speed      float64   `bson:"speed,omitempty" json:"speed,omitempty"`         // km/h
	Heading    float64   `bson:"heading,omitempty" json:"heading,omitempty"`     // degrees
	Accuracy   float64   `bson:"accuracy,omitempty" json:"accuracy,omitempty"`   // meters
	Altitude   float64   `bson:"altitude,omitempty" json:"altitude,omitempty"`   // meters
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

// DeliveryZone represents a geofenced delivery zone
type DeliveryZone struct {
	Name        string    `bson:"name" json:"name"`
	Geometry    GeoJSON   `bson:"geometry" json:"geometry"`
	Active      bool      `bson:"active" json:"active"`
	Description string    `bson:"description,omitempty" json:"description,omitempty"`
	Properties  ZoneProps `bson:"properties,omitempty" json:"properties,omitempty"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

// GeoJSON represents a GeoJSON object for MongoDB geospatial queries
type GeoJSON struct {
	Type        string      `bson:"type" json:"type"`                   // "Point", "Polygon", etc.
	Coordinates interface{} `bson:"coordinates" json:"coordinates"`     // [longitude, latitude] for Point
}

// ZoneProps contains additional properties for delivery zones
type ZoneProps struct {
	MaxCouriers    int     `bson:"max_couriers,omitempty" json:"max_couriers,omitempty"`
	Priority       int     `bson:"priority,omitempty" json:"priority,omitempty"`
	AvgDeliveryFee float64 `bson:"avg_delivery_fee,omitempty" json:"avg_delivery_fee,omitempty"`
}

// New creates a new MongoDB connection
func New(mongoURL string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create client options
	clientOptions := options.Client().ApplyURI(mongoURL)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Extract database name from URL
	dbName := "delivertrack"
	
	log.Printf("MongoDB connected successfully to database: %s", dbName)

	return &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

// GetCollection returns a collection from the database
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// CourierLocationsCollection returns the courier_locations collection
func (m *MongoDB) CourierLocationsCollection() *mongo.Collection {
	return m.GetCollection("courier_locations")
}

// DeliveryZonesCollection returns the delivery_zones collection
func (m *MongoDB) DeliveryZonesCollection() *mongo.Collection {
	return m.GetCollection("delivery_zones")
}

// NewPoint creates a GeoJSON Point from longitude and latitude
func NewPoint(longitude, latitude float64) GeoJSON {
	return GeoJSON{
		Type:        "Point",
		Coordinates: []float64{longitude, latitude},
	}
}

// NewPolygon creates a GeoJSON Polygon from coordinates
// coordinates should be [][][]float64 where:
// - First dimension: array of linear rings (outer ring + holes)
// - Second dimension: array of positions in the ring
// - Third dimension: [longitude, latitude] pair
func NewPolygon(coordinates [][][]float64) GeoJSON {
	return GeoJSON{
		Type:        "Polygon",
		Coordinates: coordinates,
	}
}
