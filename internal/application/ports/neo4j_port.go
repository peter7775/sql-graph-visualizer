package ports

import "github.com/peter7775/alevisualizer/internal/domain/aggregates/graph"

type Neo4jPort interface {
	StoreGraph(graph *graph.GraphAggregate) error
	SearchNodes(criteria string) ([]*graph.GraphAggregate, error)
	ExportGraph(query string) (interface{}, error)
	Close() error
}
