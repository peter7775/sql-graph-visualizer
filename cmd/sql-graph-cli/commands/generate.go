/*
 * SQL Graph Visualizer - Generate Command
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewGenerateCmd creates the generate command
func NewGenerateCmd() *cobra.Command {
	var (
		outputDir string
		template  string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate configuration files and examples",
		Long: `Generate configuration templates and examples for various database connection scenarios.

Available templates:
- production: Production database connection with security settings
- development: Development environment configuration
- testing: Testing configuration with data limits
- sakila: Example configuration for Sakila sample database
- minimal: Minimal configuration template`,
		Example: `  # Generate production configuration template
  sql-graph-cli generate --template production --output-dir ./config

  # Generate all example configurations
  sql-graph-cli generate --template all --output-dir ./examples

  # Generate Sakila database example
  sql-graph-cli generate --template sakila`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(generateOptions{
				OutputDir: outputDir,
				Template:  template,
				Force:     force,
			})
		},
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Output directory for generated files")
	cmd.Flags().StringVar(&template, "template", "minimal", "Template to generate: production, development, testing, sakila, minimal, all")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")

	return cmd
}

type generateOptions struct {
	OutputDir string
	Template  string
	Force     bool
}

func runGenerate(opts generateOptions) error {
	fmt.Println("TOOL SQL Graph Visualizer - Configuration Generator")
	fmt.Println("==================================================")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(opts.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("📂 Output directory: %s\n", opts.OutputDir)
	fmt.Printf("TARGET Template: %s\n", opts.Template)

	switch opts.Template {
	case "production":
		return generateProductionConfig(opts.OutputDir, opts.Force)
	case "development":
		return generateDevelopmentConfig(opts.OutputDir, opts.Force)
	case "testing":
		return generateTestingConfig(opts.OutputDir, opts.Force)
	case "sakila":
		return generateSakilaConfig(opts.OutputDir, opts.Force)
	case "minimal":
		return generateMinimalConfig(opts.OutputDir, opts.Force)
	case "all":
		return generateAllConfigs(opts.OutputDir, opts.Force)
	default:
		return fmt.Errorf("unknown template: %s", opts.Template)
	}
}

func generateProductionConfig(outputDir string, force bool) error {
	filename := filepath.Join(outputDir, "mysql-production.yml")

	config := `# SQL Graph Visualizer - Production Configuration
# Issue #10 - Direct Database Connection Implementation

mysql:
  host: "prod-mysql.company.com"
  port: 3306
  username: "readonly_analytics"
  password: "${MYSQL_ANALYTICS_PASSWORD}"  # Use environment variable
  database: "production_app"
  connection_mode: "existing"

  # Data filtering for production safety
  data_filtering:
    schema_discovery: true
    table_whitelist: ["users", "orders", "products", "categories", "payments"]  # Only analyze key tables
    table_blacklist: ["logs", "sessions", "audit_trail", "temp_"]  # Skip system tables
    row_limit_per_table: 10000  # Limit for large tables
    query_timeout: 300  # 5 minutes for complex queries
    where_conditions:
      users: "created_at >= '2023-01-01' AND status = 'active'"
      orders: "order_date >= CURDATE() - INTERVAL 1 YEAR"
      payments: "processed_at >= CURDATE() - INTERVAL 6 MONTH"

  # Strict security settings for production
  security:
    read_only: true
    connection_timeout: 30
    query_timeout: 300
    max_connections: 2
    allow_production_connections: true
    allow_root_user: false
    allowed_hosts: ["prod-mysql.company.com", "prod-replica.company.com"]
    forbidden_patterns: [".*dev.*", ".*test.*"]

  # SSL/TLS configuration for production
  ssl:
    enabled: true
    cert_file: "/etc/ssl/mysql/client-cert.pem"
    key_file: "/etc/ssl/mysql/client-key.pem"
    ca_file: "/etc/ssl/mysql/ca-cert.pem"
    insecure_skip_verify: false

  # Automatic rule generation
  auto_generated_rules:
    enabled: true
    strategy:
      table_to_node: true
      foreign_keys_to_relations: true
      naming_convention:
        node_type_format: "Pascal"      # users -> User
        relation_type_format: "UPPER_SNAKE"  # user_orders -> USER_ORDERS
    table_overrides:
      user_sessions:
        skip: true  # Skip session tables
      audit_logs:
        skip: true  # Skip audit tables

neo4j:
  uri: "bolt+s://prod-neo4j.company.com:7687"
  user: "neo4j"
  password: "${NEO4J_PASSWORD}"
  
  # Batch processing for large datasets
  batch_processing:
    batch_size: 500       # Smaller batches for production stability
    commit_frequency: 2000
    memory_limit_mb: 1024  # 1GB memory limit
`

	return writeConfigFile(filename, config, force, "production")
}

func generateDevelopmentConfig(outputDir string, force bool) error {
	filename := filepath.Join(outputDir, "mysql-development.yml")

	config := `# SQL Graph Visualizer - Development Configuration
# Issue #10 - Direct Database Connection Implementation

mysql:
  host: "localhost"
  port: 3306
  username: "dev_user"
  password: "dev_password"
  database: "myapp_development"
  connection_mode: "existing"

  # Relaxed filtering for development
  data_filtering:
    schema_discovery: true
    table_blacklist: ["migrations", "schema_migrations", "ar_internal_metadata"]
    row_limit_per_table: 1000  # Small limit for faster development
    query_timeout: 60

  # Development security settings
  security:
    read_only: true
    connection_timeout: 15
    query_timeout: 60
    max_connections: 3
    allow_root_user: true  # Allow root for development

  # No SSL required for local development
  ssl:
    enabled: false

  # Auto-generate rules for all tables
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
  password: "dev_password"
  
  batch_processing:
    batch_size: 1000
    commit_frequency: 5000
    memory_limit_mb: 512
`

	return writeConfigFile(filename, config, force, "development")
}

func generateTestingConfig(outputDir string, force bool) error {
	filename := filepath.Join(outputDir, "mysql-testing.yml")

	config := `# SQL Graph Visualizer - Testing Configuration
# Issue #10 - Direct Database Connection Implementation

mysql:
  host: "test-db.internal"
  port: 3306
  username: "test_user"
  password: "test_password"
  database: "myapp_test"
  connection_mode: "existing"

  # Strict limits for testing
  data_filtering:
    schema_discovery: true
    row_limit_per_table: 100  # Very small for fast tests
    query_timeout: 30
    where_conditions:
      # Use recent test data only
      users: "created_at >= CURDATE() - INTERVAL 30 DAY"
      orders: "created_at >= CURDATE() - INTERVAL 7 DAY"

  # Fast timeouts for testing
  security:
    read_only: true
    connection_timeout: 10
    query_timeout: 30
    max_connections: 1

  # Generate minimal rules for testing
  auto_generated_rules:
    enabled: true
    strategy:
      table_to_node: true
      foreign_keys_to_relations: true

neo4j:
  uri: "bolt://test-neo4j:7687"
  user: "neo4j"
  password: "test_password"
  
  batch_processing:
    batch_size: 100
    commit_frequency: 500
    memory_limit_mb: 256
`

	return writeConfigFile(filename, config, force, "testing")
}

func generateSakilaConfig(outputDir string, force bool) error {
	filename := filepath.Join(outputDir, "mysql-sakila.yml")

	config := `# SQL Graph Visualizer - Sakila Sample Database Configuration
# Issue #10 - Direct Database Connection Implementation
# Perfect for testing and demonstrating the direct database connection functionality

mysql:
  host: "127.0.0.1"
  port: 3308  # Sakila container port
  username: "sakila_user"
  password: "sakila123"
  database: "sakila"
  connection_mode: "existing"

  # Sakila-specific filtering
  data_filtering:
    schema_discovery: true
    table_blacklist: ["film_text"]  # Skip full-text search table
    row_limit_per_table: 100  # Sample data for demo
    query_timeout: 60

  # Standard security for sample database
  security:
    read_only: true
    connection_timeout: 30
    query_timeout: 60
    max_connections: 3

  # Generate rules for film rental domain
  auto_generated_rules:
    enabled: true
    strategy:
      table_to_node: true
      foreign_keys_to_relations: true
      naming_convention:
        node_type_format: "Pascal"      # actor -> Actor
        relation_type_format: "UPPER_SNAKE"  # film_actor -> FILM_ACTOR
    table_overrides:
      film_actor:
        # Junction table - will be detected as RELATIONSHIP automatically
      film_category:
        # Junction table - will be detected as RELATIONSHIP automatically

neo4j:
  uri: "bolt://localhost:7688"  # Sakila Neo4j container port
  user: "neo4j"
  password: "sakila123"
  
  batch_processing:
    batch_size: 1000
    commit_frequency: 5000
    memory_limit_mb: 512

# This configuration demonstrates:
# - Direct connection to existing Sakila database
# - Automatic junction table detection (film_actor, film_category)
# - Graph pattern recognition (star schemas around film, customer, store)
# - Security validation and connection testing
# - Automatic generation of Neo4j transformation rules
`

	return writeConfigFile(filename, config, force, "Sakila sample database")
}

func generateMinimalConfig(outputDir string, force bool) error {
	filename := filepath.Join(outputDir, "mysql-minimal.yml")

	config := `# SQL Graph Visualizer - Minimal Configuration Template
# Issue #10 - Direct Database Connection Implementation

mysql:
  host: "localhost"
  port: 3306
  username: "your_username"
  password: "your_password"
  database: "your_database"
  connection_mode: "existing"

  # Basic configuration
  data_filtering:
    schema_discovery: true

  security:
    read_only: true
    connection_timeout: 30

  auto_generated_rules:
    enabled: true
    strategy:
      table_to_node: true
      foreign_keys_to_relations: true

neo4j:
  uri: "bolt://localhost:7687"
  user: "neo4j"
  password: "your_neo4j_password"
`

	return writeConfigFile(filename, config, force, "minimal")
}

func generateAllConfigs(outputDir string, force bool) error {
	fmt.Println("📄 Generating all configuration templates...")

	configs := []struct {
		name string
		fn   func(string, bool) error
	}{
		{"minimal", generateMinimalConfig},
		{"development", generateDevelopmentConfig},
		{"testing", generateTestingConfig},
		{"production", generateProductionConfig},
		{"sakila", generateSakilaConfig},
	}

	for _, config := range configs {
		fmt.Printf("   • Generating %s configuration...\n", config.name)
		if err := config.fn(outputDir, force); err != nil {
			return fmt.Errorf("failed to generate %s config: %w", config.name, err)
		}
	}

	// Also generate a README
	readmeFile := filepath.Join(outputDir, "README.md")
	readme := `# SQL Graph Visualizer - Configuration Examples

This directory contains configuration examples for Issue #10 - Direct Database Connection Implementation.

## Available Configurations

- **mysql-minimal.yml** - Basic configuration template
- **mysql-development.yml** - Development environment setup
- **mysql-testing.yml** - Testing configuration with data limits
- **mysql-production.yml** - Production-ready configuration with security
- **mysql-sakila.yml** - Sakila sample database example

## Usage

1. Copy the appropriate configuration file
2. Modify the database connection parameters
3. Adjust data filtering and security settings as needed
4. Run the CLI tool:

` + "```bash" + `
# Test connection
sql-graph-cli test -c mysql-sakila.yml

# Analyze database
sql-graph-cli analyze -c mysql-production.yml --output analysis.json
` + "```" + `

## Security Notes

- Always use read-only database users for analysis
- Enable SSL/TLS for production databases
- Use environment variables for sensitive credentials
- Apply appropriate table filtering for large databases

## Generated by SQL Graph Visualizer CLI
Issue #10 - Direct Database Connection Implementation
`

	return writeConfigFile(readmeFile, readme, force, "README")
}

func writeConfigFile(filename, content string, force bool, description string) error {
	// Check if file exists
	if _, err := os.Stat(filename); err == nil && !force {
		fmt.Printf(" File %s already exists (use --force to overwrite)\n", filename)
		return nil
	}

	// Write file
	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	fmt.Printf("Generated %s configuration: %s\n", description, filename)
	return nil
}
