package ports

import (
	"github.com/peter7775/alevisualizer/internal/domain/aggregates/transform"
	"context"
)

type TransformRuleRepository interface {
	GetAllRules(ctx context.Context) ([]*transform.TransformRuleAggregate, error)
	SaveRule(ctx context.Context, rule *transform.TransformRuleAggregate) error
	DeleteRule(ctx context.Context, ruleID string) error
	UpdateRulePriority(ctx context.Context, ruleID string, priority int) error
}
