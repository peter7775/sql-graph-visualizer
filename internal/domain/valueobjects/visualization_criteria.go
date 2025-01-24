/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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
