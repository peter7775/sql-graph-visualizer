# mysql-graph-visualizer

A Golang project for converting SQL databases to Neo4j graph databases with visualization capabilities using Domain Driven Design architecture.

## Project Overview

This application transforms MySQL database structures into Neo4j graph databases based on user-defined rules, enabling visualization of database relationships as graphs using GraphQL and Neovis.JS.

### Key Features
- Converts entire or partial MySQL databases to Neo4j
- User-configurable transformation rules via YAML
- Real-time graph visualization 
- RESTful API endpoints for configuration and graph data
- Domain Driven Design architecture
- GraphQL server integration

## Tech Stack

- **Language**: Go 1.22.5+
- **Database Sources**: MySQL
- **Graph Database**: Neo4j 4.4
- **API**: GraphQL (gqlgen), REST
- **Web**: Gorilla Mux router, CORS middleware
- **Configuration**: Viper + YAML
- **Logging**: Logrus
- **Testing**: Testify

### Key Dependencies
- `github.com/99designs/gqlgen` - GraphQL server
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/neo4j/neo4j-go-driver/v4` - Neo4j driver
- `github.com/gorilla/mux` - HTTP router
- `github.com/spf13/viper` - Configuration management
- `github.com/sirupsen/logrus` - Logging

## Project Structure

```
mysql-graph-visualizer/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── application/
│   │   ├── ports/              # Interface definitions
│   │   └── services/           # Application services
│   ├── config/
│   │   └── config.go           # Configuration loading
│   ├── domain/
│   │   ├── aggregates/         # Domain aggregates
│   │   ├── entities/           # Domain entities
│   │   ├── events/             # Domain events
│   │   └── models/             # Domain models
│   ├── infrastructure/
│   │   ├── middleware/         # HTTP middleware
│   │   └── persistence/        # Database repositories
│   └── interfaces/
│       └── web/                # Web interface files
├── config/
│   ├── config.yml              # Main configuration
│   └── config-test.yml         # Test configuration
├── config.yaml                 # Neo4j configuration
├── docker-compose.yml          # Neo4j service
├── gqlgen.yml                  # GraphQL configuration
├── go.mod                      # Go module definition
└── README.md                   # Project documentation
```

## Architecture

Follows **Domain Driven Design (DDD)** principles:

- **Domain Layer**: Core business logic, entities, and aggregates
- **Application Layer**: Use cases, services, and ports (interfaces)
- **Infrastructure Layer**: External concerns (databases, web servers)
- **Interface Layer**: API endpoints and web UI

## Configuration

### Database Configuration
Configure MySQL and Neo4j connections in `config/config.yml`:

```yaml
mysql:
  host: localhost
  port: 3306
  user: username
  password: password
  database: dbname

neo4j:
  uri: bolt://localhost:7687
  user: neo4j
  password: password
```

### Transformation Rules
Define data transformation rules in the same config file to specify:
- Which MySQL tables/queries to convert
- How to create Neo4j nodes and relationships
- Property mappings between source and target
- Directional logical connections

## Development Workflow

### Prerequisites
- Go 1.22.5+
- MySQL server
- Neo4j 4.4+ (can use Docker)

### Quick Start
```bash
# Start Neo4j (using Docker)
docker-compose up -d neo4j-test

# Run the application
go run cmd/main.go

# Access visualization
open http://localhost:3000
```

### Key Commands
```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build binary
go build -o mysql-graph-visualizer cmd/main.go

# Start with custom config
LOG_LEVEL=debug go run cmd/main.go
```

### API Endpoints
- `http://localhost:8080/config` - Get current configuration
- `http://localhost:3000/api/graph` - Get graph data (JSON)
- `http://localhost:3000` - Visualization interface

## Development Notes

### Database Operations
- Application automatically clears Neo4j data on startup
- Transforms MySQL data based on configuration rules
- Creates nodes and relationships in Neo4j
- Serves graph data via REST API

### Architecture Patterns
- **Ports & Adapters**: Clean separation between domain and infrastructure
- **Repository Pattern**: Database access abstraction
- **Service Layer**: Business logic orchestration
- **Aggregate Pattern**: Consistent domain boundaries

### Error Handling
- Comprehensive logging with Logrus
- Graceful server shutdown
- Connection retry logic
- Configuration validation

## Testing

The project includes test configurations and follows Go testing conventions:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/domain/...
```

## Docker Support

Neo4j service is containerized with health checks:
- Neo4j HTTP: `http://localhost:7474`
- Neo4j Bolt: `bolt://localhost:7687`
- Default credentials: `neo4j/testpass`

## Troubleshooting

### Common Issues
1. **Port conflicts**: Application handles port 3000 conflicts automatically
2. **Neo4j connection**: Ensure Neo4j service is running and accessible
3. **MySQL connection**: Verify database credentials and connectivity
4. **Configuration**: Check YAML syntax in config files

### Logging
Set `LOG_LEVEL` environment variable for different log levels:
- `debug`, `info`, `warn`, `error`

## Future Development

The project is designed for extensibility:
- Support for other SQL databases
- Reverse transformation (Neo4j → MySQL)
- Enhanced visualization features
- Real-time data synchronization
- Advanced transformation rules
