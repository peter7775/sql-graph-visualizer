/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ports

import "mysql-graph-visualizer/internal/domain/aggregates/graph"

type Neo4jPort interface {
	StoreGraph(graph *graph.GraphAggregate) error
	SearchNodes(criteria string) ([]*graph.GraphAggregate, error)
	ExportGraph(query string) (any, error)
	FetchNodes(nodeType string) ([]map[string]any, error)
	Close() error
}
