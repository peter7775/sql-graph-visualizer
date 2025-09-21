/*
 * Mock Neo4j Repository for Railway deployment when Neo4j service is unavailable
 */

package neo4j

import (
	"context"
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/entities"

	"github.com/sirupsen/logrus"
)

type MockNeo4jRepository struct {
	logger *logrus.Logger
	nodes  []*entities.Node
	relationships []*entities.Relationship
}

func NewMockNeo4jRepository() ports.Neo4jPort {
	logrus.Info("Creating Mock Neo4j Repository for Railway deployment")
	return &MockNeo4jRepository{
		logger: logrus.StandardLogger(),
		nodes: []*entities.Node{},
		relationships: []*entities.Relationship{},
	}
}

func (m *MockNeo4jRepository) CreateNode(ctx context.Context, node *entities.Node) error {
	logrus.Debugf("Mock: Creating node %s", node.ID)
	m.nodes = append(m.nodes, node)
	return nil
}

func (m *MockNeo4jRepository) CreateRelationship(ctx context.Context, relationship *entities.Relationship) error {
	logrus.Debugf("Mock: Creating relationship %s -> %s", relationship.SourceNode.ID, relationship.TargetNode.ID)
	m.relationships = append(m.relationships, relationship)
	return nil
}

func (m *MockNeo4jRepository) ExportGraph(query string) (interface{}, error) {
	logrus.Info("Mock: Exporting graph data")
	
	// Create sample graph data for visualization
	sampleNodes := []*entities.Node{
		{ID: "1", Type: "Actor", Properties: map[string]any{"name": "Tom Hanks", "actor_id": "1"}},
		{ID: "2", Type: "Actor", Properties: map[string]any{"name": "Brad Pitt", "actor_id": "2"}},
		{ID: "3", Type: "Film", Properties: map[string]any{"title": "The Matrix", "film_id": "1", "year": 1999}},
		{ID: "4", Type: "Film", Properties: map[string]any{"title": "Inception", "film_id": "2", "year": 2010}},
		{ID: "5", Type: "Category", Properties: map[string]any{"name": "Action", "category_id": "1"}},
		{ID: "6", Type: "Category", Properties: map[string]any{"name": "Sci-Fi", "category_id": "7"}},
	}
	
	sampleRelationships := []*entities.Relationship{
		{SourceNode: sampleNodes[0], TargetNode: sampleNodes[2], Type: "ACTED_IN", Properties: map[string]any{}},
		{SourceNode: sampleNodes[1], TargetNode: sampleNodes[3], Type: "ACTED_IN", Properties: map[string]any{}},
		{SourceNode: sampleNodes[2], TargetNode: sampleNodes[4], Type: "HAS_CATEGORY", Properties: map[string]any{}},
		{SourceNode: sampleNodes[2], TargetNode: sampleNodes[5], Type: "HAS_CATEGORY", Properties: map[string]any{}},
		{SourceNode: sampleNodes[3], TargetNode: sampleNodes[4], Type: "HAS_CATEGORY", Properties: map[string]any{}},
		{SourceNode: sampleNodes[3], TargetNode: sampleNodes[5], Type: "HAS_CATEGORY", Properties: map[string]any{}},
	}
	
	// Combine with actual stored data
	allNodes := append(sampleNodes, m.nodes...)
	allRelationships := append(sampleRelationships, m.relationships...)
	
	graphAggregate := graph.NewGraphAggregate()
	for _, node := range allNodes {
		graphAggregate.AddNode(node)
	}
	for _, rel := range allRelationships {
		graphAggregate.AddRelationship(rel)
	}
	
	return graphAggregate, nil
}

func (m *MockNeo4jRepository) Close() error {
	logrus.Info("Mock: Closing Neo4j repository (no-op)")
	return nil
}

// Additional methods that might be needed
func (m *MockNeo4jRepository) NewSession(config interface{}) interface{} {
	return &MockSession{}
}

type MockSession struct{}

func (s *MockSession) Run(query string, params interface{}) (interface{}, error) {
	logrus.Debugf("Mock: Running query: %s", query)
	return &MockResult{}, nil
}

func (s *MockSession) Close() error {
	return nil
}

type MockResult struct{}