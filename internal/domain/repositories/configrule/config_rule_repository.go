package configrule

import (
	"context"
	"fmt"
	"log"
	"mysql-graph-visualizer/internal/config"
	"mysql-graph-visualizer/internal/domain/aggregates/transform"
	transformR "mysql-graph-visualizer/internal/domain/valueobjects/transform"
)

type RuleRepository struct {
	rules []*transform.TransformRuleAggregate
}

func NewRuleRepository() *RuleRepository {
	return &RuleRepository{rules: []*transform.TransformRuleAggregate{}}
}

func (r *RuleRepository) GetAllRules(ctx context.Context) ([]*transform.TransformRuleAggregate, error) {
	if len(r.rules) == 0 {
		loadedRules, err := r.LoadRulesFromConfig("config/config.yml")
		if err != nil {
			return nil, err
		}
		r.rules = loadedRules
	}
	return r.rules, nil
}

func (r *RuleRepository) SaveRule(ctx context.Context, rule *transform.TransformRuleAggregate) error {
	r.rules = append(r.rules, rule)
	return nil
}

func (r *RuleRepository) DeleteRule(ctx context.Context, ruleID string) error {
	for i, rule := range r.rules {
		if rule.ID == ruleID {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule with ID %s not found", ruleID)
}

func (r *RuleRepository) UpdateRulePriority(ctx context.Context, ruleID string, priority int) error {
	for _, rule := range r.rules {
		if rule.ID == ruleID {
			rule.Priority = priority
			return nil
		}
	}
	return fmt.Errorf("rule with ID %s not found", ruleID)
}

func (r *RuleRepository) LoadRulesFromConfig(filePath string) ([]*transform.TransformRuleAggregate, error) {
	log.Printf("Načítám pravidla z %s", filePath)

	config, err := config.LoadConfig(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not load config: %v", err)
	}

	log.Printf("Načteno %d transform rules z konfigurace", len(config.TransformRules))

	var rules []*transform.TransformRuleAggregate
	for _, rule := range config.TransformRules {
		log.Printf("Přiřazuji FieldMappings: %+v", rule.FieldMappings)
		log.Printf("Načtené pravidlo: %+v", rule)
		if rule.SourceNode.Type == "" || rule.TargetNode.Type == "" {
			log.Printf("Chyba: SourceNode nebo TargetNode je prázdný pro pravidlo: %+v", rule)
		}
		log.Printf("SourceNode: %+v, TargetNode: %+v", rule.SourceNode, rule.TargetNode)

		transformRule := transformR.TransformRule{
			Name:          rule.Name,
			SourceTable:   rule.Source.Value, // Assuming Source.Value is the correct field
			RuleType:      transformR.RuleType(rule.RuleType),
			TargetType:    rule.TargetType,
			Direction:     transformR.ParseDirection(rule.Direction),
			FieldMappings: rule.FieldMappings,
			RelationType:  rule.RelationType,
			SourceNode:    &transformR.NodeMapping{Type: rule.SourceNode.Type, Key: rule.SourceNode.Key, TargetField: rule.SourceNode.TargetField},
			TargetNode:    &transformR.NodeMapping{Type: rule.TargetNode.Type, Key: rule.TargetNode.Key, TargetField: rule.TargetNode.TargetField},
			Properties:    rule.Properties,
			Priority:      rule.Priority,
		}

		rules = append(rules, &transform.TransformRuleAggregate{Rule: transformRule})
	}

	log.Printf("Načteno %d pravidel pro transformaci", len(rules))

	return rules, nil
}