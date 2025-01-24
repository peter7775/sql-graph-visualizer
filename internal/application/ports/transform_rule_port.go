/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ports

import (
	"context"
	"mysql-graph-visualizer/internal/domain/aggregates/transform"
)

type TransformRuleRepository interface {
	GetAllRules(ctx context.Context) ([]*transform.TransformRuleAggregate, error)
	SaveRule(ctx context.Context, rule *transform.TransformRuleAggregate) error
	DeleteRule(ctx context.Context, ruleID string) error
	UpdateRulePriority(ctx context.Context, ruleID string, priority int) error
}
