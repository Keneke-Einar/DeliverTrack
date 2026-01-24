# DeliverTrack Deployment Guide

This guide provides detailed instructions for deploying DeliverTrack in various environments.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [Production Deployment](#production-deployment)
5. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- **Docker**: 20.10 or higher
- **Docker Compose**: 1.29 or higher
- **Go**: 1.21 or higher (for local development)
- **Git**: For cloning the repository

### System Requirements

**Minimum**:
- CPU: 2 cores
- RAM: 4 GB
- Disk: 10 GB

**Recommended**:
- CPU: 4 cores
- RAM: 8 GB
- Disk: 20 GB

---

## Local Development

### Step 1: Clone the Repository

```bash
git clone https://github.com/Keneke-Einar/DeliverTrack.git
cd DeliverTrack
```

### Step 2: Set Up Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit the .env file with your preferred editor
nano .env  # or vim, code, etc.
```

### Step 3: Start Infrastructure Services

Start only the required services (PostgreSQL, MongoDB, Redis, RabbitMQ):

```bash
docker-compose up -d postgres mongodb redis rabbitmq
```

Wait for services to be ready (about 30 seconds):

```bash
# Check service status
docker-compose ps
```

### Step 4: Install Dependencies

```bash
go mod download
```

### Step 5: Run the Application

```bash
go run cmd/server/main.go
```

The application will start on `http://localhost:8080`.

### Step 6: Verify Installation

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Should return:
# {
#   "status": "healthy",
#   "database": "healthy",
#   "redis": "healthy",
#   "mongodb": "healthy"
# }
```

### Step 7: Run Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

---

## Docker Deployment

### Full Stack Deployment

Deploy the entire application stack using Docker Compose.

### Step 1: Configure Environment

```bash
cp .env.example .env
# Edit .env with production values
```

**Important**: Change these values in production:
- `JWT_SECRET`: Use a strong, random secret
- Database passwords
- Redis password (if needed)
- RabbitMQ credentials

### Step 2: Build and Start Services

```bash
# Build and start all services
docker-compose up -d --build

# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f app
```

### Step 3: Verify Deployment

```bash
# Check all containers are running
docker-compose ps

# Test the API
curl http://localhost:8080/api/v1/health
```

### Step 4: Access Services

- **Application API**: http://localhost:8080
- **RabbitMQ Management**: http://localhost:15672
  - Username: `delivertrack`
  - Password: `delivertrack_password`

### Useful Docker Commands

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: Deletes all data)
docker-compose down -v

# Restart a specific service
docker-compose restart app

# View logs
docker-compose logs -f app

# Execute commands in container
docker-compose exec app sh

# Rebuild without cache
docker-compose build --no-cache
```

---

## Production Deployment

### Security Hardening

#### 1. Environment Variables

Create a secure `.env` file:

```bash
# Generate strong random secrets
JWT_SECRET=$(openssl rand -base64 32)
POSTGRES_PASSWORD=$(openssl rand -base64 32)
MONGO_PASSWORD=$(openssl rand -base64 32)
RABBITMQ_PASSWORD=$(openssl rand -base64 32)
```

#### 2. Update docker-compose.yml

For production, modify `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: always
    # Remove port mapping for security
    # ports:
    #   - "5432:5432"

  # Similar updates for other services...

  app:
    build: .
    environment:
      - DATABASE_URL=postgres://delivertrack:${POSTGRES_PASSWORD}@postgres:5432/delivertrack?sslmode=require
      - MONGODB_URL=mongodb://delivertrack:${MONGO_PASSWORD}@mongodb:27017
      - REDIS_URL=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - RABBITMQ_URL=amqp://delivertrack:${RABBITMQ_PASSWORD}@rabbitmq:5672/
      - JWT_SECRET=${JWT_SECRET}
      - ENVIRONMENT=production
    restart: always
```

### Reverse Proxy Setup (Nginx)

Create `nginx.conf`:

```nginx
upstream delivertrack {
    server app:8080;
}

server {
    listen 80;
    server_name your-domain.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass http://delivertrack;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://delivertrack;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

Add to `docker-compose.yml`:

```yaml
  nginx:
    image: nginx:alpine
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
      - ./ssl:/etc/nginx/ssl
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - app
    restart: always
```

### Database Backups

#### PostgreSQL Backup

```bash
# Create backup
docker-compose exec postgres pg_dump -U delivertrack delivertrack > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
docker-compose exec -T postgres psql -U delivertrack delivertrack < backup.sql
```

#### MongoDB Backup

```bash
# Create backup
docker-compose exec mongodb mongodump --username delivertrack --password delivertrack_password --out /backup

# Copy backup from container
docker cp delivertrack-mongodb:/backup ./mongo_backup_$(date +%Y%m%d_%H%M%S)

# Restore backup
docker-compose exec mongodb mongorestore --username delivertrack --password delivertrack_password /backup
```

### Monitoring Setup

#### Health Check Script

Create `scripts/health_check.sh`:

```bash
#!/bin/bash

URL="http://localhost:8080/api/v1/health"
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $URL)

if [ $RESPONSE -eq 200 ]; then
    echo "✓ Service is healthy"
    exit 0
else
    echo "✗ Service is unhealthy (HTTP $RESPONSE)"
    exit 1
fi
```

Add to crontab for regular checks:

```bash
*/5 * * * * /path/to/scripts/health_check.sh >> /var/log/delivertrack_health.log 2>&1
```

### Systemd Service (Alternative to Docker)

Create `/etc/systemd/system/delivertrack.service`:

```ini
[Unit]
Description=DeliverTrack Service
After=network.target postgresql.service mongodb.service redis.service rabbitmq-server.service

[Service]
Type=simple
User=delivertrack
WorkingDirectory=/opt/delivertrack
EnvironmentFile=/opt/delivertrack/.env
ExecStart=/opt/delivertrack/delivertrack
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable delivertrack
sudo systemctl start delivertrack
sudo systemctl status delivertrack
```

---

## Troubleshooting

### Common Issues

#### 1. Database Connection Failed

**Problem**: Cannot connect to PostgreSQL

**Solutions**:
```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# View PostgreSQL logs
docker-compose logs postgres

# Verify connection string in .env
DATABASE_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable

# Test connection manually
docker-compose exec postgres psql -U delivertrack -d delivertrack
```

#### 2. Redis Connection Failed

**Problem**: Cannot connect to Redis

**Solutions**:
```bash
# Check if Redis is running
docker-compose ps redis

# Test Redis connection
docker-compose exec redis redis-cli ping
# Should return: PONG

# Check Redis logs
docker-compose logs redis
```

#### 3. RabbitMQ Connection Failed

**Problem**: Cannot connect to RabbitMQ

**Solutions**:
```bash
# Check if RabbitMQ is running
docker-compose ps rabbitmq

# Access RabbitMQ management UI
# http://localhost:15672

# Check RabbitMQ logs
docker-compose logs rabbitmq

# Verify connection URL
RABBITMQ_URL=amqp://user:password@localhost:5672/
```

#### 4. Port Already in Use

**Problem**: Port 8080 is already in use

**Solutions**:
```bash
# Find process using port 8080
lsof -i :8080
# or
netstat -tulpn | grep 8080

# Kill the process or change the port in .env
PORT=8081
```

#### 5. WebSocket Connection Failed

**Problem**: Cannot establish WebSocket connection

**Solutions**:
- Check firewall settings
- Verify WebSocket endpoint: `ws://localhost:8080/ws`
- Check browser console for errors
- Verify CORS settings
- Test with WebSocket demo: `examples/websocket_demo.html`

#### 6. Application Crashes on Startup

**Problem**: Application crashes immediately after starting

**Solutions**:
```bash
# Check application logs
docker-compose logs app

# Common causes:
# - Missing environment variables
# - Database not ready
# - Port conflicts

# Try starting services in order:
docker-compose up -d postgres mongodb redis rabbitmq
sleep 30  # Wait for services to initialize
docker-compose up -d app
```

### Performance Issues

#### High Memory Usage

```bash
# Check container resource usage
docker stats

# Limit container memory in docker-compose.yml:
services:
  app:
    mem_limit: 512m
    mem_reservation: 256m
```

#### Slow Database Queries

```bash
# Enable PostgreSQL query logging
docker-compose exec postgres psql -U delivertrack -d delivertrack
ALTER SYSTEM SET log_statement = 'all';
SELECT pg_reload_conf();

# Analyze slow queries
docker-compose logs postgres | grep "duration:"
```

### Getting Help

If you encounter issues not covered here:

1. Check the logs: `docker-compose logs -f`
2. Review the [Architecture Documentation](ARCHITECTURE.md)
3. Review the [API Documentation](API.md)
4. Open an issue on GitHub

---

## Maintenance

### Regular Tasks

1. **Monitor logs**: `docker-compose logs -f`
2. **Check disk space**: `df -h`
3. **Monitor resource usage**: `docker stats`
4. **Update dependencies**: `go get -u ./...`
5. **Security updates**: `docker-compose pull`

### Backup Schedule

Recommended backup schedule:
- **Daily**: PostgreSQL database
- **Weekly**: MongoDB collections
- **Monthly**: Full system backup

### Updates

To update DeliverTrack:

```bash
# Pull latest changes
git pull origin main

# Rebuild containers
docker-compose down
docker-compose up -d --build

# Verify update
curl http://localhost:8080/api/v1/health
```
