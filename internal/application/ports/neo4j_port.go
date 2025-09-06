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


package ports

import "sql-graph-visualizer/internal/domain/aggregates/graph"

type Neo4jPort interface {
	StoreGraph(graph *graph.GraphAggregate) error
	SearchNodes(criteria string) ([]*graph.GraphAggregate, error)
	ExportGraph(query string) (any, error)
	FetchNodes(nodeType string) ([]map[string]any, error)
	Close() error
}
