# 📦 DeliverTrack

**Real-Time Package Tracking System**

A microservices-based delivery tracking platform built with Golang, featuring real-time location updates, event-driven architecture, and comprehensive analytics.

## 🚀 Tech Stack

| Category | Technology |
|----------|------------|
| **Language** | Golang |
| **Primary Database** | PostgreSQL |
| **Geospatial Database** | MongoDB |
| **Cache** | Redis |
| **Message Queue** | RabbitMQ |
| **Real-Time** | WebSockets |
| **Containerization** | Docker |
| **Monitoring** | Prometheus + Grafana |
| **Analytics** | GraphQL |

## 🏗️ Architecture

DeliverTrack consists of 4 core microservices:

```
┌───────────────────────────────────────────────────────┐
│                        API Gateway                    │
└───────────────────────────────────────────────────────┘
                             │
        ┌────────────────────┼─────────────────────┐
        ▼                    ▼                     ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│   Delivery    │   │   Tracking    │   │ Notification  │
│   Service     │   │   Service     │   │   Service     │
└───────────────┘   └───────────────┘   └───────────────┘
        │                    │                     │
        └────────────────────┼─────────────────────┘
                             ▼
                    ┌───────────────┐
                    │   Analytics   │
                    │   Service     │
                    └───────────────┘
```

## 📊 Database Schema

### PostgreSQL Tables

- **deliveries** - Core delivery records (id, customer_id, courier_id, status, timestamps)
- **couriers** - Courier information (id, name, vehicle_type, current_location)
- **customers** - Customer profiles (id, name, address, contact)

### MongoDB Collections

- **courier_locations** - Real-time GeoJSON location data with timestamps
- **delivery_zones** - Geofencing polygons for zone-based triggers

## 🔐 Authentication

JWT-based authentication with role-based access control:

| Role | Permissions |
|------|-------------|
| **Customer** | Create and view own deliveries |
| **Courier** | Update location and delivery status |
| **Admin** | Full system access |

## 📡 API Endpoints

### Delivery Service

```
POST   /deliveries              Create new delivery
GET    /deliveries/:id          Track delivery status
PUT    /deliveries/:id/status   Update delivery status
GET    /deliveries?status=      Filter deliveries by status
```

### Tracking Service

```
POST   /locations               Submit courier location update
WS     /ws/track/:delivery_id   Real-time tracking WebSocket
```

## 📨 Event-Driven Architecture

RabbitMQ events for decoupled service communication:

- `delivery.created` - New delivery order placed
- `location.updated` - Courier position changed
- `status.changed` - Delivery status transition
- `delivery.completed` - Delivery successfully finished

## ⚡ Real-Time Features

- **WebSocket Server** - Live tracking with concurrent connection handling
- **Location Broadcasting** - Real-time updates to relevant clients
- **Geofencing** - Zone entry/exit detection using MongoDB `$geoWithin`
- **ETA Calculation** - Dynamic estimates using distance matrices

## 🗄️ Caching Strategy (Redis)

| Cache Key | TTL | Description |
|-----------|-----|-------------|
| Active delivery details | Dynamic | Frequently accessed delivery info |
| Courier locations | 15 seconds | Latest courier positions |
| Customer delivery history | Long | Historical delivery records |

## 🚦 Rate Limiting

- **Courier location updates**: 1 request/second max
- **Delivery creation**: 10/hour per customer
- **API calls**: Configurable per API key

## 📈 Monitoring & Metrics

### Prometheus Metrics

- Active deliveries count
- Average delivery time
- WebSocket connection count
- Location update frequency
- API response times

### Grafana Dashboard

Operations dashboard with real-time visibility into system health and performance.

## 🔧 Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- MongoDB 6+
- Redis 7+
- RabbitMQ 3.12+

### Quick Start

```bash
# Clone the repository
git clone https://github.com/yourusername/delivertrack.git
cd delivertrack

# Start all services with Docker Compose
docker-compose up -d

# Run database migrations
make migrate

# Start the application
make run
```

### Environment Variables

```env
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=delivertrack
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secret

# MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DB=delivertrack_geo

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h
```

## 📁 Project Structure

```
delivertrack/
├── cmd/
│   ├── delivery/          # Delivery service entrypoint
│   ├── tracking/          # Tracking service entrypoint
│   ├── notification/      # Notification service entrypoint
│   └── analytics/         # Analytics service entrypoint
├── internal/
│   ├── auth/              # JWT authentication
│   ├── delivery/          # Delivery domain logic
│   ├── tracking/          # Location tracking logic
│   ├── notification/      # Notification handlers
│   ├── analytics/         # Analytics & reporting
│   └── common/            # Shared utilities
├── pkg/
│   ├── database/          # Database connections
│   ├── messaging/         # RabbitMQ client
│   ├── cache/             # Redis client
│   └── websocket/         # WebSocket handlers
├── migrations/            # Database migrations
├── docker/                # Dockerfiles
├── docker-compose.yml
├── Makefile
└── README.md
```

## 🧪 Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

## 📊 Analytics (GraphQL)

Available analytics queries:

- Delivery success rates
- Courier efficiency metrics
- Peak delivery times analysis
- Customer satisfaction scores

## 🛣️ Roadmap

- [ ] Circuit Breaker for external mapping APIs
- [ ] A/B testing for routing algorithms
- [ ] Load testing scenarios
- [ ] Admin dashboard with real-time map

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Estimated Development Time:** 1-1.5 weeks