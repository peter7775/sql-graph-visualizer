/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"sql-graph-visualizer/internal/application/bootstrap"
)

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version and build information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("sql-graph-visualizer %s\n", bootstrap.Version)
			fmt.Printf("  Go:       %s\n", runtime.Version())
			fmt.Printf("  OS/Arch:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
	return cmd
}
