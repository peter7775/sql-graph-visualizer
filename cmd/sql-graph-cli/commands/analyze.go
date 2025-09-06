/*
 * SQL Graph Visualizer - Analyze Command
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/services"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/infrastructure/factories"

	"github.com/spf13/cobra"
)

// NewAnalyzeCmd creates the analyze command
func NewAnalyzeCmd() *cobra.Command {
	var (
		// Common flags
		dbType             string
		host               string
		port               int
		username           string
		password           string
		database           string
		tableWhitelist     []string
		tableBlacklist     []string
		rowLimit           int
		outputFile         string
		outputFormat       string
		dryRun            bool
		connectionTimeout  int
		queryTimeout       int
		maxConnections     int
		
		// PostgreSQL specific flags
		schema             string
		sslMode            string
		sslCertFile        string
		sslKeyFile         string
		sslCAFile          string
		applicationName    string
		statementTimeout   int
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze existing database schema and generate transformation rules",
		Long: `Connects to an existing database (MySQL or PostgreSQL), analyzes its schema structure,
identifies relationships and graph patterns, and generates Neo4j transformation rules automatically.

Supports both MySQL and PostgreSQL databases with database-specific optimizations.`,
		Example: `  # Analyze MySQL database
  sql-graph-cli analyze --db-type mysql --host localhost --port 3306 --username user --password pass --database mydb

  # Analyze PostgreSQL database
  sql-graph-cli analyze --db-type postgresql --host localhost --port 5432 --username postgres --password pass --database chinook

  # PostgreSQL with SSL and schema
  sql-graph-cli analyze --db-type postgresql --host remote.com --database mydb --schema public --ssl-mode require

  # Analyze with table filtering
  sql-graph-cli analyze --db-type mysql --host localhost --database mydb --whitelist "users,orders,products"

  # Save analysis to JSON file
  sql-graph-cli analyze --db-type postgresql --host localhost --database chinook --output analysis.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(cmd, analyzeOptions{
				DBType:            models.DatabaseType(dbType),
				Host:              host,
				Port:              port,
				Username:          username,
				Password:          password,
				Database:          database,
				TableWhitelist:    tableWhitelist,
				TableBlacklist:    tableBlacklist,
				RowLimit:          rowLimit,
				OutputFile:        outputFile,
				OutputFormat:      outputFormat,
				DryRun:           dryRun,
				ConnectionTimeout: connectionTimeout,
				QueryTimeout:      queryTimeout,
				MaxConnections:    maxConnections,
				// PostgreSQL specific
				Schema:            schema,
				SSLMode:           sslMode,
				SSLCertFile:       sslCertFile,
				SSLKeyFile:        sslKeyFile,
				SSLCAFile:         sslCAFile,
				ApplicationName:   applicationName,
				StatementTimeout:  statementTimeout,
			})
		},
	}

	// Database type and connection flags
	cmd.Flags().StringVar(&dbType, "db-type", "mysql", "Database type: mysql, postgresql")
	cmd.Flags().StringVar(&host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&port, "port", 0, "Database port (0 = auto-detect: MySQL=3306, PostgreSQL=5432)")
	cmd.Flags().StringVar(&username, "username", "", "Database username")
	cmd.Flags().StringVar(&password, "password", "", "Database password")
	cmd.Flags().StringVar(&database, "database", "", "Database name")

	// Data filtering flags
	cmd.Flags().StringSliceVar(&tableWhitelist, "whitelist", []string{}, "Comma-separated list of tables to analyze (empty = all)")
	cmd.Flags().StringSliceVar(&tableBlacklist, "blacklist", []string{}, "Comma-separated list of tables to skip")
	cmd.Flags().IntVar(&rowLimit, "row-limit", 0, "Limit number of rows per table for analysis (0 = no limit)")

	// Output flags
	cmd.Flags().StringVar(&outputFile, "output", "", "Output file path (empty = stdout)")
	cmd.Flags().StringVar(&outputFormat, "format", "summary", "Output format: summary, json, yaml")

	// Control flags
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform analysis without generating transformation rules")

	// Connection settings
	cmd.Flags().IntVar(&connectionTimeout, "connection-timeout", 30, "Connection timeout in seconds")
	cmd.Flags().IntVar(&queryTimeout, "query-timeout", 300, "Query timeout in seconds")
	cmd.Flags().IntVar(&maxConnections, "max-connections", 3, "Maximum number of database connections")
	
	// PostgreSQL specific flags
	cmd.Flags().StringVar(&schema, "schema", "public", "PostgreSQL schema name")
	cmd.Flags().StringVar(&sslMode, "ssl-mode", "prefer", "PostgreSQL SSL mode: disable, allow, prefer, require, verify-ca, verify-full")
	cmd.Flags().StringVar(&sslCertFile, "ssl-cert", "", "PostgreSQL SSL certificate file")
	cmd.Flags().StringVar(&sslKeyFile, "ssl-key", "", "PostgreSQL SSL key file")
	cmd.Flags().StringVar(&sslCAFile, "ssl-ca", "", "PostgreSQL SSL CA file")
	cmd.Flags().StringVar(&applicationName, "app-name", "sql-graph-visualizer", "PostgreSQL application name")
	cmd.Flags().IntVar(&statementTimeout, "stmt-timeout", 30, "PostgreSQL statement timeout in seconds")

	// Required flags
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	cmd.MarkFlagRequired("database")

	return cmd
}

type analyzeOptions struct {
	// Common options
	DBType            models.DatabaseType
	Host              string
	Port              int
	Username          string
	Password          string
	Database          string
	TableWhitelist    []string
	TableBlacklist    []string
	RowLimit          int
	OutputFile        string
	OutputFormat      string
	DryRun           bool
	ConnectionTimeout int
	QueryTimeout      int
	MaxConnections    int
	
	// PostgreSQL specific options
	Schema            string
	SSLMode           string
	SSLCertFile       string
	SSLKeyFile        string
	SSLCAFile         string
	ApplicationName   string
	StatementTimeout  int
}

func runAnalyze(cmd *cobra.Command, opts analyzeOptions) error {
	fmt.Println("SQL Graph Visualizer - Database Analysis")
	fmt.Println("=============================================")

	// Auto-detect port if not specified
	if opts.Port == 0 {
		switch opts.DBType {
		case models.DatabaseTypeMySQL:
			opts.Port = 3306
		case models.DatabaseTypePostgreSQL:
			opts.Port = 5432
		}
	}

	// Build database-specific configuration
	var config models.DatabaseConfig
	switch opts.DBType {
	case models.DatabaseTypeMySQL:
		config = &models.MySQLConfig{
			Host:           opts.Host,
			Port:           opts.Port,
			Username:       opts.Username,
			Password:       opts.Password,
			Database:       opts.Database,
			ConnectionMode: models.ConnectionModeExisting,
			
			DataFiltering: models.DataFilteringConfig{
				SchemaDiscovery:  true,
				TableWhitelist:   opts.TableWhitelist,
				TableBlacklist:   opts.TableBlacklist,
				RowLimitPerTable: opts.RowLimit,
			},
			
			Security: models.SecurityConfig{
				ReadOnly:          true,
				ConnectionTimeout: opts.ConnectionTimeout,
				QueryTimeout:      opts.QueryTimeout,
				MaxConnections:    opts.MaxConnections,
			},
			
			AutoGeneratedRules: models.AutoGeneratedRulesConfig{
				Enabled: !opts.DryRun,
				Strategy: &models.RuleGenerationStrategy{
					TableToNode:            true,
					ForeignKeysToRelations: true,
					NamingConvention: &models.NamingConvention{
						NodeTypeFormat:     "Pascal",
						RelationTypeFormat: "UPPER_SNAKE",
					},
				},
			},
		}
		
	case models.DatabaseTypePostgreSQL:
		config = &models.PostgreSQLConfig{
			Host:           opts.Host,
			Port:           opts.Port,
			Username:       opts.Username,
			Password:       opts.Password,
			Database:       opts.Database,
			Schema:         opts.Schema,
			ConnectionMode: models.ConnectionModeExisting,
			
			// PostgreSQL-specific settings
			SSLConfig: models.PostgreSQLSSLConfig{
				Mode:     opts.SSLMode,
				CertFile: opts.SSLCertFile,
				KeyFile:  opts.SSLKeyFile,
				CAFile:   opts.SSLCAFile,
			},
			ApplicationName:   opts.ApplicationName,
			StatementTimeout: opts.StatementTimeout,
			
			DataFiltering: models.DataFilteringConfig{
				SchemaDiscovery:  true,
				TableWhitelist:   opts.TableWhitelist,
				TableBlacklist:   opts.TableBlacklist,
				RowLimitPerTable: opts.RowLimit,
			},
			
			Security: models.SecurityConfig{
				ReadOnly:          true,
				ConnectionTimeout: opts.ConnectionTimeout,
				QueryTimeout:      opts.QueryTimeout,
				MaxConnections:    opts.MaxConnections,
			},
			
			AutoGeneratedRules: models.AutoGeneratedRulesConfig{
				Enabled: !opts.DryRun,
				Strategy: &models.RuleGenerationStrategy{
					TableToNode:            true,
					ForeignKeysToRelations: true,
					NamingConvention: &models.NamingConvention{
						NodeTypeFormat:     "Pascal",
						RelationTypeFormat: "UPPER_SNAKE",
					},
				},
			},
		}
		
	default:
		return fmt.Errorf("unsupported database type: %s", opts.DBType)
	}

	// Create database repository based on type
	factory := factories.NewDatabaseRepositoryFactory()
	repo, err := factory.CreateRepository(opts.DBType)
	if err != nil {
		return fmt.Errorf("failed to create database repository: %w", err)
	}

	// Create universal database service
	dbService := services.NewUniversalDatabaseService(repo, config)

	// Validate configuration
	fmt.Printf("🔧 Validating configuration...\n")
	if err := dbService.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	ctx := context.Background()

	// Show connection info
	fmt.Printf("📡 Connecting to %s database: %s@%s:%d/%s\n", strings.ToUpper(string(opts.DBType)), opts.Username, opts.Host, opts.Port, opts.Database)
	if opts.DBType == models.DatabaseTypePostgreSQL && opts.Schema != "" {
		fmt.Printf("   Schema: %s\n", opts.Schema)
	}
	if opts.DBType == models.DatabaseTypePostgreSQL && opts.SSLMode != "" {
		fmt.Printf("   SSL Mode: %s\n", opts.SSLMode)
	}
	if len(opts.TableWhitelist) > 0 {
		fmt.Printf("TARGET Table whitelist: %s\n", strings.Join(opts.TableWhitelist, ", "))
	}
	if len(opts.TableBlacklist) > 0 {
		fmt.Printf("⚫ Table blacklist: %s\n", strings.Join(opts.TableBlacklist, ", "))
	}
	if opts.RowLimit > 0 {
		fmt.Printf("Row limit: %d per table\n", opts.RowLimit)
	}
	if opts.DryRun {
		fmt.Printf("Dry run mode: analysis only, no rule generation\n")
	}

	// Start analysis
	fmt.Printf("\n🔍 Starting database analysis...\n")
	startTime := time.Now()
	
	result, err := dbService.ConnectAndAnalyze(ctx)
	if err != nil {
		return fmt.Errorf("database analysis failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("analysis failed: %s", result.ErrorMessage)
	}

	duration := time.Since(startTime)
	fmt.Printf("Analysis completed in %v\n", duration)

	// Output results based on format
	switch opts.OutputFormat {
	case "json":
		return outputJSON(result, opts.OutputFile)
	case "yaml":
		return outputYAML(result, opts.OutputFile)
	default:
		return outputSummary(result, opts.OutputFile)
	}
}

func outputSummary(result *models.UniversalDatabaseAnalysisResult, outputFile string) error {
	var output strings.Builder
	
	// Header
	output.WriteString("\n" + strings.Repeat("=", 60) + "\n")
	output.WriteString("DATABASE ANALYSIS RESULTS\n")
	output.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Connection info
	output.WriteString("🔗 CONNECTION INFORMATION:\n")
	if result.DatabaseInfo != nil {
		output.WriteString(fmt.Sprintf("   Database Type: %s\n", strings.ToUpper(string(result.DatabaseType))))
		output.WriteString(fmt.Sprintf("   Database: %s@%s:%d/%s\n", 
			result.DatabaseInfo.User, result.DatabaseInfo.Host, 
			result.DatabaseInfo.Port, result.DatabaseInfo.Database))
		output.WriteString(fmt.Sprintf("   Server Version: %s\n", result.DatabaseInfo.Version))
	}
	output.WriteString(fmt.Sprintf("   Processing Time: %v\n", result.ProcessingDuration))
	if result.SecurityValidation != nil {
		output.WriteString(fmt.Sprintf("   Security Level: %s\n", result.SecurityValidation.SecurityLevel))
	}

	// Summary statistics
	if result.Summary != nil {
		summary := result.Summary
		output.WriteString("\nSTATS ANALYSIS SUMMARY:\n")
		output.WriteString(fmt.Sprintf("   Tables Analyzed: %d\n", summary.TotalTables))

		if len(summary.Warnings) > 0 {
			output.WriteString(fmt.Sprintf("   WARN  Warnings: %d\n", len(summary.Warnings)))
		}
		
		if len(summary.Recommendations) > 0 {
			output.WriteString(fmt.Sprintf("   TIP Recommendations: %d\n", len(summary.Recommendations)))
		}
	}

	// Tables with their information
	if result.SchemaAnalysis != nil && len(result.SchemaAnalysis.Tables) > 0 {
		output.WriteString("\nINFO DISCOVERED TABLES:\n")
		for _, table := range result.SchemaAnalysis.Tables {
			schemaInfo := ""
			if table.Schema != "" {
				schemaInfo = fmt.Sprintf(" (%s)", table.Schema)
			}
			output.WriteString(fmt.Sprintf("   %-20s%s - %d rows, %d columns\n", 
				table.Name, schemaInfo, table.EstimatedRows, len(table.Columns)))
			
			// Show column details for first few tables
			if len(result.SchemaAnalysis.Tables) <= 3 && len(table.Columns) > 0 {
				for i, col := range table.Columns {
					if i >= 5 {
						output.WriteString(fmt.Sprintf("     ... and %d more columns\n", len(table.Columns)-5))
						break
					}
					primaryKey := ""
					if col.IsKey && col.KeyType == "PRIMARY" {
						primaryKey = " (PK)"
					}
					output.WriteString(fmt.Sprintf("     • %s %s%s\n", col.Name, col.DataType, primaryKey))
				}
			}
		}
	}

	// Warnings and recommendations
	if result.Summary != nil {
		if len(result.Summary.Warnings) > 0 {
			output.WriteString("\nWARN  WARNINGS:\n")
			for _, warning := range result.Summary.Warnings {
				output.WriteString(fmt.Sprintf("   • %s\n", warning))
			}
		}

		if len(result.Summary.Recommendations) > 0 {
			output.WriteString("\nTIP RECOMMENDATIONS:\n")
			for _, rec := range result.Summary.Recommendations {
				output.WriteString(fmt.Sprintf("   • %s\n", rec))
			}
		}
	}

	if !result.Success {
		output.WriteString(fmt.Sprintf("\nERROR: %s\n", result.ErrorMessage))
	}

	output.WriteString("\n" + strings.Repeat("=", 60) + "\n")

	return writeOutput(output.String(), outputFile)
}

func outputJSON(result *models.UniversalDatabaseAnalysisResult, outputFile string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return writeOutput(string(data), outputFile)
}

func outputYAML(result *models.UniversalDatabaseAnalysisResult, outputFile string) error {
	// For simplicity, we'll use JSON format for now
	// In a real implementation, you'd use gopkg.in/yaml.v2
	return outputJSON(result, outputFile)
}

func writeOutput(content string, outputFile string) error {
	if outputFile == "" {
		fmt.Print(content)
		return nil
	}

	err := os.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("📄 Output written to %s\n", outputFile)
	return nil
}
