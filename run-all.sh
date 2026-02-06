#!/bin/bash

# Simple script to run DeliverTrack backend and frontend using Docker Compose

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================${NC}"
}

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_header "DeliverTrack - Start All Services"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose > /dev/null 2>&1 && ! docker compose version > /dev/null 2>&1; then
    print_error "docker-compose is not installed."
    exit 1
fi

print_info "Starting all services with Docker Compose..."
print_info "This may take a few minutes on first run..."

# Start all services
if command -v docker-compose > /dev/null 2>&1; then
    docker-compose up -d
else
    docker compose up -d
fi

print_success "Services started successfully!"

echo ""
print_info "Services will be available at:"
echo "  - Frontend (Customer Portal): http://localhost:3000"
echo "  - API Gateway: http://localhost:8084"
echo "  - Delivery Service: http://localhost:8080 (gRPC: 50051)"
echo "  - Tracking Service: http://localhost:8081 (gRPC: 50052)"
echo "  - Notification Service: http://localhost:8082 (gRPC: 50053)"
echo "  - Analytics Service: http://localhost:8083 (gRPC: 50054)"
echo ""
echo "  - PostgreSQL: localhost:5433"
echo "  - MongoDB: localhost:27017"
echo "  - Redis: localhost:6379"
echo "  - RabbitMQ: localhost:5672 (Management: http://localhost:15672)"
echo "  - Vault: http://localhost:8200"
echo ""
print_info "To stop all services, run: docker-compose down"
print_info "To view logs, run: docker-compose logs -f [service-name]"