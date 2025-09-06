/*
 * SQL Graph Visualizer - Test Command
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/services"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/infrastructure/factories"

	"github.com/spf13/cobra"
)

// NewTestCmd creates the test command
func NewTestCmd() *cobra.Command {
	var (
		// Common flags
		dbType             string
		host               string
		port               int
		username           string
		password           string
		database           string
		connectionTimeout  int
		detailed           bool
		
		// PostgreSQL specific flags
		schema             string
		sslMode            string
		sslCertFile        string
		sslKeyFile         string
		sslCAFile          string
		applicationName    string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test connection to database quickly without full analysis",
		Long: `Performs a quick connection test to validate database connectivity and permissions
without running the full schema analysis. Useful for debugging connection issues.

Supports both MySQL and PostgreSQL databases.

This command provides immediate feedback on:
- Database connectivity
- Authentication
- Basic permissions
- Server information`,
		Example: `  # Quick MySQL connection test
  sql-graph-cli test --db-type mysql --host localhost --username user --password pass --database mydb

  # PostgreSQL connection test
  sql-graph-cli test --db-type postgresql --host localhost --username postgres --password pass --database chinook

  # PostgreSQL with SSL
  sql-graph-cli test --db-type postgresql --host remote.com --username user --password pass --database mydb --ssl-mode require

  # Detailed connection test with security validation
  sql-graph-cli test --db-type mysql --host remote-db.com --port 3306 --username user --password pass --database prod --detailed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(testOptions{
				DBType:            models.DatabaseType(dbType),
				Host:              host,
				Port:              port,
				Username:          username,
				Password:          password,
				Database:          database,
				ConnectionTimeout: connectionTimeout,
				Detailed:         detailed,
				// PostgreSQL specific
				Schema:            schema,
				SSLMode:           sslMode,
				SSLCertFile:       sslCertFile,
				SSLKeyFile:        sslKeyFile,
				SSLCAFile:         sslCAFile,
				ApplicationName:   applicationName,
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

	// Test settings
	cmd.Flags().IntVar(&connectionTimeout, "connection-timeout", 10, "Connection timeout in seconds")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Perform detailed security validation")
	
	// PostgreSQL specific flags
	cmd.Flags().StringVar(&schema, "schema", "public", "PostgreSQL schema name")
	cmd.Flags().StringVar(&sslMode, "ssl-mode", "prefer", "PostgreSQL SSL mode: disable, allow, prefer, require, verify-ca, verify-full")
	cmd.Flags().StringVar(&sslCertFile, "ssl-cert", "", "PostgreSQL SSL certificate file")
	cmd.Flags().StringVar(&sslKeyFile, "ssl-key", "", "PostgreSQL SSL key file")
	cmd.Flags().StringVar(&sslCAFile, "ssl-ca", "", "PostgreSQL SSL CA file")
	cmd.Flags().StringVar(&applicationName, "app-name", "sql-graph-visualizer", "PostgreSQL application name")

	// Required flags
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	cmd.MarkFlagRequired("database")

	return cmd
}

type testOptions struct {
	// Common options
	DBType            models.DatabaseType
	Host              string
	Port              int
	Username          string
	Password          string
	Database          string
	ConnectionTimeout int
	Detailed         bool
	
	// PostgreSQL specific options
	Schema            string
	SSLMode           string
	SSLCertFile       string
	SSLKeyFile        string
	SSLCAFile         string
	ApplicationName   string
}

func runTest(opts testOptions) error {
	fmt.Println("SQL Graph Visualizer - Connection Test")
	fmt.Println("======================================")

	// Auto-detect port if not specified
	if opts.Port == 0 {
		switch opts.DBType {
		case models.DatabaseTypeMySQL:
			opts.Port = 3306
		case models.DatabaseTypePostgreSQL:
			opts.Port = 5432
		}
	}

	// Build database-specific configuration for testing
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
			
			// Minimal filtering for testing
			DataFiltering: models.DataFilteringConfig{
				SchemaDiscovery:  true,
				RowLimitPerTable: 1, // Just check table access
			},
			
			Security: models.SecurityConfig{
				ReadOnly:          true,
				ConnectionTimeout: opts.ConnectionTimeout,
				QueryTimeout:      30, // Short timeout for testing
				MaxConnections:    1,  // Single connection for testing
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
			ApplicationName: opts.ApplicationName,
			
			// Minimal filtering for testing
			DataFiltering: models.DataFilteringConfig{
				SchemaDiscovery:  true,
				RowLimitPerTable: 1, // Just check table access
			},
			
			Security: models.SecurityConfig{
				ReadOnly:          true,
				ConnectionTimeout: opts.ConnectionTimeout,
				QueryTimeout:      30, // Short timeout for testing
				MaxConnections:    1,  // Single connection for testing
			},
		}
		
	default:
		return fmt.Errorf("unsupported database type: %s", opts.DBType)
	}

	// Create database repository using factory
	factory := factories.NewDatabaseRepositoryFactory()
	repo, err := factory.CreateRepository(opts.DBType)
	if err != nil {
		return fmt.Errorf("failed to create database repository: %w", err)
	}

	// Initialize universal database service
	dbService := services.NewUniversalDatabaseService(repo, config)

	ctx := context.Background()

	// Show what we're testing
	fmt.Printf("Target: %s database - %s@%s:%d/%s\n", strings.ToUpper(string(opts.DBType)), opts.Username, opts.Host, opts.Port, opts.Database)
	if opts.DBType == models.DatabaseTypePostgreSQL && opts.Schema != "" {
		fmt.Printf("Schema: %s\n", opts.Schema)
	}
	if opts.DBType == models.DatabaseTypePostgreSQL && opts.SSLMode != "" {
		fmt.Printf("SSL Mode: %s\n", opts.SSLMode)
	}
	fmt.Printf("Timeout: %d seconds\n", opts.ConnectionTimeout)
	if opts.Detailed {
		fmt.Printf("Detailed validation: enabled\n")
	}

	fmt.Println("\nRunning connection test...")
	startTime := time.Now()

	if opts.Detailed {
		// Run full analysis for detailed validation
		result, err := dbService.ConnectAndAnalyze(ctx)
		if err != nil {
			fmt.Printf("Connection test failed: %v\n", err)
			return nil
		}

		duration := time.Since(startTime)

		if !result.Success {
			fmt.Printf("Connection test failed: %s\n", result.ErrorMessage)
			return nil
		}

		// Display detailed results
		fmt.Printf("Connection test passed in %v\n", duration)
		
		fmt.Println("\nConnection Details:")
		if result.DatabaseInfo != nil {
			fmt.Printf("   Database: %s\n", result.DatabaseInfo.Database)
			fmt.Printf("   Server Version: %s\n", result.DatabaseInfo.Version)
			fmt.Printf("   User: %s\n", result.DatabaseInfo.User)
		}
		if result.SecurityValidation != nil {
			fmt.Printf("   Security Level: %s\n", result.SecurityValidation.SecurityLevel)
		}

		if result.Summary != nil {
			fmt.Printf("   Tables Found: %d\n", result.Summary.TotalTables)
			
			if len(result.Summary.Warnings) > 0 {
				fmt.Printf("   Warnings: %d\n", len(result.Summary.Warnings))
				for _, warning := range result.Summary.Warnings {
					fmt.Printf("      • %s\n", warning)
				}
			}

			if len(result.Summary.Recommendations) > 0 {
				fmt.Println("\nRecommendations:")
				for _, rec := range result.Summary.Recommendations {
					fmt.Printf("      • %s\n", rec)
				}
			}
		}

		// Show security validation details (simplified for universal service)
		if result.SecurityValidation != nil && len(result.SecurityValidation.Validations) > 0 {
			fmt.Println("\nSecurity Validation:")
			for checkName, validation := range result.SecurityValidation.Validations {
				status := "PASS"
				if !validation.Passed {
					status = "FAIL"
				}
				fmt.Printf("   [%s] %s: %s\n", status, checkName, validation.Message)
			}

			if len(result.SecurityValidation.Recommendations) > 0 {
				fmt.Println("\nSecurity Recommendations:")
				for _, rec := range result.SecurityValidation.Recommendations {
					fmt.Printf("   • %s\n", rec)
				}
			}
		}

	} else {
		// Quick test only
		testResult, err := dbService.TestConnection(ctx)
		if err != nil {
			fmt.Printf("Connection test failed: %v\n", err)
			return nil
		}

		duration := time.Since(startTime)

		if !testResult.Success {
			fmt.Printf("Connection failed: %s\n", testResult.ErrorMessage)
			
			if len(testResult.SecurityIssues) > 0 {
				fmt.Println("\nSecurity Issues:")
				for _, issue := range testResult.SecurityIssues {
					fmt.Printf("   • %s\n", issue)
				}
			}
			return nil
		}

		// Display quick test results
		fmt.Printf("Connection test passed in %v\n", duration)
		
		fmt.Println("\nConnection Details:")
		fmt.Printf("   Database: %s\n", testResult.DatabaseName)
		fmt.Printf("   Server Version: %s\n", testResult.ServerVersion)
		fmt.Printf("   User: %s\n", testResult.UserName)
		fmt.Printf("   Tables Found: %d\n", testResult.TableCount)

		if len(testResult.Warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, warning := range testResult.Warnings {
				fmt.Printf("   • %s\n", warning)
			}
		}
	}

	fmt.Println("\nConnection test completed successfully!")
	fmt.Println("You can now run 'analyze' command for full schema analysis.")

	return nil
}
