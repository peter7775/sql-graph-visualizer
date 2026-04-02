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

package transform

// RuleParameters represents parameters for transformation rules.
type RuleParameters struct {
	conditions map[string]any
	options    map[string]any
}

// NewRuleParameters creates a new RuleParameters instance.
func NewRuleParameters(conditions map[string]any, options map[string]any) RuleParameters {
	return RuleParameters{
		conditions: conditions,
		options:    options,
	}
}

// GetCondition returns a condition value by key.
func (rp RuleParameters) GetCondition(key string) any {
	return rp.conditions[key]
}

// GetOption returns an option value by key.
func (rp RuleParameters) GetOption(key string) any {
	return rp.options[key]
}
