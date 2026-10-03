# SQL Graph Visualizer

> **Status: Active Development** - This project is under active development. APIs may change.

<div align="center">

### See your database workload heat up the table graph - live

<img src="docs/images/live-benchmark.gif" alt="Live graph benchmarking: a workload without an index turns the table graph red, one click on Apply suggested index turns it green" width="100%">

**Run a workload. Watch every table and JOIN light up in real time. Add an index. Watch the graph cool down.**

`make demo` &nbsp;&rarr;&nbsp; open **http://localhost:3000/benchmark-live** &nbsp;&rarr;&nbsp; click **Run**

[**Try the live benchmark demo**](#live-graph-benchmarking) &nbsp;|&nbsp; [How it works](#how-it-works)

</div>

<table style="border-collapse: collapse; width: 100%; margin-top: 20px; border-style: hidden">
<tr>
<td width="60%">

**Live graph benchmarking** is the flagship feature: a benchmark that shows *where* in your schema the time goes, while the workload is still running. Nodes are tables, edges are JOINs, and colour and thickness follow the measured latency and load, next to live QPS and p50/p95/p99 charts.

On top of that, SQL Graph Visualizer is a Go application that transforms SQL database structures (MySQL, PostgreSQL, Oracle, SQL Server) into Neo4j graph databases with interactive visualization and comprehensive performance analysis. Built with Domain Driven Design architecture and featuring a unified CLI, flexible transformation rules, advanced performance benchmarking, and robust database connection management.

</td>
<td width="40%">

<img src="SGV - screenshot.png" alt="Interactive Neo4j graph view of a transformed SQL database" width="100%" style="border-radius: 8px; box-shadow: 0 4px 8px rgba(0,0,0,0.1);">

</td>
</tr>
</table>

<div align="center">

[![License: Dual](https://img.shields.io/badge/License-AGPL%2BCommercial-blue.svg)](LICENSE-DUAL.md)
[![Go Version](https://img.shields.io/github/go-mod/go-version/peter7775/sql-graph-visualizer)](https://golang.org/)
[![CI/CD](https://github.com/peter7775/sql-graph-visualizer/actions/workflows/go.yml/badge.svg)](https://github.com/peter7775/sql-graph-visualizer/actions/workflows/go.yml)
[![Release](https://img.shields.io/github/v/release/peter7775/sql-graph-visualizer)](https://github.com/peter7775/sql-graph-visualizer/releases)
[![Stars](https://img.shields.io/github/stars/peter7775/sql-graph-visualizer?style=social)](https://github.com/peter7775/sql-graph-visualizer/stargazers)
[![Discussions](https://img.shields.io/github/discussions/peter7775/sql-graph-visualizer)](https://github.com/peter7775/sql-graph-visualizer/discussions)

[![MySQL](https://img.shields.io/badge/MySQL-8.0+-4479A1?logo=mysql&logoColor=white)](https://mysql.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-13+-336791?logo=postgresql&logoColor=white)](https://postgresql.org/)
[![SQL Server](https://img.shields.io/badge/SQL%20Server-2017+-CC2927?logo=microsoftsqlserver&logoColor=white)](https://www.microsoft.com/sql-server)
[![Oracle](https://img.shields.io/badge/Oracle-19c+-F80000?logo=oracle&logoColor=white)](https://www.oracle.com/database/)
[![Neo4j](https://img.shields.io/badge/Neo4j-4.4+-008CC1?logo=neo4j&logoColor=white)](https://neo4j.com/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://hub.docker.com/)

[![Live Graph Benchmarking](https://img.shields.io/badge/Live%20Graph-Benchmarking-e11d48?logo=speedtest&logoColor=white)](#live-graph-benchmarking)
[![Performance](https://img.shields.io/badge/Performance-Benchmarking-orange?logo=speedtest&logoColor=white)](#performance-benchmarking)
[![Enterprise](https://img.shields.io/badge/Enterprise-Ready-success?logo=enterprise&logoColor=white)](#enterprise-architecture)
[![API](https://img.shields.io/badge/API-GraphQL%20%7C%20REST-purple?logo=graphql&logoColor=white)](#api-documentation)

</div>
---

<div align="center">

**[Quick Start](#quick-start)** • 
**[Live Benchmarking](#live-graph-benchmarking)** • 
**<a href="https://sql-graph-visualizer-production.up.railway.app" target="_blank">Live Demo</a>** • 
**[Documentation](https://github.com/peter7775/sql-graph-visualizer/wiki)** • 
**[Discussions](https://github.com/peter7775/sql-graph-visualizer/discussions)** • 
**[Issues](https://github.com/peter7775/sql-graph-visualizer/issues)**

</div>

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Live Graph Benchmarking](#live-graph-benchmarking)
- [Installation](#installation)
- [Configuration](#configuration)
- [Transformation Rules](#transformation-rules)
- [API Documentation](#api-documentation)
- [Visualization](#visualization)
- [Testing](#testing)
- [Docker](#docker)
- [Contributing](#contributing)
- [Equity Program](#-equity-program) 
- [Roadmap](#roadmap)
- [License](#license)

## Features

### **Live Graph Benchmarking** *(flagship)*
- **Watch a workload hit your schema in real time** - tables are nodes, JOINs are edges; colour = latency, size = QPS, thickness = share of database busy time
- **Live QPS and p50/p95/p99 charts** updated every second, plus a ranked list of the slowest queries
- **One-click optimization** - apply a suggested index, re-run the same workload and see the graph turn from red to green
- **Before/after comparison** with delta badges (for example `p95 -99 %`, `QPS +8000 %`) and a ghost line of the previous run in every chart
- **Streams over Server-Sent Events** with replay and resume, plus a polling fallback; a built-in Go load generator, so no sysbench is needed
- **One command to try it:** `make demo` starts a seeded e-shop database and the whole stack in Docker - see [Live Graph Benchmarking](#live-graph-benchmarking)

### **Database Transformation**
- **Complete SQL to Neo4j conversion** with support for MySQL, PostgreSQL, Oracle, and SQL Server
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

### **Performance Analysis & Benchmarking**
- **[Live graph benchmarking](#live-graph-benchmarking)** with streaming metrics and a before/after comparison
- **Database performance benchmarking** with sysbench and custom SQL query sets
- **Live MySQL Performance Schema** metrics collection (statements, table I/O, indexes, connections)
- **Automated bottleneck detection** and hotspot analysis
- **Query pattern analysis** with optimization suggestions
- **Performance regression detection** across historical benchmark runs
- **Benchmark result persistence** with JSON/CSV export and summary reporting
- **Real-time performance monitoring** via WebSocket with visual graph load mapping

### **AI & Semantic Search** *(Planned — see [Roadmap](#roadmap))*
- **Semantic schema search** using vector embeddings of tables/columns
- **AI-assisted transformation rule suggestions** based on column similarity, even without declared foreign keys
- **Semantic clustering** of Performance Schema query patterns for more accurate hotspot detection
- **Text-to-Cypher** natural language querying via LLM + retrieval-augmented generation over the schema
- Built on a native **Neo4j 5.13+ vector index** and a pluggable `EmbeddingProvider` (OpenAI, Ollama for on-prem use)

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
│   ├── sql-graph-visualizer/   # Unified CLI entry point (cobra)
│   │   ├── main.go
│   │   └── commands/           # CLI subcommands
│   └── main.go                 # Legacy entry point (deprecated)
├── internal/
│   ├── application/            # Application Layer
│   │   ├── bootstrap/          # App initialization & lifecycle
│   │   ├── ports/              # Interface definitions
│   │   └── services/           # Application services
│   ├── domain/                 # Domain Layer
│   │   ├── aggregates/         # Domain aggregates
│   │   ├── entities/           # Domain entities
│   │   ├── events/             # Domain events
│   │   └── models/             # Domain models
│   ├── infrastructure/         # Infrastructure Layer
│   │   ├── middleware/         # HTTP middleware
│   │   └── persistence/        # Database repositories (MySQL, PostgreSQL, Oracle, MSSQL, Neo4j)
│   └── interfaces/             # Interface Layer
│       └── web/                # Web interface files
├── config/                     # Configuration files
├── docs/                       # Documentation
└── scripts/                    # Utility scripts
```

### **Tech Stack**
- **Language**: Go 1.24+
- **Source Databases**: MySQL 8.0+, PostgreSQL 13+, Oracle 19c+, SQL Server 2017+
- **Graph Database**: Neo4j 4.4+ *(upgrade to 5.13+ planned, to enable native vector index support — see [Roadmap](#roadmap))*
- **CLI Framework**: Cobra with shell completion
- **API Layer**: GraphQL (gqlgen), REST (Gorilla Mux)
- **Frontend**: HTML5, JavaScript, Neovis.js, D3.js and Chart.js (vendored, the live benchmark page works offline)
- **Live streaming**: Server-Sent Events (benchmark samples), WebSocket (Performance Schema monitoring)
- **Configuration**: Viper + YAML
- **Logging**: Logrus with structured logging
- **Testing**: Testify framework
- **Containerization**: Docker & Docker Compose
- **Performance Tools**: sysbench, custom SQL benchmark suites
- **Connection Management**: Database/sql with connection pooling

## Quick Start

### Prerequisites
- Go 1.24 or higher
- MySQL 8.0+, PostgreSQL 13+, Oracle 19c+, or SQL Server 2017+
- Neo4j 4.4+ (or use Docker)
- Git

### 1. Clone and Setup
```bash
git clone https://github.com/peter7775/sql-graph-visualizer.git
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
# Build the unified CLI
make build

# Run transformation only
./sql-graph-visualizer transform

# Start full server (transform + web UI + API)
./sql-graph-visualizer serve

# Start with specific config and debug logging
./sql-graph-visualizer serve -c config/config.yml -v
```

### 5. Access the Application
- **Visualization Interface**: http://localhost:3000
- **Live Benchmark** (demo mode, see [below](#live-graph-benchmarking)): http://localhost:3000/benchmark-live
- **GraphQL Playground**: http://localhost:8080/graphql
- **REST API**: http://localhost:8080/api/*
- **Neo4j Browser**: http://localhost:7474

<a id="live-benchmark-demo"></a>
## Live Graph Benchmarking

> **The flagship feature.** Most benchmarking tools end with a table of numbers. SQL Graph Visualizer shows you *where* in your schema the time goes - while the workload is still running.

Nodes are tables, edges are the JOINs between them. Every second the colour, size and thickness of the graph update from the running workload, next to live QPS and p50/p95/p99 charts. A slow join is not a number in a report - it is a thick red line you can point at.

<table>
<tr>
<td width="50%" align="center">

**Run A - no index on the JOIN columns**

<img src="docs/images/live-benchmark-run-a-no-index.png" alt="Table graph with red nodes and a thick red edge: every lookup scans a whole table, p95 is about 3 seconds" width="100%">

</td>
<td width="50%" align="center">

**Run B - same workload after one click**

<img src="docs/images/live-benchmark-run-b-index.png" alt="The same table graph, now green: index lookups, p95 is about 20 milliseconds" width="100%">

</td>
</tr>
</table>

### Try it in one command

```bash
make demo
```

This builds the app image, starts MySQL (seeded e-shop dataset), Neo4j and the app via
`docker-compose.demo.yml`, waits until all three are healthy and opens
**http://localhost:3000/benchmark-live** (if `xdg-open` is available). Other targets:
`make demo-logs`, `make demo-down` (stop and delete all demo data), `make demo-reseed` (start again from a fresh dataset).

### The story (30-60 seconds)

1. Pick the **Checkout peak** scenario and click **Run**. `orders.customer_id` and `order_items.product_id`
   have no index, so lookups scan whole tables: the edges around `orders` and `order_items` turn red,
   latency is in the hundreds of milliseconds to seconds and QPS is low.
2. Click **Apply suggested index**. The demo runs two `CREATE INDEX` statements
   (`orders(customer_id, total_amount)` and `order_items(product_id, quantity)`).
3. **Run** again. The same workload now runs with index lookups: the graph turns green, latency drops to
   a few milliseconds and QPS goes up by orders of magnitude. The before/after comparison shows the delta.
4. **Revert** drops the indexes again, so the demo can be repeated.

<p align="center">
<img src="docs/images/live-benchmark-compare.png" alt="Before/after comparison: p95 -99 %, QPS +8000 %, with the slowest queries ranked below the graph" width="85%">
</p>

> Typical result on a 4-core laptop (numbers vary with the load of the machine): p95 about 2.5 s -> about 30 ms, throughput about 15 QPS -> about 1,000 QPS. It is the same workload and the same data - only the index differs.

### What you see

- **Table graph** - node colour = heat (average latency of the queries that touch the table, log scale from about 5 ms green to about 500 ms red), node size = QPS, edge colour = latency of the JOIN, edge width = share of the total database busy time, particles flow along edges in proportion to QPS. The layout is computed once, so the graph never jumps.
- **KPI cards** - QPS, p95, p99 and error rate, with delta badges after a second run.
- **Charts** - throughput and p50/p95/p99 latency over the last 60 s, with the previous run as a dashed ghost line.
- **Slowest queries** - ranked by average latency, with the tables each query touches.
- **Before/after table** - Run A versus Run B with the percentage change per metric.
- **Presentation mode** (`P`) for talks and demos, light theme (`T`), `Space` to start/stop, `I` to apply/revert the index.

### How it works

```mermaid
flowchart LR
  Q["Custom query set<br/>(YAML)"] --> G["Go load generator<br/>N worker threads"]
  G --> S["1 s sampler<br/>lock-free latency histograms"]
  S --> H["Per-run sample buffer<br/>(replay + fan-out)"]
  H -->|"SSE /stream"| U["Browser<br/>D3 graph + Chart.js"]
  H -->|"GET /samples"| U
  U -->|"POST /api/demo/optimization/apply"| D[("MySQL")]
  G --> D
```

- **Load generator** - the `custom` benchmark tool runs a weighted set of `SELECT`/`INSERT`/`UPDATE` queries on N threads against the source database. No external tool such as sysbench is needed.
- **Sampler** - every second the collector turns constant-memory log-scale latency histograms (about 5 % bucket width) into a *live sample*: interval QPS, avg/p50/p95/p99/max latency, errors, per-query statistics and the table graph (node heat, edge latency and load).
- **Streaming** - samples are pushed over **Server-Sent Events** (`GET /api/performance/benchmarks/{id}/stream`). A new connection first replays the buffered samples, `Last-Event-ID` resumes after a dropped connection, and `GET .../samples?since=N` is a polling fallback. The UI switches to polling by itself if the stream keeps failing.
- **Graph mapping** - each query lists the `tables` it touches, ordered along its JOIN path; every adjacent pair becomes a graph edge. If `tables` is omitted they are derived from the `FROM`/`JOIN` clauses.
- **Comparison** - `GET /api/performance/benchmarks/compare?a=<id>&b=<id>` returns both summaries and the percentage change.
- **Safe by design** - benchmark queries are limited to `SELECT`/`INSERT`/`UPDATE`. The demo endpoints that create or drop indexes exist **only when `DEMO_MODE=true`** and accept nothing but plain `CREATE INDEX` / `DROP INDEX` statements from the configuration.

### Use it with your own queries

Benchmark streaming works for any custom query set. Define the queries (with the `tables` they use) and start a run through the API:

```yaml
performance:
  monitoring:
    enabled: true              # initialises the performance services
  benchmarks:
    enabled: true
    custom_queries:
      - name: checkout-peak
        threads: 12
        duration: 30s
        queries:
          - description: Recent orders of a customer
            query: "SELECT o.id, o.total_amount, c.email FROM orders o JOIN customers c ON c.id = o.customer_id WHERE o.customer_id = ? ORDER BY o.id DESC LIMIT 10"
            parameters: [4242]
            weight: 35
            tables: [orders, customers]      # adjacent pairs become graph edges
```

```bash
curl -s -X POST localhost:8080/api/performance/benchmarks \
  -d '{"tool":"custom","query_set":"checkout-peak","duration_seconds":30,"threads":12}'
curl -N localhost:8080/api/performance/benchmarks/<id>/stream     # live samples as SSE
```

The `/benchmark-live` page is driven by the `live_demo` section of the configuration (scenarios, the index to apply and the graph topology). It is only served when `DEMO_MODE=true`, because it can create and drop indexes. See `config/demo-config.yml` for a complete example.

<details>
<summary>Demo API reference (<code>DEMO_MODE=true</code>)</summary>

```bash
GET  /api/demo/scenarios                       # scenario presets
POST /api/demo/scenarios/{name}/run            # optional body: {"threads": 8, "duration_seconds": 20}
GET  /api/demo/topology                        # tables (nodes) and JOINs (edges)
GET  /api/demo/optimization                    # the suggested index and whether it exists
POST /api/demo/optimization/apply              # CREATE INDEX (idempotent)
POST /api/demo/optimization/revert             # DROP INDEX (idempotent)
```

</details>

### Scenarios

| Scenario | Threads | Workload |
| --- | --- | --- |
| `checkout-peak` | 12 | OLTP: recent orders of a customer, order lines, product pages, small idempotent updates. Biggest effect of the index. |
| `reporting-heavy` | 4 | Analytics: top products per category, top customers per city, daily revenue, category sales summary. |
| `mixed-oltp` | 8 | Mix of the above plus light reporting and writes. |

Scenarios, query sets, the index to apply and the graph topology are defined in `config/demo-config.yml`
(`live_demo.*` and `performance.benchmarks.custom_queries`). Writes are idempotent `UPDATE`s of single rows, so the
dataset does not grow while benchmarking.

### Dataset

Generated deterministically by `demo/mysql/02-seed.sql` on the first start: 20 categories, 20,000 customers,
2,500 products, 200,000 orders and about 600,000 order items (about 55 MB). The relations are logical only
(no foreign keys): InnoDB creates an index for every foreign key, which would remove the "slow without index" part
of the story. `scripts/demo-query-timing.sh [apply|revert|explain]` times the key queries directly in MySQL.

### Requirements

- Docker with Compose v2 (`docker compose`), about 2 GB of free RAM (MySQL ~0.4 GB, Neo4j ~0.6 GB) and 4 CPU cores recommended.
- First start: image download + build takes a few minutes; MySQL seeding takes about a minute. Later starts reuse the volume.
- Free host ports: 3000 (UI), 8080 (API), 3306 (MySQL), 7474/7687 (Neo4j). Ports are published on `127.0.0.1` only.
  Override with `DEMO_APP_PORT`, `DEMO_API_PORT`, `DEMO_MYSQL_PORT`, `DEMO_NEO4J_HTTP_PORT`, `DEMO_NEO4J_BOLT_PORT`,
  e.g. `DEMO_APP_PORT=3100 make demo`. Set `DEMO_BIND_ADDR=0.0.0.0` to expose the demo on your network
  (the demo API can create/drop indexes without authentication, so only do this on a trusted network).

### Troubleshooting

- **Port already in use**: use the `DEMO_*_PORT` variables above, or stop the process that owns the port (`ss -ltnp`).
- **`make demo` times out**: the first MySQL start is the slow part. Follow it with `docker logs -f mysql-demo`; raise the limit with `DEMO_WAIT_TIMEOUT=900 make demo`.
- **Results look the same before and after the index**: the indexes may already exist from a previous run. Use **Revert** in the UI or `make demo-reseed`.
- **Latencies are lower than described**: the dataset is sized for a laptop; on a fast machine, increase the row counts in `demo/mysql/02-seed.sql` and run `make demo-reseed`.
- **Changing credentials**: the demo credentials are fixed (`demopass123`) in `docker-compose.demo.yml` and `config/demo-config.yml`; keep both in sync.
- **Inspect the app**: `make demo-logs`, `curl http://localhost:8080/api/health`.

## Installation

### From Source
```bash
git clone https://github.com/peter7775/sql-graph-visualizer.git
cd sql-graph-visualizer
make build
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

The application includes comprehensive performance benchmarking capabilities to analyze database performance and optimize graph transformations. Benchmarks are configured under `performance.benchmarks` in `config/config.yml` and executed via the [Performance Benchmarking API](#performance-benchmarking-api).

### Supported Benchmark Tools

#### sysbench (MySQL/PostgreSQL)
```yaml
performance:
  benchmarks:
    enabled: true
    default_duration: "2m"
    max_duration: "30m"
    results_dir: "data/performance/benchmarks"
    sysbench:
      executable_path: "/usr/bin/sysbench"
      defaults:
        table_size: 100000
        threads: 4
        time: 120
```
Supported sysbench test types: `oltp_read_write`, `oltp_read_only`, `oltp_write_only`, `oltp_point_select`, `oltp_insert`, `oltp_update_index`, `oltp_update_non_index`, `oltp_delete`, `select_random_points`, `select_random_ranges`, `bulk_insert`.

### Custom Query Benchmarks

Define named sets of SELECT/INSERT/UPDATE queries to benchmark against the active source database (DDL and DELETE/TRUNCATE statements are rejected as a safety measure):

```yaml
performance:
  benchmarks:
    custom_queries:
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
Run it with `POST /api/performance/benchmarks` using `"tool": "custom"` and `"query_set": "user_relationships"`.

### Performance Analysis Features

#### Automated Bottleneck & Hotspot Detection
- **Bottleneck identification** from benchmark metrics and slow queries
- **Hotspot detection** across benchmark history, scored by latency/frequency/resource weight
- **Query pattern analysis** to group similar queries and flag anti-patterns
- **Regression detection** comparing the latest run against the previous one

#### Optimization Suggestions & Reporting
- **Automatic optimization suggestions** (indexing, query rewrites, schema, configuration)
- **Overall performance scoring** with a rating per benchmark run
- **Summary reports** via `GET /api/performance/reports/summary` combining bottlenecks, hotspots, query patterns, and regressions
- **Export** persisted benchmark history as JSON or CSV via `GET /api/performance/export`

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
# Get application configuration
GET /config

# Get graph data (JSON format: nodes + relationships)
GET /api/graph

# Health check
GET /api/health

# Deployment/debug info
GET /api/debug
```

#### Performance Benchmarking API
```bash
# List benchmark executions
GET /api/performance/benchmarks

# Start a new benchmark
POST /api/performance/benchmarks
{
  "tool": "sysbench",
  "test_type": "oltp_read_write",
  "duration_seconds": 300,
  "threads": 4,
  "database_type": "mysql"
}
# For custom query sets: {"tool": "custom", "query_set": "user_relationships", "duration_seconds": 120}

# Get benchmark status / results
GET /api/performance/benchmarks/{id}
GET /api/performance/benchmarks/{id}/results

# Stop a running benchmark
POST /api/performance/benchmarks/{id}/stop

# Live benchmarking: stream samples (Server-Sent Events, replay + Last-Event-ID resume)
GET /api/performance/benchmarks/{id}/stream
# ...or poll them
GET /api/performance/benchmarks/{id}/samples?since=<seq>

# Compare two finished runs (percentage change of b versus a)
GET /api/performance/benchmarks/compare?a=<id>&b=<id>

# Current Performance Schema snapshot (optionally with graph data)
GET /api/performance/data?include_graph=true

# Persisted benchmark history
GET /api/performance/data/history

# Performance data mapped onto the graph
GET /api/performance/data/graph

# Metrics summaries
GET /api/performance/metrics/summary
GET /api/performance/metrics/tables
GET /api/performance/metrics/queries

# Summarized report: bottlenecks, hotspots, query patterns, regressions, optimizations
GET /api/performance/reports/summary

# Export persisted benchmark history
GET /api/performance/export?format=json   # or format=csv

# Real-time monitoring
GET /api/performance/realtime/status
GET /ws/performance   # WebSocket stream of live graph performance data
```

### GraphQL Schema

The GraphQL endpoint provides a flexible query interface for graph data:

```graphql
query {
  graph {
    nodes { id label properties }
    relationships { from to type properties }
  }
  nodesByType(type: "User") {
    id
    properties
  }
  node(id: "123") {
    id
    label
    properties
  }
  relationshipsByType(type: "MEMBER_OF") {
    from
    to
    properties
  }
  searchNodes(query: "alice") {
    id
    label
  }
  config {
    neo4j { uri username }
  }
}

mutation {
  transformData
}

subscription {
  graphUpdates {
    nodes { id label }
  }
}
```

**GraphQL Playground**: http://localhost:8080/graphql

> Performance benchmarking and monitoring are exposed via the [REST API](#performance-benchmarking-api); the GraphQL schema currently covers graph data only.

## Visualization

The web interface provides an interactive graph visualization. For the live, streaming view of a running workload see [Live Graph Benchmarking](#live-graph-benchmarking) (`/benchmark-live`); the performance dashboard is at `/performance`.

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

## Equity Program for Contributors

**Join the commercial success! We offer equity sharing for qualified contributors.**

This project has significant commercial potential and we believe in sharing success with those who help build it.

### How It Works

**Contribute meaningfully → Earn equity stake → Share in commercial licensing revenue**

- **Equity Tiers**: 0.1% - 2.0% based on contribution impact
- **Revenue Sharing**: From commercial licensing and enterprise deployments
- **Vesting**: 50% after 6 months of active contribution, 50% after 12 months
- **High-Impact Areas**: Core algorithms, enterprise features, performance optimization

### Qualification Criteria

**Automatic Qualification** (0.1% - 0.5% equity):
- Merge 3+ significant PRs (marked with `equity-eligible` label)
- Resolve complex issues (marked with `high-impact` label)
- Maintain active contribution for 3+ months

**High-Impact Qualification** (0.5% - 2.0% equity):
- Lead major feature development
- Contribute breakthrough innovations
- Drive adoption and community growth
- Enterprise client development

### Commercial Licensing Context

This project operates under a **Dual License model**:
- **Open Source**: Free for non-commercial use (AGPL-3.0)
- **Commercial**: Paid licensing for enterprise use ($2,500+/year)

**Revenue Sources**:
- Enterprise software licensing
- SaaS platform integrations
- Custom development contracts
- Support and consulting services

### Express Your Interest

Ready to contribute and earn equity? **[Create a Contributor Intent Issue](https://github.com/petrstepanov/sql-graph-visualizer/issues/new?assignees=&labels=contributor-intent&template=contributor_intent.yml)**

**Or contact directly**: petrstepanek99@gmail.com

---

**Important**: See [CONTRIBUTORS.md](CONTRIBUTORS.md) for complete equity program terms and legal framework.

## Roadmap

### Completed 
- [x] Basic MySQL to Neo4j transformation
- [x] **PostgreSQL support** with full feature parity
- [x] Rule-based configuration system
- [x] GraphQL API implementation
- [x] Web-based visualization
- [x] Docker containerization
- [x] CI/CD pipeline
- [x] **Performance benchmarking integration** (sysbench, custom SQL query sets)
- [x] **MySQL Performance Schema** live monitoring with statement/table/index/connection metrics
- [x] **Automated bottleneck & hotspot detection** with optimization suggestions
- [x] **Live graph benchmarking** - streaming per-second metrics (SSE), table graph heat map, one-click index optimization and before/after comparison, one-command Docker demo
- [x] **Real-time performance dashboard** with WebSocket updates and graph load overlays
- [x] **Benchmark result persistence** with historical reporting and JSON/CSV export
- [x] **Robust connection management** with pooling and failover
- [x] **Multi-database connection** support
- [x] **Oracle Database Support** with full schema discovery
- [x] **SQL Server (MSSQL) Support** with INFORMATION_SCHEMA + sys.* queries
- [x] **Unified CLI** with cobra subcommands (`transform`, `serve`, `check`, `analyze`, `config`, `generate`, `version`)

### In Progress
- [ ] **Predictive performance insights** exposed via API (trend/anomaly detection engine implemented, REST endpoint pending)
- [ ] **Enterprise authentication** and authorization
- [ ] **Neo4j upgrade to 5.13+** ([#28](https://github.com/peter7775/sql-graph-visualizer/issues/28)) and **neo4j-go-driver v4 → v5 migration** ([#29](https://github.com/peter7775/sql-graph-visualizer/issues/29)) — required foundation for native vector index support

### AI & Vector Search (Planned)
- [ ] **Native Neo4j 5.x vector index** for schema embeddings ([#30](https://github.com/peter7775/sql-graph-visualizer/issues/30))
- [ ] **EmbeddingProvider port** with OpenAI and Ollama (on-prem) adapters ([#31](https://github.com/peter7775/sql-graph-visualizer/issues/31))
- [ ] **Semantic schema search** over table/column embeddings ([#32](https://github.com/peter7775/sql-graph-visualizer/issues/32))
- [ ] **AI-assisted transformation rule suggestions** via column similarity ([#33](https://github.com/peter7775/sql-graph-visualizer/issues/33))
- [ ] **Semantic clustering** of Performance Schema query patterns ([#34](https://github.com/peter7775/sql-graph-visualizer/issues/34))
- [ ] **Text-to-Cypher** via LLM + RAG over the schema ([#35](https://github.com/peter7775/sql-graph-visualizer/issues/35))

### Future Plans 
- [ ] **Reverse Transformation**: Neo4j to SQL conversion
- [ ] **Advanced Analytics**: Graph algorithms integration (PageRank, Community Detection)
- [ ] **Cloud Deployment**: Kubernetes manifests and Helm charts
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
./sql-graph-visualizer serve -v
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

**Custom Query Benchmark Rejected**
```bash
# Only SELECT/INSERT/UPDATE statements are allowed in custom query sets.
# DDL (CREATE/DROP/ALTER) and DELETE/TRUNCATE statements are rejected.
```

**Benchmark Permission Errors**
```bash
# Ensure the database user configured for benchmarking has sufficient
# permissions for the statements used:
# - sysbench OLTP tests need SELECT, INSERT, UPDATE, DELETE, CREATE TABLE, DROP TABLE
# - custom query benchmarks need SELECT, INSERT, UPDATE only
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

## Community & Support

### Connect With Us
- **Discussions**: [GitHub Discussions](https://github.com/peter7775/sql-graph-visualizer/discussions) - Ask questions, share ideas
- **Email**: [petrstepanek99@gmail.com](mailto:petrstepanek99@gmail.com) - Direct contact & partnerships  
- **LinkedIn**: Connect for professional networking
- **Twitter**: Follow for updates and announcements

### Community Updates
- **Newsletter**: Monthly development updates and feature releases
- **Blog**: Technical deep-dives and case studies
- **Webinars**: Live demos and Q&A sessions

### Show Your Support
If this project helps you, consider:
- **Star** this repository
- **Fork** and contribute
- **Share** with your network
- **Sponsor** development efforts

## Acknowledgments

- [Neo4j](https://neo4j.com/) for the excellent graph database
- [Neovis.js](https://github.com/neo4j-contrib/neovis.js) for graph visualization
- [gqlgen](https://github.com/99designs/gqlgen) for GraphQL implementation
- All contributors who have helped improve this project

---

<div align="center">

**Made with love by the SQL Graph Visualizer Team**

[![GitHub Stars](https://img.shields.io/github/stars/peter7775/sql-graph-visualizer?style=social)](https://github.com/peter7775/sql-graph-visualizer/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/peter7775/sql-graph-visualizer?style=social)](https://github.com/peter7775/sql-graph-visualizer/network/members)
[![GitHub Watchers](https://img.shields.io/github/watchers/peter7775/sql-graph-visualizer?style=social)](https://github.com/peter7775/sql-graph-visualizer/watchers)

</div>
