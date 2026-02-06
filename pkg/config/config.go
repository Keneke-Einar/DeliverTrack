package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Keneke-Einar/delivertrack/pkg/secrets"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Service     ServiceConfig     `mapstructure:"service"`
	Services    ServicesConfig    `mapstructure:"services"`
	Database    DatabaseConfig    `mapstructure:"database"`
	MongoDB     MongoDBConfig     `mapstructure:"mongodb"`
	Redis       RedisConfig       `mapstructure:"redis"`
	RabbitMQ    RabbitMQConfig    `mapstructure:"rabbitmq"`
	Auth        AuthConfig        `mapstructure:"auth"`
	Vault       VaultConfig       `mapstructure:"vault"`
	Logging     LoggingConfig     `mapstructure:"logging"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Port    string `mapstructure:"port"`
	Version string `mapstructure:"version"`
}

// ServicesConfig holds URLs for other services
type ServicesConfig struct {
	Delivery     string `mapstructure:"delivery"`
	Tracking     string `mapstructure:"tracking"`
	Notification string `mapstructure:"notification"`
	Analytics    string `mapstructure:"analytics"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

// MongoDBConfig holds MongoDB configuration
type MongoDBConfig struct {
	URL string `mapstructure:"url"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL string `mapstructure:"url"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string        `mapstructure:"jwt_secret"`
	JWTExpiration time.Duration `mapstructure:"jwt_expiration"`
}

// VaultConfig holds Vault configuration
type VaultConfig struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
	Path    string `mapstructure:"path"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	Output    string `mapstructure:"output"`
	Sampling  SamplingConfig `mapstructure:"sampling"`
	Rotation  RotationConfig `mapstructure:"rotation"`
}

// SamplingConfig holds log sampling configuration
type SamplingConfig struct {
	Initial    int `mapstructure:"initial"`
	Thereafter int `mapstructure:"thereafter"`
}

// RotationConfig holds log rotation configuration
type RotationConfig struct {
	MaxSize    int  `mapstructure:"max_size"`
	MaxAge     int  `mapstructure:"max_age"`
	MaxBackups int  `mapstructure:"max_backups"`
	Compress   bool `mapstructure:"compress"`
}

// Load loads configuration from environment variables, config files, and Vault
func Load(serviceName string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(fmt.Sprintf("./config/%s", serviceName))

	// Enable reading from environment variables
	viper.SetEnvPrefix(serviceName)
	viper.AutomaticEnv()

	// Set defaults
	setDefaults(serviceName)

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with env vars and defaults
		log.Printf("Config file not found, using environment variables and defaults")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Override vault config with direct env vars
	if addr := os.Getenv("VAULT_ADDR"); addr != "" {
		config.Vault.Address = addr
	}
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		config.Vault.Token = token
	}

	// Initialize Vault client and load secrets
	vaultClient, err := secrets.NewVaultClient(config.Vault.Address, config.Vault.Token, config.Vault.Path)
	if err != nil {
		log.Printf("Failed to initialize Vault client: %v, continuing with defaults", err)
	} else {
		// Override sensitive configs with Vault secrets
		if secret, err := vaultClient.GetSecret("database_url"); err == nil && secret != "" {
			config.Database.URL = secret
		}
		if secret, err := vaultClient.GetSecret("mongodb_url"); err == nil && secret != "" {
			config.MongoDB.URL = secret
		}
		if secret, err := vaultClient.GetSecret("redis_url"); err == nil && secret != "" {
			config.Redis.URL = secret
		}
		if secret, err := vaultClient.GetSecret("rabbitmq_url"); err == nil && secret != "" {
			config.RabbitMQ.URL = secret
		}
		if secret, err := vaultClient.GetSecret("jwt_secret"); err == nil && secret != "" {
			config.Auth.JWTSecret = secret
		}
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(serviceName string) {
	port := "8080"
	switch serviceName {
	case "delivery":
		port = "8080"
	case "tracking":
		port = "8081"
	case "notification":
		port = "8082"
	case "analytics":
		port = "8083"
	case "gateway":
		port = "8084"
	}

	viper.SetDefault("service.port", port)
	viper.SetDefault("service.version", "dev")
	viper.SetDefault("services.delivery", "localhost:50051")
	viper.SetDefault("services.tracking", "localhost:50052")
	viper.SetDefault("services.notification", "localhost:50053")
	viper.SetDefault("services.analytics", "localhost:50054")
	viper.SetDefault("database.url", "postgres://postgres:postgres@localhost:5432/delivertrack?sslmode=disable")
	viper.SetDefault("mongodb.url", "mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin")
	viper.SetDefault("redis.url", "localhost:6379")
	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("auth.jwt_secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.jwt_expiration", "24h")
	viper.SetDefault("vault.address", "http://localhost:8200")
	viper.SetDefault("vault.token", "root")
	viper.SetDefault("vault.path", fmt.Sprintf("secret/data/%s", serviceName))
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "console")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.sampling.initial", 100)
	viper.SetDefault("logging.sampling.thereafter", 100)
	viper.SetDefault("logging.rotation.max_size", 100)
	viper.SetDefault("logging.rotation.max_age", 30)
	viper.SetDefault("logging.rotation.max_backups", 3)
	viper.SetDefault("logging.rotation.compress", true)
}

// GetEnv is a helper function to get environment variable with fallback
func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
