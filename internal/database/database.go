package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	PG    *sql.DB
	Mongo *mongo.Client
	Redis *redis.Client
}

func NewPostgres(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return db, nil
}

func NewMongoDB(mongoURL string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	log.Println("Connected to MongoDB")
	return client, nil
}

func NewRedis(redisURL, password string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil
	}

	log.Println("Connected to Redis")
	return client
}

func InitializeSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS packages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tracking_number VARCHAR(100) UNIQUE NOT NULL,
		sender_name VARCHAR(255) NOT NULL,
		sender_address TEXT NOT NULL,
		recipient_name VARCHAR(255) NOT NULL,
		recipient_address TEXT NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		current_location VARCHAR(255),
		weight DECIMAL(10, 2),
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS deliveries (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		package_id UUID REFERENCES packages(id) ON DELETE CASCADE,
		driver_id UUID REFERENCES users(id),
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		scheduled_at TIMESTAMP,
		delivered_at TIMESTAMP,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_packages_tracking_number ON packages(tracking_number);
	CREATE INDEX IF NOT EXISTS idx_packages_status ON packages(status);
	CREATE INDEX IF NOT EXISTS idx_deliveries_package_id ON deliveries(package_id);
	CREATE INDEX IF NOT EXISTS idx_deliveries_driver_id ON deliveries(driver_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("Database schema initialized")
	return nil
}
