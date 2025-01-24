package visualization

import (
	"github.com/peter7775/alevisualizer/internal/application/ports"
	"github.com/peter7775/alevisualizer/internal/domain/aggregates/graph"
	"github.com/peter7775/alevisualizer/internal/domain/valueobjects"
	"context"
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
