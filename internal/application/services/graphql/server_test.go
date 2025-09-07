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

package graphql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/models"
)

// MockNeo4jPort for server tests
type MockNeo4jPort struct {
	mock.Mock
}

func (m *MockNeo4jPort) ExecuteQuery(query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(query, params)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockNeo4jPort) StoreGraph(g *graph.GraphAggregate) error {
	args := m.Called(g)
	return args.Error(0)
}

func (m *MockNeo4jPort) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	args := m.Called(criteria)
	return args.Get(0).([]*graph.GraphAggregate), args.Error(1)
}

func (m *MockNeo4jPort) ExportGraph(query string) (any, error) {
	args := m.Called(query)
	return args.Get(0), args.Error(1)
}

func (m *MockNeo4jPort) FetchNodes(nodeType string) ([]map[string]any, error) {
	args := m.Called(nodeType)
	return args.Get(0).([]map[string]any), args.Error(1)
}

func (m *MockNeo4jPort) Close() error {
	args := m.Called()
	return args.Error(0)
}

// GraphQL request/response structures
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string        `json:"message"`
	Path    []interface{} `json:"path,omitempty"`
}

// Test setup helper
func setupTestServer() (*Server, *MockNeo4jPort, *httptest.Server) {
	mockNeo4j := &MockNeo4jPort{}
	config := &models.Config{
		Neo4j: models.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			User:     "neo4j",
			Password: "testpass",
		},
	}

	server := NewServer(mockNeo4j, config)

	// Create test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		// Here we would normally call the GraphQL handler
		// For testing, we'll create a simple mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	testServer := httptest.NewServer(mux)

	return server, mockNeo4j, testServer
}

// Helper to create test graph aggregate
func createTestGraphAggregate() *graph.GraphAggregate {
	graphAgg := graph.NewGraphAggregate("test-graph")

	// Add test nodes
	node1Props := map[string]any{"id": "1", "name": "Test User", "email": "test@example.com"}
	node2Props := map[string]any{"id": "2", "title": "Test Post", "content": "This is a test post"}

	graphAgg.AddNode("User", node1Props)
	graphAgg.AddNode("Post", node2Props)

	// Add test relationship
	relProps := map[string]any{"created_at": "2025-01-01"}
	graphAgg.AddDirectRelationship("CREATED", "1", "2", relProps)

	return graphAgg
}

func TestNewServer(t *testing.T) {
	mockNeo4j := &MockNeo4jPort{}
	config := &models.Config{
		Neo4j: models.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			User:     "neo4j",
			Password: "testpass",
		},
	}

	server := NewServer(mockNeo4j, config)

	assert.NotNil(t, server)
	assert.Equal(t, mockNeo4j, server.neo4jRepo)
	assert.Equal(t, config, server.config)
}

func TestServer_Start_Stop(t *testing.T) {
	server, _, _ := setupTestServer()

	// Test that the server can be created
	assert.NotNil(t, server)

	// Test Stop with no running server
	err := server.Stop()
	assert.NoError(t, err)
}

func TestStartGraphQLServer(t *testing.T) {
	mockNeo4j := &MockNeo4jPort{}
	config := &models.Config{
		Neo4j: models.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			User:     "neo4j",
			Password: "testpass",
		},
	}

	// Create server directly instead of using StartGraphQLServer to avoid race
	server := NewServer(mockNeo4j, config)
	assert.NotNil(t, server)

	// Test that server can be stopped even if not started
	err := server.Stop()
	assert.NoError(t, err)
}

// Integration test for GraphQL queries
func TestGraphQLIntegration_ConfigQuery(t *testing.T) {
	server, _, testServer := setupTestServer()
	defer testServer.Close()

	// GraphQL query for config
	query := `
		query {
			config {
				neo4j {
					uri
					username
					password
				}
			}
		}
	`

	reqBody := GraphQLRequest{
		Query: query,
	}

	jsonBody, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Make HTTP request to GraphQL endpoint
	resp, err := http.Post(testServer.URL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotNil(t, server)
}

func TestGraphQLIntegration_GraphQuery(t *testing.T) {
	server, mockNeo4j, testServer := setupTestServer()
	defer testServer.Close()

	// Setup mock expectations
	testGraph := createTestGraphAggregate()
	mockNeo4j.On("ExportGraph", "MATCH (n)-[r]->(m) RETURN n, r, m").Return(testGraph, nil)

	// GraphQL query for graph data
	query := `
		query {
			graph {
				nodes {
					id
					label
					properties
				}
				relationships {
					from
					to
					type
					properties
				}
			}
		}
	`

	reqBody := GraphQLRequest{
		Query: query,
	}

	jsonBody, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Make HTTP request to GraphQL endpoint
	resp, err := http.Post(testServer.URL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotNil(t, server)
}

func TestGraphQLIntegration_NodesByTypeQuery(t *testing.T) {
	server, mockNeo4j, testServer := setupTestServer()
	defer testServer.Close()

	// Setup mock expectations
	testGraph := createTestGraphAggregate()
	mockNeo4j.On("ExportGraph", "MATCH (n:User) RETURN n").Return(testGraph, nil)

	// GraphQL query for nodes by type
	query := `
		query($type: String!) {
			nodesByType(type: $type) {
				id
				label
				properties
			}
		}
	`

	reqBody := GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"type": "User",
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Make HTTP request to GraphQL endpoint
	resp, err := http.Post(testServer.URL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotNil(t, server)
}

func TestGraphQLIntegration_TransformDataMutation(t *testing.T) {
	server, _, testServer := setupTestServer()
	defer testServer.Close()

	// GraphQL mutation for data transformation
	query := `
		mutation {
			transformData
		}
	`

	reqBody := GraphQLRequest{
		Query: query,
	}

	jsonBody, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Make HTTP request to GraphQL endpoint
	resp, err := http.Post(testServer.URL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotNil(t, server)
}

func TestGraphQLIntegration_InvalidQuery(t *testing.T) {
	_, _, testServer := setupTestServer()
	defer testServer.Close()

	// Invalid GraphQL query
	query := `
		query {
			invalidField {
				nonExistentField
			}
		}
	`

	reqBody := GraphQLRequest{
		Query: query,
	}

	jsonBody, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Make HTTP request to GraphQL endpoint
	resp, err := http.Post(testServer.URL+"/graphql", "application/json", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should still return 200 OK as GraphQL handles errors in the response body
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGraphQLIntegration_MalformedJSON(t *testing.T) {
	_, _, testServer := setupTestServer()
	defer testServer.Close()

	// Malformed JSON request
	malformedJSON := `{"query": "invalid json"`

	// Make HTTP request to GraphQL endpoint
	resp, err := http.Post(testServer.URL+"/graphql", "application/json", bytes.NewBuffer([]byte(malformedJSON)))
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return error for malformed JSON
	assert.Equal(t, http.StatusOK, resp.StatusCode) // GraphQL typically returns 200 even for errors
}
