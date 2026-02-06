#!/bin/bash

# DeliverTrack - Start/Stop all services

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

print_header "DeliverTrack Service Management"

case "${1:-start}" in
    start)
        print_info "Starting all DeliverTrack services..."
        docker-compose up -d
        print_success "Services started!"
        echo ""
        print_info "Available services:"
        echo "  - Frontend (Customer Portal): http://localhost:3000"
        echo "  - Gateway API: http://localhost:8084"
        echo "  - Delivery Service: http://localhost:8080"
        echo "  - Tracking Service: http://localhost:8081"
        echo "  - Notification Service: http://localhost:8082"
        echo "  - Analytics Service: http://localhost:8083"
        echo "  - Vault (Secrets): http://localhost:8200"
        echo "  - PostgreSQL: localhost:5433"
        echo "  - MongoDB: localhost:27017"
        echo "  - Redis: localhost:6379"
        echo "  - RabbitMQ: localhost:5672 (Management: localhost:15672)"
        ;;
    stop)
        print_info "Stopping all DeliverTrack services..."
        docker-compose down
        print_success "Services stopped!"
        ;;
    restart)
        print_info "Restarting all DeliverTrack services..."
        docker-compose restart
        print_success "Services restarted!"
        ;;
    status)
        print_info "Service status:"
        docker-compose ps
        ;;
    logs)
        service="${2:-}"
        if [ -n "$service" ]; then
            docker-compose logs -f "$service"
        else
            docker-compose logs -f
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs [service]}"
        echo ""
        echo "Commands:"
        echo "  start   - Start all services (default)"
        echo "  stop    - Stop all services"
        echo "  restart - Restart all services"
        echo "  status  - Show service status"
        echo "  logs    - Show logs (optionally for specific service)"
        exit 1
        ;;
esac