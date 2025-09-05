/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package neo4j

import (
	"fmt"
	"mysql-graph-visualizer/internal/domain/aggregates/graph"

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
	defer session.Close()

	// Store nodes
	for _, node := range graph.GetNodes() {
		query := "CREATE (n:" + node.Type + ") SET n = $props"
		if _, err := session.Run(query, map[string]interface{}{
			"props": node.Properties,
		}); err != nil {
			return err
		}
		logrus.Infof("Node saved: type=%s, properties=%+v", node.Type, node.Properties)
	}

	// Store relationships
	for _, rel := range graph.GetRelationships() {
		query := "MATCH (a {id: $sourceId}), (b {id: $targetId}) CREATE (a)-[r:" + rel.Type + "]->(b) SET r = $props"
		if _, err := session.Run(query, map[string]interface{}{
			"sourceId": rel.SourceNode.ID,
			"targetId": rel.TargetNode.ID,
			"props":    rel.Properties,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *Neo4jRepository) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

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
		graphAgg.AddNode("Person", map[string]interface{}{
			"id":   i + 1,
			"name": fmt.Sprintf("Person %d", i+1),
		})
	}

	return []*graph.GraphAggregate{graphAgg}, nil
}

func (r *Neo4jRepository) ExportGraph(query string) (interface{}, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(`
	MATCH (n)
	OPTIONAL MATCH (n)-[r]->(m)
	RETURN n, r, m
	UNION
	MATCH (n)
	WHERE NOT (n)--()
	RETURN n, null AS r, null AS m
	`, nil)
	if err != nil {
		return nil, err
	}

	graphAgg := graph.NewGraphAggregate("")

	for result.Next() {
		record := result.Record()
		node := record.GetByIndex(0).(neo4j.Node)

		// Add node to graph
		graphAgg.AddNode(node.Labels[0], node.Props)
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return graphAgg, nil
}

func (r *Neo4jRepository) Close() error {
	return r.driver.Close()
}

func (r *Neo4jRepository) NewSession(config neo4j.SessionConfig) neo4j.Session {
	return r.driver.NewSession(config)
}

func (r *Neo4jRepository) FetchNodes(nodeType string) ([]map[string]interface{}, error) {
	session := r.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	query := fmt.Sprintf("MATCH (n:%s) RETURN n", nodeType)
	result, err := session.Run(query, nil)
	if err != nil {
		return nil, err
	}

	var nodes []map[string]interface{}
	for result.Next() {
		record := result.Record()
		node := record.GetByIndex(0).(neo4j.Node)
		nodes = append(nodes, node.Props)
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return nodes, nil
}
