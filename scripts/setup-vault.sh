#!/bin/bash

# Setup Vault secrets for DeliverTrack services

set -e

# Wait for Vault to be ready
echo "Waiting for Vault to be ready..."
until vault status > /dev/null 2>&1; do
  sleep 2
done

echo "Vault is ready. Setting up secrets..."

# Enable KV secrets engine if not already enabled
vault secrets enable -path=secret kv-v2 || echo "KV secrets engine already enabled"

# Store secrets for delivery service
vault kv put secret/delivery \
  database_url="postgres://postgres:postgres@postgres:5432/delivertrack?sslmode=disable" \
  mongodb_url="mongodb://admin:admin123@mongodb:27017/delivertrack?authSource=admin" \
  redis_url="redis:6379" \
  rabbitmq_url="amqp://guest:guest@rabbitmq:5672/" \
  jwt_secret="your-super-secret-jwt-key-change-in-production"

# Store secrets for tracking service
vault kv put secret/tracking \
  database_url="postgres://postgres:postgres@postgres:5432/delivertrack?sslmode=disable" \
  mongodb_url="mongodb://admin:admin123@mongodb:27017/delivertrack?authSource=admin" \
  redis_url="redis:6379" \
  rabbitmq_url="amqp://guest:guest@rabbitmq:5672/" \
  jwt_secret="your-super-secret-jwt-key-change-in-production"

# Store secrets for notification service
vault kv put secret/notification \
  database_url="postgres://postgres:postgres@postgres:5432/delivertrack?sslmode=disable" \
  mongodb_url="mongodb://admin:admin123@mongodb:27017/delivertrack?authSource=admin" \
  redis_url="redis:6379" \
  rabbitmq_url="amqp://guest:guest@rabbitmq:5672/" \
  jwt_secret="your-super-secret-jwt-key-change-in-production"

# Store secrets for analytics service
vault kv put secret/analytics \
  database_url="postgres://postgres:postgres@postgres:5432/delivertrack?sslmode=disable" \
  mongodb_url="mongodb://admin:admin123@mongodb:27017/delivertrack?authSource=admin" \
  redis_url="redis:6379" \
  rabbitmq_url="amqp://guest:guest@rabbitmq:5672/" \
  jwt_secret="your-super-secret-jwt-key-change-in-production"

echo "Vault secrets setup complete!"