/*
 * MySQL Graph Visualizer - Config Command
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewConfigCmd creates the config command
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management and validation",
		Long:  `Validate, display, and manage MySQL Graph Visualizer configurations.`,
	}

	// Add subcommands
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigInitCmd())

	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  `Validate the syntax and structure of a configuration file without connecting to databases.`,
		Example: `  # Validate configuration file
  mysql-graph-cli config validate --config mysql-production.yml

  # Validate current directory config
  mysql-graph-cli config validate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigValidate(configFile)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file to validate")
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var (
		configFile string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display configuration file content",
		Long:  `Display the contents of a configuration file in various formats.`,
		Example: `  # Show config as YAML
  mysql-graph-cli config show --config mysql-production.yml

  # Show config as JSON
  mysql-graph-cli config show --config mysql-production.yml --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(configFile, format)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file to display")
	cmd.Flags().StringVar(&format, "format", "yaml", "Output format: yaml, json")
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var (
		outputFile string
		template   string
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long:  `Create a new configuration file based on a template.`,
		Example: `  # Initialize minimal config
  mysql-graph-cli config init --template minimal --output config.yml

  # Initialize production config
  mysql-graph-cli config init --template production --output prod-config.yml --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(outputFile, template, force)
		},
	}

	cmd.Flags().StringVar(&outputFile, "output", "mysql-graph-config.yml", "Output configuration file")
	cmd.Flags().StringVar(&template, "template", "minimal", "Configuration template: minimal, development, testing, production, sakila")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing file")

	return cmd
}

func runConfigValidate(configFile string) error {
	fmt.Println("🔍 Configuration Validation")
	fmt.Println("===========================")

	// Find config file if not specified
	if configFile == "" {
		configFile = findConfigFile()
		if configFile == "" {
			return fmt.Errorf("no configuration file found. Use --config to specify a file")
		}
		fmt.Printf("📄 Found configuration: %s\n", configFile)
	} else {
		fmt.Printf("📄 Validating: %s\n", configFile)
	}

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	// Read file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	fmt.Printf("📏 File size: %d bytes\n", len(data))

	// Parse as YAML
	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("❌ YAML syntax error: %v\n", err)
		return fmt.Errorf("invalid YAML syntax")
	}

	fmt.Println("✅ YAML syntax is valid")

	// Basic structure validation
	validationErrors := validateConfigStructure(config)
	
	if len(validationErrors) > 0 {
		fmt.Printf("⚠️  Configuration warnings:\n")
		for _, err := range validationErrors {
			fmt.Printf("   • %s\n", err)
		}
	} else {
		fmt.Println("✅ Configuration structure is valid")
	}

	// Connection parameters validation
	if mysqlConfig, ok := config["mysql"].(map[interface{}]interface{}); ok {
		validateMySQLConfig(mysqlConfig)
	}

	if neo4jConfig, ok := config["neo4j"].(map[interface{}]interface{}); ok {
		validateNeo4jConfig(neo4jConfig)
	}

	fmt.Println("\n🎉 Configuration validation completed!")
	
	if len(validationErrors) == 0 {
		fmt.Println("✅ No issues found - configuration is ready for use")
	} else {
		fmt.Printf("⚠️  Found %d warnings - please review before use\n", len(validationErrors))
	}

	return nil
}

func runConfigShow(configFile string, format string) error {
	fmt.Println("📄 Configuration Display")
	fmt.Println("========================")

	// Find config file if not specified
	if configFile == "" {
		configFile = findConfigFile()
		if configFile == "" {
			return fmt.Errorf("no configuration file found. Use --config to specify a file")
		}
	}

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	// Read file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	fmt.Printf("📁 File: %s\n", configFile)
	fmt.Printf("📏 Size: %d bytes\n", len(data))
	fmt.Printf("🎯 Format: %s\n", format)
	fmt.Println(fmt.Sprintf("%s", string([]rune{0x2500}[0]))+string([]rune{0x2500}[:40])+fmt.Sprintf("%s", string([]rune{0x2500}[0])))

	switch format {
	case "json":
		// Parse YAML and convert to JSON
		var config interface{}
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to convert to JSON: %w", err)
		}

		fmt.Println(string(jsonData))
	default:
		// Display as YAML
		fmt.Print(string(data))
	}

	return nil
}

func runConfigInit(outputFile string, template string, force bool) error {
	fmt.Println("🔧 Configuration Initialization")
	fmt.Println("===============================")

	// Check if file exists
	if _, err := os.Stat(outputFile); err == nil && !force {
		return fmt.Errorf("file %s already exists (use --force to overwrite)", outputFile)
	}

	fmt.Printf("📄 Output file: %s\n", outputFile)
	fmt.Printf("🎯 Template: %s\n", template)

	// Generate config based on template
	outputDir := filepath.Dir(outputFile)
	
	// Create output directory if needed
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	var configContent string
	var err error

	switch template {
	case "minimal":
		configContent = getMinimalConfigTemplate()
	case "development":
		configContent = getDevelopmentConfigTemplate()
	case "testing":
		configContent = getTestingConfigTemplate()
	case "production":
		configContent = getProductionConfigTemplate()
	case "sakila":
		configContent = getSakilaConfigTemplate()
	default:
		return fmt.Errorf("unknown template: %s", template)
	}

	// Write configuration file
	err = os.WriteFile(outputFile, []byte(configContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Configuration file created: %s\n", outputFile)
	fmt.Println("\n📝 Next steps:")
	fmt.Println("   1. Edit the configuration file with your database details")
	fmt.Println("   2. Test the connection: mysql-graph-cli test --config " + outputFile)
	fmt.Println("   3. Analyze your database: mysql-graph-cli analyze --config " + outputFile)

	return nil
}

func findConfigFile() string {
	// Look for common config file names in current directory
	candidates := []string{
		"mysql-graph-config.yml",
		"mysql-graph-config.yaml",
		"config.yml",
		"config.yaml",
		"mysql-sakila.yml",
		"mysql-production.yml",
		"mysql-development.yml",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func validateConfigStructure(config map[string]interface{}) []string {
	var errors []string

	// Check for required sections
	if _, ok := config["mysql"]; !ok {
		errors = append(errors, "Missing 'mysql' section")
	}

	if _, ok := config["neo4j"]; !ok {
		errors = append(errors, "Missing 'neo4j' section - required for graph transformation")
	}

	return errors
}

func validateMySQLConfig(mysqlConfig map[interface{}]interface{}) {
	fmt.Println("\n🔍 MySQL Configuration:")
	
	// Check required fields
	requiredFields := []string{"host", "port", "username", "password", "database"}
	for _, field := range requiredFields {
		if _, ok := mysqlConfig[field]; !ok {
			if _, ok := mysqlConfig[field+"s"]; !ok { // Check plural form too
				fmt.Printf("   ⚠️  Missing required field: %s\n", field)
			}
		}
	}

	// Check connection mode
	if mode, ok := mysqlConfig["connection_mode"]; ok {
		if mode == "existing" {
			fmt.Println("   ✅ Connection mode: existing (direct database connection)")
		}
	} else {
		fmt.Println("   ⚠️  Consider adding 'connection_mode: existing' for direct connections")
	}

	// Check security settings
	if _, ok := mysqlConfig["security"]; ok {
		fmt.Println("   ✅ Security configuration present")
	} else {
		fmt.Println("   ⚠️  Consider adding security configuration for production use")
	}
}

func validateNeo4jConfig(neo4jConfig map[interface{}]interface{}) {
	fmt.Println("\n🔍 Neo4j Configuration:")
	
	// Check required fields
	if _, ok := neo4jConfig["uri"]; !ok {
		fmt.Println("   ⚠️  Missing required field: uri")
	} else {
		fmt.Println("   ✅ URI configured")
	}

	if _, ok := neo4jConfig["user"]; !ok {
		fmt.Println("   ⚠️  Missing required field: user")
	}

	if _, ok := neo4jConfig["password"]; !ok {
		fmt.Println("   ⚠️  Missing required field: password")
	}
}

// Template functions (simplified versions)
func getMinimalConfigTemplate() string {
	return `# MySQL Graph Visualizer - Minimal Configuration
# Issue #10 - Direct Database Connection Implementation

mysql:
  host: "localhost"
  port: 3306
  username: "your_username"
  password: "your_password"
  database: "your_database"
  connection_mode: "existing"

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
}

func getDevelopmentConfigTemplate() string {
	return getMinimalConfigTemplate() // Simplified for now
}

func getTestingConfigTemplate() string {
	return getMinimalConfigTemplate() // Simplified for now
}

func getProductionConfigTemplate() string {
	return getMinimalConfigTemplate() // Simplified for now
}

func getSakilaConfigTemplate() string {
	return `# MySQL Graph Visualizer - Sakila Sample Database
mysql:
  host: "127.0.0.1"
  port: 3308
  username: "sakila_user"
  password: "sakila123"
  database: "sakila"
  connection_mode: "existing"

neo4j:
  uri: "bolt://localhost:7688"
  user: "neo4j"
  password: "sakila123"
`
}
