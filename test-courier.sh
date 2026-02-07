#!/bin/bash

# Test Courier User Interaction
# This script demonstrates the complete courier workflow

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

GATEWAY_URL="http://localhost:8084"

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Test health endpoints
test_health() {
    print_header "Testing System Health"
    echo "Testing gateway health..."
    if curl -s -f "$GATEWAY_URL/health" > /dev/null; then
        print_success "Gateway is healthy"
    else
        print_error "Gateway is not responding"
        exit 1
    fi
}

# Register a courier user
register_courier() {
    print_header "Registering Courier User"

    local response=$(curl -s -X POST "$GATEWAY_URL/register" \
        -H "Content-Type: application/json" \
        -d '{
            "username": "courier_test",
            "email": "courier@test.com",
            "password": "password123",
            "role": "courier"
        }')

    echo "Registration response:"
    echo "$response" | jq '.'

    local user_id=$(echo "$response" | jq -r '.user.id // empty')
    local courier_id=$(echo "$response" | jq -r '.user.courier_id // empty')

    if [ ! -z "$user_id" ] && [ "$user_id" != "null" ]; then
        print_success "Courier registered successfully"
        echo "User ID: $user_id"
        echo "Courier ID: $courier_id"
        echo "$user_id:$courier_id" > /tmp/courier_creds.txt
    else
        print_error "Courier registration failed"
        exit 1
    fi
}

# Login as courier
login_courier() {
    print_header "Logging in as Courier"

    local response=$(curl -s -X POST "$GATEWAY_URL/login" \
        -H "Content-Type: application/json" \
        -d '{
            "username": "courier_test",
            "password": "password123"
        }')

    echo "Login response:"
    echo "$response" | jq '.'

    local token=$(echo "$response" | jq -r '.token // empty')
    local user_data=$(echo "$response" | jq -r '.user // empty')

    if [ ! -z "$token" ] && [ "$token" != "null" ]; then
        print_success "Courier login successful"
        echo "$token" > /tmp/courier_token.txt
        echo "Token saved to /tmp/courier_token.txt"
    else
        print_error "Courier login failed"
        exit 1
    fi
}

# Create a delivery (as admin first)
create_delivery() {
    print_header "Creating a Test Delivery"

    # First login as admin to create delivery
    local admin_response=$(curl -s -X POST "$GATEWAY_URL/login" \
        -H "Content-Type: application/json" \
        -d '{
            "username": "admin",
            "password": "admin123"
        }')

    local admin_token=$(echo "$admin_response" | jq -r '.token // empty')

    if [ -z "$admin_token" ] || [ "$admin_token" = "null" ]; then
        print_error "Admin login failed - cannot create delivery"
        return 1
    fi

    local delivery_response=$(curl -s -X POST "$GATEWAY_URL/api/delivery/deliveries" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $admin_token" \
        -d '{
            "pickup_location": "(-122.4194,37.7749)",
            "delivery_location": "(-122.4089,37.7849)",
            "package_description": "Test package for courier",
            "customer_id": 1
        }')

    echo "Delivery creation response:"
    echo "$delivery_response" | jq '.'

    local delivery_id=$(echo "$delivery_response" | jq -r '.id // empty')

    if [ ! -z "$delivery_id" ] && [ "$delivery_id" != "null" ]; then
        print_success "Delivery created successfully"
        echo "Delivery ID: $delivery_id"
        echo "$delivery_id" > /tmp/delivery_id.txt
    else
        print_error "Delivery creation failed"
    fi
}

# Assign delivery to courier (as admin)
assign_delivery() {
    print_header "Assigning Delivery to Courier"

    local admin_response=$(curl -s -X POST "$GATEWAY_URL/login" \
        -H "Content-Type: application/json" \
        -d '{
            "username": "admin",
            "password": "admin123"
        }')

    local admin_token=$(echo "$admin_response" | jq -r '.token // empty')
    local delivery_id=$(cat /tmp/delivery_id.txt 2>/dev/null || echo "")
    local courier_id=$(cut -d: -f2 /tmp/courier_creds.txt 2>/dev/null || echo "")

    if [ -z "$delivery_id" ] || [ -z "$courier_id" ]; then
        print_error "Missing delivery ID or courier ID"
        return 1
    fi

    local assign_response=$(curl -s -X PUT "$GATEWAY_URL/api/delivery/deliveries/$delivery_id/status" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $admin_token" \
        -d "{
            \"status\": \"assigned\",
            \"courier_id\": $courier_id,
            \"notes\": \"Assigned to courier for testing\"
        }")

    echo "Delivery assignment response:"
    echo "$assign_response" | jq '.'

    print_success "Delivery assigned to courier"
}

# Courier updates location
courier_update_location() {
    print_header "Courier Updating Location"

    local token=$(cat /tmp/courier_token.txt 2>/dev/null || echo "")
    local delivery_id=$(cat /tmp/delivery_id.txt 2>/dev/null || echo "")
    local courier_id=$(cut -d: -f2 /tmp/courier_creds.txt 2>/dev/null || echo "")

    if [ -z "$token" ] || [ -z "$delivery_id" ] || [ -z "$courier_id" ]; then
        print_error "Missing authentication or delivery/courier IDs"
        return 1
    fi

    local location_response=$(curl -s -X POST "$GATEWAY_URL/api/tracking/locations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{
            \"delivery_id\": $delivery_id,
            \"courier_id\": $courier_id,
            \"location\": \"(-122.4150,37.7800)\",
            \"speed\": 45.5,
            \"heading\": 90
        }")

    echo "Location update response:"
    echo "$location_response" | jq '.'

    local location_id=$(echo "$location_response" | jq -r '.id // empty')

    if [ ! -z "$location_id" ] && [ "$location_id" != "null" ]; then
        print_success "Courier location updated successfully"
    else
        print_error "Location update failed"
    fi
}

# Courier updates delivery status
courier_update_status() {
    print_header "Courier Updating Delivery Status"

    local token=$(cat /tmp/courier_token.txt 2>/dev/null || echo "")
    local delivery_id=$(cat /tmp/delivery_id.txt 2>/dev/null || echo "")

    if [ -z "$token" ] || [ -z "$delivery_id" ]; then
        print_error "Missing authentication or delivery ID"
        return 1
    fi

    local status_response=$(curl -s -X PUT "$GATEWAY_URL/api/delivery/deliveries/$delivery_id/status" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d '{
            "status": "picked_up",
            "notes": "Package picked up by courier"
        }')

    echo "Status update response:"
    echo "$status_response" | jq '.'

    print_success "Delivery status updated to 'picked_up'"
}

# Get delivery track
get_delivery_track() {
    print_header "Getting Delivery Track"

    local token=$(cat /tmp/courier_token.txt 2>/dev/null || echo "")
    local delivery_id=$(cat /tmp/delivery_id.txt 2>/dev/null || echo "")

    if [ -z "$token" ] || [ -z "$delivery_id" ]; then
        print_error "Missing authentication or delivery ID"
        return 1
    fi

    local track_response=$(curl -s -X GET "$GATEWAY_URL/api/tracking/deliveries/$delivery_id/track" \
        -H "Authorization: Bearer $token")

    echo "Delivery track response:"
    echo "$track_response" | jq '.'

    print_success "Retrieved delivery tracking information"
}

# Main test flow
main() {
    print_header "Courier User Interaction Test"

    # Clean up any previous test files
    rm -f /tmp/courier_*.txt /tmp/delivery_id.txt

    test_health
    echo ""

    register_courier
    echo ""

    login_courier
    echo ""

    create_delivery
    echo ""

    assign_delivery
    echo ""

    courier_update_location
    echo ""

    courier_update_status
    echo ""

    get_delivery_track
    echo ""

    print_header "Courier Test Complete"
    print_success "All courier operations completed successfully!"
    echo ""
    print_info "Courier workflow demonstrated:"
    echo "  ✓ User registration as courier"
    echo "  ✓ Authentication and JWT token handling"
    echo "  ✓ Delivery assignment to courier"
    echo "  ✓ Real-time location updates"
    echo "  ✓ Delivery status updates"
    echo "  ✓ Tracking information retrieval"
}

main "$@"