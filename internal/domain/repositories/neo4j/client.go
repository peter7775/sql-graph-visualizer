package neo4j

import (
	"fmt"

	"github.com/peter7775/alevisualizer/internal/domain/models"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
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

	// Implementace ukládání grafu
	return nil
}
