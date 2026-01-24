package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL  string
	MongoDBURL   string
	RedisURL     string
	RedisPassword string
	RabbitMQURL  string
	Port         string
	Host         string
	JWTSecret    string
	Environment  string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://delivertrack:delivertrack_password@localhost:5432/delivertrack?sslmode=disable"),
		MongoDBURL:    getEnv("MONGODB_URL", "mongodb://delivertrack:delivertrack_password@localhost:27017"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RabbitMQURL:   getEnv("RABBITMQ_URL", "amqp://delivertrack:delivertrack_password@localhost:5672/"),
		Port:          getEnv("PORT", "8080"),
		Host:          getEnv("HOST", "0.0.0.0"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-this"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
