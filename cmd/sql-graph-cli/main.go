/*
 * SQL Graph Visualizer - CLI Application for Direct Database Connection
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package main

import (
	"os"

	"github.com/spf13/cobra"
	"sql-graph-visualizer/cmd/sql-graph-cli/commands"
)

var rootCmd = &cobra.Command{
	Use:   "sql-graph-cli",
	Short: "SQL Graph Visualizer - Direct Database Connection CLI",
	Long: `SQL Graph Visualizer CLI provides tools for connecting to existing SQL databases (MySQL, PostgreSQL),
analyzing their schema, and automatically generating Neo4j transformation rules.

This tool supports multiple database engines and provides seamless graph transformation capabilities.`,
	Version: "1.0.0",
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress output except errors")

	// Add commands
	rootCmd.AddCommand(commands.NewAnalyzeCmd())
	rootCmd.AddCommand(commands.NewTestCmd())
	rootCmd.AddCommand(commands.NewGenerateCmd())
	rootCmd.AddCommand(commands.NewConfigCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
