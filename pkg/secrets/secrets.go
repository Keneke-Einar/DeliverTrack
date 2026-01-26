package secrets

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
)

// VaultClient wraps the Vault API client
type VaultClient struct {
	client *api.Client
	path   string
}

// NewVaultClient creates a new Vault client
func NewVaultClient(address, token, path string) (*VaultClient, error) {
	config := &api.Config{
		Address: address,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	client.SetToken(token)

	return &VaultClient{
		client: client,
		path:   path,
	}, nil
}

// GetSecret retrieves a secret from Vault
func (v *VaultClient) GetSecret(key string) (string, error) {
	secret, err := v.client.Logical().Read(v.path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		log.Printf("No secret data found at path %s", v.path)
		return "", fmt.Errorf("no secret data found")
	}

	value, ok := secret.Data[key].(string)
	if !ok {
		return "", fmt.Errorf("secret key %s not found or not a string", key)
	}

	return value, nil
}

// GetSecretWithFallback retrieves a secret from Vault, with fallback to environment variable
func (v *VaultClient) GetSecretWithFallback(key, envKey, fallback string) string {
	if value, err := v.GetSecret(key); err == nil && value != "" {
		return value
	}

	// Fallback to environment variable or default
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}

	return fallback
}