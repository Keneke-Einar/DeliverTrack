# DeliverTrack - Real-Time Package Tracking System

A comprehensive real-time package tracking system built with modern technologies to provide seamless tracking, monitoring, and delivery management capabilities.

## Tech Stack

- **Backend**: Golang (Go 1.21+)
- **Databases**: 
  - PostgreSQL (Package and user data)
  - MongoDB (Geospatial location tracking)
  - Redis (Caching and session management)
- **Message Queue**: RabbitMQ (Event-driven architecture)
- **Real-Time Communication**: WebSockets
- **Containerization**: Docker & Docker Compose

## Features

### Core Functionality
- ✅ Package creation and management
- ✅ Real-time package status tracking
- ✅ Geospatial location history
- ✅ RESTful API for package operations
- ✅ WebSocket support for live updates
- ✅ Redis caching for improved performance
- ✅ Event-driven architecture with RabbitMQ
- ✅ MongoDB for efficient geospatial queries

### API Endpoints

#### Health & Monitoring
- `GET /api/v1/health` - System health check
- `GET /api/v1/stats` - System statistics

#### Package Management
- `GET /api/v1/packages` - List all packages
- `POST /api/v1/packages` - Create a new package
- `GET /api/v1/packages/{tracking_number}` - Get package details
- `PUT /api/v1/packages/{tracking_number}/status` - Update package status
- `GET /api/v1/packages/{tracking_number}/locations` - Get location history

#### WebSocket
- `WS /ws` - WebSocket connection for real-time updates

## Quick Start

### Prerequisites
- Docker and Docker Compose installed
- Go 1.21+ (for local development)

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/Keneke-Einar/DeliverTrack.git
cd DeliverTrack
```

2. Create environment file:
```bash
cp .env.example .env
```

3. Start all services:
```bash
docker-compose up -d
```

4. The API will be available at `http://localhost:8080`

### Local Development

1. Start the infrastructure services:
```bash
docker-compose up -d postgres mongodb redis rabbitmq
```

2. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your local configuration
```

3. Install dependencies:
```bash
go mod download
```

4. Run the application:
```bash
go run cmd/server/main.go
```

## Usage Examples

### Creating a Package

```bash
curl -X POST http://localhost:8080/api/v1/packages \
  -H "Content-Type: application/json" \
  -d '{
    "tracking_number": "TRK123456789",
    "sender_name": "John Doe",
    "sender_address": "123 Main St, City, State",
    "recipient_name": "Jane Smith",
    "recipient_address": "456 Oak Ave, City, State",
    "weight": 2.5,
    "description": "Electronics package"
  }'
```

### Getting Package Details

```bash
curl http://localhost:8080/api/v1/packages/TRK123456789
```

### Updating Package Status

```bash
curl -X PUT http://localhost:8080/api/v1/packages/TRK123456789/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_transit",
    "location": "Distribution Center - Newark",
    "latitude": 40.7357,
    "longitude": -74.1724
  }'
```

### Getting Location History

```bash
curl http://localhost:8080/api/v1/packages/TRK123456789/locations
```

### WebSocket Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  // Subscribe to package updates
  ws.send(JSON.stringify({
    type: 'subscribe',
    tracking_number: 'TRK123456789'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Package update:', data);
};
```

## Architecture

### System Components

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Client    │◄────►│  API Server  │◄────►│ PostgreSQL  │
│ (Web/Mobile)│      │   (Golang)   │      └─────────────┘
└─────────────┘      └──────────────┘
       │                    │                ┌─────────────┐
       │                    ├───────────────►│   MongoDB   │
       │                    │                │(Geospatial) │
       │                    │                └─────────────┘
       │                    │
       │                    │                ┌─────────────┐
       │                    ├───────────────►│    Redis    │
       │                    │                │  (Caching)  │
       │                    │                └─────────────┘
       │                    │
       │                    │                ┌─────────────┐
       │                    ├───────────────►│  RabbitMQ   │
       │                    │                │(Messaging)  │
       │                    │                └─────────────┘
       │                    │
       └────────────────────┘
            WebSocket
```

### Data Flow

1. **Package Creation**: REST API → PostgreSQL → Response
2. **Status Update**: REST API → PostgreSQL → RabbitMQ → WebSocket Broadcast
3. **Location Tracking**: REST API → MongoDB (Geospatial Index)
4. **Cache Strategy**: Redis caches package data for 5 minutes
5. **Real-time Updates**: RabbitMQ → WebSocket Hub → Connected Clients

## Package Status Flow

```
pending → in_transit → out_for_delivery → delivered
                    ↓
                 failed/cancelled
```

### Status Types
- `pending`: Package created, awaiting pickup
- `in_transit`: Package in transit to destination
- `out_for_delivery`: Package out for final delivery
- `delivered`: Package successfully delivered
- `failed`: Delivery attempt failed
- `cancelled`: Package shipment cancelled

## Development

### Project Structure

```
DeliverTrack/
├── cmd/
│   └── server/          # Main application entry point
│       └── main.go
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database connections
│   ├── handlers/        # HTTP request handlers
│   ├── messaging/       # RabbitMQ integration
│   ├── models/          # Data models
│   └── websocket/       # WebSocket hub
├── pkg/
│   └── utils/           # Utility functions
├── docker-compose.yml   # Docker services configuration
├── Dockerfile           # Application container
├── .env.example         # Environment variables template
└── README.md
```

### Building the Application

```bash
# Build binary
go build -o delivertrack ./cmd/server

# Run tests
go test ./...

# Build Docker image
docker build -t delivertrack:latest .
```

## Monitoring

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2026-01-24T10:40:00Z",
  "database": "healthy",
  "redis": "healthy",
  "mongodb": "healthy"
}
```

### Statistics
```bash
curl http://localhost:8080/api/v1/stats
```

Response:
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

## Service Ports

- **Application**: 8080
- **PostgreSQL**: 5432
- **MongoDB**: 27017
- **Redis**: 6379
- **RabbitMQ**: 5672 (AMQP), 15672 (Management UI)

## RabbitMQ Management

Access the RabbitMQ Management UI at `http://localhost:15672`
- Username: `delivertrack`
- Password: `delivertrack_password`

## Security Considerations

⚠️ **Important**: The default credentials in `.env.example` are for development only. Always use strong, unique credentials in production environments.

### Production Recommendations
- Use environment-specific secrets management
- Enable SSL/TLS for all connections
- Implement JWT authentication for API endpoints
- Use API rate limiting
- Enable CORS only for trusted domains
- Regular security audits and updates

## Troubleshooting

### Services Not Starting
```bash
# Check service logs
docker-compose logs -f

# Restart specific service
docker-compose restart app

# Rebuild and restart
docker-compose up -d --build
```

### Database Connection Issues
```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check database logs
docker-compose logs postgres
```

### WebSocket Connection Issues
- Ensure firewall allows WebSocket connections
- Check that the WebSocket endpoint is accessible
- Verify CORS settings for cross-origin connections

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write or update tests
5. Submit a pull request

## License

This project is licensed under the MIT License.

## Support

For issues, questions, or contributions, please open an issue on the GitHub repository.