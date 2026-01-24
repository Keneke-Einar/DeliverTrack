#!/bin/bash

# DeliverTrack API Test Script
# This script demonstrates the complete package lifecycle

BASE_URL="http://localhost:8080/api/v1"
TRACKING_NUMBER="TRK$(date +%s)"

echo "DeliverTrack API Test"
echo "===================="
echo ""

# Color codes for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Health Check
echo -e "${BLUE}1. Testing Health Check...${NC}"
curl -s "$BASE_URL/health" | jq '.'
echo ""
echo ""

# Test 2: Create Package
echo -e "${BLUE}2. Creating a new package (Tracking: $TRACKING_NUMBER)...${NC}"
CREATE_RESPONSE=$(curl -s -X POST "$BASE_URL/packages" \
  -H "Content-Type: application/json" \
  -d "{
    \"tracking_number\": \"$TRACKING_NUMBER\",
    \"sender_name\": \"John Doe\",
    \"sender_address\": \"123 Main St, New York, NY 10001\",
    \"recipient_name\": \"Jane Smith\",
    \"recipient_address\": \"456 Oak Ave, Los Angeles, CA 90001\",
    \"weight\": 2.5,
    \"description\": \"Test Electronics Package\"
  }")
echo "$CREATE_RESPONSE" | jq '.'
echo ""
echo ""

# Test 3: Get Package Details
echo -e "${BLUE}3. Getting package details...${NC}"
curl -s "$BASE_URL/packages/$TRACKING_NUMBER" | jq '.'
echo ""
echo ""

# Test 4: Update to In Transit
echo -e "${BLUE}4. Updating status to 'in_transit'...${NC}"
curl -s -X PUT "$BASE_URL/packages/$TRACKING_NUMBER/status" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_transit",
    "location": "Newark Distribution Center",
    "latitude": 40.7357,
    "longitude": -74.1724
  }' | jq '.'
echo ""
sleep 1
echo ""

# Test 5: Update to Out for Delivery
echo -e "${BLUE}5. Updating status to 'out_for_delivery'...${NC}"
curl -s -X PUT "$BASE_URL/packages/$TRACKING_NUMBER/status" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "out_for_delivery",
    "location": "Los Angeles Local Hub",
    "latitude": 34.0522,
    "longitude": -118.2437
  }' | jq '.'
echo ""
sleep 1
echo ""

# Test 6: Get Location History
echo -e "${BLUE}6. Getting location history...${NC}"
curl -s "$BASE_URL/packages/$TRACKING_NUMBER/locations" | jq '.'
echo ""
echo ""

# Test 7: Update to Delivered
echo -e "${BLUE}7. Updating status to 'delivered'...${NC}"
curl -s -X PUT "$BASE_URL/packages/$TRACKING_NUMBER/status" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "delivered",
    "location": "456 Oak Ave, Los Angeles, CA 90001",
    "latitude": 34.0522,
    "longitude": -118.2437
  }' | jq '.'
echo ""
echo ""

# Test 8: Get Final Package State
echo -e "${BLUE}8. Getting final package state...${NC}"
curl -s "$BASE_URL/packages/$TRACKING_NUMBER" | jq '.'
echo ""
echo ""

# Test 9: List All Packages
echo -e "${BLUE}9. Listing all packages (first 5)...${NC}"
curl -s "$BASE_URL/packages" | jq '.[:5]'
echo ""
echo ""

# Test 10: Get Stats
echo -e "${BLUE}10. Getting system statistics...${NC}"
curl -s "$BASE_URL/stats" | jq '.'
echo ""
echo ""

echo -e "${GREEN}✓ All tests completed!${NC}"
echo -e "${YELLOW}Your test package tracking number: $TRACKING_NUMBER${NC}"
