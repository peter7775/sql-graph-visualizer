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

// Package visualization provides graph visualization services.
package visualization

import (
	"context"
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/valueobjects"
)

// VisualizationService provides graph visualization capabilities.
//nolint:revive // VisualizationService is descriptive and follows project conventions
type VisualizationService struct {
	neo4jPort ports.Neo4jPort
}

// NewVisualizationService creates a new visualization service instance.
func NewVisualizationService(neo4jPort ports.Neo4jPort) *VisualizationService {
	return &VisualizationService{
		neo4jPort: neo4jPort,
	}
}

// GetGraphData retrieves graph data based on search criteria.
func (s *VisualizationService) GetGraphData(ctx context.Context, criteria valueobjects.SearchCriteria) ([]*graph.GraphAggregate, error) {
	return s.neo4jPort.SearchNodes(criteria.ToString())
}

// ExportGraph exports graph data in the specified format.
func (s *VisualizationService) ExportGraph(ctx context.Context, format string) (any, error) {
	query := s.buildExportQuery(format)
	return s.neo4jPort.ExportGraph(query)
}

func (s *VisualizationService) buildExportQuery(format string) string {
	switch format {
	case "json":
		return `
			MATCH (n)-[r]->(m)
			RETURN {
				nodes: collect(distinct {
					id: id(n),
					labels: labels(n),
					properties: properties(n)
				}),
				relationships: collect({
					id: id(r),
					type: type(r),
					properties: properties(r),
					source: id(n),
					target: id(m)
				})
			} as graph
		`
	default:
		return `
			MATCH (n)-[r]->(m)
			RETURN n, r, m
			LIMIT 100
		`
	}
}

// GetConfig returns visualization configuration options.
func (s *VisualizationService) GetConfig() map[string]any {
	return map[string]any{
		"nodeTypes":         []string{"Table", "Column", "ForeignKey"},
		"relationshipTypes": []string{"HAS_COLUMN", "REFERENCES"},
	}
}
