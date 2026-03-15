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

package configrule

import (
	"context"
	"fmt"
	transformAgg "sql-graph-visualizer/internal/domain/aggregates/transform"
	"sql-graph-visualizer/internal/domain/repositories/config"
	transformVal "sql-graph-visualizer/internal/domain/valueobjects/transform"

	"github.com/sirupsen/logrus"
)

// RuleRepository manages transformation rule configuration.
type RuleRepository struct {
	rules []*transformAgg.RuleAggregate
}

// NewRuleRepository creates a new rule repository instance.
func NewRuleRepository() *RuleRepository {
	return &RuleRepository{rules: []*transformAgg.RuleAggregate{}}
}

// GetAllRules retrieves all transformation rules from configuration.
func (r *RuleRepository) GetAllRules(ctx context.Context) ([]*transformAgg.RuleAggregate, error) {
	logrus.Infof("GetAllRules called - current rules count: %d", len(r.rules))
	if len(r.rules) == 0 {
		logrus.Infof("Loading rules from config file...")
		loadedRules, err := r.LoadRulesFromConfig()
		if err != nil {
			logrus.Errorf("Failed to load rules: %v", err)
			return nil, err
		}
		r.rules = loadedRules
		logrus.Infof("Successfully loaded %d rules", len(r.rules))
	}
	logrus.Infof("Returning %d rules from GetAllRules", len(r.rules))
	return r.rules, nil
}

// SaveRule saves a transformation rule to configuration.
func (r *RuleRepository) SaveRule(ctx context.Context, rule *transformAgg.RuleAggregate) error {
	r.rules = append(r.rules, rule)
	return nil
}

// DeleteRule removes a transformation rule from configuration.
func (r *RuleRepository) DeleteRule(ctx context.Context, ruleID string) error {
	for i, rule := range r.rules {
		if rule.ID == ruleID {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule with ID %s not found", ruleID)
}

// UpdateRulePriority updates the priority of a transformation rule.
func (r *RuleRepository) UpdateRulePriority(ctx context.Context, ruleID string, priority int) error {
	for _, rule := range r.rules {
		if rule.ID == ruleID {
			rule.Priority = priority
			return nil
		}
	}
	return fmt.Errorf("rule with ID %s not found", ruleID)
}

// LoadRulesFromConfig loads transformation rules from the configuration file.
func (r *RuleRepository) LoadRulesFromConfig() ([]*transformAgg.RuleAggregate, error) {
	logrus.Infof("Loading rules from configuration file")

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("could not load config: %v", err)
	}

	logrus.Infof("Loaded TransformRules from config: %+v", cfg.TransformRules)
	logrus.Infof("Number of TransformRules: %d", len(cfg.TransformRules))

	var rules []*transformAgg.RuleAggregate
	for _, configRule := range cfg.TransformRules {
		logrus.Infof("Processing rule: %+v", configRule)

		transformRule := transformVal.TransformRule{
			Name:          configRule.Name,
			RuleType:      transformVal.RuleType(configRule.RuleType),
			TargetType:    configRule.TargetType,
			FieldMappings: configRule.FieldMappings,
			RelationType:  configRule.RelationType,
			Direction:     transformVal.ParseDirection(string(configRule.Direction)),
			Properties:    configRule.Properties,
		}

		if configRule.Source.Type == "query" {
			transformRule.SourceSQL = configRule.Source.Value
		}

		if configRule.RuleType == "relationship" {
			if configRule.SourceNode.Type != "" {
				transformRule.SourceNode = &transformVal.NodeMapping{
					Type:        configRule.SourceNode.Type,
					Key:         configRule.SourceNode.Key,
					TargetField: configRule.SourceNode.TargetField,
				}
			}

			if configRule.TargetNode.Type != "" {
				transformRule.TargetNode = &transformVal.NodeMapping{
					Type:        configRule.TargetNode.Type,
					Key:         configRule.TargetNode.Key,
					TargetField: configRule.TargetNode.TargetField,
				}
			}
		}

		if configRule.Properties != nil {
			transformRule.Properties = configRule.Properties
		}

		logrus.Infof("Created rule:")
		logrus.Infof("- Name: %s", transformRule.Name)
		logrus.Infof("- Type: %s", transformRule.RuleType)
		logrus.Infof("- Target Type: %s", transformRule.TargetType)
		logrus.Infof("- Field Mappings: %+v", transformRule.FieldMappings)
		logrus.Infof("- Source Node: %+v", transformRule.SourceNode)
		logrus.Infof("- Target Node: %+v", transformRule.TargetNode)
		logrus.Infof("- Properties: %+v", transformRule.Properties)

		rules = append(rules, &transformAgg.RuleAggregate{
			Rule: transformRule,
			Name: transformRule.Name,
		})
	}

	logrus.Infof("Total loaded %d rules", len(rules))
	return rules, nil
}
