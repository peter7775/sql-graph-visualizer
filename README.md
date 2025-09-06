# SQL Graph Visualizer

[![License: Dual](https://img.shields.io/badge/License-AGPL%2BCommercial-blue.svg)](LICENSE-DUAL.md)

> **Status: Active Development** - This project is under active development. APIs may change.

A powerful Go application that transforms SQL database structures (MySQL, PostgreSQL) into Neo4j graph databases with interactive visualization capabilities. Built with Domain Driven Design architecture and featuring flexible, user-configurable transformation rules.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [Transformation Rules](#transformation-rules)
- [API Documentation](#api-documentation)
- [Visualization](#visualization)
- [Testing](#testing)
- [Docker](#docker)
- [Contributing](#contributing)
- [Roadmap](#roadmap)
- [License](#license)

## Features

### **Database Transformation**
- **Complete SQL to Neo4j conversion** with support for MySQL and PostgreSQL
- **Flexible rule-based mapping** with custom transformation rules
- **Custom SQL query support** - transform not just tables, but any SQL query result
- **Relationship modeling** - define directional logical links between nodes
- **Property mapping** - map SQL columns to Neo4j node properties
- **Aggregation support** - create analytical nodes from complex queries

### **Visualization & Analysis**
- **Interactive graph visualization** using Neovis.js and D3.js
- **Real-time data exploration** with GraphQL queries
- **RESTful API** for programmatic access
- **Customizable node appearance** and relationship styling
- **Filter and search** capabilities within the graph

### **Enterprise Architecture**
- **Domain Driven Design (DDD)** - clean, maintainable codebase
- **Layered architecture** - domain, application, infrastructure, and interface layers
- **Dependency injection** with ports and adapters pattern
- **Comprehensive logging** with structured logging support
- **Configuration management** with YAML-based rules

### **Developer Experience**
- **Docker support** for easy deployment
- **Comprehensive testing** suite
- **GitHub Actions CI/CD** pipeline
- **Detailed documentation** and examples
- **Issue templates** for bug reports and feature requests

## Architecture

This project follows **Domain Driven Design (DDD)** principles with a clean layered architecture:

```
sql-graph-visualizer/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── application/            # Application Layer
│   │   ├── ports/              # Interface definitions
│   │   └── services/           # Application services
│   ├── domain/                 # Domain Layer
│   │   ├── aggregates/         # Domain aggregates
│   │   ├── entities/           # Domain entities
│   │   ├── events/             # Domain events
│   │   └── models/             # Domain models
│   ├── infrastructure/         # Infrastructure Layer
│   │   ├── middleware/         # HTTP middleware
│   │   └── persistence/        # Database repositories
│   └── interfaces/             # Interface Layer
│       └── web/                # Web interface files
├── config/                     # Configuration files
├── docs/                       # Documentation
└── scripts/                    # Utility scripts
```

### **Tech Stack**
- **Language**: Go 1.24+
- **Source Databases**: MySQL 8.0+, PostgreSQL 13+ (planned)
- **Graph Database**: Neo4j 4.4+
- **API Layer**: GraphQL (gqlgen), REST (Gorilla Mux)
- **Frontend**: HTML5, JavaScript, Neovis.js
- **Configuration**: Viper + YAML
- **Logging**: Logrus with structured logging
- **Testing**: Testify framework
- **Containerization**: Docker & Docker Compose

## Quick Start

### Prerequisites
- Go 1.24 or higher
- MySQL 8.0+
- Neo4j 4.4+ (or use Docker)
- Git

### 1. Clone and Setup
```bash
git clone https://github.com/yourusername/sql-graph-visualizer.git
cd sql-graph-visualizer
go mod tidy
```

### 2. Start Neo4j (using Docker)
```bash
docker-compose up -d neo4j-test
```

### 3. Configure Database Connections
```bash
cp config/config.yml.example config/config.yml
# Edit config/config.yml with your database credentials
```

### 4. Run the Application
```bash
# Development mode with debug logging
LOG_LEVEL=debug go run cmd/main.go

# Or build and run
go build -o sql-graph-visualizer cmd/main.go
./sql-graph-visualizer
```

### 5. Access the Application
- **Visualization Interface**: http://localhost:3000
- **GraphQL Playground**: http://localhost:8080/graphql
- **REST API**: http://localhost:8080/api/*
- **Neo4j Browser**: http://localhost:7474

## Installation

### From Source
```bash
git clone https://github.com/yourusername/sql-graph-visualizer.git
cd sql-graph-visualizer
go build -o sql-graph-visualizer cmd/main.go
```

### Using Docker
```bash
docker-compose up -d
```

### Using Go Install
```bash
go install github.com/yourusername/sql-graph-visualizer@latest
```

## Configuration

The application uses YAML configuration files. The main configuration file is `config/config.yml`:

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

transform_rules:
  - name: "users_to_nodes"
    rule_type: "node"
    source:
      type: "query"
      value: "SELECT * FROM users WHERE is_active = 1"
    target_type: "User"
    field_mappings:
      id: "id"
      username: "username"
      email: "email"
```

### Environment Variables
- `LOG_LEVEL`: Set logging level (`debug`, `info`, `warn`, `error`)
- `CONFIG_PATH`: Path to configuration file (default: `config/config.yml`)
- `PORT`: HTTP server port (default: `3000`)
- `API_PORT`: API server port (default: `8080`)

## Transformation Rules

Transformation rules define how MySQL data is converted to Neo4j. There are two main rule types:

### Node Rules
Create Neo4j nodes from MySQL data:

```yaml
- name: "users_to_nodes"
  rule_type: "node"
  source:
    type: "query"  # or "table"
    value: "SELECT u.*, CONCAT(u.first_name, ' ', u.last_name) as full_name FROM users u"
  target_type: "User"
  field_mappings:
    id: "id"
    username: "username"
    full_name: "name"  # Neo4j property name
```

### Relationship Rules
Create Neo4j relationships between nodes:

```yaml
- name: "user_team_membership"
  rule_type: "relationship"
  relationship_type: "MEMBER_OF"
  direction: "outgoing"  # outgoing, incoming, or both
  source:
    type: "query"
    value: "SELECT user_id, team_id, role, joined_at FROM team_members"
  source_node:
    type: "User"
    key: "user_id"
    target_field: "id"
  target_node:
    type: "Team"
    key: "team_id"
    target_field: "id"
  properties:
    role: "role"
    joined_at: "joined_at"
```

### Advanced Features
- **Custom Aggregations**: Create analytical nodes from complex SQL queries
- **Conditional Logic**: Apply rules based on data conditions
- **Property Transformation**: Transform data types and formats
- **Relationship Properties**: Add metadata to relationships

## API Documentation

### REST API Endpoints

```bash
# Get current configuration
GET /api/config

# Get graph data (JSON format)
GET /api/graph

# Get specific node data
GET /api/nodes/{type}

# Get relationships
GET /api/relationships/{type}

# Health check
GET /api/health
```

### GraphQL Schema

The GraphQL endpoint provides a flexible query interface:

```graphql
query {
  nodes(type: "User") {
    id
    properties
  }
  relationships(type: "MEMBER_OF") {
    source
    target
    properties
  }
}
```

**GraphQL Playground**: http://localhost:8080/graphql

## Visualization

The web interface provides an interactive graph visualization:

### Features
- **Interactive Navigation**: Pan, zoom, and drag nodes
- **Node Filtering**: Filter by node types and properties
- **Relationship Highlighting**: Highlight specific relationship types
- **Search Functionality**: Find nodes by name or properties
- **Layout Options**: Different graph layout algorithms
- **Export Capabilities**: Export graph data or screenshots

### Customization
Customize the visualization by modifying the configuration:

```yaml
visualization:
  node_colors:
    User: "#4CAF50"
    Team: "#2196F3"
    Project: "#FF9800"
  relationship_colors:
    MEMBER_OF: "#757575"
    LEADS: "#F44336"
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Tests with Coverage
```bash
go test -cover ./...
```

### Run Specific Package Tests
```bash
go test ./internal/domain/...
go test ./internal/application/...
```

### Integration Tests
```bash
# Start test databases
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./...
```

### Load Testing
```bash
# Using included load test script
./scripts/load-test.sh
```

## Docker

### Development Setup
```bash
# Start all services (MySQL, Neo4j, Application)
docker-compose up -d

# View logs
docker-compose logs -f sql-graph-visualizer

# Stop services
docker-compose down
```

### Production Deployment
```bash
# Build production image
docker build -t sql-graph-visualizer:latest .

# Run with production configuration
docker run -d \
  --name sql-graph-visualizer \
  -p 3000:3000 \
  -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  sql-graph-visualizer:latest
```

### Health Checks
The Docker container includes health checks:

```bash
docker ps  # Check health status
docker inspect sql-graph-visualizer  # Detailed health info
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Run tests and ensure they pass
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to your branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Code Standards
- Follow Go best practices and idioms
- Maintain DDD architecture principles
- Write comprehensive tests
- Update documentation
- Use conventional commit messages

### Issue Templates
We provide issue templates for:
- [Bug Reports](.github/ISSUE_TEMPLATE/bug_report.yml)
- [Feature Requests](.github/ISSUE_TEMPLATE/feature_request.yml)
- [Database Connection Issues](.github/ISSUE_TEMPLATE/database_connection.yml)
- [Performance Issues](.github/ISSUE_TEMPLATE/performance.yml)

## Roadmap

### Completed
- [x] Basic MySQL to Neo4j transformation
- [x] Rule-based configuration system
- [x] GraphQL API implementation
- [x] Web-based visualization
- [x] Docker containerization
- [x] CI/CD pipeline

### In Progress
- [ ] Advanced visualization features
- [ ] Performance optimizations
- [ ] Real-time data synchronization

### Future Plans
- [ ] **Reverse Transformation**: Neo4j to MySQL conversion
- [ ] **Multi-database Support**: PostgreSQL, SQLite, Oracle
- [ ] **Advanced Analytics**: Graph algorithms integration
- [ ] **Cloud Deployment**: Kubernetes manifests
- [ ] **Authentication**: User management and access control
- [ ] **Monitoring**: Metrics and observability
- [ ] **Plugin System**: Custom transformation plugins

## Performance

### Benchmarks
- **Small datasets** (< 10k nodes): < 5 seconds
- **Medium datasets** (10k-100k nodes): < 30 seconds
- **Large datasets** (100k+ nodes): Configurable batch processing

### Optimization Tips
- Use indexed columns in transformation queries
- Configure appropriate batch sizes
- Monitor memory usage during large transformations
- Use connection pooling for high-throughput scenarios

## Troubleshooting

### Common Issues

**Connection Errors**
```bash
# Test MySQL connection
mysql -h localhost -u username -p

# Test Neo4j connection
cypher-shell -a bolt://localhost:7687
```

**Port Conflicts**
The application automatically handles port conflicts and will find available ports.

**Memory Issues**
For large datasets, increase the batch size in configuration:

```yaml
processing:
  batch_size: 1000
  max_memory_mb: 2048
```

**Debug Mode**
```bash
LOG_LEVEL=debug go run cmd/main.go
```

## License

### ⚠️ IMPORTANT: License Change Notice

**This project changed from MIT to Dual License on January 6, 2025.**

- **Prior clones (before Jan 6, 2025)**: Continue under MIT License ✅
- **New features & innovations**: Require Dual License compliance 🔒
- **See [LEGAL_NOTICE.md](LEGAL_NOTICE.md) for complete details**

### Current License (From January 6, 2025)

This project is available under a **Dual License**:

### Open Source (AGPL-3.0)
- ✅ **FREE** for open source projects, educational use, and research
- ✅ Source code must remain open source (copyleft)
- ✅ Perfect for learning, contributing, and non-commercial use

### Commercial License
- 💼 **Required** for commercial use, SaaS platforms, and enterprise deployments
- 💰 **Pricing**: Starting at $2,500/year for startups
- 🚀 **Includes**: Proprietary use rights, enterprise support, custom development

**Commercial licensing required for:**
- Database management SaaS platforms
- Enterprise monitoring tools integration
- Commercial database consulting services
- White-label or OEM distributions

**Contact:** petrstepanek99@gmail.com for commercial licensing

### Patent-Pending Innovations
This software contains breakthrough innovations in:
- Database consistency validation through graph transformation
- Performance benchmark integration with visual load mapping
- Automated schema discovery and rule generation

See [LICENSE](LICENSE) for complete terms.

## Acknowledgments

- [Neo4j](https://neo4j.com/) for the excellent graph database
- [Neovis.js](https://github.com/neo4j-contrib/neovis.js) for graph visualization
- [gqlgen](https://github.com/99designs/gqlgen) for GraphQL implementation
- All contributors who have helped improve this project

---

<p align="center">Made with love by the SQL Graph Visualizer Team</p>
