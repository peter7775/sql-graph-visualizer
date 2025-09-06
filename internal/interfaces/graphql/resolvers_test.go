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
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/models"
)

// MockNeo4jPort is a mock implementation of the Neo4jPort
type MockNeo4jPort struct {
	mock.Mock
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

// Test setup helper
func setupTestResolver() (*Resolver, *MockNeo4jPort) {
	mockNeo4j := &MockNeo4jPort{}
	config := &models.Config{
		Neo4j: models.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			User:     "neo4j",
			Password: "testpass",
		},
	}

	resolver := &Resolver{
		Neo4jRepo: mockNeo4j,
		Config:    config,
	}

	return resolver, mockNeo4j
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

func TestQueryResolver_Graph(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	testGraph := createTestGraphAggregate()

	// Mock the ExportGraph call
	mockNeo4j.On("ExportGraph", "MATCH (n)-[r]->(m) RETURN n, r, m").Return(testGraph, nil)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.Graph(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Nodes, 2)
	assert.Len(t, result.Relationships, 1)

	// Check node details
	assert.Equal(t, "User_1", result.Nodes[0].ID)
	assert.Equal(t, "User", result.Nodes[0].Label)

	var node1Props map[string]any
	err = json.Unmarshal([]byte(result.Nodes[0].Properties), &node1Props)
	assert.NoError(t, err)
	assert.Equal(t, "Test User", node1Props["name"])

	// Check relationship details
	assert.Equal(t, "User_1", result.Relationships[0].From)
	assert.Equal(t, "Post_2", result.Relationships[0].To)
	assert.Equal(t, "CREATED", result.Relationships[0].Type)

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_NodesByType(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	testGraph := createTestGraphAggregate()

	// Mock the ExportGraph call for User nodes
	mockNeo4j.On("ExportGraph", "MATCH (n:User) RETURN n").Return(testGraph, nil)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.NodesByType(ctx, "User")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1) // Only User nodes should be returned

	assert.Equal(t, "User_1", result[0].ID)
	assert.Equal(t, "User", result[0].Label)

	var nodeProps map[string]any
	err = json.Unmarshal([]byte(result[0].Properties), &nodeProps)
	assert.NoError(t, err)
	assert.Equal(t, "Test User", nodeProps["name"])

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_Node(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	testGraph := createTestGraphAggregate()

	// Mock the ExportGraph call
	mockNeo4j.On("ExportGraph", "MATCH (n) WHERE id(n) = $nodeId RETURN n").Return(testGraph, nil)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.Node(ctx, "User_1")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "User_1", result.ID)
	assert.Equal(t, "User", result.Label)

	var nodeProps map[string]any
	err = json.Unmarshal([]byte(result.Properties), &nodeProps)
	assert.NoError(t, err)
	assert.Equal(t, "Test User", nodeProps["name"])

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_Node_NotFound(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	testGraph := createTestGraphAggregate()

	// Mock the ExportGraph call
	mockNeo4j.On("ExportGraph", "MATCH (n) WHERE id(n) = $nodeId RETURN n").Return(testGraph, nil)

	// Execute the query for non-existent node
	ctx := context.Background()
	result, err := queryResolver.Node(ctx, "NonExistent_999")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "node with ID NonExistent_999 not found")

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_RelationshipsByType(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	testGraph := createTestGraphAggregate()

	// Mock the ExportGraph call
	mockNeo4j.On("ExportGraph", "MATCH (n)-[r:CREATED]->(m) RETURN n, r, m").Return(testGraph, nil)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.RelationshipsByType(ctx, "CREATED")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)

	assert.Equal(t, "User_1", result[0].From)
	assert.Equal(t, "Post_2", result[0].To)
	assert.Equal(t, "CREATED", result[0].Type)

	var relProps map[string]any
	err = json.Unmarshal([]byte(result[0].Properties), &relProps)
	assert.NoError(t, err)
	assert.Equal(t, "2025-01-01", relProps["created_at"])

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_SearchNodes(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	testGraph := createTestGraphAggregate()

	// Mock the ExportGraph call for search
	expectedCypher := "MATCH (n) WHERE ANY(prop IN keys(n) WHERE toString(n[prop]) CONTAINS 'Test') RETURN n"
	mockNeo4j.On("ExportGraph", expectedCypher).Return(testGraph, nil)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.SearchNodes(ctx, "Test")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2) // Both nodes contain "Test" in their properties

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_Config(t *testing.T) {
	resolver, _ := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.Config(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Neo4j)
	assert.Equal(t, "bolt://localhost:7687", result.Neo4j.URI)
	assert.Equal(t, "neo4j", result.Neo4j.Username)
	assert.Equal(t, "testpass", result.Neo4j.Password)
}

func TestMutationResolver_TransformData(t *testing.T) {
	resolver, _ := setupTestResolver()
	mutationResolver := &mutationResolver{resolver}

	// Execute the mutation
	ctx := context.Background()
	result, err := mutationResolver.TransformData(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestQueryResolver_Graph_InvalidType(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	// Mock the ExportGraph call to return invalid type
	mockNeo4j.On("ExportGraph", "MATCH (n)-[r]->(m) RETURN n, r, m").Return("invalid-type", nil)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.Graph(ctx)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid graph type")

	mockNeo4j.AssertExpectations(t)
}

func TestQueryResolver_Graph_ExportError(t *testing.T) {
	resolver, mockNeo4j := setupTestResolver()
	queryResolver := &queryResolver{resolver}

	// Mock the ExportGraph call to return an error
	mockNeo4j.On("ExportGraph", "MATCH (n)-[r]->(m) RETURN n, r, m").Return(nil, assert.AnError)

	// Execute the query
	ctx := context.Background()
	result, err := queryResolver.Graph(ctx)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to export graph")

	mockNeo4j.AssertExpectations(t)
}
