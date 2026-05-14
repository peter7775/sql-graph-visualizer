/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sql-graph-visualizer/internal/application/bootstrap"
)

// NewServeCmd creates the serve command.
func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start web visualization and API servers",
		Long: `Runs the full application: transforms data, then starts the web visualization
server, GraphQL playground, and REST API. The process stays running until
interrupted (Ctrl+C).

This is the default interactive mode equivalent to the legacy entry point.`,
		Example: `  # Start with default config
  sql-graph-visualizer serve

  # Start with specific config and debug logging
  sql-graph-visualizer serve -c config/config-chinook.yml -v

  # Start in quiet mode
  sql-graph-visualizer serve -c config/config.yml -q`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe()
		},
	}

	return cmd
}

func runServe() error {
	ctx := context.Background()

	res, err := bootstrap.InitAll(ctx, GetConfigPath())
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	defer res.Cleanup()

	// Run transformation
	if err := res.RunTransform(ctx); err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// Start servers
	servers, err := res.StartServers()
	if err != nil {
		return fmt.Errorf("failed to start servers: %w", err)
	}

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	logrus.Info("Application is running. Press Ctrl+C to stop.")
	<-quit

	logrus.Info("Shutting down servers...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	servers.Shutdown(shutdownCtx)

	logrus.Info("Servers successfully shut down")
	return nil
}
