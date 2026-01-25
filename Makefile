.PHONY: run test migrate test-integration test-coverage build build-all lint clean docker-build docker-push

# Variables
SERVICES := delivery tracking notification analytics
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY ?= ghcr.io
IMAGE_PREFIX ?= $(shell basename $(CURDIR))

# Run all services locally
run:
	docker compose up -d

# Run a specific service
run-%:
	cd cmd/$* && go run .

# Run all tests
test:
	@for service in $(SERVICES); do \
		echo "Testing $$service..."; \
		cd cmd/$$service && go test ./... && cd ../..; \
	done
	go test ./pkg/... ./internal/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@echo "mode: atomic" > coverage.out
	@for service in $(SERVICES); do \
		cd cmd/$$service && go test -coverprofile=coverage.tmp -covermode=atomic ./... && cd ../..; \
		if [ -f cmd/$$service/coverage.tmp ]; then \
			tail -n +2 cmd/$$service/coverage.tmp >> coverage.out; \
			rm cmd/$$service/coverage.tmp; \
		fi \
	done
	@go tool cover -func=coverage.out | tail -1

# Run integration tests
test-integration:
	docker compose -f docker-compose.yml -f docker-compose.test.yml up -d
	go test -tags=integration ./...
	docker compose -f docker-compose.yml -f docker-compose.test.yml down -v

# Run linter
lint:
	golangci-lint run ./...

# Run database migrations
migrate:
	@echo "Running migrations..."
	go run ./cmd/migrate/main.go up

migrate-down:
	go run ./cmd/migrate/main.go down

# Build a specific service
build-%:
	cd cmd/$* && CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=$(VERSION)" -o ../../bin/$* .

# Build all services
build-all: $(addprefix build-,$(SERVICES))

# Build Docker image for a specific service
docker-build-%:
	docker build -t $(REGISTRY)/$(IMAGE_PREFIX)/$*:$(VERSION) -f docker/Dockerfile.$* .

# Build all Docker images
docker-build: $(addprefix docker-build-,$(SERVICES))

# Push Docker image for a specific service
docker-push-%:
	docker push $(REGISTRY)/$(IMAGE_PREFIX)/$*:$(VERSION)

# Push all Docker images
docker-push: $(addprefix docker-push-,$(SERVICES))

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out
	docker compose down -v --remove-orphans

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Generate code (protobuf, mocks, etc.)
generate:
	go generate ./...

# Help
help:
	@echo "Available targets:"
	@echo "  run              - Run all services with docker compose"
	@echo "  run-<service>    - Run a specific service locally"
	@echo "  test             - Run all tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-integration - Run integration tests"
	@echo "  lint             - Run linter"
	@echo "  build-<service>  - Build a specific service"
	@echo "  build-all        - Build all services"
	@echo "  docker-build     - Build all Docker images"
	@echo "  docker-push      - Push all Docker images"
	@echo "  clean            - Clean build artifacts"
	@echo "  tools            - Install development tools"
	@echo "  mongo-shell      - Connect to MongoDB shell"
	@echo "  mongo-test       - Test MongoDB geospatial setup"
	@echo "  mongo-backup     - Backup MongoDB data"

# MongoDB operations
mongo-shell:
	mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin

mongo-test:
	./scripts/test-mongodb.sh

mongo-backup:
	@echo "Creating MongoDB backup..."
	@mkdir -p backups
	mongodump --uri="mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin" --out=backups/mongo-$(shell date +%Y%m%d-%H%M%S)
	@echo "Backup created in backups/"

mongo-restore:
	@echo "Restoring MongoDB from latest backup..."
	@LATEST=$$(ls -td backups/mongo-* | head -1); \
	mongorestore --uri="mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin" $$LATEST
	@echo "Restore complete"

