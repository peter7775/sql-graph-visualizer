/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package neo4j

import (
	"fmt"

	"mysql-graph-visualizer/internal/domain/models"

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

	// Implementace vložení dat
	return nil
}

func (c *Client) SearchNodes(query string, params map[string]interface{}) ([]models.SearchResult, error) {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.Run(query, params)
	if err != nil {
		return nil, err
	}

	var searchResults []models.SearchResult
	for result.Next() {
		record := result.Record()
		searchResults = append(searchResults, models.SearchResult{
			ID:     record.GetByIndex(0).(string),
			Name:   record.GetByIndex(1).(string),
			Labels: record.GetByIndex(2).([]string),
		})
	}

	return searchResults, nil
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

func (c *Client) StoreGraph(graph *models.Graph) error {
	session := c.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	logrus.Infof("Počet uzlů k uložení: %d", len(graph.Nodes))

	// Uložení uzlů
	for _, node := range graph.Nodes {
		query := "CREATE (n:" + node.Label + ") SET n = $props"
		if _, err := session.Run(query, map[string]interface{}{
			"props": node.Properties,
		}); err != nil {
			return err
		}
		logrus.Infof("Uložen uzel: typ=%s, vlastnosti=%+v", node.Label, node.Properties)
	}

	// Uložení vztahů
	for _, rel := range graph.Relations {
		query := "MATCH (a {id: $sourceId}), (b {id: $targetId}) CREATE (a)-[r:" + rel.Type + "]->(b) SET r = $props"
		if _, err := session.Run(query, map[string]interface{}{
			"sourceId": rel.From,
			"targetId": rel.To,
			"props":    rel.Properties,
		}); err != nil {
			return err
		}
	}

	return nil
}
