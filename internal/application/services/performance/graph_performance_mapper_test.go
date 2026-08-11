package performance

import (
	"context"
	"testing"

	"sql-graph-visualizer/internal/domain/models"
)

func TestGraphPerformanceMapper_MapPerformanceToGraph(t *testing.T) {
	mapper := NewGraphPerformanceMapper(newTestLogger(), nil, nil, nil)

	baseGraph := &models.Graph{
		Nodes: []*models.Node{
			{Label: "User", Properties: map[string]any{"id": "1"}},
			{Label: "Team", Properties: map[string]any{"id": "2"}},
		},
		Relations: []*models.Relation{
			{Type: "MEMBER_OF", From: "1", To: "2", Properties: map[string]any{}},
		},
	}
	perfData := &PerformanceSchemaData{
		ConnectionStats: &ConnectionStatistics{},
	}

	graphData, err := mapper.MapPerformanceToGraph(context.Background(), baseGraph, perfData)
	if err != nil {
		t.Fatalf("MapPerformanceToGraph() error = %v", err)
	}
	if len(graphData.Nodes) != len(baseGraph.Nodes) {
		t.Errorf("MapPerformanceToGraph() returned %d nodes, want %d", len(graphData.Nodes), len(baseGraph.Nodes))
	}
	if len(graphData.Edges) != len(baseGraph.Relations) {
		t.Errorf("MapPerformanceToGraph() returned %d edges, want %d", len(graphData.Edges), len(baseGraph.Relations))
	}
	if graphData.Metadata.NodeCount != len(baseGraph.Nodes) {
		t.Errorf("Metadata.NodeCount = %d, want %d", graphData.Metadata.NodeCount, len(baseGraph.Nodes))
	}
}

func TestGraphPerformanceMapper_MapPerformanceToGraph_RequiresInputs(t *testing.T) {
	mapper := NewGraphPerformanceMapper(newTestLogger(), nil, nil, nil)
	ctx := context.Background()

	if _, err := mapper.MapPerformanceToGraph(ctx, nil, &PerformanceSchemaData{}); err == nil {
		t.Error("MapPerformanceToGraph() with nil base graph: expected error, got nil")
	}
	if _, err := mapper.MapPerformanceToGraph(ctx, &models.Graph{}, nil); err == nil {
		t.Error("MapPerformanceToGraph() with nil performance data: expected error, got nil")
	}
}

func TestGraphPerformanceMapper_CreatePerformanceNode_UsesIDProperty(t *testing.T) {
	mapper := NewGraphPerformanceMapper(newTestLogger(), nil, nil, nil)

	node := &models.Node{Label: "User", Properties: map[string]any{"id": "abc-123"}}
	perfNode := mapper.createPerformanceNode(node, map[string]*TablePerformanceInfo{})

	if perfNode.ID != "abc-123" {
		t.Errorf("createPerformanceNode() ID = %q, want abc-123", perfNode.ID)
	}
	if perfNode.TableName != "User" {
		t.Errorf("createPerformanceNode() TableName = %q, want User", perfNode.TableName)
	}
}
