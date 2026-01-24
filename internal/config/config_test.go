package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test default values
	cfg := Load()

	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	if cfg.Port == "" {
		t.Error("Port should have a default value")
	}

	if cfg.Host == "" {
		t.Error("Host should have a default value")
	}
}

func TestGetEnv(t *testing.T) {
	testKey := "TEST_KEY_12345"
	testValue := "test_value"
	defaultValue := "default"

	// Test default value when env var is not set
	result := getEnv(testKey, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected %s, got %s", defaultValue, result)
	}

	// Test actual value when env var is set
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	result = getEnv(testKey, defaultValue)
	if result != testValue {
		t.Errorf("Expected %s, got %s", testValue, result)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	testKey := "TEST_INT_KEY_12345"
	defaultValue := 42

	// Test default value when env var is not set
	result := getEnvAsInt(testKey, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected %d, got %d", defaultValue, result)
	}

	// Test actual value when env var is set
	os.Setenv(testKey, "100")
	defer os.Unsetenv(testKey)

	result = getEnvAsInt(testKey, defaultValue)
	if result != 100 {
		t.Errorf("Expected 100, got %d", result)
	}

	// Test invalid value falls back to default
	os.Setenv(testKey, "invalid")
	result = getEnvAsInt(testKey, defaultValue)
	if result != defaultValue {
		t.Errorf("Expected %d for invalid input, got %d", defaultValue, result)
	}
}
