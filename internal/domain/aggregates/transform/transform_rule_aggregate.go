package transform

import (
	"github.com/peter7775/alevisualizer/internal/domain/entities"
	"github.com/peter7775/alevisualizer/internal/domain/valueobjects/transform"
	"fmt"
)

type TransformRuleAggregate struct {
	entities.BaseEntity
	Rule transform.TransformRule
}

func (t *TransformRuleAggregate) ApplyRules(data []map[string]interface{}) []interface{} {
	var results []interface{}
	for _, record := range data {
		result, err := t.ApplyRule(record)
		if err == nil && result != nil {
			results = append(results, result)
		}
	}
	return results
}

func (t *TransformRuleAggregate) ApplyRule(data map[string]interface{}) (interface{}, error) {
	switch t.Rule.RuleType {
	case transform.NodeRule:
		return t.transformToNode(data)
	case transform.RelationshipRule:
		return t.transformToRelationship(data)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", t.Rule.RuleType)
	}
}

func (t *TransformRuleAggregate) transformToNode(data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["_type"] = t.Rule.TargetType

	for sourceField, targetField := range t.Rule.FieldMappings {
		if value, exists := data[sourceField]; exists {
			result[targetField] = value
		}
	}

	return result, nil
}

func (t *TransformRuleAggregate) transformToRelationship(data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["_type"] = t.Rule.RelationType
	result["_direction"] = t.Rule.Direction

	// Nastavení source a target node
	result["source"] = map[string]interface{}{
		"type":  t.Rule.SourceNode.Type,
		"key":   data[t.Rule.SourceNode.Key],
		"field": t.Rule.SourceNode.TargetField,
	}

	result["target"] = map[string]interface{}{
		"type":  t.Rule.TargetNode.Type,
		"key":   data[t.Rule.TargetNode.Key],
		"field": t.Rule.TargetNode.TargetField,
	}

	// Přidání vlastností relace
	properties := make(map[string]interface{})
	for sourceField, targetField := range t.Rule.Properties {
		if value, exists := data[sourceField]; exists {
			properties[targetField] = value
		}
	}
	result["properties"] = properties

	return result, nil
}
