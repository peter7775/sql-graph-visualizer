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

import (
	"context"
	"mysql-graph-visualizer/internal/domain/aggregates/transform"
)

type TransformRuleRepository interface {
	GetAllRules(ctx context.Context) ([]*transform.RuleAggregate, error)
	SaveRule(ctx context.Context, rule *transform.RuleAggregate) error
	DeleteRule(ctx context.Context, ruleID string) error
	UpdateRulePriority(ctx context.Context, ruleID string, priority int) error
}
