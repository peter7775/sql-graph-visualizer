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


package neo4j

import (
	"fmt"
	"log"
	"sql-graph-visualizer/internal/domain/aggregates/graph"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/sirupsen/logrus"
)

type Neo4jRepository struct {
	driver neo4j.Driver
}

func NewNeo4jRepository(uri, username, password string) (*Neo4jRepository, error) {
	logrus.Infof("Creating Neo4j driver with URI: %s, user: %s", uri, username)
	driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		logrus.Errorf("Error creating Neo4j driver: %v", err)
		return nil, err
	}
	logrus.Infof("Neo4j driver created successfully")
	return &Neo4jRepository{driver: driver}, nil
}

func (r *Neo4jRepository) StoreGraph(graph *graph.GraphAggregate) error {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("Error closing session: %v", err)
		}
	}()

	// Store nodes
	for _, node := range graph.GetNodes() {
		query := "CREATE (n:" + node.Type + ") SET n = $props"
		if _, err := session.Run(query, map[string]any{
			"props": node.Properties,
		}); err != nil {
			return err
		}
		logrus.Infof("Node saved: type=%s, properties=%+v", node.Type, node.Properties)
	}

	// Store relationships
	logrus.Infof("Number of relationships to save: %d", len(graph.GetRelationships()))
	for _, rel := range graph.GetRelationships() {
		// Get the actual IDs from node properties instead of node entity IDs
		sourceID, exists := rel.SourceNode.Properties["id"]
		if !exists {
			logrus.Warnf("Source node missing id property for relationship %s", rel.Type)
			continue
		}
		targetID, exists := rel.TargetNode.Properties["id"]
		if !exists {
			logrus.Warnf("Target node missing id property for relationship %s", rel.Type)
			continue
		}

		logrus.Infof("Creating relationship %s: %v -> %v", rel.Type, sourceID, targetID)

		// Create relationship with proper source and target matching
		query := "MATCH (a {id: $sourceId}), (b {id: $targetId}) CREATE (a)-[r:" + rel.Type + "]->(b) SET r = $props"
		params := map[string]any{
			"sourceId": sourceID,
			"targetId": targetID,
			"props":    rel.Properties,
		}

		result, err := session.Run(query, params)
		if err != nil {
			logrus.Errorf("Failed to create relationship %s from %v to %v: %v", rel.Type, sourceID, targetID, err)
			return err
		}

		// Check if relationship was actually created
		summary, err := result.Consume()
		if err != nil {
			logrus.Warnf("Error consuming result for relationship %s: %v", rel.Type, err)
		} else {
			logrus.Infof("Relationship %s created successfully. Relationships created: %d", rel.Type, summary.Counters().RelationshipsCreated())
		}
	}

	return nil
}

func (r *Neo4jRepository) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("Error closing session: %v", err)
		}
	}()

	result, err := session.Run(criteria, nil)
	if err != nil {
		return nil, err
	}

	if !result.Next() {
		return nil, nil
	}

	record := result.Record()
	count := record.Values[0].(int64)

	// Create one GraphAggregate with nodes
	graphAgg := graph.NewGraphAggregate("")
	for i := int64(0); i < count; i++ {
		if err := graphAgg.AddNode("Person", map[string]any{
			"id":   i + 1,
			"name": fmt.Sprintf("Person %d", i+1),
		}); err != nil {
			logrus.Errorf("Error adding Person node: %v", err)
		}
	}

	return []*graph.GraphAggregate{graphAgg}, nil
}

func (r *Neo4jRepository) ExportGraph(query string) (any, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("Error closing session: %v", err)
		}
	}()

	graphAgg := graph.NewGraphAggregate("")

	// First, fetch all nodes
	nodeResult, err := session.Run(`MATCH (n) RETURN n`, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nodes: %w", err)
	}

	processedNodes := make(map[int64]bool)
	for nodeResult.Next() {
		record := nodeResult.Record()
		node := record.Values[0].(neo4j.Node)

		// Check if we've already processed this node
		if processedNodes[node.Id] {
			continue
		}
		processedNodes[node.Id] = true

		// Add node to graph
		nodeProps := make(map[string]any)
		for key, value := range node.Props {
			nodeProps[key] = value
		}

		// Set ID if not present in props
		if _, hasID := nodeProps["id"]; !hasID {
			nodeProps["id"] = node.Id
		}

		label := "Unknown"
		if len(node.Labels) > 0 {
			label = node.Labels[0]
		}

		logrus.Debugf("Adding node to graph: ID=%d, Label=%s, Props=%+v", node.Id, label, nodeProps)
		if err := graphAgg.AddNode(label, nodeProps); err != nil {
			logrus.Errorf("Error adding %s node: %v", label, err)
		}
	}

	if err = nodeResult.Err(); err != nil {
		return nil, fmt.Errorf("error processing nodes: %w", err)
	}

	// Then, fetch all relationships
	relResult, err := session.Run(`MATCH (n)-[r]->(m) RETURN n, r, m`, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relationships: %w", err)
	}

	for relResult.Next() {
		record := relResult.Record()
		sourceNode := record.Values[0].(neo4j.Node)
		rel := record.Values[1].(neo4j.Relationship)
		targetNode := record.Values[2].(neo4j.Node)

		// Create relationship properties
		relProps := make(map[string]any)
		for key, value := range rel.Props {
			relProps[key] = value
		}

		// Add relationship to graph
		err = graphAgg.AddDirectRelationship(
			rel.Type,
			sourceNode.Id,
			targetNode.Id,
			relProps,
		)
		if err != nil {
			logrus.Warnf("Failed to add relationship %s: %v", rel.Type, err)
			continue
		}

		logrus.Debugf("Adding relationship: Type=%s, Source=%d, Target=%d, Props=%+v",
			rel.Type, sourceNode.Id, targetNode.Id, relProps)
	}

	if err = relResult.Err(); err != nil {
		return nil, fmt.Errorf("error processing relationships: %w", err)
	}

	logrus.Infof("ExportGraph complete: %d nodes, %d relationships",
		len(graphAgg.GetNodes()), len(graphAgg.GetRelationships()))

	return graphAgg, nil
}

func (r *Neo4jRepository) Close() error {
	return r.driver.Close()
}

func (r *Neo4jRepository) NewSession(config neo4j.SessionConfig) neo4j.Session {
	return r.driver.NewSession(config)
}

func (r *Neo4jRepository) FetchNodes(nodeType string) ([]map[string]any, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	query := fmt.Sprintf("MATCH (n:%s) RETURN n", nodeType)
	result, err := session.Run(query, nil)
	if err != nil {
		return nil, err
	}

	var nodes []map[string]any
	for result.Next() {
		record := result.Record()
		node := record.Values[0].(neo4j.Node)
		nodes = append(nodes, node.Props)
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return nodes, nil
}

// ExecuteQuery executes a generic Cypher query and returns results as maps
func (r *Neo4jRepository) ExecuteQuery(query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	result, err := session.Run(query, params)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for result.Next() {
		record := result.Record()
		row := make(map[string]interface{})
		
		// Extract all values from the record
		for i, key := range record.Keys {
			row[key] = record.Values[i]
		}
		
		results = append(results, row)
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
