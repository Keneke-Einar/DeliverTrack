# DeliverTrack API Testing Guide

This guide provides comprehensive instructions for testing all implemented features of the DeliverTrack system via console/command-line interface.

## Prerequisites

- Docker and Docker Compose (for running PostgreSQL, MongoDB, Redis, RabbitMQ)
- Go 1.21+ installed
- `jq` for JSON parsing (install with: `sudo apt-get install jq` or `brew install jq`)
- `curl` for making HTTP requests

## Quick Start

### 1. Start All Services

```bash
./scripts/start-services.sh
```

This script will:
- Start Docker containers (PostgreSQL, MongoDB, Redis, RabbitMQ)
- Start Gateway Service (port 8084)
- Start Delivery Service (port 8080)
- Start Tracking Service (port 8081)
- Start Notification Service (port 8082)
- Perform health checks on all services

### 2. Run Automated Tests

```bash
./scripts/quick-test.sh
```

This script automatically tests:
- ✓ Health endpoints for all services
- ✓ User registration (customer, courier, admin)
- ✓ User login and JWT authentication
- ✓ Delivery creation
- ✓ Delivery retrieval and listing
- ✓ Delivery status updates
- ✓ Location tracking
- ✓ Real-time location queries
- ✓ Notification retrieval

### 3. Interactive Testing

```bash
./scripts/test-api.sh
```

This provides an interactive menu for testing individual features:
- Authentication (register, login)
- Delivery operations (create, read, update)
- Tracking operations (record location, get track, calculate ETA)
- Notification operations (send, read, mark as read)

### 4. Stop All Services

```bash
./scripts/stop-services.sh
```

## Service Architecture

### Gateway Service (Port 8084)
- **Purpose**: API Gateway with authentication, rate limiting, and routing
- **Endpoints**:
  - `POST /login` - User authentication
  - `POST /register` - User registration
  - `GET /health` - Health check
  - `/api/delivery/*` - Routes to Delivery Service
  - `/api/tracking/*` - Routes to Tracking Service
  - `/api/notification/*` - Routes to Notification Service
  - `/api/analytics/*` - Routes to Analytics Service

### Delivery Service (Port 8080)
- **Purpose**: Manage delivery orders
- **Endpoints**:
  - `POST /deliveries` - Create a new delivery
  - `GET /deliveries/:id` - Get delivery by ID
  - `GET /deliveries` - List deliveries (with filters)
  - `PUT /deliveries/:id/status` - Update delivery status

### Tracking Service (Port 8081)
- **Purpose**: Real-time location tracking
- **Endpoints**:
  - `POST /locations` - Record a location update
  - `GET /deliveries/:id/track` - Get full delivery track
  - `GET /deliveries/:id/location` - Get current location
  - `GET /couriers/:id/location` - Get courier's current location
  - `POST /deliveries/:id/eta` - Calculate ETA
  - `WS /ws/deliveries/:id` - WebSocket for real-time updates

### Notification Service (Port 8082)
- **Purpose**: User notifications and event handling
- **Endpoints**:
  - `POST /notifications` - Send a notification
  - `GET /notifications` - Get user's notifications
  - `PUT /notifications/:id/read` - Mark notification as read

## Manual Testing Examples

### 1. Health Check

```bash
curl http://localhost:8084/health | jq '.'
```

Expected response:
```json
{
  "status": "ok",
  "service": "gateway"
}
```

### 2. Register a Customer

```bash
curl -X POST http://localhost:8084/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "SecurePass123!",
    "role": "customer"
  }' | jq '.'
```

Expected response:
```json
{
  "user": {
    "id": 1,
    "username": "john_doe",
    "email": "john@example.com",
    "role": "customer",
    "customer_id": 1,
    "created_at": "2026-02-03T10:00:00Z"
  }
}
```

### 3. Login

```bash
curl -X POST http://localhost:8084/login \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "john_doe",
    "password": "SecurePass123!"
  }' | jq '.'
```

Expected response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "john_doe",
    "email": "john@example.com",
    "role": "customer",
    "customer_id": 1
  }
}
```

**Save the token for subsequent requests:**
```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 4. Create a Delivery

```bash
curl -X POST http://localhost:8084/api/delivery/deliveries/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "customer_id": 1,
    "pickup_location": "POINT(-122.4194 37.7749)",
    "delivery_location": "POINT(-122.4089 37.7849)",
    "scheduled_date": "2026-02-03T15:00:00Z",
    "notes": "Please handle with care"
  }' | jq '.'
```

Expected response:
```json
{
  "id": 1,
  "customer_id": 1,
  "courier_id": null,
  "status": "pending",
  "pickup_location": "POINT(-122.4194 37.7749)",
  "delivery_location": "POINT(-122.4089 37.7849)",
  "scheduled_date": "2026-02-03T15:00:00Z",
  "notes": "Please handle with care",
  "created_at": "2026-02-03T10:05:00Z",
  "updated_at": "2026-02-03T10:05:00Z"
}
```

### 5. Get Delivery by ID

```bash
curl -X GET http://localhost:8084/api/delivery/deliveries/1 \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

### 6. List Deliveries

```bash
# List all deliveries
curl -X GET http://localhost:8084/api/delivery/deliveries \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Filter by status
curl -X GET "http://localhost:8084/api/delivery/deliveries?status=pending" \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Filter by customer
curl -X GET "http://localhost:8084/api/delivery/deliveries?customer_id=1" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

### 7. Update Delivery Status

```bash
curl -X PUT http://localhost:8084/api/delivery/deliveries/1/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "status": "assigned",
    "notes": "Assigned to courier #5"
  }' | jq '.'
```

Available statuses:
- `pending` - Initial state
- `assigned` - Assigned to a courier
- `picked_up` - Courier picked up the package
- `in_transit` - Package is being delivered
- `delivered` - Successfully delivered
- `cancelled` - Delivery cancelled

### 8. Record Location Update

```bash
curl -X POST http://localhost:8084/api/tracking/locations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "delivery_id": 1,
    "courier_id": 1,
    "location": "POINT(-122.4150 37.7800)",
    "speed": 45.5,
    "heading": 90
  }' | jq '.'
```

### 9. Get Delivery Track

```bash
curl -X GET http://localhost:8084/api/tracking/deliveries/1/track \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

Expected response:
```json
[
  {
    "id": "507f1f77bcf86cd799439011",
    "delivery_id": 1,
    "courier_id": 1,
    "location": "POINT(-122.4150 37.7800)",
    "speed": 45.5,
    "heading": 90,
    "timestamp": "2026-02-03T10:15:00Z"
  }
]
```

### 10. Get Current Location

```bash
curl -X GET http://localhost:8084/api/tracking/deliveries/1/location \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

### 11. Get Courier Location

```bash
curl -X GET http://localhost:8084/api/tracking/couriers/1/location \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

### 12. Calculate ETA

```bash
curl -X POST http://localhost:8084/api/tracking/deliveries/1/eta \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "current_location": "POINT(-122.4150 37.7800)"
  }' | jq '.'
```

### 13. Get Notifications

```bash
curl -X GET http://localhost:8084/api/notification/notifications \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

Expected response:
```json
[
  {
    "id": 1,
    "user_id": 1,
    "type": "delivery_created",
    "message": "Your delivery #1 has been created",
    "read": false,
    "created_at": "2026-02-03T10:05:00Z"
  }
]
```

### 14. Mark Notification as Read

```bash
curl -X PUT http://localhost:8084/api/notification/notifications/1/read \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

### 15. WebSocket Real-Time Tracking

```javascript
// In a browser console or Node.js
const ws = new WebSocket('ws://localhost:8081/ws/deliveries/1?token=' + TOKEN);

ws.onmessage = (event) => {
  console.log('Location update:', JSON.parse(event.data));
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

## Testing Scenarios

### Scenario 1: Complete Delivery Flow

1. **Register a customer**
2. **Register a courier**
3. **Login as customer**
4. **Create a delivery**
5. **Login as admin/courier**
6. **Assign delivery to courier** (update status to 'assigned')
7. **Update status to 'picked_up'**
8. **Record multiple location updates**
9. **Update status to 'in_transit'**
10. **Calculate ETA**
11. **Update status to 'delivered'**
12. **Check notifications**

### Scenario 2: Multi-User Testing

1. **Create multiple customers**
2. **Create multiple couriers**
3. **Create deliveries for different customers**
4. **Test authorization** (customer can only see their deliveries)
5. **Assign deliveries to different couriers**
6. **Track multiple deliveries simultaneously**

### Scenario 3: Real-Time Tracking

1. **Create a delivery**
2. **Open WebSocket connection**
3. **Record location updates**
4. **Observe real-time updates via WebSocket**
5. **Query historical track**
6. **Calculate ETA at different points**

## Viewing Logs

### Real-time logs for all services:
```bash
tail -f logs/*.log
```

### Individual service logs:
```bash
tail -f logs/gateway.log
tail -f logs/delivery.log
tail -f logs/tracking.log
tail -f logs/notification.log
```

## Troubleshooting

### Services won't start

1. Check if ports are already in use:
```bash
lsof -i :8080,8081,8082,8084
```

2. Check Docker containers:
```bash
docker-compose ps
```

3. Check database migrations:
```bash
docker-compose exec postgres psql -U postgres -d delivertrack -c "\dt"
```

### Authentication errors

1. Verify token is included in Authorization header
2. Check token hasn't expired (default: 24 hours)
3. Re-login to get a fresh token

### Database connection errors

1. Ensure PostgreSQL and MongoDB containers are running
2. Check connection strings in config files
3. Verify database credentials

### No location data

1. Ensure tracking service is running
2. Verify MongoDB is accessible
3. Record at least one location update
4. Check courier_id matches the authenticated user

## Additional Resources

- **Architecture Documentation**: [docs/ARCHITECTURE_DIAGRAMS.md](../docs/ARCHITECTURE_DIAGRAMS.md)
- **JWT Authentication**: [docs/JWT_AUTH_IMPLEMENTATION.md](../docs/JWT_AUTH_IMPLEMENTATION.md)
- **MongoDB Setup**: [docs/MONGODB_SETUP_SUMMARY.md](../docs/MONGODB_SETUP_SUMMARY.md)
- **API Gateway Flow**: [docs/AUTH_FLOW_DIAGRAM.md](../docs/AUTH_FLOW_DIAGRAM.md)

## Support

For issues or questions:
1. Check service logs in `logs/` directory
2. Verify all prerequisites are installed
3. Ensure Docker containers are running
4. Review error messages and status codes
