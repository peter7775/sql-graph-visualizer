# SQL Graph Visualizer

[![License: Dual](https://img.shields.io/badge/License-AGPL%2BCommercial-blue.svg)](LICENSE-DUAL.md)

> **Status: Active Development** - This project is under active development. APIs may change.

A powerful Go application that transforms SQL database structures (MySQL, PostgreSQL) into Neo4j graph databases with interactive visualization and comprehensive performance analysis capabilities. Built with Domain Driven Design architecture and featuring flexible transformation rules, advanced performance benchmarking, and robust database connection management.

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
- **Complete SQL to Neo4j conversion** with full support for MySQL and PostgreSQL
- **Flexible rule-based mapping** with custom transformation rules
- **Custom SQL query support** - transform not just tables, but any SQL query result
- **Relationship modeling** - define directional logical links between nodes
- **Property mapping** - map SQL columns to Neo4j node properties
- **Aggregation support** - create analytical nodes from complex queries
- **Direct database connections** - robust connection management with automatic failover
- **Connection pooling** - optimized performance for high-throughput scenarios

### **Visualization & Analysis**
- **Interactive graph visualization** using Neovis.js and D3.js
- **Real-time data exploration** with GraphQL queries
- **RESTful API** for programmatic access
- **Customizable node appearance** and relationship styling
- **Filter and search** capabilities within the graph

### **Performance Analysis & Benchmarking**
- **Database performance benchmarking** with sysbench, pgbench integration
- **Automated bottleneck detection** and hotspot analysis
- **Query performance analysis** with optimization suggestions
- **Critical path identification** through database relationships
- **Performance trend analysis** with historical data tracking
- **Custom benchmark scenarios** for specific workload testing
- **Real-time performance monitoring** with visual load mapping

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
- **Source Databases**: MySQL 8.0+, PostgreSQL 13+
- **Graph Database**: Neo4j 4.4+
- **API Layer**: GraphQL (gqlgen), REST (Gorilla Mux)
- **Frontend**: HTML5, JavaScript, Neovis.js
- **Configuration**: Viper + YAML
- **Logging**: Logrus with structured logging
- **Testing**: Testify framework
- **Containerization**: Docker & Docker Compose
- **Performance Tools**: sysbench, pgbench integration
- **Connection Management**: Database/sql with connection pooling

## Quick Start

### Prerequisites
- Go 1.24 or higher
- MySQL 8.0+ or PostgreSQL 13+
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
# MySQL Configuration
mysql:
  host: localhost
  port: 3306
  user: username
  password: password
  database: dbname
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

# PostgreSQL Configuration (alternative to MySQL)
postgresql:
  host: localhost
  port: 5432
  user: username
  password: password
  database: dbname
  sslmode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

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

## Performance Benchmarking

The application includes comprehensive performance benchmarking capabilities to analyze database performance and optimize graph transformations.

### Supported Benchmark Tools

#### sysbench (MySQL/PostgreSQL)
```yaml
benchmark:
  sysbench:
    enabled: true
    executable_path: "/usr/bin/sysbench"
    test_types:
      - "oltp_read_write"
      - "oltp_read_only"
      - "oltp_write_only"
      - "select_random_points"
```

#### pgbench (PostgreSQL)
```yaml
benchmark:
  pgbench:
    enabled: true
    executable_path: "/usr/bin/pgbench"
    default_scale: 10
    test_duration: "5m"
```

### Custom Benchmark Configuration

```yaml
custom_benchmarks:
  - name: "user_relationships"
    description: "Test user-to-team relationship queries"
    duration: "2m"
    threads: 4
    queries:
      - query: "SELECT u.*, t.name FROM users u JOIN team_members tm ON u.id = tm.user_id JOIN teams t ON tm.team_id = t.id WHERE u.is_active = 1"
        weight: 70
        description: "Active user team memberships"
      - query: "SELECT COUNT(*) FROM users u JOIN team_members tm ON u.id = tm.user_id GROUP BY tm.team_id"
        weight: 30
        description: "Team member counts"
```

### Performance Analysis Features

#### Automated Bottleneck Detection
- **Query performance analysis** with execution plan inspection
- **Index usage monitoring** and missing index detection
- **Join efficiency analysis** with optimization suggestions
- **Lock contention detection** and deadlock prevention

#### Hotspot Analysis
- **Table access patterns** identification
- **High-load relationship** detection
- **Resource utilization** tracking (CPU, I/O, memory)
- **Critical path analysis** through database relationships

#### Optimization Suggestions
- **Automatic index recommendations** based on query patterns
- **Query optimization hints** with before/after comparisons
- **Schema improvement** suggestions
- **Connection pooling** optimization

## Database Connection Management

The application provides robust database connection management with automatic failover, connection pooling, and comprehensive error handling.

### Connection Features

#### Automatic Connection Management
- **Connection pooling** with configurable limits
- **Automatic reconnection** on connection failures
- **Health checks** for database availability
- **Graceful degradation** when databases are unavailable

#### Multi-Database Support
```yaml
# Configure multiple databases
databases:
  primary:
    type: "mysql"  # or "postgresql"
    host: "primary-db.example.com"
    port: 3306
    database: "main_db"
    # Connection pool settings
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: "5m"
    conn_max_idle_time: "10m"
  
  secondary:
    type: "postgresql"
    host: "secondary-db.example.com"
    port: 5432
    database: "analytics_db"
    sslmode: "require"
    max_open_conns: 15
    max_idle_conns: 3
```

#### Connection Error Handling
- **Retry mechanisms** with exponential backoff
- **Circuit breaker** pattern for failing connections
- **Detailed error logging** with connection diagnostics
- **Fallback strategies** for multi-database setups

#### Security Features
- **SSL/TLS encryption** support for all database types
- **Connection string validation** to prevent injection
- **Credential management** with environment variable support
- **Connection timeout** configuration

### Performance Optimization

#### Connection Pooling Best Practices
```yaml
connection_pools:
  # Production settings
  production:
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: "1h"
    conn_max_idle_time: "15m"
  
  # Development settings
  development:
    max_open_conns: 10
    max_idle_conns: 2
    conn_max_lifetime: "30m"
    conn_max_idle_time: "5m"
```

#### Monitoring and Diagnostics
- **Connection pool metrics** (active, idle, waiting connections)
- **Query execution timing** and slow query detection
- **Database health monitoring** with periodic checks
- **Performance metrics** export to monitoring systems

## API Documentation

### REST API Endpoints

#### Core Graph API
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

#### Performance Benchmarking API
```bash
# Start a new benchmark
POST /api/performance/benchmark
{
  "tool": "sysbench",
  "test_type": "oltp_read_write",
  "duration": "5m",
  "threads": 4,
  "database_type": "mysql"
}

# Get benchmark results
GET /api/performance/benchmark/{execution_id}

# List all benchmark executions
GET /api/performance/benchmarks

# Get performance analysis
GET /api/performance/analysis/{execution_id}

# Get bottlenecks
GET /api/performance/bottlenecks

# Get optimization suggestions
GET /api/performance/optimizations
```

#### Database Connection API
```bash
# Get connection status
GET /api/connections/status

# Test database connection
POST /api/connections/test
{
  "type": "postgresql",
  "host": "localhost",
  "port": 5432,
  "database": "testdb"
}

# Get connection pool metrics
GET /api/connections/metrics
```

### GraphQL Schema

The GraphQL endpoint provides a flexible query interface:

```graphql
# Basic graph queries
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

# Performance analysis queries
query {
  performanceAnalysis(executionId: "abc123") {
    overallScore {
      score
      rating
    }
    bottlenecks {
      type
      severity
      description
      recommendations
    }
    optimizations {
      type
      title
      impact {
        latencyImprovement
        throughputImprovement
      }
    }
  }
}

# Database connections status
query {
  connectionStatus {
    database
    status
    poolMetrics {
      activeConnections
      idleConnections
      maxConnections
    }
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

### Completed ✅
- [x] Basic MySQL to Neo4j transformation
- [x] **PostgreSQL support** with full feature parity
- [x] Rule-based configuration system
- [x] GraphQL API implementation
- [x] Web-based visualization
- [x] Docker containerization
- [x] CI/CD pipeline
- [x] **Performance benchmarking integration** (sysbench, pgbench)
- [x] **Automated bottleneck detection** and optimization suggestions
- [x] **Robust connection management** with pooling and failover
- [x] **Multi-database connection** support

### In Progress 🚧
- [ ] **Real-time performance monitoring** dashboard
- [ ] **Advanced visualization features** with performance overlays
- [ ] **Trend analysis** and predictive performance insights
- [ ] **Enterprise authentication** and authorization

### Future Plans 🚀
- [ ] **Oracle Database Support**: Enterprise database integration
- [ ] **SQL Server Support**: Microsoft ecosystem compatibility
- [ ] **Reverse Transformation**: Neo4j to SQL conversion
- [ ] **Advanced Analytics**: Graph algorithms integration (PageRank, Community Detection)
- [ ] **Cloud Deployment**: Kubernetes manifests and Helm charts
- [ ] **Machine Learning**: Automated optimization recommendations
- [ ] **Monitoring Integration**: Prometheus, Grafana, DataDog
- [ ] **Plugin System**: Custom transformation and analysis plugins
- [ ] **Multi-tenant SaaS**: Cloud-hosted solution
- [ ] **Streaming Data**: Real-time database change detection

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

### PostgreSQL Connection Issues

**SSL Connection Problems**
```bash
# Test SSL connection
psql "postgresql://username:password@localhost:5432/dbname?sslmode=require"

# Disable SSL for development
psql "postgresql://username:password@localhost:5432/dbname?sslmode=disable"
```

**Authentication Issues**
```yaml
# Update pg_hba.conf for password authentication
postgresql:
  host: localhost
  port: 5432
  user: username
  password: password
  database: dbname
  sslmode: disable
```

### Performance Benchmarking Issues

**sysbench Not Found**
```bash
# Install sysbench on Ubuntu/Debian
sudo apt-get install sysbench

# Install on macOS
brew install sysbench

# Verify installation
sysbench --version
```

**pgbench Configuration**
```bash
# Initialize pgbench tables
pgbench -i -s 10 your_database

# Test pgbench connection
pgbench -c 4 -j 2 -T 60 your_database
```

**Benchmark Permission Errors**
```yaml
# Ensure benchmark user has sufficient permissions
benchmark:
  database_permissions:
    - "SELECT, INSERT, UPDATE, DELETE"
    - "CREATE TABLE, DROP TABLE"
    - "REFERENCES, INDEX"
```

### Connection Pool Issues

**Too Many Connections**
```yaml
# Reduce connection pool size
connection_pools:
  max_open_conns: 10  # Reduce from default 25
  max_idle_conns: 2   # Reduce from default 5
```

**Connection Timeouts**
```yaml
# Increase timeout values
connection_timeout: "30s"
read_timeout: "60s"
write_timeout: "60s"
```

## License

### WARNING IMPORTANT: License Change Notice

**This project changed from MIT to Dual License on January 6, 2025.**

- **Prior clones (before Jan 6, 2025)**: Continue under MIT License ✅
- **New features & innovations**: Require Dual License compliance 🔒
- **See [LEGAL_NOTICE.md](LEGAL_NOTICE.md) for complete details**

### Current License (From January 6, 2025)

This project is available under a **Dual License**:

### Open Source (AGPL-3.0)
- - **FREE** for open source projects, educational use, and research
- - Source code must remain open source (copyleft)
- - Perfect for learning, contributing, and non-commercial use

### Commercial License
- **Required** for commercial use, SaaS platforms, and enterprise deployments
- **Pricing**: Starting at $2,500/year for startups
- **Includes**: Proprietary use rights, enterprise support, custom development

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
