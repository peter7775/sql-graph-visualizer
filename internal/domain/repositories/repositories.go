package repositories

import (
	"github.com/peter7775/alevisualizer/internal/domain/aggregates/graph"
	"github.com/peter7775/alevisualizer/internal/domain/aggregates/transform"
	"github.com/peter7775/alevisualizer/internal/domain/models"
	"github.com/peter7775/alevisualizer/internal/domain/valueobjects"
	"context"
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
	FetchData() ([]map[string]interface{}, error)
	Close() error
	// Další metody podle potřeby
}

type Neo4jRepository interface {
	StoreGraph(graph *models.Graph) error
	SearchNodes(query string, params map[string]interface{}) ([]models.SearchResult, error)
	ExportGraph(query string) (interface{}, error)
	Close() error
	// Další metody podle potřeby
}

type TransformRuleRepository interface {
	Repository
	GetAllRules(ctx context.Context) ([]*transform.TransformRuleAggregate, error)
	SaveRule(ctx context.Context, rule *transform.TransformRuleAggregate) error
	DeleteRule(ctx context.Context, ruleID string) error
	UpdateRulePriority(ctx context.Context, ruleID string, priority int) error
}
