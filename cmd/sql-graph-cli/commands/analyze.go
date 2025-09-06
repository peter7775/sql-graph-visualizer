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
	"sql-graph-visualizer/internal/infrastructure/persistence/mysql"

	"github.com/spf13/cobra"
)

// NewAnalyzeCmd creates the analyze command
func NewAnalyzeCmd() *cobra.Command {
	var (
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
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze existing MySQL database schema and generate transformation rules",
		Long: `Connects to an existing MySQL database, analyzes its schema structure,
identifies relationships and graph patterns, and generates Neo4j transformation rules automatically.

This command demonstrates the core functionality of Issue #10 - Direct Database Connection Implementation.`,
		Example: `  # Analyze local database
  sql-graph-cli analyze --host localhost --port 3306 --username user --password pass --database mydb

  # Analyze with table filtering
  sql-graph-cli analyze --host localhost --database mydb --whitelist "users,orders,products"

  # Save analysis to JSON file
  sql-graph-cli analyze --host localhost --database mydb --output analysis.json

  # Dry run without generating rules
  sql-graph-cli analyze --host localhost --database mydb --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(cmd, analyzeOptions{
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
			})
		},
	}

	// Database connection flags
	cmd.Flags().StringVar(&host, "host", "localhost", "MySQL database host")
	cmd.Flags().IntVar(&port, "port", 3306, "MySQL database port")
	cmd.Flags().StringVar(&username, "username", "", "MySQL username")
	cmd.Flags().StringVar(&password, "password", "", "MySQL password")
	cmd.Flags().StringVar(&database, "database", "", "MySQL database name")

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

	// Required flags
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	cmd.MarkFlagRequired("database")

	return cmd
}

type analyzeOptions struct {
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
}

func runAnalyze(cmd *cobra.Command, opts analyzeOptions) error {
	fmt.Println("🔍 SQL Graph Visualizer - Database Analysis")
	fmt.Println("=============================================")

	// Build configuration
	config := &models.MySQLConfig{
		Host:     opts.Host,
		Port:     opts.Port,
		Username: opts.Username,
		Password: opts.Password,
		Database: opts.Database,
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

	// Initialize services
	mysqlRepo := mysql.NewMySQLRepository(nil)
	directDBService := services.NewDirectDatabaseService(mysqlRepo, config)

	// Validate configuration
	fmt.Printf("🔧 Validating configuration...\n")
	if err := directDBService.ValidateConfiguration(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	ctx := context.Background()

	// Show connection info
	fmt.Printf("📡 Connecting to %s@%s:%d/%s\n", opts.Username, opts.Host, opts.Port, opts.Database)
	if len(opts.TableWhitelist) > 0 {
		fmt.Printf("🎯 Table whitelist: %s\n", strings.Join(opts.TableWhitelist, ", "))
	}
	if len(opts.TableBlacklist) > 0 {
		fmt.Printf("⚫ Table blacklist: %s\n", strings.Join(opts.TableBlacklist, ", "))
	}
	if opts.RowLimit > 0 {
		fmt.Printf("📊 Row limit: %d per table\n", opts.RowLimit)
	}
	if opts.DryRun {
		fmt.Printf("🧪 Dry run mode: analysis only, no rule generation\n")
	}

	// Start analysis
	fmt.Printf("\n🔍 Starting database analysis...\n")
	startTime := time.Now()
	
	result, err := directDBService.ConnectAndAnalyze(ctx)
	if err != nil {
		return fmt.Errorf("database analysis failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("analysis failed: %s", result.ErrorMessage)
	}

	duration := time.Since(startTime)
	fmt.Printf("✅ Analysis completed in %v\n", duration)

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

func outputSummary(result *models.DirectDatabaseAnalysisResult, outputFile string) error {
	var output strings.Builder
	
	// Header
	output.WriteString("\n" + strings.Repeat("=", 60) + "\n")
	output.WriteString("📊 DATABASE ANALYSIS RESULTS\n")
	output.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Connection info
	output.WriteString("🔗 CONNECTION INFORMATION:\n")
	output.WriteString(fmt.Sprintf("   Database: %s@%s:%d/%s\n", 
		result.DatabaseInfo.User, result.DatabaseInfo.Host, 
		result.DatabaseInfo.Port, result.DatabaseInfo.Database))
	output.WriteString(fmt.Sprintf("   Server Version: %s\n", result.DatabaseInfo.Version))
	output.WriteString(fmt.Sprintf("   Processing Time: %v\n", result.ProcessingDuration))
	output.WriteString(fmt.Sprintf("   Security Level: %s\n", result.SecurityValidation.SecurityLevel))

	// Summary statistics
	if result.Summary != nil {
		summary := result.Summary
		output.WriteString("\n📈 ANALYSIS SUMMARY:\n")
		output.WriteString(fmt.Sprintf("   Tables Analyzed: %d\n", summary.TotalTables))
		output.WriteString(fmt.Sprintf("   Generated Rules: %d (%d nodes, %d relationships)\n", 
			summary.TotalRules, summary.NodeRules, summary.RelationshipRules))
		output.WriteString(fmt.Sprintf("   Graph Patterns: %d\n", summary.TotalPatterns))

		if len(summary.Warnings) > 0 {
			output.WriteString(fmt.Sprintf("   ⚠️  Warnings: %d\n", len(summary.Warnings)))
		}
		
		if len(summary.Recommendations) > 0 {
			output.WriteString(fmt.Sprintf("   💡 Recommendations: %d\n", len(summary.Recommendations)))
		}
	}

	// Dataset information
	if result.SchemaAnalysis != nil && result.SchemaAnalysis.DatasetInfo != nil {
		dataset := result.SchemaAnalysis.DatasetInfo
		output.WriteString("\n📊 DATASET INFORMATION:\n")
		output.WriteString(fmt.Sprintf("   Total Rows: %d\n", dataset.TotalRows))
		output.WriteString(fmt.Sprintf("   Estimated Size: %.2f MB\n", dataset.EstimatedSizeMB))
	}

	// Tables with their types
	if result.SchemaAnalysis != nil && len(result.SchemaAnalysis.Tables) > 0 {
		output.WriteString("\n📋 DISCOVERED TABLES:\n")
		for _, table := range result.SchemaAnalysis.Tables {
			graphType := "NODE"
			if table.GraphType == "RELATIONSHIP" {
				graphType = "RELATIONSHIP"
			}
			output.WriteString(fmt.Sprintf("   %-20s (%s) - %d rows, %d columns\n", 
				table.Name, graphType, table.EstimatedRows, len(table.Columns)))
		}
	}

	// Graph patterns
	if result.SchemaAnalysis != nil && len(result.SchemaAnalysis.GraphPatterns) > 0 {
		output.WriteString("\n🕸️  IDENTIFIED GRAPH PATTERNS:\n")
		for _, pattern := range result.SchemaAnalysis.GraphPatterns {
			output.WriteString(fmt.Sprintf("   %s: %s (%.1f%% confidence)\n", 
				pattern.PatternType, pattern.Description, pattern.Confidence*100))
		}
	}

	// Sample generated rules
	if result.SchemaAnalysis != nil && len(result.SchemaAnalysis.GeneratedRules) > 0 {
		output.WriteString("\n🔄 GENERATED TRANSFORMATION RULES:\n")
		nodeCount := 0
		relCount := 0
		
		for _, rule := range result.SchemaAnalysis.GeneratedRules {
			if rule.RuleType == "NODE_CREATION" {
				nodeCount++
			} else if rule.RuleType == "RELATIONSHIP_CREATION" {
				relCount++
			}
		}
		
		output.WriteString(fmt.Sprintf("   Node Creation Rules: %d\n", nodeCount))
		output.WriteString(fmt.Sprintf("   Relationship Rules: %d\n", relCount))
		
		// Show first few rules as examples
		output.WriteString("\n   Sample Rules:\n")
		count := 0
		for _, rule := range result.SchemaAnalysis.GeneratedRules {
			if count >= 3 {
				break
			}
			output.WriteString(fmt.Sprintf("   • %s (%s): %s\n", 
				rule.RuleID, rule.RuleType, rule.Description))
			count++
		}
		if len(result.SchemaAnalysis.GeneratedRules) > 3 {
			output.WriteString(fmt.Sprintf("   ... and %d more rules\n", 
				len(result.SchemaAnalysis.GeneratedRules)-3))
		}
	}

	// Warnings and recommendations
	if result.Summary != nil {
		if len(result.Summary.Warnings) > 0 {
			output.WriteString("\n⚠️  WARNINGS:\n")
			for _, warning := range result.Summary.Warnings {
				output.WriteString(fmt.Sprintf("   • %s\n", warning))
			}
		}

		if len(result.Summary.Recommendations) > 0 {
			output.WriteString("\n💡 RECOMMENDATIONS:\n")
			for _, rec := range result.Summary.Recommendations {
				output.WriteString(fmt.Sprintf("   • %s\n", rec))
			}
		}
	}

	output.WriteString("\n" + strings.Repeat("=", 60) + "\n")

	return writeOutput(output.String(), outputFile)
}

func outputJSON(result *models.DirectDatabaseAnalysisResult, outputFile string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return writeOutput(string(data), outputFile)
}

func outputYAML(result *models.DirectDatabaseAnalysisResult, outputFile string) error {
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
