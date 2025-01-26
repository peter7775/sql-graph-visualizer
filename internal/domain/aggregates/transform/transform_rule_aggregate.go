/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

import (
	"fmt"
	"log"
	"mysql-graph-visualizer/internal/domain/entities"
	"mysql-graph-visualizer/internal/domain/valueobjects/transform"
)

type TransformRuleAggregate struct {
	entities.BaseEntity
	Rule transform.TransformRule
	Name string
	Description string
	Priority int
	IsActive bool
	Conditions map[string]interface{}
	Actions map[string]interface{}
}

func (t *TransformRuleAggregate) ApplyRules(data []map[string]interface{}) []interface{} {
	var results []interface{}
	for _, record := range data {
		log.Printf("Aplikuji pravidlo na záznam: %+v", record)
		result, err := t.ApplyRule(record)
		if err == nil && result != nil {
			results = append(results, result)
		}
	}
	return results
}

func (t *TransformRuleAggregate) ApplyRule(data map[string]interface{}) (interface{}, error) {
	log.Printf("Aplikuji pravidlo: %+v", t.Rule)
	log.Printf("Aktuální FieldMappings: %+v", t.Rule.FieldMappings)
	log.Printf("Kontrola FieldMappings před transformací: %+v", t.Rule.FieldMappings)
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

	log.Printf("FieldMappings: %+v", t.Rule.FieldMappings)
	// log.Printf("Zpracovávám data: %+v", data)

	for sourceField, targetField := range t.Rule.FieldMappings {
		if value, exists := data[sourceField]; exists {
			log.Printf("Mapping field %s to %s with value %v", sourceField, targetField, value)
			result[targetField] = value
		} else {
			log.Printf("Field %s not found in data", sourceField)
		}
	}

	return result, nil
}

func (t *TransformRuleAggregate) transformToRelationship(data map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	result["_type"] = t.Rule.RelationType
	result["_direction"] = t.Rule.Direction

	// Kontrola, zda SourceNode a TargetNode nejsou nil
	if t.Rule.SourceNode == nil || t.Rule.TargetNode == nil {
		return nil, fmt.Errorf("SourceNode nebo TargetNode je nil")
	}

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
