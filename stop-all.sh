#!/bin/bash

# Simple script to stop all DeliverTrack services

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

print_header "DeliverTrack - Stop All Services"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running."
    exit 1
fi

print_info "Stopping all services..."

# Stop all services
if command -v docker-compose > /dev/null 2>&1; then
    docker-compose down
else
    docker compose down
fi

print_success "All services stopped!"