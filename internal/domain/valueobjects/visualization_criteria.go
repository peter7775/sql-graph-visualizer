/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */

package valueobjects

// VisualizationFormat represents supported visualization output formats.
type VisualizationFormat string

const (
	// FormatJSON represents JSON output format
	FormatJSON  VisualizationFormat = "json"
	// FormatGraphML represents GraphML output format  
	FormatGraphML VisualizationFormat = "graphml"
)

// VisualizationCriteria represents criteria for graph visualization.
type VisualizationCriteria struct {
	SearchCriteria
	Format VisualizationFormat
	Limit  int
}

// NewVisualizationCriteria creates new visualization criteria.
func NewVisualizationCriteria(format VisualizationFormat, limit int) *VisualizationCriteria {
	return &VisualizationCriteria{
		Format: format,
		Limit:  limit,
	}
}
