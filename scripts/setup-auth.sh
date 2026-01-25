#!/bin/bash

# Setup script for JWT authentication module

echo "Setting up JWT authentication module..."

# Navigate to auth module
cd internal/auth

echo "Installing Go dependencies..."
go mod tidy
go get github.com/golang-jwt/jwt/v5@v5.2.0
go get golang.org/x/crypto@v0.19.0

echo "Running tests..."
go test -v

echo ""
echo "✅ JWT authentication module setup complete!"
echo ""
echo "Next steps:"
echo "1. Run database migration: psql -U postgres -d delivertrack -f migrations/002_create_users.up.sql"
echo "2. Set environment variable: export JWT_SECRET='your-secret-key'"
echo "3. Check out the example server: examples/auth_server.go"
echo "4. Read the documentation: internal/auth/README.md"
