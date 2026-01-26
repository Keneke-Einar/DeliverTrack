# Architecture Diagrams

## Layered Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Entry Point                          │
│                    (cmd/delivery/main.go)                   │
│                                                             │
│  • Initialize database connection                          │
│  • Wire dependencies (dependency injection)                │
│  • Start HTTP server                                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    ADAPTERS LAYER                           │
│                  (Infrastructure)                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  HTTP Handlers          Database Repos       JWT Service   │
│  ┌───────────┐         ┌──────────────┐    ┌────────────┐ │
│  │ POST /... │         │  PostgreSQL  │    │ Generate   │ │
│  │ GET  /... │ ───────▶│  Repository  │    │ Validate   │ │
│  │ PUT  /... │         │              │    │ Token      │ │
│  └───────────┘         └──────────────┘    └────────────┘ │
│        │                      ▲                    ▲       │
└────────┼──────────────────────┼────────────────────┼───────┘
         │                      │                    │
         │                      │                    │
         │                      │                    │
┌────────▼──────────────────────┴────────────────────┴───────┐
│                    PORTS LAYER                              │
│                   (Interfaces)                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  DeliveryService         DeliveryRepository    TokenService│
│  ┌───────────────┐       ┌──────────────────┐ ┌──────────┐│
│  │ CreateDelivery│       │ Create(...)      │ │Generate()││
│  │ GetDelivery   │       │ GetByID(...)     │ │Validate()││
│  │ ListDeliveries│       │ GetByStatus(...) │ └──────────┘│
│  │ UpdateStatus  │       │ UpdateStatus(...)│             │
│  └───────────────┘       └──────────────────┘             │
│         ▲                         ▲                ▲       │
└─────────┼─────────────────────────┼────────────────┼───────┘
          │                         │                │
          │                         │                │
┌─────────▼─────────────────────────┴────────────────┴───────┐
│                    APP LAYER                                │
│               (Use Cases / Services)                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  DeliveryService                                           │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ CreateDelivery(ctx, customerID, location...)         │ │
│  │   1. Validate input                                  │ │
│  │   2. Create domain entity                            │ │
│  │   3. Apply business rules                            │ │
│  │   4. Call repository to persist                      │ │
│  │   5. Return result                                   │ │
│  └──────────────────────────────────────────────────────┘ │
│                           ▼                                 │
└─────────────────────────────────────────────────────────────┘
                            │
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                    DOMAIN LAYER                             │
│              (Business Entities & Rules)                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Delivery Entity                                           │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ Fields:                                              │ │
│  │   • ID, CustomerID, CourierID                        │ │
│  │   • Status, Locations, Dates                         │ │
│  │                                                      │ │
│  │ Methods:                                             │ │
│  │   • NewDelivery() - Create with validation          │ │
│  │   • UpdateStatus() - Validate status transition     │ │
│  │   • AssignCourier() - Business rules                │ │
│  │   • CanBeModifiedBy() - Authorization logic         │ │
│  │                                                      │ │
│  │ Validations:                                         │ │
│  │   • Status must be valid                            │ │
│  │   • Locations cannot be empty                       │ │
│  │   • Customer ID must be positive                    │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                             │
│  No external dependencies - Pure Go                        │
└─────────────────────────────────────────────────────────────┘
```

## Request Flow Example

### Creating a Delivery

```
1. HTTP Request
   POST /deliveries
   {
     "customer_id": 123,
     "pickup_location": "Address A",
     "delivery_location": "Address B"
   }
        │
        ▼
2. HTTP Adapter (adapters/http_handler.go)
   • Parse JSON
   • Extract auth context
   • Validate request format
        │
        ▼
3. Application Service (app/service.go)
   • Check authorization
   • Validate business rules
   • Create domain entity
        │
        ▼
4. Domain Entity (domain/delivery.go)
   • Validate customer_id > 0
   • Validate locations not empty
   • Set initial status to "pending"
   • Set timestamps
        │
        ▼
5. Repository Interface (ports/repository.go)
   interface DeliveryRepository {
     Create(ctx, delivery) error
   }
        │
        ▼
6. PostgreSQL Adapter (adapters/postgres_repository.go)
   • Convert domain entity to SQL types
   • Execute INSERT query
   • Return created entity with ID
        │
        ▼
7. Response
   HTTP 201 Created
   {
     "id": 456,
     "customer_id": 123,
     "status": "pending",
     ...
   }
```

## Dependency Injection Flow

```
main.go
│
├─ 1. Initialize Infrastructure
│   ├─ Database Connection
│   ├─ Config Loading
│   └─ Logger Setup
│
├─ 2. Create Adapters (Outer Layer)
│   ├─ PostgreSQL Repository ─────────┐
│   ├─ JWT Token Service ──────────┐  │
│   └─ MongoDB Client (if needed)  │  │
│                                   │  │
├─ 3. Create Application Services   │  │
│   │   (Inject via Interfaces)    │  │
│   │                               │  │
│   └─ DeliveryService ◄────────────┴──┘
│       │                           │
│       └─ AuthService ◄────────────┘
│
├─ 4. Create HTTP Handlers
│   │   (Inject Services)
│   │
│   ├─ DeliveryHandler(deliveryService)
│   └─ AuthHandler(authService)
│
└─ 5. Start HTTP Server
    └─ Route Requests to Handlers
```

## Service Dependencies

```
┌──────────────────────────────────────────────────────────┐
│                    Delivery Service                       │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  Depends on (via interfaces):                           │
│  • DeliveryRepository (for persistence)                 │
│                                                          │
│  Independent of:                                         │
│  • HTTP framework                                        │
│  • Database implementation                               │
│  • Authentication mechanism                              │
└──────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│                      Auth Service                         │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  Depends on (via interfaces):                           │
│  • UserRepository (for user data)                       │
│  • TokenService (for JWT generation/validation)         │
│                                                          │
│  Independent of:                                         │
│  • HTTP framework                                        │
│  • Database implementation                               │
│  • Token format (JWT vs OAuth)                          │
└──────────────────────────────────────────────────────────┘
```

## Testing Strategy Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    UNIT TESTS                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Domain Layer Tests (No Mocks Needed)                  │
│  ┌───────────────────────────────────────────────────┐ │
│  │ TestNewDelivery()                                 │ │
│  │ TestUpdateStatus()                                │ │
│  │ TestValidation()                                  │ │
│  │                                                   │ │
│  │ Fast, Pure, No Dependencies                       │ │
│  └───────────────────────────────────────────────────┘ │
│                                                         │
│  Application Layer Tests (Mock Repositories)           │
│  ┌───────────────────────────────────────────────────┐ │
│  │ mockRepo := &MockDeliveryRepository{}             │ │
│  │ service := app.NewDeliveryService(mockRepo)       │ │
│  │                                                   │ │
│  │ Test business logic without database              │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                 INTEGRATION TESTS                       │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Adapter Tests (Real Database)                         │
│  ┌───────────────────────────────────────────────────┐ │
│  │ repo := adapters.NewPostgresRepo(testDB)          │ │
│  │ delivery := domain.NewDelivery(...)               │ │
│  │ err := repo.Create(ctx, delivery)                 │ │
│  │                                                   │ │
│  │ Test with real PostgreSQL                         │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│                     E2E TESTS                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Full Stack Tests (HTTP → DB)                          │
│  ┌───────────────────────────────────────────────────┐ │
│  │ POST /deliveries                                  │ │
│  │ GET  /deliveries/123                              │ │
│  │ PUT  /deliveries/123/status                       │ │
│  │                                                   │ │
│  │ Test complete request flow                        │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## Package Dependencies

```
cmd/delivery/main.go
    ↓
    imports
    ↓
┌───────────────────────────────────────────────────────────┐
│  github.com/Keneke-Einar/delivertrack/internal/delivery/        │
│                                                           │
│  adapters/                                                │
│      ↓ imports                                            │
│  ports/ ← app/                                            │
│      ↓ imports                                            │
│  domain/                                                  │
└───────────────────────────────────────────────────────────┘
    ↓
    imports
    ↓
┌───────────────────────────────────────────────────────────┐
│  github.com/delivertrack/auth/                            │
│                                                           │
│  adapters/                                                │
│      ↓ imports                                            │
│  ports/ ← app/                                            │
│      ↓ imports                                            │
│  domain/                                                  │
└───────────────────────────────────────────────────────────┘
    ↓
    imports
    ↓
┌───────────────────────────────────────────────────────────┐
│  External Dependencies                                    │
│  • github.com/lib/pq (PostgreSQL)                         │
│  • github.com/golang-jwt/jwt                              │
│  • golang.org/x/crypto/bcrypt                             │
└───────────────────────────────────────────────────────────┘
```

## Key Principles Visualized

### Dependency Rule

```
        ┌─────────────────┐
        │   Adapters      │  ◄── Depends on everything
        │  (Framework)    │      Can import all layers
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │   Ports         │  ◄── Defines interfaces
        │ (Interfaces)    │      No imports (just Go interfaces)
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │   App           │  ◄── Imports ports & domain
        │ (Use Cases)     │      No framework dependencies
        └────────┬────────┘
                 │
        ┌────────▼────────┐
        │   Domain        │  ◄── No imports!
        │ (Pure Logic)    │      100% pure Go
        └─────────────────┘
```

This architecture ensures:
- Business logic is independent
- Easy to test at every layer
- Can swap implementations easily
- Clear boundaries between concerns
