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

type VisualizationFormat string

const (
	FormatJSON  VisualizationFormat = "json"
	FormatBasic VisualizationFormat = "basic"
)

type VisualizationCriteria struct {
	SearchCriteria
	Format VisualizationFormat
	Limit  int
}

func NewVisualizationCriteria(format VisualizationFormat, limit int) *VisualizationCriteria {
	return &VisualizationCriteria{
		Format: format,
		Limit:  limit,
	}
}
