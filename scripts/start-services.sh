#!/bin/bash

# Start all DeliverTrack services

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

print_header "DeliverTrack Service Startup"

# Check if services are already running
print_info "Checking for existing processes..."
existing=$(ps aux | grep -E "go run.*cmd/(gateway|delivery|tracking|notification)" | grep -v grep | wc -l)
if [ "$existing" -gt 0 ]; then
    print_info "Found $existing existing service processes. Stopping them..."
    pkill -f "go run.*cmd/" || true
    sleep 2
fi

# Check if Docker containers are running
print_info "Checking Docker containers..."
if ! docker-compose ps | grep -q "Up"; then
    print_info "Starting Docker containers (postgres, mongodb, redis, rabbitmq)..."
    docker-compose up -d postgres mongodb redis rabbitmq
    print_success "Docker containers started"
    print_info "Waiting for containers to be ready (10 seconds)..."
    sleep 10
else
    print_success "Docker containers are already running"
fi

# Start Gateway Service
print_info "Starting Gateway Service on port 8084..."
cd /home/keneke/projects/DeliverTrack
nohup go run ./cmd/gateway/main.go > logs/gateway.log 2>&1 &
GATEWAY_PID=$!
echo $GATEWAY_PID > logs/gateway.pid
print_success "Gateway Service started (PID: $GATEWAY_PID)"
sleep 2

# Start Delivery Service
print_info "Starting Delivery Service on port 8080..."
nohup go run ./cmd/delivery/main.go > logs/delivery.log 2>&1 &
DELIVERY_PID=$!
echo $DELIVERY_PID > logs/delivery.pid
print_success "Delivery Service started (PID: $DELIVERY_PID)"
sleep 2

# Start Tracking Service
print_info "Starting Tracking Service on port 8081..."
nohup go run ./cmd/tracking/main.go > logs/tracking.log 2>&1 &
TRACKING_PID=$!
echo $TRACKING_PID > logs/tracking.pid
print_success "Tracking Service started (PID: $TRACKING_PID)"
sleep 2

# Start Notification Service
print_info "Starting Notification Service on port 8082..."
nohup go run ./cmd/notification/main.go > logs/notification.log 2>&1 &
NOTIFICATION_PID=$!
echo $NOTIFICATION_PID > logs/notification.pid
print_success "Notification Service started (PID: $NOTIFICATION_PID)"
sleep 2

print_header "Service Status"

echo ""
print_info "Gateway Service:      http://localhost:8084 (PID: $GATEWAY_PID)"
print_info "Delivery Service:     http://localhost:8080 (PID: $DELIVERY_PID)"
print_info "Tracking Service:     http://localhost:8081 (PID: $TRACKING_PID)"
print_info "Notification Service: http://localhost:8082 (PID: $NOTIFICATION_PID)"
echo ""

# Wait a moment for services to fully start
print_info "Waiting for services to initialize (5 seconds)..."
sleep 5

# Check health endpoints
print_header "Health Check"

check_health() {
    local service=$1
    local url=$2
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [ "$response" == "200" ]; then
        print_success "$service is healthy"
        return 0
    else
        print_error "$service is not responding (HTTP $response)"
        return 1
    fi
}

all_healthy=true

check_health "Gateway" "http://localhost:8084/health" || all_healthy=false
check_health "Delivery" "http://localhost:8080/health" || all_healthy=false
check_health "Tracking" "http://localhost:8081/health" || all_healthy=false
check_health "Notification" "http://localhost:8082/health" || all_healthy=false

echo ""

if [ "$all_healthy" = true ]; then
    print_success "All services are running!"
    echo ""
    print_info "Logs are available in:"
    print_info "  - logs/gateway.log"
    print_info "  - logs/delivery.log"
    print_info "  - logs/tracking.log"
    print_info "  - logs/notification.log"
    echo ""
    print_info "To view logs in real-time:"
    print_info "  tail -f logs/*.log"
    echo ""
    print_info "To stop all services:"
    print_info "  ./scripts/stop-services.sh"
    echo ""
    print_info "To run automated tests:"
    print_info "  ./scripts/quick-test.sh"
    echo ""
    print_info "To run interactive tests:"
    print_info "  ./scripts/test-api.sh"
else
    print_error "Some services failed to start. Check the logs for details."
    exit 1
fi
