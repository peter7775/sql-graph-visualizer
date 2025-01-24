/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
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

func (s *VisualizationService) ExportGraph(ctx context.Context, format string) (interface{}, error) {
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

func (s *VisualizationService) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"nodeTypes":         []string{"Table", "Column", "ForeignKey"},
		"relationshipTypes": []string{"HAS_COLUMN", "REFERENCES"},
	}
}
