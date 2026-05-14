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

	"github.com/spf13/cobra"

	"sql-graph-visualizer/internal/application/bootstrap"
)

// NewTransformCmd creates the transform command.
func NewTransformCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transform",
		Short: "Run a one-shot SQL → Neo4j transformation",
		Long: `Connects to the configured SQL database, transforms the data according to
the transformation rules in the configuration file, stores the result in Neo4j,
and exits.

This is the non-interactive mode — no web server is started.`,
		Example: `  # Transform using default config
  sql-graph-visualizer transform

  # Transform with specific config
  sql-graph-visualizer transform --config config/config-chinook.yml

  # Transform with debug logging
  sql-graph-visualizer transform -c config/config.yml --log-level debug`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTransform()
		},
	}

	return cmd
}

func runTransform() error {
	ctx := context.Background()

	res, err := bootstrap.InitAll(ctx, GetConfigPath())
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	defer res.Cleanup()

	if err := res.RunTransform(ctx); err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	fmt.Println("Transformation completed successfully.")
	return nil
}
