#!/bin/bash

# API Testing Script for DeliverTrack
# This script provides interactive testing for all implemented endpoints

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Base URLs
GATEWAY_URL="http://localhost:8084"
DELIVERY_URL="http://localhost:8080"
TRACKING_URL="http://localhost:8081"
NOTIFICATION_URL="http://localhost:8082"

# Global variables
TOKEN=""
USER_ID=""
CUSTOMER_ID=""
COURIER_ID=""
ROLE=""

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

# Make HTTP request and pretty print response
make_request() {
    local method=$1
    local url=$2
    local data=$3
    local auth_header=$4

    echo -e "${YELLOW}Request: $method $url${NC}"
    if [ ! -z "$data" ]; then
        echo -e "${YELLOW}Data: $data${NC}"
    fi
    
    if [ ! -z "$auth_header" ]; then
        response=$(curl -s -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$data")
    else
        response=$(curl -s -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    echo -e "${GREEN}Response:${NC}"
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    echo ""
    
    echo "$response"
}

# Test health endpoints
test_health() {
    print_header "Testing Health Endpoints"
    
    print_info "Gateway Health..."
    make_request GET "$GATEWAY_URL/health" "" ""
    
    print_info "Delivery Service Health..."
    make_request GET "$DELIVERY_URL/health" "" ""
    
    print_info "Tracking Service Health..."
    make_request GET "$TRACKING_URL/health" "" ""
    
    print_info "Notification Service Health..."
    make_request GET "$NOTIFICATION_URL/health" "" ""
}

# Register a new user
register_user() {
    print_header "User Registration"
    
    echo "Select user type:"
    echo "1) Customer"
    echo "2) Courier"
    echo "3) Admin"
    read -p "Choice: " choice
    
    case $choice in
        1) ROLE="customer" ;;
        2) ROLE="courier" ;;
        3) ROLE="admin" ;;
        *) ROLE="customer" ;;
    esac
    
    read -p "Username: " username
    read -p "Email: " email
    read -sp "Password: " password
    echo ""
    
    data="{\"username\":\"$username\",\"email\":\"$email\",\"password\":\"$password\",\"role\":\"$ROLE\"}"
    
    response=$(make_request POST "$GATEWAY_URL/register" "$data" "")
    
    # Extract user info from response
    USER_ID=$(echo "$response" | jq -r '.user.id // empty')
    CUSTOMER_ID=$(echo "$response" | jq -r '.user.customer_id // empty')
    COURIER_ID=$(echo "$response" | jq -r '.user.courier_id // empty')
    
    if [ ! -z "$USER_ID" ]; then
        print_success "User registered successfully! User ID: $USER_ID"
    fi
}

# Login
login() {
    print_header "User Login"
    
    read -p "Username: " username
    read -sp "Password: " password
    echo ""
    
    data="{\"username\":\"$username\",\"password\":\"$password\"}"
    
    response=$(make_request POST "$GATEWAY_URL/login" "$data" "")
    
    # Extract token and user info
    TOKEN=$(echo "$response" | jq -r '.token // empty')
    USER_ID=$(echo "$response" | jq -r '.user.id // empty')
    ROLE=$(echo "$response" | jq -r '.user.role // empty')
    CUSTOMER_ID=$(echo "$response" | jq -r '.user.customer_id // empty')
    COURIER_ID=$(echo "$response" | jq -r '.user.courier_id // empty')
    
    if [ ! -z "$TOKEN" ]; then
        print_success "Login successful!"
        print_info "Token: ${TOKEN:0:20}..."
        print_info "User ID: $USER_ID"
        print_info "Role: $ROLE"
        [ ! -z "$CUSTOMER_ID" ] && [ "$CUSTOMER_ID" != "null" ] && print_info "Customer ID: $CUSTOMER_ID"
        [ ! -z "$COURIER_ID" ] && [ "$COURIER_ID" != "null" ] && print_info "Courier ID: $COURIER_ID"
    else
        print_error "Login failed!"
    fi
}

# Create a delivery
create_delivery() {
    print_header "Create Delivery"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Customer ID [$CUSTOMER_ID]: " cust_id
    cust_id=${cust_id:-$CUSTOMER_ID}
    
    echo "Pickup Location (format: (-122.4194,37.7749) coordinates)"
    read -p "Default: (-122.4194,37.7749) [San Francisco]: " pickup
    pickup=${pickup:-"(-122.4194,37.7749)"}
    
    echo "Delivery Location (format: (-122.4194,37.7849) coordinates)"
    read -p "Default: (-122.4089,37.7849) [San Francisco Downtown]: " delivery
    delivery=${delivery:-"(-122.4089,37.7849)"}
    
    echo "Scheduled Date (format: 2026-02-03T12:00:00Z)"
    read -p "Default: $(date -u -d '+1 hour' '+%Y-%m-%dT%H:%M:%SZ'): " scheduled
    scheduled=${scheduled:-$(date -u -d '+1 hour' '+%Y-%m-%dT%H:%M:%SZ')}
    
    read -p "Notes (optional): " notes
    
    data="{\"customer_id\":$cust_id,\"pickup_location\":\"$pickup\",\"delivery_location\":\"$delivery\",\"scheduled_date\":\"$scheduled\""
    [ ! -z "$notes" ] && data="$data,\"notes\":\"$notes\""
    data="$data}"
    
    response=$(make_request POST "$GATEWAY_URL/api/delivery/deliveries/" "$data" "auth")
    
    DELIVERY_ID=$(echo "$response" | jq -r '.id // empty')
    if [ ! -z "$DELIVERY_ID" ]; then
        print_success "Delivery created! ID: $DELIVERY_ID"
    fi
}

# Get delivery by ID
get_delivery() {
    print_header "Get Delivery"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Delivery ID: " delivery_id
    
    make_request GET "$GATEWAY_URL/api/delivery/deliveries/$delivery_id" "" "auth"
}

# List deliveries
list_deliveries() {
    print_header "List Deliveries"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Filter by status (pending/assigned/picked_up/in_transit/delivered/cancelled) [optional]: " status
    read -p "Filter by customer ID [optional]: " customer_id
    
    url="$GATEWAY_URL/api/delivery/deliveries?"
    [ ! -z "$status" ] && url="${url}status=$status&"
    [ ! -z "$customer_id" ] && url="${url}customer_id=$customer_id&"
    
    make_request GET "$url" "" "auth"
}

# Update delivery status
update_delivery_status() {
    print_header "Update Delivery Status"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Delivery ID: " delivery_id
    
    echo "Select new status:"
    echo "1) pending"
    echo "2) assigned"
    echo "3) picked_up"
    echo "4) in_transit"
    echo "5) delivered"
    echo "6) cancelled"
    read -p "Choice: " choice
    
    case $choice in
        1) status="pending" ;;
        2) status="assigned" ;;
        3) status="picked_up" ;;
        4) status="in_transit" ;;
        5) status="delivered" ;;
        6) status="cancelled" ;;
        *) status="pending" ;;
    esac
    
    read -p "Notes (optional): " notes
    
    data="{\"status\":\"$status\""
    [ ! -z "$notes" ] && data="$data,\"notes\":\"$notes\""
    data="$data}"
    
    make_request PUT "$GATEWAY_URL/api/delivery/deliveries/$delivery_id/status" "$data" "auth"
}

# Record location
record_location() {
    print_header "Record Location"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Delivery ID: " delivery_id
    read -p "Courier ID [$COURIER_ID]: " courier_id
    courier_id=${courier_id:-$COURIER_ID}
    
    echo "Location (format: (longitude,latitude) coordinates)"
    read -p "Default: (-122.4150,37.7800): " location
    location=${location:-"(-122.4150,37.7800)"}
    
    read -p "Speed (km/h) [optional]: " speed
    read -p "Heading (degrees) [optional]: " heading
    
    data="{\"delivery_id\":$delivery_id,\"courier_id\":$courier_id,\"location\":\"$location\""
    [ ! -z "$speed" ] && data="$data,\"speed\":$speed"
    [ ! -z "$heading" ] && data="$data,\"heading\":$heading"
    data="$data}"
    
    make_request POST "$GATEWAY_URL/api/tracking/locations" "$data" "auth"
}

# Get delivery track
get_delivery_track() {
    print_header "Get Delivery Track"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Delivery ID: " delivery_id
    
    make_request GET "$GATEWAY_URL/api/tracking/deliveries/$delivery_id/track" "" "auth"
}

# Get current location
get_current_location() {
    print_header "Get Current Location"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Delivery ID: " delivery_id
    
    make_request GET "$GATEWAY_URL/api/tracking/deliveries/$delivery_id/location" "" "auth"
}

# Get courier location
get_courier_location() {
    print_header "Get Courier Location"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Courier ID [$COURIER_ID]: " courier_id
    courier_id=${courier_id:-$COURIER_ID}
    
    make_request GET "$GATEWAY_URL/api/tracking/couriers/$courier_id/location" "" "auth"
}

# Calculate ETA
calculate_eta() {
    print_header "Calculate ETA"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Delivery ID: " delivery_id
    
    echo "Current Location (format: (longitude,latitude) coordinates)"
    read -p "Default: (-122.4150,37.7800): " current_location
    current_location=${current_location:-"(-122.4150,37.7800)"}
    
    data="{\"current_location\":\"$current_location\"}"
    
    make_request POST "$GATEWAY_URL/api/tracking/deliveries/$delivery_id/eta" "$data" "auth"
}

# Send notification
send_notification() {
    print_header "Send Notification"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "User ID: " user_id
    read -p "Message: " message
    
    echo "Select notification type:"
    echo "1) delivery_created"
    echo "2) delivery_assigned"
    echo "3) delivery_picked_up"
    echo "4) delivery_in_transit"
    echo "5) delivery_delivered"
    echo "6) delivery_cancelled"
    echo "7) location_update"
    read -p "Choice: " choice
    
    case $choice in
        1) type="delivery_created" ;;
        2) type="delivery_assigned" ;;
        3) type="delivery_picked_up" ;;
        4) type="delivery_in_transit" ;;
        5) type="delivery_delivered" ;;
        6) type="delivery_cancelled" ;;
        7) type="location_update" ;;
        *) type="delivery_created" ;;
    esac
    
    data="{\"user_id\":$user_id,\"message\":\"$message\",\"type\":\"$type\"}"
    
    make_request POST "$GATEWAY_URL/api/notification/notifications/" "$data" "auth"
}

# Get user notifications
get_notifications() {
    print_header "Get User Notifications"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    make_request GET "$GATEWAY_URL/api/notification/notifications" "" "auth"
}

# Mark notification as read
mark_notification_read() {
    print_header "Mark Notification as Read"
    
    if [ -z "$TOKEN" ]; then
        print_error "Please login first!"
        return
    fi
    
    read -p "Notification ID: " notification_id
    
    make_request PUT "$GATEWAY_URL/api/notification/notifications/$notification_id/read" "" "auth"
}

# Main menu
main_menu() {
    while true; do
        echo ""
        print_header "DeliverTrack API Testing Menu"
        
        if [ ! -z "$TOKEN" ]; then
            print_info "Logged in as: $ROLE (User ID: $USER_ID)"
        else
            print_info "Not logged in"
        fi
        
        echo ""
        echo "=== System ==="
        echo "1) Test Health Endpoints"
        echo ""
        echo "=== Authentication ==="
        echo "2) Register User"
        echo "3) Login"
        echo ""
        echo "=== Delivery Service ==="
        echo "4) Create Delivery"
        echo "5) Get Delivery by ID"
        echo "6) List Deliveries"
        echo "7) Update Delivery Status"
        echo ""
        echo "=== Tracking Service ==="
        echo "8) Record Location"
        echo "9) Get Delivery Track"
        echo "10) Get Current Location"
        echo "11) Get Courier Location"
        echo "12) Calculate ETA"
        echo ""
        echo "=== Notification Service ==="
        echo "13) Send Notification"
        echo "14) Get User Notifications"
        echo "15) Mark Notification as Read"
        echo ""
        echo "0) Exit"
        echo ""
        read -p "Select option: " option
        
        case $option in
            1) test_health ;;
            2) register_user ;;
            3) login ;;
            4) create_delivery ;;
            5) get_delivery ;;
            6) list_deliveries ;;
            7) update_delivery_status ;;
            8) record_location ;;
            9) get_delivery_track ;;
            10) get_current_location ;;
            11) get_courier_location ;;
            12) calculate_eta ;;
            13) send_notification ;;
            14) get_notifications ;;
            15) mark_notification_read ;;
            0) 
                print_info "Goodbye!"
                exit 0
                ;;
            *)
                print_error "Invalid option!"
                ;;
        esac
    done
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    print_error "jq is not installed. Install it for better JSON formatting."
    print_info "On Ubuntu/Debian: sudo apt-get install jq"
    print_info "On macOS: brew install jq"
    exit 1
fi

# Start the menu
main_menu
