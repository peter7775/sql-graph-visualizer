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

	"sql-graph-visualizer/internal/domain/aggregates/graph"

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

func (c *Client) InsertData(data any) error {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	// Implementation of data insertion
	return nil
}

func (c *Client) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	// Implement the logic to search nodes based on criteria
	// This is a placeholder implementation
	return []*graph.GraphAggregate{}, nil
}

func (c *Client) ExportGraph(query string) (any, error) {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	result, err := session.Run(query, nil)
	if err != nil {
		return nil, err
	}

	if result.Next() {
		return result.Record().Values[0], nil
	}

	return nil, nil
}

func (c *Client) StoreGraph(graph *graph.GraphAggregate) error {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	logrus.Infof("Number of nodes to save: %d", len(graph.GetNodes()))

	// Store node properties as individual keys and values
	for _, node := range graph.GetNodes() {
		query := "CREATE (n:Node {id: $id, type: $type"
		params := map[string]any{
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

		// Create relationship with proper name (not generic RELATION)
		query := "MATCH (a:Node {id: $fromId}), (b:Node {id: $toId}) CREATE (a)-[r:" + rel.Type + "]->(b) SET r = $props"
		params := map[string]any{
			"fromId": sourceID,
			"toId":   targetID,
			"props":  rel.Properties,
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

func (c *Client) FetchNodes(nodeType string) ([]map[string]any, error) {
	logrus.Infof("Loading nodes of type: %s", nodeType)
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	result, err := session.Run("MATCH (n:Node {type: $nodeType}) RETURN n", map[string]any{
		"nodeType": nodeType,
	})
	if err != nil {
		return nil, err
	}

	var nodes []map[string]any
	for result.Next() {
		record := result.Record()
		node := record.Values[0].(neo4j.Node)
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
