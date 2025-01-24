/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package repositories

import (
	"context"
	"mysql-graph-visualizer/internal/domain/aggregates/transform"
)

type InMemoryRuleRepository struct {
	rules []*transform.TransformRuleAggregate
}

func NewInMemoryRuleRepository() TransformRuleRepository {
	return &InMemoryRuleRepository{
		rules: make([]*transform.TransformRuleAggregate, 0),
	}
}

func (r *InMemoryRuleRepository) GetAllRules(ctx context.Context) ([]*transform.TransformRuleAggregate, error) {
	return r.rules, nil
}

func (r *InMemoryRuleRepository) SaveRule(ctx context.Context, rule *transform.TransformRuleAggregate) error {
	r.rules = append(r.rules, rule)
	return nil
}

func (r *InMemoryRuleRepository) DeleteRule(ctx context.Context, ruleID string) error {
	for i, rule := range r.rules {
		if rule.ID == ruleID {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *InMemoryRuleRepository) UpdateRulePriority(ctx context.Context, ruleID string, priority int) error {
	for _, rule := range r.rules {
		if rule.ID == ruleID {
			rule.Rule.Priority = priority
			return nil
		}
	}
	return nil
}

func (r *InMemoryRuleRepository) WithTransaction(ctx context.Context, fn func(tx Transaction) error) error {
	return fn(nil) // Pro in-memory implementaci nepotřebujeme transakce
}
