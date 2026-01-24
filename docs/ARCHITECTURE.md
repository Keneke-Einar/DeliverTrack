# DeliverTrack Architecture

## System Overview

DeliverTrack is a microservices-based real-time package tracking system built with a modern tech stack designed for scalability, reliability, and real-time performance.

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Standard library with Gorilla Mux (routing) and Gorilla WebSocket

### Databases
- **PostgreSQL**: Primary data store for packages, users, and deliveries
- **MongoDB**: Geospatial data and location history
- **Redis**: Caching layer for high-performance data access

### Messaging & Real-Time
- **RabbitMQ**: Message broker for event-driven architecture
- **WebSockets**: Real-time bidirectional communication with clients

### DevOps
- **Docker**: Containerization
- **Docker Compose**: Multi-container orchestration

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Layer                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  Web Client │  │ Mobile App  │  │  IoT Device │            │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
└─────────┼─────────────────┼─────────────────┼───────────────────┘
          │                 │                 │
          │    HTTP/REST    │                 │
          └────────┬────────┴─────────────────┘
                   │                           
          ┌────────▼────────┐                  
          │   API Gateway   │◄────── WebSocket
          │  (Go Server)    │                  
          └────────┬────────┘                  
                   │                           
       ┌───────────┼───────────┐              
       │           │           │              
┌──────▼──────┐ ┌──▼──────┐ ┌─▼────────┐     
│  Handler    │ │WebSocket│ │Middleware│     
│   Layer     │ │   Hub   │ │  Layer   │     
└──────┬──────┘ └──┬──────┘ └──────────┘     
       │           │                          
       │           │                          
┌──────▼───────────▼─────────────────────────┐
│         Business Logic Layer                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │ Package  │  │ Delivery │  │ Location │ │
│  │ Service  │  │ Service  │  │ Service  │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘ │
└───────┼─────────────┼─────────────┼────────┘
        │             │             │         
        │             │             │         
┌───────┼─────────────┼─────────────┼────────┐
│       │    Data Access Layer      │        │
│  ┌────▼─────┐  ┌────▼──────┐  ┌──▼──────┐ │
│  │PostgreSQL│  │  MongoDB  │  │  Redis  │ │
│  │   (SQL)  │  │(Geospatial)│ │ (Cache) │ │
│  └──────────┘  └───────────┘  └─────────┘ │
└────────────────────────────────────────────┘
        │                           
        │                           
┌───────▼────────────────────────────────────┐
│      Message Queue Layer                   │
│  ┌──────────────────────────────────────┐ │
│  │          RabbitMQ Broker              │ │
│  │  ┌──────────┐      ┌──────────┐      │ │
│  │  │ Exchange │─────►│  Queue   │      │ │
│  │  └──────────┘      └────┬─────┘      │ │
│  └───────────────────────────┼───────────┘ │
└──────────────────────────────┼─────────────┘
                               │              
                    ┌──────────▼──────────┐   
                    │   Event Consumers   │   
                    │  (Async Workers)    │   
                    └─────────────────────┘   
```

## Component Descriptions

### API Gateway (Go Server)
- **Responsibilities**:
  - Route HTTP requests to appropriate handlers
  - Manage WebSocket connections
  - Apply middleware (CORS, logging, authentication)
  - Serve as the single entry point for all client requests

- **Key Features**:
  - RESTful API endpoints
  - WebSocket support for real-time updates
  - Request/response logging
  - CORS handling

### Handler Layer
- **Responsibilities**:
  - Process incoming HTTP requests
  - Validate input data
  - Coordinate business logic
  - Format responses

- **Handlers**:
  - Package handlers (CRUD operations)
  - Status update handlers
  - Location tracking handlers
  - Health check and statistics

### WebSocket Hub
- **Responsibilities**:
  - Manage WebSocket client connections
  - Handle client subscriptions
  - Broadcast updates to subscribed clients
  - Connection lifecycle management

- **Features**:
  - Client registration/unregistration
  - Selective broadcasting based on subscriptions
  - Concurrent connection handling

### Data Access Layer

#### PostgreSQL
- **Purpose**: Primary relational data store
- **Data Stored**:
  - User accounts and profiles
  - Package information
  - Delivery records
  - Transactional data

- **Schema Design**:
  - Normalized tables with proper indexing
  - Foreign key constraints
  - ACID compliance

#### MongoDB
- **Purpose**: Geospatial data and location history
- **Data Stored**:
  - Package location coordinates
  - Location timestamps
  - Address information

- **Features**:
  - Geospatial indexes for efficient queries
  - Document-based storage
  - Time-series data handling

#### Redis
- **Purpose**: High-speed caching layer
- **Cached Data**:
  - Package details (5-minute TTL)
  - Session data
  - Frequently accessed information

- **Benefits**:
  - Reduced database load
  - Faster response times
  - Improved scalability

### Message Queue Layer (RabbitMQ)

- **Purpose**: Asynchronous event processing
- **Use Cases**:
  - Package status change notifications
  - Location update broadcasts
  - Decoupling services

- **Architecture**:
  - Topic exchange for flexible routing
  - Durable queues for reliability
  - Message acknowledgment

## Data Flow Patterns

### 1. Package Creation Flow
```
Client Request
    ↓
API Gateway
    ↓
Handler validates input
    ↓
Insert into PostgreSQL
    ↓
Return response to client
```

### 2. Status Update Flow
```
Client Request (Status Update)
    ↓
API Gateway
    ↓
Handler validates input
    ↓
Update PostgreSQL
    ↓
Store location in MongoDB (if coordinates provided)
    ↓
Invalidate Redis cache
    ↓
Publish event to RabbitMQ
    ↓
Event Consumer receives message
    ↓
WebSocket Hub broadcasts to subscribed clients
    ↓
Clients receive real-time update
```

### 3. Package Query Flow
```
Client Request (Get Package)
    ↓
API Gateway
    ↓
Check Redis cache
    ├─ Cache HIT → Return cached data
    └─ Cache MISS
        ↓
    Query PostgreSQL
        ↓
    Store in Redis cache
        ↓
    Return response to client
```

### 4. Location History Query Flow
```
Client Request (Get Locations)
    ↓
API Gateway
    ↓
Get Package ID from PostgreSQL
    ↓
Query MongoDB with Package ID
    ↓
Return location array to client
```

## Scalability Considerations

### Horizontal Scaling
- **Application Layer**: Multiple Go server instances behind a load balancer
- **Database Layer**: Read replicas for PostgreSQL, MongoDB sharding
- **Cache Layer**: Redis cluster for distributed caching
- **Message Queue**: RabbitMQ clustering

### Vertical Scaling
- Increase server resources (CPU, RAM)
- Optimize database queries and indexes
- Connection pooling

### Performance Optimization
- Database query optimization
- Redis caching strategy
- Connection pooling (PostgreSQL, MongoDB, Redis)
- Asynchronous processing (RabbitMQ)
- WebSocket connection management

## Security Architecture

### Authentication & Authorization
- JWT-based authentication (configurable)
- Role-based access control (RBAC)
- API key authentication for IoT devices

### Data Security
- Encrypted connections (SSL/TLS)
- Password hashing (bcrypt)
- Environment-based configuration
- Secrets management

### Network Security
- CORS configuration
- Rate limiting
- Input validation and sanitization
- SQL injection prevention (parameterized queries)

## Monitoring & Observability

### Health Checks
- `/api/v1/health` endpoint
- Database connectivity checks
- Service dependency status

### Metrics
- System statistics endpoint
- Package counts by status
- Real-time connection counts
- Queue depths (RabbitMQ)

### Logging
- Request/response logging
- Error logging
- Performance metrics
- Audit trails

## Deployment Architecture

### Docker Compose (Development/Testing)
```
┌─────────────────────────────────────┐
│        Docker Host                  │
│  ┌────────────┐  ┌────────────┐    │
│  │    App     │  │ PostgreSQL │    │
│  └────────────┘  └────────────┘    │
│  ┌────────────┐  ┌────────────┐    │
│  │  MongoDB   │  │   Redis    │    │
│  └────────────┘  └────────────┘    │
│  ┌────────────┐                     │
│  │  RabbitMQ  │                     │
│  └────────────┘                     │
└─────────────────────────────────────┘
```

### Production (Kubernetes - Future)
```
┌─────────────────────────────────────────┐
│         Kubernetes Cluster              │
│  ┌────────────────────────────────┐    │
│  │    Load Balancer/Ingress       │    │
│  └────────────┬───────────────────┘    │
│               │                         │
│  ┌────────────▼───────────────────┐    │
│  │  DeliverTrack Pods (Replicas)  │    │
│  └────────────┬───────────────────┘    │
│               │                         │
│  ┌────────────▼───────────────────┐    │
│  │   Database Services (StatefulSets)│ │
│  │  PostgreSQL │ MongoDB │ Redis   │   │
│  └────────────┬───────────────────┘    │
│               │                         │
│  ┌────────────▼───────────────────┐    │
│  │    Message Queue (RabbitMQ)    │    │
│  └────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

## Design Patterns Used

### 1. Repository Pattern
- Abstraction layer for data access
- Separation of business logic from data access

### 2. Pub/Sub Pattern
- RabbitMQ for event-driven architecture
- Decoupled components

### 3. Cache-Aside Pattern
- Redis for read-through caching
- Improved read performance

### 4. Observer Pattern
- WebSocket hub for real-time notifications
- Event-driven updates

## Future Enhancements

### Potential Improvements
1. **Microservices**: Break into separate services
2. **API Gateway**: Dedicated gateway (e.g., Kong, Traefik)
3. **Service Mesh**: Istio for service-to-service communication
4. **Monitoring**: Prometheus + Grafana
5. **Logging**: ELK Stack or Loki
6. **Tracing**: Jaeger or Zipkin
7. **CI/CD**: GitHub Actions, Jenkins
8. **Mobile Apps**: Native iOS/Android applications
9. **Machine Learning**: Delivery time prediction
10. **Analytics**: Business intelligence dashboard
