/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"
	"mysql-graph-visualizer/internal/domain/aggregates/serialization"
	transform_agg "mysql-graph-visualizer/internal/domain/aggregates/transform"
	"mysql-graph-visualizer/internal/domain/entities"
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

	convertMapValues := func(item map[string]any) map[string]any {
		result := make(map[string]any)
		for k, v := range item {
			switch val := v.(type) {
			case map[string]any:
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

	tableData := make(map[string][]map[string]any)
	for _, item := range data {
		if tableName, ok := item["_table"].(string); ok {
			convertedItem := convertMapValues(item)
			tableData[tableName] = append(tableData[tableName], convertedItem)
		}
	}

	// First pass: Process all node rules to create nodes
	logrus.Infof("First pass: Creating nodes")
	for _, rule := range rules {
		if rule.Rule.RuleType != transform.NodeRule {
			continue
		}

		logrus.Infof("Processing node rule: %s", rule.Rule.Name)

		var items []map[string]any
		var err error

		if rule.Rule.SourceSQL != "" {
			// Rule has custom SQL query
			logrus.Infof("Executing SQL query: %s", rule.Rule.SourceSQL)
			items, err = s.mysqlPort.ExecuteQuery(rule.Rule.SourceSQL)
			if err != nil {
				return fmt.Errorf("error executing SQL query for rule %s: %v", rule.Rule.Name, err)
			}
		} else {
			// Rule uses table data (legacy approach)
			sourceTable := rule.Rule.SourceTable
			logrus.Infof("Applying rule to table: %s", sourceTable)
			var ok bool
			items, ok = tableData[sourceTable]
			if !ok {
				items = []map[string]any{}
			}
		}

		logrus.Infof("Data returned for node rule %s: %d records", rule.Rule.Name, len(items))

		// Convert map properties to supported types before transformation
		for i, item := range items {
			items[i] = s.convertMapProperties(item)
		}

		// Apply transformation rules
		transformedData := rule.ApplyRules(items)
		logrus.Infof("Transformed %d records for node rule %s", len(transformedData), rule.Rule.Name)

		// Add transformed data to graph
		for _, item := range transformedData {
			if mapItem, ok := item.(map[string]any); ok {
				mapItem = s.convertMapProperties(mapItem)
				if err := s.updateGraph(mapItem, graphAggregate); err != nil {
					logrus.Warnf("Warning updating graph for node rule %s: %v (continuing)", rule.Rule.Name, err)
				}
			} else {
				logrus.Warnf("Unexpected data format for node rule %s: %T", rule.Rule.Name, item)
			}
		}
	}

	// Second pass: Process relationship rules to create relationships
	logrus.Infof("Second pass: Creating relationships")
	for _, rule := range rules {
		if rule.Rule.RuleType != transform.RelationshipRule {
			continue
		}

		logrus.Infof("Processing relationship rule: %s", rule.Rule.Name)

		if rule.Rule.SourceSQL != "" {
			// Rule has custom SQL query - process like before
			logrus.Infof("Executing SQL query for relationship: %s", rule.Rule.SourceSQL)
			items, err := s.mysqlPort.ExecuteQuery(rule.Rule.SourceSQL)
			if err != nil {
				logrus.Warnf("Error executing SQL query for relationship rule %s: %v (continuing)", rule.Rule.Name, err)
				continue
			}

			// Convert map properties to supported types before transformation
			for i, item := range items {
				items[i] = s.convertMapProperties(item)
			}

			// Apply transformation rules
			transformedData := rule.ApplyRules(items)
			logrus.Infof("Transformed %d records for relationship rule %s", len(transformedData), rule.Rule.Name)

			// Add transformed relationships to graph
			for _, item := range transformedData {
				if mapItem, ok := item.(map[string]any); ok {
					if err := s.updateGraph(mapItem, graphAggregate); err != nil {
						logrus.Warnf("Warning updating graph for relationship rule %s: %v (continuing)", rule.Rule.Name, err)
					}
				} else {
					logrus.Warnf("Unexpected data format for relationship rule %s: %T", rule.Rule.Name, item)
				}
			}
		} else {
			// For relationship rules without SQL, create relationships based on existing nodes
			logrus.Infof("Processing relationship rule without SQL: %s", rule.Rule.Name)
			err := s.createRelationshipsFromExistingNodes(rule, graphAggregate)
			if err != nil {
				logrus.Warnf("Error creating relationships for rule %s: %v (continuing)", rule.Rule.Name, err)
			}
		}
	}

	logrus.Infof("Number of nodes to save: %d", len(graphAggregate.GetNodes()))
	logrus.Infof("Saving graph to Neo4j")
	return s.neo4jPort.StoreGraph(graphAggregate)
}

func (s *TransformService) updateGraph(data any, graph *graph.GraphAggregate) error {
	switch transformed := data.(type) {
	case map[string]any:
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

func (s *TransformService) createNode(nodeType string, data map[string]any, graph *graph.GraphAggregate) error {
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
		case map[string]any:
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

func (s *TransformService) createRelationship(data map[string]any, graph *graph.GraphAggregate) error {
	relType, ok := data["_type"].(string)
	if !ok {
		return fmt.Errorf("relationship missing _type field")
	}

	direction, ok := data["_direction"].(transform.Direction)
	if !ok {
		return fmt.Errorf("relationship missing _direction field")
	}

	// Handle source field - might be map or JSON string
	var source map[string]any
	if sourceRaw, exists := data["source"]; exists {
		if sourceMap, ok := sourceRaw.(map[string]any); ok {
			source = sourceMap
		} else if sourceStr, ok := sourceRaw.(string); ok {
			// Try to parse JSON string
			var sourceJSON map[string]any
			if err := json.Unmarshal([]byte(sourceStr), &sourceJSON); err == nil {
				source = sourceJSON
			} else {
				return fmt.Errorf("failed to parse source JSON: %v", err)
			}
		} else {
			return fmt.Errorf("source field has invalid type: %T", sourceRaw)
		}
	} else {
		return fmt.Errorf("relationship missing source field")
	}

	// Handle target field - might be map or JSON string
	var target map[string]any
	if targetRaw, exists := data["target"]; exists {
		if targetMap, ok := targetRaw.(map[string]any); ok {
			target = targetMap
		} else if targetStr, ok := targetRaw.(string); ok {
			// Try to parse JSON string
			var targetJSON map[string]any
			if err := json.Unmarshal([]byte(targetStr), &targetJSON); err == nil {
				target = targetJSON
			} else {
				return fmt.Errorf("failed to parse target JSON: %v", err)
			}
		} else {
			return fmt.Errorf("target field has invalid type: %T", targetRaw)
		}
	} else {
		return fmt.Errorf("relationship missing target field")
	}

	// Handle properties field - might be map or JSON string
	var properties map[string]any
	if propRaw, exists := data["properties"]; exists {
		if propMap, ok := propRaw.(map[string]any); ok {
			properties = propMap
		} else if propStr, ok := propRaw.(string); ok {
			// Try to parse JSON string
			var propJSON map[string]any
			if err := json.Unmarshal([]byte(propStr), &propJSON); err == nil {
				properties = propJSON
			} else {
				return fmt.Errorf("failed to parse properties JSON: %v", err)
			}
		} else {
			return fmt.Errorf("properties field has invalid type: %T", propRaw)
		}
	} else {
		// Properties are optional
		properties = make(map[string]any)
	}

	logrus.Infof("Saving relationship to graph: type=%s, direction=%s, source=%+v, target=%+v, properties=%+v", relType, direction, source, target, properties)

	// Extract required fields from source and target
	sourceType, ok := source["type"].(string)
	if !ok {
		return fmt.Errorf("source missing type field")
	}
	targetType, ok := target["type"].(string)
	if !ok {
		return fmt.Errorf("target missing type field")
	}
	sourceField, ok := source["field"].(string)
	if !ok {
		return fmt.Errorf("source missing field")
	}
	targetField, ok := target["field"].(string)
	if !ok {
		return fmt.Errorf("target missing field")
	}

	return graph.AddRelationship(
		relType,
		direction,
		sourceType,
		source["key"],
		sourceField,
		targetType,
		target["key"],
		targetField,
		properties,
	)
}

// Create relationships from existing nodes in the graph based on rule definitions
func (s *TransformService) createRelationshipsFromExistingNodes(rule *transform_agg.RuleAggregate, graph *graph.GraphAggregate) error {
	if rule.Rule.SourceNode == nil || rule.Rule.TargetNode == nil {
		return fmt.Errorf("rule %s missing source or target node configuration", rule.Rule.Name)
	}

	logrus.Infof("Creating relationships from existing nodes for rule: %s", rule.Rule.Name)
	logrus.Infof("Source node type: %s, key: %s, target_field: %s",
		rule.Rule.SourceNode.Type, rule.Rule.SourceNode.Key, rule.Rule.SourceNode.TargetField)
	logrus.Infof("Target node type: %s, key: %s, target_field: %s",
		rule.Rule.TargetNode.Type, rule.Rule.TargetNode.Key, rule.Rule.TargetNode.TargetField)

	// Get all existing nodes
	existingNodes := graph.GetNodes()
	logrus.Infof("Found %d existing nodes in graph", len(existingNodes))

	// Find source and target nodes
	var sourceNodes, targetNodes []*entities.Node
	for _, node := range existingNodes {
		if node.Type == rule.Rule.SourceNode.Type {
			sourceNodes = append(sourceNodes, node)
		}
		if node.Type == rule.Rule.TargetNode.Type {
			targetNodes = append(targetNodes, node)
		}
	}

	logrus.Infof("Found %d source nodes of type %s and %d target nodes of type %s",
		len(sourceNodes), rule.Rule.SourceNode.Type, len(targetNodes), rule.Rule.TargetNode.Type)

	// Create relationships based on matching fields
	relationshipCount := 0
	for _, sourceNode := range sourceNodes {
		// Get the key value from source node
		sourceKeyValue, exists := sourceNode.Properties[rule.Rule.SourceNode.Key]
		if !exists {
			logrus.Warnf("Source node %s missing key field %s", sourceNode.ID, rule.Rule.SourceNode.Key)
			continue
		}

		for _, targetNode := range targetNodes {
			// Get the key value from target node
			targetKeyValue, exists := targetNode.Properties[rule.Rule.TargetNode.Key]
			if !exists {
				logrus.Warnf("Target node %s missing key field %s", targetNode.ID, rule.Rule.TargetNode.Key)
				continue
			}

			// Check if keys match (convert to strings for comparison)
			sourceKeyStr := fmt.Sprintf("%v", sourceKeyValue)
			targetKeyStr := fmt.Sprintf("%v", targetKeyValue)

			if sourceKeyStr == targetKeyStr {
				// Create relationship properties
				properties := make(map[string]any)
				for srcProp, tgtProp := range rule.Rule.Properties {
					if value, exists := sourceNode.Properties[srcProp]; exists {
						properties[tgtProp] = value
					} else if value, exists := targetNode.Properties[srcProp]; exists {
						properties[tgtProp] = value
					}
				}

				// Add the relationship
				err := graph.AddDirectRelationship(
					rule.Rule.RelationType,
					sourceNode.ID,
					targetNode.ID,
					properties,
				)
				if err != nil {
					logrus.Warnf("Failed to create relationship %s between %s and %s: %v",
						rule.Rule.RelationType, sourceNode.ID, targetNode.ID, err)
					continue
				}
				relationshipCount++
				logrus.Infof("Created relationship %s: %s(%s) -> %s(%s)",
					rule.Rule.RelationType, sourceNode.Type, sourceKeyStr, targetNode.Type, targetKeyStr)
			}
		}
	}

	logrus.Infof("Created %d relationships for rule %s", relationshipCount, rule.Rule.Name)
	return nil
}

// Define a helper function to convert map properties to supported types
func (s *TransformService) convertMapProperties(item map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range item {
		switch val := v.(type) {
		case map[string]any:
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
