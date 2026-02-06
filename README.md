# 📦 DeliverTrack

**Real-Time Package Tracking System**

A microservices-based delivery tracking platform built with Golang, featuring real-time location updates, event-driven architecture, comprehensive analytics, and web-based user interfaces for customers, couriers, and administrators.

## 🚀 Tech Stack

| Category | Technology |
|----------|------------|
| **Language** | Golang |
| **Architecture** | Hexagonal (Ports & Adapters) |
| **Primary Database** | PostgreSQL |
| **Geospatial Database** | MongoDB |
| **Cache** | Redis |
| **Message Queue** | RabbitMQ |
| **Real-Time** | WebSockets |
| **Web Framework** | React + TypeScript (Advanced Phase) |
| **Containerization** | Docker |
| **Monitoring** | Prometheus + Grafana |
| **Analytics** | GraphQL |

## 🏗️ Architecture

### Service Architecture

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

### Layered Architecture (Hexagonal)

Each service follows a layered architecture with clear separation of concerns:

```
Service Structure:
├── domain/      # Business entities & rules (pure logic)
├── app/         # Use cases & orchestration
├── ports/       # Interfaces (contracts)
└── adapters/    # Infrastructure (HTTP, DB, JWT, etc.)
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

## 🌐 Web Interfaces

### Customer/Courier Portal
- **Real-time delivery tracking** with interactive maps
- **Order management** for customers (create, view, track deliveries)
- **Location updates** for couriers with GPS integration
- **WebSocket-powered live updates** for delivery status changes
- **Responsive design** optimized for desktop and tablet use

### Admin Dashboard
- **Real-time system monitoring** with live metrics
- **Delivery management** with bulk operations
- **Courier performance analytics** and route optimization
- **System health monitoring** with alerting capabilities
- **Interactive maps** showing all active deliveries and couriers

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

# Start all services with a single command
./run.sh start

# Or manually with Docker Compose
docker-compose up -d
```

**Services will be available at:**
- **Frontend (Customer Portal)**: http://localhost:3000
- **API Gateway**: http://localhost:8084
- **Delivery Service**: http://localhost:8080
- **Tracking Service**: http://localhost:8081
- **Notification Service**: http://localhost:8082
- **Analytics Service**: http://localhost:8083

**Infrastructure:**
- **PostgreSQL**: localhost:5433
- **MongoDB**: localhost:27017
- **Redis**: localhost:6379
- **RabbitMQ**: localhost:5672 (Management UI: http://localhost:15672)
- **Vault**: http://localhost:8200

**Service Management:**
```bash
# Start services
./run.sh start

# Stop services
./run.sh stop

# Restart services
./run.sh restart

# Check status
./run.sh status

# View logs
./run.sh logs [service-name]
```

```bash
# Stop all services
./stop-all.sh

# Or manually
docker-compose down
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
├── web/                   # Web applications (Advanced Phase)
│   ├── customer-portal/   # React app for customers/couriers
│   └── admin-dashboard/   # React app for administrators
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
- [ ] Customer/Courier web portal with real-time tracking
- [ ] Admin dashboard with real-time system monitoring
