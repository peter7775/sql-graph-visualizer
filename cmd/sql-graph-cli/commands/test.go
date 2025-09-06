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
	"time"

	"sql-graph-visualizer/internal/application/services"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/infrastructure/persistence/mysql"

	"github.com/spf13/cobra"
)

// NewTestCmd creates the test command
func NewTestCmd() *cobra.Command {
	var (
		host               string
		port               int
		username           string
		password           string
		database           string
		connectionTimeout  int
		detailed           bool
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test connection to MySQL database quickly without full analysis",
		Long: `Performs a quick connection test to validate database connectivity and permissions
without running the full schema analysis. Useful for debugging connection issues.

This command provides immediate feedback on:
- Database connectivity
- Authentication
- Basic permissions
- Server information`,
		Example: `  # Quick connection test
  sql-graph-cli test --host localhost --username user --password pass --database mydb

  # Detailed connection test with security validation
  sql-graph-cli test --host remote-db.com --port 3306 --username user --password pass --database prod --detailed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(testOptions{
				Host:              host,
				Port:              port,
				Username:          username,
				Password:          password,
				Database:          database,
				ConnectionTimeout: connectionTimeout,
				Detailed:         detailed,
			})
		},
	}

	// Database connection flags
	cmd.Flags().StringVar(&host, "host", "localhost", "MySQL database host")
	cmd.Flags().IntVar(&port, "port", 3306, "MySQL database port")
	cmd.Flags().StringVar(&username, "username", "", "MySQL username")
	cmd.Flags().StringVar(&password, "password", "", "MySQL password")
	cmd.Flags().StringVar(&database, "database", "", "MySQL database name")

	// Test settings
	cmd.Flags().IntVar(&connectionTimeout, "connection-timeout", 10, "Connection timeout in seconds")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Perform detailed security validation")

	// Required flags
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	cmd.MarkFlagRequired("database")

	return cmd
}

type testOptions struct {
	Host              string
	Port              int
	Username          string
	Password          string
	Database          string
	ConnectionTimeout int
	Detailed         bool
}

func runTest(opts testOptions) error {
	fmt.Println("SQL Graph Visualizer - Connection Test")
	fmt.Println("======================================")

	// Build minimal configuration for testing
	config := &models.MySQLConfig{
		Host:     opts.Host,
		Port:     opts.Port,
		Username: opts.Username,
		Password: opts.Password,
		Database: opts.Database,
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

	// Initialize services
	mysqlRepo := mysql.NewMySQLRepository(nil)
	directDBService := services.NewDirectDatabaseService(mysqlRepo, config)

	ctx := context.Background()

	// Show what we're testing
	fmt.Printf("Target: %s@%s:%d/%s\n", opts.Username, opts.Host, opts.Port, opts.Database)
	fmt.Printf("Timeout: %d seconds\n", opts.ConnectionTimeout)
	if opts.Detailed {
		fmt.Printf("Detailed validation: enabled\n")
	}

	fmt.Println("\nRunning connection test...")
	startTime := time.Now()

	if opts.Detailed {
		// Run full analysis for detailed validation
		result, err := directDBService.ConnectAndAnalyze(ctx)
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
		fmt.Printf("   Database: %s\n", result.DatabaseInfo.Database)
		fmt.Printf("   Server Version: %s\n", result.DatabaseInfo.Version)
		fmt.Printf("   User: %s\n", result.DatabaseInfo.User)
		fmt.Printf("   Security Level: %s\n", result.SecurityValidation.SecurityLevel)

		if result.Summary != nil {
			fmt.Printf("   Tables Found: %d\n", result.Summary.TotalTables)
			
			if len(result.Summary.Warnings) > 0 {
				fmt.Printf("   Warnings: %d\n", len(result.Summary.Warnings))
				for _, warning := range result.Summary.Warnings {
					fmt.Printf("      • %s\n", warning)
				}
			}
		}

		// Show security validation details
		if result.SecurityValidation != nil && len(result.SecurityValidation.Validations) > 0 {
			fmt.Println("\nSecurity Validation:")
			for checkName, validation := range result.SecurityValidation.Validations {
				status := "PASS"
				if !validation.Passed {
					status = "FAIL"
				}
				fmt.Printf("   [%s] %s: %s\n", status, checkName, validation.Message)
			}
		}

		if len(result.SecurityValidation.Recommendations) > 0 {
			fmt.Println("\nSecurity Recommendations:")
			for _, rec := range result.SecurityValidation.Recommendations {
				fmt.Printf("   • %s\n", rec)
			}
		}

	} else {
		// Quick test only
		testResult, err := directDBService.TestConnection(ctx)
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
