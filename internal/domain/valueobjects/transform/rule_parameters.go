/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

type RuleParameters struct {
	conditions map[string]any
	options    map[string]any
}

func NewRuleParameters(conditions map[string]any, options map[string]any) RuleParameters {
	return RuleParameters{
		conditions: conditions,
		options:    options,
	}
}

func (rp RuleParameters) GetCondition(key string) any {
	return rp.conditions[key]
}

func (rp RuleParameters) GetOption(key string) any {
	return rp.options[key]
}
