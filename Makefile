# Feature Flag & Experimentation Platform Makefile

.PHONY: help dev test lint seed up down clean install deps build run-control-plane run-edge run-ingestor run-analytics

# Default target
help: ## Show this help message
	@echo "Feature Flag & Experimentation Platform"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development
install: ## Install dependencies
	go mod download
	go mod tidy

deps: install ## Alias for install

dev: ## Start development environment
	docker-compose -f deploy/docker-compose.yml up -d
	@echo "Development environment started"
	@echo "Services available at:"
	@echo "  - Control Plane API: http://localhost:8080"
	@echo "  - Edge Evaluator: http://localhost:8081"
	@echo "  - Event Ingestor: http://localhost:8082"
	@echo "  - Analytics Engine: http://localhost:8083"
	@echo "  - Admin UI: http://localhost:3000"

up: dev ## Alias for dev

down: ## Stop development environment
	docker-compose -f deploy/docker-compose.yml down

clean: down ## Clean up containers and volumes
	docker-compose -f deploy/docker-compose.yml down -v
	docker system prune -f

# Building
build: ## Build all services
	@echo "Building all services..."
	go build -o bin/control-plane ./cmd/control-plane
	go build -o bin/edge-evaluator ./cmd/edge-evaluator
	go build -o bin/event-ingestor ./cmd/event-ingestor
	go build -o bin/analytics-engine ./cmd/analytics-engine
	@echo "Build complete"

build-docker: ## Build Docker images for all services
	docker build -t ff-control-plane -f cmd/control-plane/Dockerfile .
	docker build -t ff-edge-evaluator -f cmd/edge-evaluator/Dockerfile .
	docker build -t ff-event-ingestor -f cmd/event-ingestor/Dockerfile .
	docker build -t ff-analytics-engine -f cmd/analytics-engine/Dockerfile .

# Running services locally
run-control-plane: ## Run control plane service locally
	go run ./cmd/control-plane

run-edge: ## Run edge evaluator service locally
	go run ./cmd/edge-evaluator

run-ingestor: ## Run event ingestor service locally
	go run ./cmd/event-ingestor

run-analytics: ## Run analytics engine service locally
	go run ./cmd/analytics-engine

# Testing
test: ## Run all tests
	@echo "Running tests..."
	go test -v -race -cover ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -race -cover -tags=integration ./...

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race -cover -short ./...

test-coverage: ## Generate test coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Linting
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	goimports -w .

# Database operations
migrate-up: ## Run database migrations (up)
	@echo "Running Postgres migrations..."
	migrate -path db/migrations/postgres -database "postgres://postgres:password@localhost:5432/feature_flags?sslmode=disable" up
	@echo "Running ClickHouse migrations..."
	migrate -path db/migrations/clickhouse -database "clickhouse://localhost:9000/default" up

migrate-down: ## Run database migrations (down)
	@echo "Running Postgres migrations down..."
	migrate -path db/migrations/postgres -database "postgres://postgres:password@localhost:5432/feature_flags?sslmode=disable" down
	@echo "Running ClickHouse migrations down..."
	migrate -path db/migrations/clickhouse -database "clickhouse://localhost:9000/default" down

migrate-create: ## Create new migration (usage: make migrate-create name=migration_name)
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
	migrate create -ext sql -dir db/migrations/postgres $(name)

# Data seeding
seed: ## Seed database with demo data
	@echo "Seeding database with demo data..."
	go run ./scripts/seed

# API documentation
docs-generate: ## Generate API documentation
	@which swagger > /dev/null || go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/control-plane/main.go -o api/docs

proto-generate: ## Generate protobuf code
	@which buf > /dev/null || go install github.com/bufbuild/buf/cmd/buf@latest
	buf generate

# Load testing
load-test: ## Run load tests
	@which k6 > /dev/null || (echo "k6 not found. Install from https://k6.io/docs/getting-started/installation/" && exit 1)
	k6 run scripts/load-test.js

# Security
security-scan: ## Run security vulnerability scan
	@which gosec > /dev/null || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	gosec ./...

# Release
release: lint test build ## Prepare for release (lint + test + build)
	@echo "Release preparation complete"

# Development utilities
logs: ## Show logs from all services
	docker-compose -f deploy/docker-compose.yml logs -f

ps: ## Show running containers
	docker-compose -f deploy/docker-compose.yml ps

restart: ## Restart all services
	docker-compose -f deploy/docker-compose.yml restart

# CI/CD
ci: lint test build ## Run CI pipeline locally
	@echo "CI pipeline completed successfully"
