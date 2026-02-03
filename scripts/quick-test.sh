#!/bin/bash

# Quick Test Script for DeliverTrack
# Runs automated tests for all services

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

GATEWAY_URL="http://localhost:8084"
TOKEN=""
DELIVERY_ID=""

print_step() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

# Test 1: Health Checks
print_step "Testing Health Endpoints"

echo "Gateway Health..."
response=$(curl -s "$GATEWAY_URL/health")
echo "$response" | jq '.'
if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
    print_success "Gateway is healthy"
else
    print_error "Gateway health check failed"
fi

echo ""
echo "Delivery Service Health..."
response=$(curl -s "http://localhost:8080/health")
echo "$response" | jq '.'
if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
    print_success "Delivery service is healthy"
else
    print_error "Delivery service health check failed"
fi

echo ""
echo "Tracking Service Health..."
response=$(curl -s "http://localhost:8081/health")
echo "$response" | jq '.'
if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
    print_success "Tracking service is healthy"
else
    print_error "Tracking service health check failed"
fi

echo ""
echo "Notification Service Health..."
response=$(curl -s "http://localhost:8082/health")
echo "$response" | jq '.'
if echo "$response" | jq -e '.status == "ok"' > /dev/null; then
    print_success "Notification service is healthy"
else
    print_error "Notification service health check failed"
fi

# Test 2: Register a test customer
print_step "Registering Test Customer"

timestamp=$(date +%s)
username="testcustomer_$timestamp"
email="test_$timestamp@example.com"
password="TestPass123!"

register_data="{\"username\":\"$username\",\"email\":\"$email\",\"password\":\"$password\",\"role\":\"customer\"}"

echo "Registration request: $register_data"
response=$(curl -s -X POST "$GATEWAY_URL/register" \
    -H "Content-Type: application/json" \
    -d "$register_data")

echo "Registration response:"
echo "$response" | jq '.'

USER_ID=$(echo "$response" | jq -r '.id // empty')
CUSTOMER_ID=$(echo "$response" | jq -r '.customer_id // empty')

# For customers, use user_id as customer_id if not explicitly set
if [ ! -z "$USER_ID" ] && [ "$USER_ID" != "null" ]; then
    if [ -z "$CUSTOMER_ID" ] || [ "$CUSTOMER_ID" == "null" ]; then
        CUSTOMER_ID=$USER_ID
    fi
    print_success "Customer registered successfully! User ID: $USER_ID, Customer ID: $CUSTOMER_ID"
else
    print_error "Customer registration failed"
fi

# Test 3: Login
print_step "Logging In"

login_data="{\"username\":\"$username\",\"password\":\"$password\"}"

echo "Login request: $login_data"
response=$(curl -s -X POST "$GATEWAY_URL/login" \
    -H "Content-Type: application/json" \
    -d "$login_data")

echo "Login response:"
echo "$response" | jq '.'

TOKEN=$(echo "$response" | jq -r '.token // empty')

if [ ! -z "$TOKEN" ]; then
    print_success "Login successful! Token: ${TOKEN:0:30}..."
else
    print_error "Login failed"
fi

# Test 4: Register a courier
print_step "Registering Test Courier"

courier_username="testcourier_$timestamp"
courier_email="courier_$timestamp@example.com"
courier_password="CourierPass123!"

courier_data="{\"username\":\"$courier_username\",\"email\":\"$courier_email\",\"password\":\"$courier_password\",\"role\":\"courier\"}"

echo "Courier registration request: $courier_data"
response=$(curl -s -X POST "$GATEWAY_URL/register" \
    -H "Content-Type: application/json" \
    -d "$courier_data")

echo "Courier registration response:"
echo "$response" | jq '.'

COURIER_USER_ID=$(echo "$response" | jq -r '.id // empty')
COURIER_ID=$(echo "$response" | jq -r '.courier_id // empty')

# For couriers, use user_id as courier_id if not explicitly set
if [ ! -z "$COURIER_USER_ID" ] && [ "$COURIER_USER_ID" != "null" ]; then
    if [ -z "$COURIER_ID" ] || [ "$COURIER_ID" == "null" ]; then
        COURIER_ID=$COURIER_USER_ID
    fi
    print_success "Courier registered successfully! User ID: $COURIER_USER_ID, Courier ID: $COURIER_ID"
else
    print_error "Courier registration failed"
fi

# Test 5: Create a Delivery
print_step "Creating a Delivery"

delivery_data="{
    \"customer_id\": $CUSTOMER_ID,
    \"pickup_location\": \"(-122.4194,37.7749)\",
    \"delivery_location\": \"(-122.4089,37.7849)\",
    \"scheduled_date\": \"$(date -u -d '+2 hours' '+%Y-%m-%dT%H:%M:%SZ')\",
    \"notes\": \"Automated test delivery\"
}"

echo "Create delivery request: $delivery_data"
response=$(curl -s -X POST "$GATEWAY_URL/api/delivery/deliveries/" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "$delivery_data")

echo "Create delivery response:"
echo "$response" | jq '.'

DELIVERY_ID=$(echo "$response" | jq -r '.id // empty')

if [ ! -z "$DELIVERY_ID" ] && [ "$DELIVERY_ID" != "null" ]; then
    print_success "Delivery created successfully! Delivery ID: $DELIVERY_ID"
else
    print_error "Delivery creation failed"
fi

# Test 6: Get Delivery by ID
print_step "Getting Delivery by ID"

response=$(curl -s -X GET "$GATEWAY_URL/api/delivery/deliveries/$DELIVERY_ID" \
    -H "Authorization: Bearer $TOKEN")

echo "Get delivery response:"
echo "$response" | jq '.'

delivery_id_check=$(echo "$response" | jq -r '.id // empty')
if [ "$delivery_id_check" == "$DELIVERY_ID" ]; then
    print_success "Successfully retrieved delivery"
else
    print_error "Failed to retrieve delivery"
fi

# Test 7: List Deliveries
print_step "Listing Deliveries"

response=$(curl -s -X GET "$GATEWAY_URL/api/delivery/deliveries?customer_id=$CUSTOMER_ID" \
    -H "Authorization: Bearer $TOKEN")

echo "List deliveries response:"
echo "$response" | jq '.'

delivery_count=$(echo "$response" | jq 'length')
if [ "$delivery_count" -gt 0 ]; then
    print_success "Successfully listed $delivery_count deliveries"
else
    print_error "Failed to list deliveries"
fi

# Test 8: Update Delivery Status
print_step "Updating Delivery Status to 'assigned'"

# Login as admin or courier to assign delivery
admin_username="admin_$timestamp"
admin_email="admin_$timestamp@example.com"
admin_password="AdminPass123!"

admin_data="{\"username\":\"$admin_username\",\"email\":\"$admin_email\",\"password\":\"$admin_password\",\"role\":\"admin\"}"

echo "Creating admin user..."
response=$(curl -s -X POST "$GATEWAY_URL/register" \
    -H "Content-Type: application/json" \
    -d "$admin_data")

echo "$response" | jq '.'

# Login as admin
    admin_login_data="{\"username\":\"$admin_username\",\"password\":\"$admin_password\"}"
    -H "Content-Type: application/json" \
    -d "$admin_login_data")

ADMIN_TOKEN=$(echo "$response" | jq -r '.token // empty')

if [ ! -z "$ADMIN_TOKEN" ]; then
    print_success "Admin login successful"
    
    # Update status
    status_data="{\"status\":\"assigned\",\"notes\":\"Assigned to courier $COURIER_ID\"}"
    
    echo "Update status request: $status_data"
    response=$(curl -s -X PUT "$GATEWAY_URL/api/delivery/deliveries/$DELIVERY_ID/status" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d "$status_data")
    
    echo "Update status response:"
    echo "$response" | jq '.'
    
    new_status=$(echo "$response" | jq -r '.status // empty')
    if [ "$new_status" == "assigned" ]; then
        print_success "Successfully updated delivery status"
    else
        print_error "Failed to update delivery status"
    fi
else
    print_error "Admin login failed"
fi

# Test 9: Record Location
print_step "Recording Location"

location_data="{
    \"delivery_id\": $DELIVERY_ID,
    \"courier_id\": $COURIER_ID,
    \"location\": \"(-122.4150,37.7800)\",
    \"speed\": 45.5,
    \"heading\": 90
}"

echo "Record location request: $location_data"
response=$(curl -s -X POST "$GATEWAY_URL/api/tracking/locations" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d "$location_data")

echo "Record location response:"
echo "$response" | jq '.'

location_id=$(echo "$response" | jq -r '.id // empty')
if [ ! -z "$location_id" ] && [ "$location_id" != "null" ]; then
    print_success "Successfully recorded location"
else
    print_error "Failed to record location"
fi

# Test 10: Get Delivery Track
print_step "Getting Delivery Track"

response=$(curl -s -X GET "$GATEWAY_URL/api/tracking/deliveries/$DELIVERY_ID/track" \
    -H "Authorization: Bearer $TOKEN")

echo "Get delivery track response:"
echo "$response" | jq '.'

track_count=$(echo "$response" | jq 'length')
if [ "$track_count" -gt 0 ]; then
    print_success "Successfully retrieved delivery track with $track_count location(s)"
else
    echo "No track data yet (may be expected)"
fi

# Test 11: Get Current Location
print_step "Getting Current Location"

response=$(curl -s -X GET "$GATEWAY_URL/api/tracking/deliveries/$DELIVERY_ID/location" \
    -H "Authorization: Bearer $TOKEN")

echo "Get current location response:"
echo "$response" | jq '.'

current_loc=$(echo "$response" | jq -r '.location // empty')
if [ ! -z "$current_loc" ]; then
    print_success "Successfully retrieved current location"
else
    echo "No current location data yet (may be expected)"
fi

# Test 12: Get Courier Location
print_step "Getting Courier Location"

response=$(curl -s -X GET "$GATEWAY_URL/api/tracking/couriers/$COURIER_ID/location" \
    -H "Authorization: Bearer $TOKEN")

echo "Get courier location response:"
echo "$response" | jq '.'

courier_loc=$(echo "$response" | jq -r '.location // empty')
if [ ! -z "$courier_loc" ]; then
    print_success "Successfully retrieved courier location"
else
    echo "No courier location data yet (may be expected)"
fi

# Test 13: Get User Notifications
print_step "Getting User Notifications"

response=$(curl -s -X GET "$GATEWAY_URL/api/notification/notifications" \
    -H "Authorization: Bearer $TOKEN")

echo "Get notifications response:"
echo "$response" | jq '.'

notification_count=$(echo "$response" | jq 'length // 0')
print_success "Retrieved $notification_count notification(s)"

# Summary
print_step "Test Summary"
print_success "All tests completed!"
echo ""
echo "Test Data Created:"
echo "  - Customer ID: $CUSTOMER_ID"
echo "  - Courier ID: $COURIER_ID"
echo "  - Delivery ID: $DELIVERY_ID"
echo "  - Customer Token: ${TOKEN:0:30}..."
echo "  - Admin Token: ${ADMIN_TOKEN:0:30}..."
echo ""
echo "You can use these values for further testing with the interactive script:"
echo "  ./scripts/test-api.sh"
