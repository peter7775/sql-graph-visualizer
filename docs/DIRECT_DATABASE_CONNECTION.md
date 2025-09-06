# Direct Database Connection Implementation (Issue #10)

## Overview

The MySQL Graph Visualizer now supports direct connections to existing MySQL databases, automatically analyzing their schema structure and generating Neo4j transformation rules. This feature eliminates the need for SQL dump files and enables real-time analysis of production databases.

## Features

### Core Capabilities
- **Direct Database Connection**: Connect to any existing MySQL database
- **Automatic Schema Discovery**: Analyze table structure, relationships, and constraints
- **Junction Table Detection**: Automatically identify many-to-many relationship tables
- **Graph Pattern Recognition**: Detect star schemas, hierarchical structures, and other patterns
- **Security Validation**: Multi-layer security checks and read-only access verification
- **Rule Generation**: Automatically create Neo4j transformation rules
- **Configuration Management**: Template-based configuration system
- **Performance Optimization**: Efficient processing with configurable limits

### Security Features
- Read-only database access validation
- SSL/TLS connection support
- Production database connection policies
- Credential strength validation
- Permission verification
- Network security checks

## Installation

### Prerequisites
- Go 1.19 or later
- MySQL database access
- Neo4j database (for transformation)

### Building the CLI
```bash
cd mysql-graph-visualizer
go build ./cmd/mysql-graph-cli
```

## CLI Commands

### 1. Connection Testing
Test database connectivity quickly without full analysis:

```bash
# Quick connection test
mysql-graph-cli test --host localhost --username user --password pass --database mydb

# Detailed test with security validation
mysql-graph-cli test --host prod-db.com --username readonly_user --password secret --database production --detailed
```

**Options:**
- `--host`: Database host (default: localhost)
- `--port`: Database port (default: 3306)
- `--username`: MySQL username (required)
- `--password`: MySQL password (required)
- `--database`: Database name (required)
- `--connection-timeout`: Connection timeout in seconds (default: 10)
- `--detailed`: Enable detailed security validation

### 2. Database Analysis
Perform comprehensive schema analysis and rule generation:

```bash
# Basic analysis
mysql-graph-cli analyze --host localhost --username user --password pass --database mydb

# Advanced analysis with filtering
mysql-graph-cli analyze \
  --host prod-db.company.com \
  --username readonly_analytics \
  --password $DB_PASSWORD \
  --database ecommerce_prod \
  --whitelist "users,orders,products,categories" \
  --blacklist "logs,sessions,temp_tables" \
  --row-limit 10000 \
  --output analysis.json \
  --format json

# Dry run without rule generation
mysql-graph-cli analyze --host localhost --database mydb --dry-run
```

**Options:**
- `--whitelist`: Comma-separated list of tables to analyze
- `--blacklist`: Comma-separated list of tables to skip
- `--row-limit`: Maximum rows per table for analysis
- `--output`: Output file path (default: stdout)
- `--format`: Output format - summary, json, yaml (default: summary)
- `--dry-run`: Analyze without generating transformation rules
- `--connection-timeout`: Connection timeout in seconds
- `--query-timeout`: Query timeout in seconds
- `--max-connections`: Maximum database connections

### 3. Configuration Management
Generate and manage configuration files:

```bash
# Generate configuration templates
mysql-graph-cli generate --template all --output-dir ./config-examples

# Generate specific template
mysql-graph-cli generate --template production --output-dir ./config

# Initialize new configuration
mysql-graph-cli config init --template minimal --output config.yml

# Validate configuration
mysql-graph-cli config validate --config production.yml

# Display configuration
mysql-graph-cli config show --config production.yml --format json
```

**Available Templates:**
- `minimal`: Basic configuration template
- `development`: Development environment setup
- `testing`: Testing configuration with data limits
- `production`: Production-ready configuration with security
- `sakila`: Example configuration for Sakila sample database
- `all`: Generate all templates

## Configuration Files

### Basic Configuration Structure

```yaml
mysql:
  host: "localhost"
  port: 3306
  username: "your_username"
  password: "your_password"
  database: "your_database"
  connection_mode: "existing"

  data_filtering:
    schema_discovery: true
    table_whitelist: ["users", "orders", "products"]
    table_blacklist: ["logs", "sessions", "cache"]
    row_limit_per_table: 10000
    where_conditions:
      users: "created_at >= '2023-01-01' AND status = 'active'"
      orders: "order_date >= CURDATE() - INTERVAL 1 YEAR"

  security:
    read_only: true
    connection_timeout: 30
    query_timeout: 300
    max_connections: 3

  ssl:
    enabled: false
    cert_file: "/path/to/client-cert.pem"
    key_file: "/path/to/client-key.pem"
    ca_file: "/path/to/ca-cert.pem"
    insecure_skip_verify: false

  auto_generated_rules:
    enabled: true
    strategy:
      table_to_node: true
      foreign_keys_to_relations: true
      naming_convention:
        node_type_format: "Pascal"
        relation_type_format: "UPPER_SNAKE"

neo4j:
  uri: "bolt://localhost:7687"
  user: "neo4j"
  password: "your_neo4j_password"
  
  batch_processing:
    batch_size: 1000
    commit_frequency: 5000
    memory_limit_mb: 1024
```

### Production Configuration Example

```yaml
mysql:
  host: "prod-mysql.company.com"
  port: 3306
  username: "readonly_analytics"
  password: "${MYSQL_ANALYTICS_PASSWORD}"
  database: "production_app"
  connection_mode: "existing"

  data_filtering:
    schema_discovery: true
    table_whitelist: ["users", "orders", "products", "categories", "payments"]
    table_blacklist: ["logs", "sessions", "audit_trail", "temp_"]
    row_limit_per_table: 10000
    where_conditions:
      users: "created_at >= '2023-01-01' AND status = 'active'"
      orders: "order_date >= CURDATE() - INTERVAL 1 YEAR"

  security:
    read_only: true
    connection_timeout: 30
    query_timeout: 300
    max_connections: 2
    allow_production_connections: true
    allow_root_user: false
    allowed_hosts: ["prod-mysql.company.com", "prod-replica.company.com"]
    forbidden_patterns: [".*dev.*", ".*test.*"]

  ssl:
    enabled: true
    cert_file: "/etc/ssl/mysql/client-cert.pem"
    key_file: "/etc/ssl/mysql/client-key.pem"
    ca_file: "/etc/ssl/mysql/ca-cert.pem"
    insecure_skip_verify: false

  auto_generated_rules:
    enabled: true
    strategy:
      table_to_node: true
      foreign_keys_to_relations: true
      naming_convention:
        node_type_format: "Pascal"
        relation_type_format: "UPPER_SNAKE"
    table_overrides:
      user_sessions:
        skip: true
      audit_logs:
        skip: true

neo4j:
  uri: "bolt+s://prod-neo4j.company.com:7687"
  user: "neo4j"
  password: "${NEO4J_PASSWORD}"
  
  batch_processing:
    batch_size: 500
    commit_frequency: 2000
    memory_limit_mb: 1024
```

## Schema Analysis Features

### Junction Table Detection
The system automatically detects junction tables (many-to-many relationship tables) using heuristics:
- Tables with multiple foreign key relationships
- High ratio of foreign keys to total columns
- Naming patterns suggesting relationships (e.g., `user_roles`, `film_category`)

These tables are automatically classified as RELATIONSHIP entities rather than NODE entities.

### Graph Pattern Recognition
The analyzer identifies common database patterns:

**Star Schema Detection:**
- Central tables with many incoming foreign key relationships
- Confidence scoring based on relationship count
- Automatic identification of fact and dimension tables

**Hierarchical Structures:**
- Tables with self-referencing foreign keys
- Parent-child relationship identification
- Tree structure recognition

### Automatic Rule Generation
Based on schema analysis, the system generates:

**Node Creation Rules:**
```cypher
CREATE (n:User {
  id: row.id,
  username: row.username,
  email: row.email,
  created_at: row.created_at
})
```

**Relationship Creation Rules:**
```cypher
MATCH (u:User {id: row.user_id}), (r:Role {id: row.role_id})
CREATE (u)-[:HAS_ROLE]->(r)
```

## Security Considerations

### Database User Permissions
- **Recommended**: Create dedicated read-only users for analysis
- **Avoid**: Using root or administrative accounts
- **Required Permissions**: SELECT on target tables and INFORMATION_SCHEMA

### Network Security
- Use SSL/TLS for connections over public networks
- Implement firewall rules for database access
- Consider VPN connections for remote analysis

### Production Safety
- Enable `read_only` mode in configuration
- Set appropriate connection and query timeouts
- Use table filtering to limit analysis scope
- Monitor database performance during analysis

## Performance Optimization

### Large Database Handling
- Use `row_limit_per_table` to limit data processing
- Apply `table_whitelist` to focus on important tables
- Implement `where_conditions` for data filtering
- Configure appropriate timeouts

### Memory Management
- Set `memory_limit_mb` in Neo4j batch processing
- Use smaller `batch_size` for memory-constrained environments
- Monitor system resources during analysis

### Connection Management
- Limit `max_connections` to avoid overwhelming database
- Use connection pooling for multiple operations
- Implement proper connection cleanup

## Error Handling and Troubleshooting

### Common Issues

**Connection Failures:**
```
ERROR: Connection failed: Access denied for user 'username'@'host'
```
- Verify username and password
- Check user permissions
- Ensure database allows connections from client IP

**SSL/TLS Issues:**
```
ERROR: SSL certificate verification failed
```
- Verify certificate paths and validity
- Check CA certificate configuration
- Consider using `insecure_skip_verify` for testing only

**Schema Discovery Failures:**
```
WARNING: Failed to analyze table 'tablename': permission denied
```
- Verify SELECT permissions on all tables
- Check INFORMATION_SCHEMA access
- Review table blacklist configuration

**Memory Issues:**
```
ERROR: Out of memory during batch processing
```
- Reduce `batch_size` in configuration
- Lower `memory_limit_mb` setting
- Implement table filtering

### Logging and Debugging
Enable verbose logging:
```bash
mysql-graph-cli analyze --verbose --config production.yml
```

Check configuration validity:
```bash
mysql-graph-cli config validate --config production.yml
```

## Integration Examples

### E-commerce Database Analysis
```bash
mysql-graph-cli analyze \
  --host ecommerce-db.company.com \
  --username analytics_readonly \
  --password $DB_PASSWORD \
  --database ecommerce_prod \
  --whitelist "users,orders,products,categories,order_items" \
  --where-conditions "orders=order_date >= CURDATE() - INTERVAL 6 MONTH" \
  --row-limit 50000 \
  --output ecommerce-analysis.json \
  --format json
```

### Multi-tenant Application
```bash
mysql-graph-cli analyze \
  --host saas-db.company.com \
  --username tenant_analyzer \
  --password $DB_PASSWORD \
  --database tenant_db \
  --blacklist "logs,sessions,cache_*,temp_*" \
  --row-limit 10000 \
  --output tenant-schema.json
```

### Legacy System Migration
```bash
mysql-graph-cli analyze \
  --host legacy-system.company.com \
  --username migration_user \
  --password $LEGACY_PASSWORD \
  --database legacy_crm \
  --query-timeout 600 \
  --connection-timeout 60 \
  --output legacy-migration-rules.json \
  --format json
```

## API Reference

### DirectDatabaseService
Main service class for database analysis operations.

**Methods:**
- `ConnectAndAnalyze(ctx context.Context) (*DirectDatabaseAnalysisResult, error)`
- `TestConnection(ctx context.Context) (*ConnectionTestResult, error)`
- `GetDataSizeEstimation(ctx context.Context) (*DatasetInfo, error)`

### SchemaAnalyzerService
Service for schema discovery and pattern recognition.

**Methods:**
- `AnalyzeSchemaFromConnection(ctx, db, filterConfig) (*SchemaAnalysisResult, error)`
- `generateTransformationRules(*SchemaAnalysisResult) error`

### SecurityValidationService
Service for connection security validation.

**Methods:**
- `ValidateConnectionSecurity(ctx, config) (*SecurityValidationResult, error)`

## Best Practices

### Database Analysis
1. **Start with Connection Testing**: Use `test` command before full analysis
2. **Use Read-Only Users**: Create dedicated users with minimal permissions
3. **Implement Filtering**: Use whitelist/blacklist for focused analysis
4. **Set Reasonable Limits**: Configure row limits for large databases
5. **Monitor Performance**: Watch database load during analysis

### Configuration Management
1. **Use Templates**: Start with appropriate configuration templates
2. **Environment Variables**: Use env vars for sensitive credentials
3. **Version Control**: Store configurations in version control
4. **Validate Configurations**: Always validate before use
5. **Document Changes**: Maintain configuration change logs

### Security
1. **SSL/TLS**: Always use encrypted connections for production
2. **Network Security**: Implement proper firewall rules
3. **Credential Management**: Use secure credential storage
4. **Access Logging**: Monitor database access logs
5. **Regular Audits**: Periodically review access permissions

### Performance
1. **Resource Monitoring**: Monitor CPU, memory, and I/O during analysis
2. **Batch Processing**: Use appropriate batch sizes for data volume
3. **Connection Limits**: Don't overwhelm database with connections
4. **Query Optimization**: Use efficient WHERE conditions
5. **Timeout Configuration**: Set appropriate timeouts for operations

## Migration from File-Based Approach

### Before (File-Based)
```bash
# Export database
mysqldump -u user -p database > dump.sql

# Import to test environment
mysql -u user -p test_db < dump.sql

# Run transformation
mysql-graph-visualizer --config config.yml
```

### After (Direct Connection)
```bash
# Direct analysis
mysql-graph-cli analyze \
  --host production-db.com \
  --username readonly_user \
  --password $DB_PASSWORD \
  --database production_db \
  --output analysis.json
```

### Benefits
- **No Data Export**: Eliminates need for database dumps
- **Real-Time Analysis**: Works with live data
- **Security**: No sensitive data in files
- **Efficiency**: Faster analysis without I/O overhead
- **Automation**: Can be automated in CI/CD pipelines

## Conclusion

The Direct Database Connection implementation provides a powerful, secure, and efficient way to analyze existing MySQL databases and generate Neo4j transformation rules. By following the guidelines and best practices in this documentation, users can successfully integrate this feature into their data analysis and migration workflows.

For additional support or feature requests, please refer to the project's issue tracker on GitHub.
