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

package repositories

import (
	"context"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/aggregates/transform"
	"sql-graph-visualizer/internal/domain/valueobjects"
)

type Transaction interface {
	Commit() error
	Rollback() error
}

type Repository interface {
	WithTransaction(ctx context.Context, fn func(tx Transaction) error) error
}

type GraphRepository interface {
	Repository
	Save(ctx context.Context, graph *graph.GraphAggregate) error
	FindById(ctx context.Context, id string) (*graph.GraphAggregate, error)
	FindByCriteria(ctx context.Context, criteria valueobjects.SearchCriteria) ([]*graph.GraphAggregate, error)
}

type MySQLRepository interface {
	FetchData() ([]map[string]any, error)
	Close() error
}

type Neo4jRepository interface {
	StoreGraph(graph *graph.GraphAggregate) error
	SearchNodes(criteria string) ([]*graph.GraphAggregate, error)
	ExportGraph(query string) (any, error)
	FetchNodes(nodeType string) ([]map[string]any, error)
	Close() error
}

type TransformRuleRepository interface {
	Repository
	GetAllRules(ctx context.Context) ([]*transform.RuleAggregate, error)
	SaveRule(ctx context.Context, rule *transform.RuleAggregate) error
	DeleteRule(ctx context.Context, ruleID string) error
	UpdateRulePriority(ctx context.Context, ruleID string, priority int) error
}
