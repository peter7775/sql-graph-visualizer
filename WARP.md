# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Essential Development Commands

### Setup and Dependencies
```bash
# Install dependencies and required tools
make install

# Generate GraphQL code (required after schema changes)
make generate
```

### Building and Running
```bash
# Build the application
make build

# Run in development mode with debug logging
LOG_LEVEL=debug go run cmd/main.go

# Run using make
make run

# Quick rebuild and test cycle
make quick
```

### Testing
```bash
# Run unit tests with coverage
make test

# Run integration tests (requires Docker services running)
make test-integration

# Run all CI checks locally
make ci-check
```

### Docker and Services
```bash
# Start Neo4j test database
make docker-up

# Stop all Docker services
make docker-down

# Start full stack with Docker Compose
docker-compose up -d
```

### Code Quality and Security
```bash
# Format Go code
make format

# Run security scans (govulncheck, gosec)
make sec-scan

# Development environment setup (install + generate + format + test)
make dev
```

### Single Test Execution
```bash
# Run specific package tests
go test ./internal/domain/...
go test ./internal/application/...

# Run specific test with verbose output
go test -v ./internal/application/services/transform/... -run TestTransformService

# Run integration tests with tags
go test -v -timeout 15m -tags=integration ./internal/tests/integration/...
```

## High-Level Architecture

This application follows **Domain Driven Design (DDD)** with clean architecture principles:

### Core Architecture Pattern
- **Domain Layer** (`internal/domain/`): Business logic, entities, aggregates, and value objects
- **Application Layer** (`internal/application/`): Use cases, services, and port definitions  
- **Infrastructure Layer** (`internal/infrastructure/`): Database repositories, external services
- **Interface Layer** (`internal/interfaces/`): Web UI, GraphQL/REST APIs, handlers

### Key Architectural Concepts

**Transformation Engine**: The heart of the system is the rule-based transformation service (`internal/application/services/transform/`) that converts SQL data to Neo4j graphs using configurable YAML rules.

**Dual Database Strategy**: The application maintains connections to both source databases (MySQL/PostgreSQL) and the target graph database (Neo4j), orchestrating data flow between them.

**Port-Adapter Pattern**: All external dependencies (databases, APIs) are abstracted through ports (`internal/application/ports/`) with concrete implementations as adapters in the infrastructure layer.

**Multi-Interface Support**: The system exposes data through multiple interfaces:
- Web visualization (`internal/interfaces/web/`)
- GraphQL API (`internal/interfaces/graphql/`)
- REST API (`internal/interfaces/api/`)

### Critical Configuration System
The transformation behavior is entirely driven by YAML configuration in `config/config.yml`. The configuration supports:
- **Node Rules**: Transform SQL queries/tables into Neo4j nodes
- **Relationship Rules**: Create directed relationships between nodes
- **Complex Aggregations**: Generate analytical nodes from complex SQL queries
- **Property Mappings**: Map SQL columns to Neo4j properties with transformations

## Important Implementation Details

### GraphQL Code Generation
This project uses gqlgen for GraphQL. After modifying `schema/schema.graphqls`, you **must** run `make generate` to regenerate the GraphQL code before building.

### Database Initialization Behavior
The main application (`cmd/main.go`) **automatically deletes all Neo4j data** on startup to ensure clean transformations. This is intentional for development/demo purposes.

### Configuration Loading Logic
Configuration loading follows this priority:
1. `CONFIG_PATH` environment variable (absolute or relative to project root)
2. Test config (`config/config-test.yml`) when `GO_ENV=test`
3. Default config (`config/config.yml`)

### Multi-Database Support Architecture
While primarily MySQL-focused, the codebase has infrastructure for PostgreSQL support:
- Factory pattern in `internal/infrastructure/factories/`
- Separate repository implementations for each database type
- Database-specific configuration handling

### Security and Validation
- gosec security scanning with custom exclusions for generated code
- Input validation in configuration loading (prevents directory traversal)
- Security validation service for transformation rules

## Development Workflow Patterns

### Adding New Transformation Rules
1. Modify `config/config.yml` with new rules
2. Test with sample data
3. Add integration tests in `internal/tests/integration/`
4. Update documentation if adding new rule types

### GraphQL Schema Changes
1. Edit `schema/schema.graphqls`
2. Run `make generate` to regenerate Go code
3. Update resolvers in `internal/interfaces/graphql/`
4. Test with GraphQL playground at http://localhost:8080/graphql

### Adding New Database Support
1. Create repository in `internal/infrastructure/persistence/[database]/`
2. Implement ports in `internal/application/ports/`
3. Update factory in `internal/infrastructure/factories/`
4. Add configuration support in `internal/domain/models/`

## Service Endpoints

When running locally:
- **Web Visualization**: http://localhost:3000
- **GraphQL Playground**: http://localhost:8080/graphql  
- **REST API**: http://localhost:8080/api/*
- **Neo4j Browser**: http://localhost:7474 (when using Docker)

## Environment Variables

- `LOG_LEVEL`: Controls logging verbosity (debug, info, warn, error)
- `CONFIG_PATH`: Override default configuration file path
- `GO_ENV`: Set to "test" to use test configuration
- `PORT`: HTTP server port (default: 3000)
- `API_PORT`: API server port (default: 8080)

## License and Commercial Usage

**Important**: This project changed from MIT to Dual License on January 6, 2025. Commercial use requires separate licensing. See `LICENSE-DUAL.md` for details.

## Testing Data Requirements

Integration tests require test databases with sample data. The Docker Compose setup provides MySQL and Neo4j instances with appropriate test data for the default transformation rules.
