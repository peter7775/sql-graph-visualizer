/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

// Package commands provides the CLI command implementations for sql-graph-visualizer.
package commands

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgFile  string
	logLevel string
	verbose  bool
	quiet    bool
)

// NewRootCmd creates the root command for sql-graph-visualizer.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sql-graph-visualizer",
		Short: "SQL Graph Visualizer — transform SQL databases into Neo4j graphs",
		Long: `SQL Graph Visualizer transforms SQL database structures (MySQL, PostgreSQL) into
Neo4j graph databases with interactive visualization and performance analysis.

Use subcommands to control the application:

  transform   Run a one-shot SQL → Neo4j transformation
  serve       Start web visualization and API servers
  check       Test database connectivity
  analyze     Analyze database schema and generate rules
  config      Manage configuration files
  generate    Generate configuration templates
  version     Show version and build information`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		SilenceUsage: true,
	}

	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to configuration file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose (debug) logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress output except errors")

	// Register subcommands
	rootCmd.AddCommand(NewTransformCmd())
	rootCmd.AddCommand(NewServeCmd())
	rootCmd.AddCommand(NewCheckCmd())
	rootCmd.AddCommand(NewAnalyzeCmd())
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.AddCommand(NewGenerateCmd())
	rootCmd.AddCommand(NewVersionCmd())

	return rootCmd
}

// GetConfigPath returns the config file path from the global flag.
func GetConfigPath() string {
	return cfgFile
}

func initLogging() {
	if quiet {
		logrus.SetLevel(logrus.ErrorLevel)
		return
	}
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
		return
	}

	// Check flag first, then env var
	level := logLevel
	if level == "info" {
		if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
			level = envLevel
		}
	}

	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(parsed)
	}
}
