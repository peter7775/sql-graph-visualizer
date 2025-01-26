/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

import (
	"fmt"
	"mysql-graph-visualizer/internal/domain/entities"
	"mysql-graph-visualizer/internal/domain/valueobjects/transform"

	"github.com/sirupsen/logrus"
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

type NodeMapping struct {
	Type        string
	Key         string
	TargetField string
}

func (t *TransformRuleAggregate) ApplyRules(data []map[string]interface{}) []interface{} {
	var results []interface{}
	for _, record := range data {
		logrus.Infof("Aplikuji pravidlo na záznam: %+v", record)
		logrus.Infof("Zpracovávám záznam: %+v", record)
		logrus.Infof("SourceNode: %+v", t.Rule.SourceNode)
		logrus.Infof("TargetNode: %+v", t.Rule.TargetNode)
		result, err := t.ApplyRule(record)
		if err == nil && result != nil {
			results = append(results, result)
		}
	}
	return results
}

func (t *TransformRuleAggregate) ApplyRule(data map[string]interface{}) (interface{}, error) {
	logrus.Infof("Aplikuji pravidlo: %+v", t.Rule)
	logrus.Infof("Aktuální FieldMappings: %+v", t.Rule.FieldMappings)
	logrus.Infof("Kontrola FieldMappings před transformací: %+v", t.Rule.FieldMappings)
	logrus.Infof("FieldMappings: %+v", t.Rule.FieldMappings)
	logrus.Infof("Kontrola SourceNode: %+v", t.Rule.SourceNode)
	logrus.Infof("Kontrola TargetNode: %+v", t.Rule.TargetNode)
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

	logrus.Infof("FieldMappings: %+v", t.Rule.FieldMappings)
	// log.Printf("Zpracovávám data: %+v", data)

	for sourceField, targetField := range t.Rule.FieldMappings {
		if value, exists := data[sourceField]; exists {
			logrus.Infof("Mapping field %s to %s with value %v", sourceField, targetField, value)
			result[targetField] = value
		} else {
			logrus.Warnf("Field %s not found in data", sourceField)
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
