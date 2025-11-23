# Ordering Platform

A full-stack ordering platform with Go microservices backend and React frontend.

## Project Structure

```
.
├── cmd/
│   └── api/              # API server entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── database/         # Database connection and migrations
│   ├── domain/           # Domain entities and business rules
│   ├── logger/           # Structured logging
│   ├── repository/       # Data access layer
│   ├── server/           # HTTP server setup
│   ├── service/          # Business logic layer
│   └── transport/        # HTTP handlers and middleware
├── migrations/           # Database migrations (Goose)
├── frontend/             # React frontend application
└── .env                  # Environment configuration
```

## Prerequisites

- Go 1.24+
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for frontend)

## Getting Started

### 1. Clone and Setup

```bash
# Install Go dependencies
make deps

# Copy environment file
cp .env.example .env
# Edit .env with your configuration
```

### 2. Start Infrastructure

```bash
# Start PostgreSQL and Redis
make docker-up
```

### 3. Run Migrations

```bash
# Run database migrations
make migrate-up

# Check migration status
make migrate-status
```

### 4. Run the Application

```bash
# Build and run
make build
make run

# Or run with live reload
make watch
```

The API server will be available at `http://localhost:8080`

## Development

### Available Make Commands

- `make build` - Build the application
- `make run` - Run the application
- `make watch` - Run with live reload
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make docker-up` - Start Docker containers
- `make docker-down` - Stop Docker containers
- `make migrate-create` - Create a new migration
- `make migrate-up` - Run migrations
- `make migrate-down` - Rollback migrations
- `make clean` - Clean build artifacts
- `make fmt` - Format code
- `make lint` - Run linter

### Creating Migrations

```bash
make migrate-create
# Enter migration name when prompted
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

## Architecture

The application follows clean architecture principles with clear separation of concerns:

- **Domain Layer**: Core business entities and rules
- **Service Layer**: Business logic and orchestration
- **Repository Layer**: Data access and persistence
- **Transport Layer**: HTTP handlers, middleware, and API contracts

## Configuration

Configuration is managed through environment variables. See `.env` for available options:

- `SERVER_PORT` - API server port (default: 8080)
- `SERVER_ENV` - Environment (development/production)
- `DB_*` - Database configuration
- `REDIS_*` - Redis configuration
- `JWT_*` - JWT token configuration

## API Documentation

API documentation will be available at `/api/docs` once implemented.

## License

MIT
