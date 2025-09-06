/*
 * MySQL Graph Visualizer - CLI Application for Direct Database Connection
 *
 * Copyright (c) 2024
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package main

import (
	"os"

	"mysql-graph-visualizer/cmd/mysql-graph-cli/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mysql-graph-cli",
	Short: "MySQL Graph Visualizer - Direct Database Connection CLI",
	Long: `MySQL Graph Visualizer CLI provides tools for connecting to existing MySQL databases,
analyzing their schema, and automatically generating Neo4j transformation rules.

This tool implements Issue #10 - Direct Database Connection Implementation.`,
	Version: "1.0.0-alpha",
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
