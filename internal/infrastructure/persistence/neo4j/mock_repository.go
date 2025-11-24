package neo4j

import (
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/entities"

	"github.com/sirupsen/logrus"
)

type MockNeo4jRepository struct {
	logger    *logrus.Logger
	nodes     []*entities.Node
	relations []*entities.Relation
}

func NewMockNeo4jRepository() ports.Neo4jPort {
	logrus.Info("Creating Mock Neo4j Repository for Railway deployment")
	return &MockNeo4jRepository{
		logger:    logrus.StandardLogger(),
		nodes:     []*entities.Node{},
		relations: []*entities.Relation{},
	}
}

func (m *MockNeo4jRepository) StoreGraph(graph *graph.GraphAggregate) error {
	logrus.Info("Mock: Storing graph data")
	return nil
}

func (m *MockNeo4jRepository) SearchNodes(criteria string) ([]*graph.GraphAggregate, error) {
	logrus.Infof("Mock: Searching nodes with criteria: %s", criteria)
	return []*graph.GraphAggregate{}, nil
}

func (m *MockNeo4jRepository) ExportGraph(query string) (any, error) {
	logrus.Info("Mock: Exporting graph data")

	sampleNodes := []*entities.Node{
		entities.NewNode("1", "Actor"),
		entities.NewNode("2", "Actor"),
		entities.NewNode("3", "Film"),
		entities.NewNode("4", "Film"),
		entities.NewNode("5", "Category"),
		entities.NewNode("6", "Category"),
	}

	sampleNodes[0].Properties["name"] = "Tom Hanks"
	sampleNodes[0].Properties["actor_id"] = "1"
	sampleNodes[1].Properties["name"] = "Brad Pitt"
	sampleNodes[1].Properties["actor_id"] = "2"
	sampleNodes[2].Properties["title"] = "The Matrix"
	sampleNodes[2].Properties["film_id"] = "1"
	sampleNodes[2].Properties["year"] = 1999
	sampleNodes[3].Properties["title"] = "Inception"
	sampleNodes[3].Properties["film_id"] = "2"
	sampleNodes[3].Properties["year"] = 2010
	sampleNodes[4].Properties["name"] = "Action"
	sampleNodes[4].Properties["category_id"] = "1"
	sampleNodes[5].Properties["name"] = "Sci-Fi"
	sampleNodes[5].Properties["category_id"] = "7"

	sampleRelations := []*entities.Relation{
		entities.NewRelation("1", "ACTED_IN", sampleNodes[0], sampleNodes[2]),
		entities.NewRelation("2", "ACTED_IN", sampleNodes[1], sampleNodes[3]),
		entities.NewRelation("3", "HAS_CATEGORY", sampleNodes[2], sampleNodes[4]),
		entities.NewRelation("4", "HAS_CATEGORY", sampleNodes[2], sampleNodes[5]),
		entities.NewRelation("5", "HAS_CATEGORY", sampleNodes[3], sampleNodes[4]),
		entities.NewRelation("6", "HAS_CATEGORY", sampleNodes[3], sampleNodes[5]),
	}

	// Combine with actual stored data
	allNodes := append(sampleNodes, m.nodes...)
	allRelations := append(sampleRelations, m.relations...)

	graphAggregate := graph.NewGraphAggregate("mock-graph-1")

	// Add nodes using proper GraphAggregate method
	for _, node := range allNodes {
		if err := graphAggregate.AddNode(node.Label, node.Properties); err != nil {
			logrus.WithError(err).Error("Failed to add node to graph")
		}
	}

	// Add relationships using direct relationship method
	for _, rel := range allRelations {
		// Extract IDs from properties
		sourceID := rel.FromNode.Properties["actor_id"]
		if sourceID == nil {
			sourceID = rel.FromNode.Properties["film_id"]
		}
		if sourceID == nil {
			sourceID = rel.FromNode.Properties["category_id"]
		}

		targetID := rel.ToNode.Properties["actor_id"]
		if targetID == nil {
			targetID = rel.ToNode.Properties["film_id"]
		}
		if targetID == nil {
			targetID = rel.ToNode.Properties["category_id"]
		}

		if sourceID != nil && targetID != nil {
			if err := graphAggregate.AddDirectRelationship(rel.Type, sourceID, targetID, rel.Properties); err != nil {
				logrus.WithError(err).Error("Failed to add relationship to graph")
			}
		}
	}

	return graphAggregate, nil
}

func (m *MockNeo4jRepository) FetchNodes(nodeType string) ([]map[string]any, error) {
	logrus.Infof("Mock: Fetching nodes of type: %s", nodeType)

	// Return sample nodes based on type
	switch nodeType {
	case "Actor":
		return []map[string]any{
			{"name": "Tom Hanks", "actor_id": "1"},
			{"name": "Brad Pitt", "actor_id": "2"},
		}, nil
	case "Film":
		return []map[string]any{
			{"title": "The Matrix", "film_id": "1", "year": 1999},
			{"title": "Inception", "film_id": "2", "year": 2010},
		}, nil
	case "Category":
		return []map[string]any{
			{"name": "Action", "category_id": "1"},
			{"name": "Sci-Fi", "category_id": "7"},
		}, nil
	default:
		return []map[string]any{}, nil
	}
}

func (m *MockNeo4jRepository) ExecuteQuery(query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	logrus.Infof("Mock: Executing query: %s", query)
	return []map[string]interface{}{}, nil
}

func (m *MockNeo4jRepository) Close() error {
	logrus.Info("Mock: Closing Neo4j repository (no-op)")
	return nil
}
