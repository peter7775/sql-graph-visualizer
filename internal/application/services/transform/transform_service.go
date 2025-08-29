/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	"mysql-graph-visualizer/internal/domain/aggregates/serialization"
	"mysql-graph-visualizer/internal/domain/valueobjects/transform"

	"github.com/sirupsen/logrus"
)

type TransformService struct {
	mysqlPort ports.MySQLPort
	neo4jPort ports.Neo4jPort
	ruleRepo  ports.TransformRuleRepository
}

func NewTransformService(
	mysqlPort ports.MySQLPort,
	neo4jPort ports.Neo4jPort,
	ruleRepo ports.TransformRuleRepository,
) *TransformService {
	return &TransformService{
		mysqlPort: mysqlPort,
		neo4jPort: neo4jPort,
		ruleRepo:  ruleRepo,
	}
}

func (s *TransformService) TransformAndStore(ctx context.Context) error {
	data, err := s.mysqlPort.FetchData()
	if err != nil {
		return err
	}

	logrus.Infof("Loaded %d records from MySQL", len(data))

	rules, err := s.ruleRepo.GetAllRules(ctx)
	logrus.Infof("Rules: %+v", rules)
	if err != nil {
		return err
	}

	graphAggregate := graph.NewGraphAggregate("")

	convertMapValues := func(item map[string]interface{}) map[string]interface{} {
		result := make(map[string]interface{})
		for k, v := range item {
			switch val := v.(type) {
			case map[string]interface{}:
				if jsonStr, err := json.Marshal(val); err == nil {
					result[k] = string(jsonStr)
				} else {
					result[k] = fmt.Sprintf("%v", val)
				}
			default:
				result[k] = v
			}
		}
		return result
	}

	tableData := make(map[string][]map[string]interface{})
	for _, item := range data {
		if tableName, ok := item["_table"].(string); ok {
			convertedItem := convertMapValues(item)
			tableData[tableName] = append(tableData[tableName], convertedItem)
		}
	}

	nodePHPActionNodes, err := s.neo4jPort.FetchNodes("NodePHPAction")
	if err != nil {
		return fmt.Errorf("error loading NodePHPAction nodes from Neo4j: %v", err)
	}

	phpActionNodes, err := s.neo4jPort.FetchNodes("PHPAction")
	if err != nil {
		return fmt.Errorf("error loading PHPAction nodes from Neo4j: %v", err)
	}

	for _, nodeData := range phpActionNodes {
		if err := graphAggregate.AddNode("PHPAction", nodeData); err != nil {
			return fmt.Errorf("error adding node to GraphAggregate: %v", err)
		}
	}

	for _, rule := range rules {
		if rule.Rule.RuleType == transform.RelationshipRule {
			logrus.Infof("Processing relationship rule: %+v", rule)
			transformedData := rule.ApplyRules(append(nodePHPActionNodes, phpActionNodes...))
			logrus.Infof("Transformed %d records", len(transformedData))
			for _, item := range transformedData {
				if err := s.updateGraph(item, graphAggregate); err != nil {
					return err
				}
			}
		} else if rule.Rule.SourceSQL != "" && rule.Rule.RuleType != transform.RelationshipRule {
			logrus.Infof("Executing SQL query: %s", rule.Rule.SourceSQL)
			items, err := s.mysqlPort.ExecuteQuery(rule.Rule.SourceSQL)
			if err != nil {
				return fmt.Errorf("error executing SQL query: %v", err)
			}
			logrus.Infof("Data returned by SQL query: %+v", items)
			transformedData := rule.ApplyRules(items)
			logrus.Infof("Transformed %d records", len(transformedData))
			for _, item := range transformedData {
				if err := s.updateGraph(item, graphAggregate); err != nil {
					return err
				}
			}
		} else {
			sourceTable := rule.Rule.SourceTable
			logrus.Infof("Applying rule to table: %s", sourceTable)
			items, ok := tableData[sourceTable]
			if !ok {
				items = []map[string]interface{}{}
			}

			// Convert map properties to supported types before transformation
			for i, item := range items {
				items[i] = s.convertMapProperties(item)
			}

			transformedData := rule.ApplyRules(items)
			logrus.Infof("Transformed %d records", len(transformedData))

			for _, item := range transformedData {
				if mapItem, ok := item.(map[string]interface{}); ok {
					mapItem = s.convertMapProperties(mapItem)
					if err := s.updateGraph(mapItem, graphAggregate); err != nil {
						return err
					}
				} else {
					logrus.Warnf("Unexpected data format: %T", item)
				}
			}
		}
	}

	logrus.Infof("Number of nodes to save: %d", len(graphAggregate.GetNodes()))
	logrus.Infof("Saving graph to Neo4j")
	return s.neo4jPort.StoreGraph(graphAggregate)
}

func (s *TransformService) updateGraph(data interface{}, graph *graph.GraphAggregate) error {
	switch transformed := data.(type) {
	case map[string]interface{}:
		if nodeType, ok := transformed["_type"].(string); ok {
			if _, hasSource := transformed["source"]; hasSource {
				logrus.Infof("Adding relationship to graph: %+v", transformed)
				return s.createRelationship(transformed, graph)
			}
			logrus.Infof("Adding node to graph: %+v", transformed)
			if _, hasID := transformed["id"]; !hasID {
				transformed["id"] = serialization.GenerateUniqueID()
			}
			if _, hasName := transformed["name"]; !hasName {
				transformed["name"] = "default_name"
			}
			return s.createNode(nodeType, transformed, graph)
		}
	}
	return fmt.Errorf("invalid transform result format")
}

// Define a maximum length for text properties
const maxTextLength = 10000

func (s *TransformService) createNode(nodeType string, data map[string]interface{}, graph *graph.GraphAggregate) error {
	if _, hasID := data["id"]; !hasID {
		return fmt.Errorf("node data missing required 'id' field")
	}
	if _, hasName := data["name"]; !hasName {
		return fmt.Errorf("node data missing required 'name' field")
	}

	for key, value := range data {
		logrus.Infof("Key: %s, Value: %v, Type: %T", key, value, value)
		switch v := value.(type) {
		case []byte:
			data[key] = string(v)
		case string:
			if len(v) > maxTextLength {
				logrus.Warnf("Truncating long string for key %s to %d characters", key, maxTextLength)
				data[key] = v[:maxTextLength]
			}
		case int64:
			data[key] = fmt.Sprintf("%d", v)
		case int, float64, bool:
			// Primitive types are fine
		case map[string]interface{}:
			logrus.Warnf("Converting map to string for key %s", key)
			data[key] = fmt.Sprintf("%v", v)
		default:
			logrus.Warnf("Unexpected data type for key %s: %T", key, value)
			data[key] = fmt.Sprintf("%v", value)
		}
	}

	logrus.Infof("Final node data for Neo4j: %+v", data)

	delete(data, "_type")
	logrus.Infof("Saving node to graph: type=%s, data=%+v", nodeType, data)
	return graph.AddNode(nodeType, data)
}

// Helper function to check if a string is base64 encoded
func isBase64Encoded(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func (s *TransformService) createRelationship(data map[string]interface{}, graph *graph.GraphAggregate) error {
	relType := data["_type"].(string)
	direction := data["_direction"].(transform.Direction)
	source := data["source"].(map[string]interface{})
	target := data["target"].(map[string]interface{})
	properties := data["properties"].(map[string]interface{})

	logrus.Infof("Saving relationship to graph: type=%s, direction=%s, source=%+v, target=%+v, properties=%+v", relType, direction, source, target, properties)

	return graph.AddRelationship(
		relType,
		direction,
		source["type"].(string),
		source["key"],
		source["field"].(string),
		target["type"].(string),
		target["key"],
		target["field"].(string),
		properties,
	)
}

// Define a helper function to convert map properties to supported types
func (s *TransformService) convertMapProperties(item map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range item {
		switch val := v.(type) {
		case map[string]interface{}:
			// Convert map to JSON string
			if jsonStr, err := json.Marshal(val); err == nil {
				result[k] = string(jsonStr)
			} else {
				result[k] = fmt.Sprintf("%v", val)
			}
		default:
			result[k] = v
		}
	}
	return result
}
