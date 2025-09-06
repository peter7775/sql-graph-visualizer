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


package visualization

import (
	"context"
	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	"mysql-graph-visualizer/internal/domain/valueobjects"
)

type VisualizationService struct {
	neo4jPort ports.Neo4jPort
}

func NewVisualizationService(neo4jPort ports.Neo4jPort) *VisualizationService {
	return &VisualizationService{
		neo4jPort: neo4jPort,
	}
}

func (s *VisualizationService) GetGraphData(ctx context.Context, criteria valueobjects.SearchCriteria) ([]*graph.GraphAggregate, error) {
	return s.neo4jPort.SearchNodes(criteria.ToString())
}

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

func (s *VisualizationService) GetConfig() map[string]any {
	return map[string]any{
		"nodeTypes":         []string{"Table", "Column", "ForeignKey"},
		"relationshipTypes": []string{"HAS_COLUMN", "REFERENCES"},
	}
}
