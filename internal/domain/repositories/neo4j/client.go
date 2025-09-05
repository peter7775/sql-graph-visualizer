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

type Neo4jConfig struct {
	URI      string
	User     string
	Password string
}

type Client struct {
	driver neo4j.Driver
}

func NewNeo4jClient(config Neo4jConfig) (*Client, error) {
	driver, err := neo4j.NewDriver(config.URI,
		neo4j.BasicAuth(config.User, config.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	return &Client{driver: driver}, nil
}

func (c *Client) Close() error {
	return c.driver.Close()
}

func (c *Client) InsertData(data interface{}) error {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Implementation of data insertion
	return nil
}

func (c *Client) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	// Implement the logic to search nodes based on criteria
	// This is a placeholder implementation
	return []*graph.GraphAggregate{}, nil
}

func (c *Client) ExportGraph(query string) (interface{}, error) {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(query, nil)
	if err != nil {
		return nil, err
	}

	if result.Next() {
		return result.Record().GetByIndex(0), nil
	}

	return nil, nil
}

func (c *Client) StoreGraph(graph *graph.GraphAggregate) error {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	logrus.Infof("Number of nodes to save: %d", len(graph.GetNodes()))

	// Store node properties as individual keys and values
	for _, node := range graph.GetNodes() {
		query := "CREATE (n:Node {id: $id, type: $type"
		params := map[string]interface{}{
			"id":   node.ID,
			"type": node.Type,
		}
		for k, v := range node.Properties {
			query += ", " + k + ": $" + k
			params[k] = v
		}
		query += "})"
		if _, err := session.Run(query, params); err != nil {
			return err
		}
		logrus.Infof("Node saved: type=%s, properties=%+v", node.Type, node.Properties)
	}

	// Store relationships
	for _, rel := range graph.GetRelationships() {
		query := "MATCH (a:Node {id: $fromId}), (b:Node {id: $toId}) CREATE (a)-[r:RELATION {type: $type, properties: $properties}]->(b)"
		if _, err := session.Run(query, map[string]interface{}{
			"fromId":     rel.SourceNode.ID,
			"toId":       rel.TargetNode.ID,
			"type":       rel.Type,
			"properties": rel.Properties,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) FetchNodes(nodeType string) ([]map[string]interface{}, error) {
	logrus.Infof("Loading nodes of type: %s", nodeType)
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run("MATCH (n:Node {type: $nodeType}) RETURN n", map[string]interface{}{
		"nodeType": nodeType,
	})
	if err != nil {
		return nil, err
	}

	var nodes []map[string]interface{}
	for result.Next() {
		record := result.Record()
		node := record.GetByIndex(0).(neo4j.Node)
		properties := node.Props
		logrus.Infof("Loaded node: %v", properties)
		nodes = append(nodes, properties)
	}

	logrus.Infof("Loaded %d nodes", len(nodes))
	return nodes, nil
}

// GetDriver returns Neo4j driver for direct access
func (c *Client) GetDriver() neo4j.Driver {
	return c.driver
}
