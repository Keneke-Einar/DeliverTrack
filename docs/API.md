# DeliverTrack API Reference

## Base URL
```
http://localhost:8080/api/v1
```

## Endpoints

### 1. Health Check

Check the health status of the system and all dependencies.

**Endpoint:** `GET /api/v1/health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-01-24T10:40:00Z",
  "database": "healthy",
  "redis": "healthy",
  "mongodb": "healthy"
}
```

**Status Codes:**
- `200 OK`: All systems operational
- `503 Service Unavailable`: One or more services degraded

---

### 2. System Statistics

Get system-wide statistics about packages.

**Endpoint:** `GET /api/v1/stats`

**Response:**
```json
{
  "total_packages": 1250,
  "packages_today": 45,
  "status_counts": {
    "pending": 10,
    "in_transit": 150,
    "out_for_delivery": 25,
    "delivered": 1050,
    "failed": 10,
    "cancelled": 5
  }
}
```

---

### 3. List Packages

Get a list of all packages (limited to 100 most recent).

**Endpoint:** `GET /api/v1/packages`

**Response:**
```json
[
  {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "tracking_number": "TRK123456789",
    "sender_name": "John Doe",
    "recipient_name": "Jane Smith",
    "status": "in_transit",
    "current_location": "Newark Distribution Center",
    "created_at": "2026-01-24T10:00:00Z",
    "updated_at": "2026-01-24T12:30:00Z"
  }
]
```

---

### 4. Create Package

Create a new package for tracking.

**Endpoint:** `POST /api/v1/packages`

**Request Body:**
```json
{
  "tracking_number": "TRK123456789",
  "sender_name": "John Doe",
  "sender_address": "123 Main St, New York, NY 10001",
  "recipient_name": "Jane Smith",
  "recipient_address": "456 Oak Ave, Los Angeles, CA 90001",
  "weight": 2.5,
  "description": "Electronics package"
}
```

**Response:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "tracking_number": "TRK123456789",
  "sender_name": "John Doe",
  "sender_address": "123 Main St, New York, NY 10001",
  "recipient_name": "Jane Smith",
  "recipient_address": "456 Oak Ave, Los Angeles, CA 90001",
  "status": "pending",
  "weight": 2.5,
  "description": "Electronics package",
  "created_at": "2026-01-24T10:00:00Z",
  "updated_at": "2026-01-24T10:00:00Z"
}
```

**Status Codes:**
- `201 Created`: Package created successfully
- `400 Bad Request`: Invalid request body
- `500 Internal Server Error`: Server error

---

### 5. Get Package Details

Get detailed information about a specific package.

**Endpoint:** `GET /api/v1/packages/{tracking_number}`

**Response:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "tracking_number": "TRK123456789",
  "sender_name": "John Doe",
  "sender_address": "123 Main St, New York, NY 10001",
  "recipient_name": "Jane Smith",
  "recipient_address": "456 Oak Ave, Los Angeles, CA 90001",
  "status": "in_transit",
  "current_location": "Newark Distribution Center",
  "weight": 2.5,
  "description": "Electronics package",
  "created_at": "2026-01-24T10:00:00Z",
  "updated_at": "2026-01-24T12:30:00Z"
}
```

**Headers:**
- `X-Cache: HIT` - Response served from cache
- `X-Cache: MISS` - Response served from database

**Status Codes:**
- `200 OK`: Package found
- `404 Not Found`: Package not found

---

### 6. Update Package Status

Update the status and location of a package.

**Endpoint:** `PUT /api/v1/packages/{tracking_number}/status`

**Request Body:**
```json
{
  "status": "in_transit",
  "location": "Newark Distribution Center",
  "latitude": 40.7357,
  "longitude": -74.1724
}
```

**Response:**
```json
{
  "message": "Package status updated successfully",
  "status": "in_transit"
}
```

**Status Codes:**
- `200 OK`: Status updated successfully
- `400 Bad Request`: Invalid request body
- `404 Not Found`: Package not found
- `500 Internal Server Error`: Server error

**Side Effects:**
- Invalidates Redis cache for the package
- Publishes update event to RabbitMQ
- Broadcasts update to WebSocket clients
- Stores location in MongoDB (if coordinates provided)

---

### 7. Get Package Location History

Get the complete location history for a package.

**Endpoint:** `GET /api/v1/packages/{tracking_number}/locations`

**Response:**
```json
[
  {
    "package_id": "123e4567-e89b-12d3-a456-426614174000",
    "latitude": 40.7357,
    "longitude": -74.1724,
    "address": "Newark Distribution Center",
    "timestamp": "2026-01-24T12:30:00Z"
  },
  {
    "package_id": "123e4567-e89b-12d3-a456-426614174000",
    "latitude": 40.7128,
    "longitude": -74.0060,
    "address": "New York Origin Facility",
    "timestamp": "2026-01-24T10:30:00Z"
  }
]
```

**Status Codes:**
- `200 OK`: Locations retrieved successfully
- `404 Not Found`: Package not found
- `500 Internal Server Error`: Server error

---

## WebSocket API

### Connection

Connect to the WebSocket endpoint to receive real-time package updates.

**Endpoint:** `ws://localhost:8080/ws`

### Subscribe to Package

Send a subscription message to receive updates for a specific package.

**Message Format:**
```json
{
  "type": "subscribe",
  "tracking_number": "TRK123456789"
}
```

### Receive Updates

Updates will be pushed to the client in real-time.

**Update Message Format:**
```json
{
  "type": "package_update",
  "tracking_number": "TRK123456789",
  "data": {
    "status": "in_transit",
    "location": "Newark Distribution Center",
    "latitude": 40.7357,
    "longitude": -74.1724,
    "timestamp": "2026-01-24T12:30:00Z"
  }
}
```

---

## Package Status Values

- `pending`: Package created, awaiting pickup
- `in_transit`: Package is being transported
- `out_for_delivery`: Package is out for final delivery
- `delivered`: Package has been delivered
- `failed`: Delivery attempt failed
- `cancelled`: Package shipment was cancelled

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message description"
}
```

**Common Status Codes:**
- `400 Bad Request`: Invalid input
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable
