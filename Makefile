# Makefile for Ordering Platform

.PHONY: all build run test clean docker-up docker-down migrate-up migrate-down migrate-create help

# Build the application
all: build test

build:
	@echo "Building API server..."
	@go build -o bin/api ./cmd/api

# Run the application
run:
	@echo "Running API server..."
	@go run cmd/api/main.go

# Run with live reload
watch:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
		air; \
	fi

# Start Docker containers
docker-up:
	@echo "Starting Docker containers..."
	@docker compose up -d

# Stop Docker containers
docker-down:
	@echo "Stopping Docker containers..."
	@docker compose down

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v -cover

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

# Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	goose -dir migrations create $$name sql

# Run migrations
migrate-up:
	@echo "Running migrations..."
	@goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=ordering_platform sslmode=disable" up

# Rollback migrations
migrate-down:
	@echo "Rolling back migrations..."
	@goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=ordering_platform sslmode=disable" down

# Migration status
migrate-status:
	@goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=ordering_platform sslmode=disable" status

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  watch          - Run with live reload"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  migrate-create - Create a new migration"
	@echo "  migrate-up     - Run migrations"
	@echo "  migrate-down   - Rollback migrations"
	@echo "  migrate-status - Show migration status"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
