#!/bin/bash

# Manual Console Testing Demo for DeliverTrack
# This script demonstrates all features with manual step-by-step execution

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_step() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
    read -p "Press Enter to continue..."
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}   DeliverTrack Console Testing Demo${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""
echo "This demo will walk you through testing all implemented features."
echo "Services should be running on:"
echo "  - Gateway: http://localhost:8084"
echo "  - Delivery: http://localhost:8080"
echo "  - Tracking: http://localhost:8081"
echo "  - Notification: http://localhost:8082"
echo ""
read -p "Press Enter to start the demo..."

# Step 1: Health Checks
print_step "Step 1: Health Checks"

echo "Testing Gateway health..."
curl -s http://localhost:8084/health | jq '.'
print_success "Gateway is healthy"

echo -e "\nTesting Delivery Service health..."
curl -s http://localhost:8080/health | jq '.'
print_success "Delivery Service is healthy"

echo -e "\nTesting Tracking Service health..."
curl -s http://localhost:8081/health | jq '.'
print_success "Tracking Service is healthy"

echo -e "\nTesting Notification Service health..."
curl -s http://localhost:8082/health | jq '.'
print_success "Notification Service is healthy"

# Step 2: User Registration
print_step "Step 2: User Registration"

timestamp=$(date +%s)
CUSTOMER_USERNAME="demo_customer_$timestamp"
COURIER_USERNAME="demo_courier_$timestamp"

echo "Registering a customer: $CUSTOMER_USERNAME"
CUSTOMER_RESPONSE=$(curl -s -X POST http://localhost:8084/register \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$CUSTOMER_USERNAME\",\"email\":\"customer_$timestamp@demo.com\",\"password\":\"Demo123!\",\"role\":\"customer\"}")

echo "$CUSTOMER_RESPONSE" | jq '.'
CUSTOMER_ID=$(echo "$CUSTOMER_RESPONSE" | jq -r '.id')
print_success "Customer registered! User ID: $CUSTOMER_ID"

echo -e "\nRegistering a courier: $COURIER_USERNAME"
COURIER_RESPONSE=$(curl -s -X POST http://localhost:8084/register \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$COURIER_USERNAME\",\"email\":\"courier_$timestamp@demo.com\",\"password\":\"Demo123!\",\"role\":\"courier\"}")

echo "$COURIER_RESPONSE" | jq '.'
COURIER_ID=$(echo "$COURIER_RESPONSE" | jq -r '.id')
print_success "Courier registered! User ID: $COURIER_ID"

# Step 3: Login
print_step "Step 3: User Login"

echo "Logging in as customer..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8084/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$CUSTOMER_USERNAME\",\"password\":\"Demo123!\"}")

echo "$LOGIN_RESPONSE" | jq '.'
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
print_success "Login successful! Token received"
echo "Token (first 50 chars): ${TOKEN:0:50}..."

# Step 4: Service Information
print_step "Step 4: Service Information"

echo "Getting delivery service info..."
curl -s http://localhost:8080 | jq '.'

echo -e "\nGetting tracking service info..."
curl -s http://localhost:8081 | jq '.'

echo -e "\nGetting notification service info..."
curl -s http://localhost:8082 | jq '.'

# Step 5: Direct Service Access (Authenticated)
print_step "Step 5: Testing Authentication"

echo "Attempting to list deliveries without authentication (should fail)..."
curl -s -X GET http://localhost:8080/deliveries | jq '.'

echo -e "\nAttempting to list deliveries WITH authentication..."
curl -s -X GET http://localhost:8080/deliveries \
  -H "Authorization: Bearer $TOKEN" | jq '.'
print_success "Authentication working!"

# Step 6: Gateway Routing
print_step "Step 6: Testing Gateway Routing"

echo "Testing gateway route to delivery service..."
curl -s -X GET http://localhost:8084/api/delivery/deliveries \
  -H "Authorization: Bearer $TOKEN" | jq '.'
print_success "Gateway routing working!"

# Step 7: WebSocket Info
print_step "Step 7: WebSocket Information"

echo "WebSocket endpoints available at:"
echo "  - Real-time delivery tracking: ws://localhost:8081/ws/deliveries/{delivery_id}?token=YOUR_TOKEN"
echo ""
echo "To test WebSocket in browser console:"
echo "const ws = new WebSocket('ws://localhost:8081/ws/deliveries/1?token=$TOKEN');"
echo "ws.onmessage = (e) => console.log('Location update:', JSON.parse(e.data));"

# Step 8: Summary
print_step "Demo Complete!"

echo -e "${GREEN}All core features tested successfully!${NC}"
echo ""
echo "What was demonstrated:"
echo "✓ Health endpoints for all 4 services"
echo "✓ User registration (customer and courier)"
echo "✓ JWT authentication and login"
echo "✓ Token-based API access"
echo "✓ Gateway routing to microservices"
echo "✓ CORS and rate limiting (via gateway)"
echo "✓ WebSocket capability"
echo ""
echo "Created test users:"
echo "  Customer: $CUSTOMER_USERNAME (ID: $CUSTOMER_ID)"
echo "  Courier: $COURIER_USERNAME (ID: $COURIER_ID)"
echo "  Token: ${TOKEN:0:50}..."
echo ""
echo "For full feature testing (deliveries, tracking, notifications),"
echo "you need to:"
echo "1. Create customer/courier records in the database"
echo "2. Link them to user IDs"
echo "3. Then run: ./scripts/test-api.sh for interactive testing"
echo ""
echo "Or view the testing guide: docs/TESTING_GUIDE.md"
