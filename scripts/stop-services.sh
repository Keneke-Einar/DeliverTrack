#!/bin/bash

# Stop all DeliverTrack services

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info "Stopping DeliverTrack services..."

# Kill processes by PID files
if [ -f logs/gateway.pid ]; then
    kill $(cat logs/gateway.pid) 2>/dev/null || true
    rm logs/gateway.pid
    print_success "Gateway stopped"
fi

if [ -f logs/delivery.pid ]; then
    kill $(cat logs/delivery.pid) 2>/dev/null || true
    rm logs/delivery.pid
    print_success "Delivery stopped"
fi

if [ -f logs/tracking.pid ]; then
    kill $(cat logs/tracking.pid) 2>/dev/null || true
    rm logs/tracking.pid
    print_success "Tracking stopped"
fi

if [ -f logs/notification.pid ]; then
    kill $(cat logs/notification.pid) 2>/dev/null || true
    rm logs/notification.pid
    print_success "Notification stopped"
fi

# Fallback: kill by process name
pkill -f "go run.*cmd/gateway" 2>/dev/null || true
pkill -f "go run.*cmd/delivery" 2>/dev/null || true
pkill -f "go run.*cmd/tracking" 2>/dev/null || true
pkill -f "go run.*cmd/notification" 2>/dev/null || true

print_success "All services stopped"

# Optionally stop Docker containers
read -p "Stop Docker containers too? (y/N): " stop_docker
if [ "$stop_docker" = "y" ] || [ "$stop_docker" = "Y" ]; then
    print_info "Stopping Docker containers..."
    docker-compose down
    print_success "Docker containers stopped"
fi
