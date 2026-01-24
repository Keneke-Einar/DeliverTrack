# DeliverTrack Quick Start Guide

Get DeliverTrack up and running in 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- Internet connection (for pulling Docker images)

## Quick Start Steps

### 1. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/Keneke-Einar/DeliverTrack.git
cd DeliverTrack

# Create environment file (using defaults for development)
cp .env.example .env
```

### 2. Start Everything

```bash
# Start all services (this will download Docker images on first run)
docker-compose up -d

# Wait about 30 seconds for all services to initialize
sleep 30

# Check that all services are running
docker-compose ps
```

You should see all services with status "Up":
- delivertrack-app
- delivertrack-postgres
- delivertrack-mongodb
- delivertrack-redis
- delivertrack-rabbitmq

### 3. Verify Installation

```bash
# Check API health
curl http://localhost:8080/api/v1/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2026-01-24T10:50:00Z",
  "database": "healthy",
  "redis": "healthy",
  "mongodb": "healthy"
}
```

### 4. Test the System

#### Create a Test Package

```bash
curl -X POST http://localhost:8080/api/v1/packages \
  -H "Content-Type: application/json" \
  -d '{
    "tracking_number": "TRK123456789",
    "sender_name": "John Doe",
    "sender_address": "123 Main St, New York, NY",
    "recipient_name": "Jane Smith",
    "recipient_address": "456 Oak Ave, Los Angeles, CA",
    "weight": 2.5,
    "description": "Test Package"
  }'
```

#### Track the Package

```bash
curl http://localhost:8080/api/v1/packages/TRK123456789
```

#### Update Package Status

```bash
curl -X PUT http://localhost:8080/api/v1/packages/TRK123456789/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_transit",
    "location": "Newark Distribution Center",
    "latitude": 40.7357,
    "longitude": -74.1724
  }'
```

### 5. Test Real-Time Updates

Open the WebSocket demo in your browser:

```bash
# On macOS
open examples/websocket_demo.html

# On Linux
xdg-open examples/websocket_demo.html

# On Windows
start examples/websocket_demo.html
```

Or simply open the file in your browser manually.

1. Click "Connect" to establish WebSocket connection
2. Enter tracking number: `TRK123456789`
3. Click "Subscribe"
4. In another terminal, update the package status (see command above)
5. Watch the real-time update appear in the browser!

### 6. Run Automated Test Script

```bash
# Make sure you have jq installed (for JSON formatting)
# On macOS: brew install jq
# On Ubuntu/Debian: sudo apt-get install jq

# Run the test script
./scripts/test_api.sh
```

This will create a test package and run it through the complete lifecycle.

## Accessing Services

- **DeliverTrack API**: http://localhost:8080
- **API Health Check**: http://localhost:8080/api/v1/health
- **RabbitMQ Management UI**: http://localhost:15672
  - Username: `delivertrack`
  - Password: `delivertrack_password`

## Common Commands

```bash
# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f app

# Stop all services
docker-compose down

# Restart a service
docker-compose restart app

# Rebuild and restart
docker-compose up -d --build
```

## What's Next?

- Read the [API Documentation](docs/API.md) for all available endpoints
- Check out the [Architecture](docs/ARCHITECTURE.md) to understand the system design
- Review [Deployment Guide](docs/DEPLOYMENT.md) for production setup
- Explore the source code to customize for your needs

## Troubleshooting

### Services won't start

```bash
# Check if ports are already in use
lsof -i :8080
lsof -i :5432
lsof -i :27017
lsof -i :6379
lsof -i :5672

# If ports are in use, either stop those services or change ports in .env
```

### "Connection refused" errors

```bash
# Make sure all services are running
docker-compose ps

# Check logs for errors
docker-compose logs

# Restart services
docker-compose down
docker-compose up -d
```

### Need help?

- Check the logs: `docker-compose logs -f`
- Review the [Deployment Guide](docs/DEPLOYMENT.md)
- Open an issue on GitHub

## Sample Data

Want to test with more data? Here's a script to create 10 test packages:

```bash
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/packages \
    -H "Content-Type: application/json" \
    -d "{
      \"tracking_number\": \"TRK$(date +%s)${i}\",
      \"sender_name\": \"Sender ${i}\",
      \"sender_address\": \"${i}00 Sender St, City\",
      \"recipient_name\": \"Recipient ${i}\",
      \"recipient_address\": \"${i}00 Recipient Ave, City\",
      \"weight\": $((i + 1)).5,
      \"description\": \"Test Package ${i}\"
    }"
  sleep 1
done

# List all packages
curl http://localhost:8080/api/v1/packages | jq '.'
```

## Clean Up

To completely remove DeliverTrack and all data:

```bash
# Stop and remove containers, networks, volumes
docker-compose down -v

# Remove the cloned directory (if desired)
cd ..
rm -rf DeliverTrack
```

---

**Congratulations!** You now have DeliverTrack running locally. 🚀

Happy tracking! 📦
