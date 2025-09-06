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

import (
	"fmt"
	"sql-graph-visualizer/internal/domain/entities"
	"sql-graph-visualizer/internal/domain/valueobjects/transform"

	"github.com/sirupsen/logrus"
)

type RuleAggregate struct {
	entities.BaseEntity
	Rule        transform.TransformRule
	Name        string
	Description string
	Priority    int
	IsActive    bool
	Conditions  map[string]any
	Actions     map[string]any
}

type NodeMapping struct {
	Type        string
	Key         string
	TargetField string
}

func (t *RuleAggregate) ApplyRules(data []map[string]any) []any {
	var results []any
	for _, record := range data {
		logrus.Infof("Applying rule to record: %+v", record)
		logrus.Infof("Processing record: %+v", record)
		logrus.Infof("SourceNode: %+v", t.Rule.SourceNode)
		logrus.Infof("TargetNode: %+v", t.Rule.TargetNode)
		result, err := t.ApplyRule(record)
		if err == nil && result != nil {
			results = append(results, result)
		}
	}
	return results
}

func (t *RuleAggregate) ApplyRule(data map[string]any) (any, error) {
	logrus.Infof("Applying rule: %+v", t.Rule)
	logrus.Infof("Current FieldMappings: %+v", t.Rule.FieldMappings)
	logrus.Infof("Checking FieldMappings before transformation: %+v", t.Rule.FieldMappings)
	logrus.Infof("FieldMappings: %+v", t.Rule.FieldMappings)
	logrus.Infof("Checking SourceNode: %+v", t.Rule.SourceNode)
	logrus.Infof("Checking TargetNode: %+v", t.Rule.TargetNode)
	switch t.Rule.RuleType {
	case transform.NodeRule:
		return t.transformToNode(data)
	case transform.RelationshipRule:
		return t.transformToRelationship(data)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", t.Rule.RuleType)
	}
}

func (t *RuleAggregate) transformToNode(data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	result["_type"] = t.Rule.TargetType

	logrus.Infof("FieldMappings: %+v", t.Rule.FieldMappings)

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

func (t *RuleAggregate) transformToRelationship(data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	result["_type"] = t.Rule.RelationType
	result["_direction"] = t.Rule.Direction

	result["source"] = map[string]any{
		"type":  t.Rule.SourceNode.Type,
		"key":   data[t.Rule.SourceNode.Key],
		"field": t.Rule.SourceNode.TargetField,
	}

	result["target"] = map[string]any{
		"type":  t.Rule.TargetNode.Type,
		"key":   data[t.Rule.TargetNode.Key],
		"field": t.Rule.TargetNode.TargetField,
	}

	properties := make(map[string]any)
	for sourceField, targetField := range t.Rule.Properties {
		if value, exists := data[sourceField]; exists {
			properties[targetField] = value
		}
	}
	result["properties"] = properties

	return result, nil
}
