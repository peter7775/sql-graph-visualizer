package performance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
)

func ecommerceQuerySets() map[string]ports.CustomBenchmarkConfig {
	return map[string]ports.CustomBenchmarkConfig{
		"checkout": {Name: "checkout", Queries: []ports.CustomQueryDefinition{
			{Query: "SELECT 1", Tables: []string{"customers", "orders"}},
			{Query: "SELECT 2", Tables: []string{"Orders", "order_items", "products"}},
			{Query: "SELECT 3", Tables: []string{"categories", "products"}},
		}},
		"reporting": {Name: "reporting", Queries: []ports.CustomQueryDefinition{
			{Query: "SELECT c.name FROM customers c JOIN orders o ON o.customer_id = c.id"},
		}},
	}
}

func TestTopologyService_DerivesChainsNotStars(t *testing.T) {
	svc := NewTopologyService(newTestLogger(), nil, "", ecommerceQuerySets(), nil)

	topo, err := svc.Build(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if topo.Source != "queries+schema" {
		t.Errorf("source = %q, want queries+schema", topo.Source)
	}

	var nodeIDs []string
	for _, n := range topo.Nodes {
		nodeIDs = append(nodeIDs, n.ID)
	}
	if got := strings.Join(nodeIDs, ","); got != "categories,customers,order_items,orders,products" {
		t.Errorf("nodes = %s", got)
	}

	edges := map[string]ports.TopologyEdge{}
	for _, e := range topo.Edges {
		edges[e.ID] = e
	}
	for _, id := range []string{"customers-orders", "order_items-orders", "order_items-products", "categories-products"} {
		if _, ok := edges[id]; !ok {
			t.Errorf("missing edge %s in %v", id, edges)
		}
	}
	if _, star := edges["orders-products"]; star || len(edges) != 4 {
		t.Errorf("edges must follow the JOIN chain, got %v", edges)
	}

	for _, n := range topo.Nodes {
		if n.ID == "order_items" && n.Label != "Order Items" {
			t.Errorf("label = %q, want Order Items", n.Label)
		}
	}
}

func TestTopologyService_RestrictsToQuerySet(t *testing.T) {
	svc := NewTopologyService(newTestLogger(), nil, "", ecommerceQuerySets(), nil)

	topo, err := svc.Build(context.Background(), "reporting")
	if err != nil {
		t.Fatal(err)
	}
	if len(topo.Nodes) != 2 || len(topo.Edges) != 1 || topo.Edges[0].ID != "customers-orders" {
		t.Errorf("reporting topology (tables derived from SQL) = %+v", topo)
	}

	if _, err := svc.Build(context.Background(), "nope"); !errors.Is(err, ErrUnknownQuerySet) {
		t.Errorf("unknown query set error = %v, want ErrUnknownQuerySet", err)
	}
}

func TestTopologyService_MergesForeignKeys(t *testing.T) {
	svc := NewTopologyService(newTestLogger(), nil, "", ecommerceQuerySets(), nil)
	svc.loadForeignKeys = func(context.Context) ([]foreignKey, error) {
		return []foreignKey{
			{table: "orders", column: "customer_id", refTable: "customers", refColumn: "id"},
			{table: "orders", column: "x", refTable: "unused_table", refColumn: "id"}, // not in the query sets
			{table: "products", column: "category_id", refTable: "categories", refColumn: "id"},
		}, nil
	}

	topo, err := svc.Build(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	relations := map[string]string{}
	for _, e := range topo.Edges {
		relations[e.ID] = e.Relation
	}
	if relations["customers-orders"] != "orders.customer_id -> customers.id" {
		t.Errorf("FK relation not used: %q", relations["customers-orders"])
	}
	if relations["order_items-orders"] != "joined in queries" {
		t.Errorf("non-FK edge must keep the query relation: %q", relations["order_items-orders"])
	}
	if len(topo.Edges) != 4 {
		t.Errorf("FK to an unused table must not add an edge: %v", relations)
	}

	// A failing FK lookup degrades gracefully to the query-derived topology.
	svc.loadForeignKeys = func(context.Context) ([]foreignKey, error) { return nil, errors.New("boom") }
	topo, err = svc.Build(context.Background(), "")
	if err != nil || len(topo.Edges) != 4 {
		t.Errorf("topology after FK failure = %v, %v", topo, err)
	}
}

func TestTopologyService_ConfigOverrideWins(t *testing.T) {
	override := &models.LiveDemoTopology{
		Nodes: []models.LiveDemoNode{{ID: "orders"}, {ID: "customers", Label: "Customers", Table: "customers"}},
		Edges: []models.LiveDemoEdge{{Source: "orders", Target: "customers", Relation: "orders.customer_id -> customers.id"}},
	}
	svc := NewTopologyService(newTestLogger(), nil, "", ecommerceQuerySets(), override)

	topo, err := svc.Build(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if topo.Source != "config" || len(topo.Nodes) != 2 || topo.Nodes[0].Label != "orders" || topo.Nodes[1].Label != "Customers" {
		t.Errorf("override topology = %+v", topo)
	}
	if topo.Edges[0].ID != ports.GraphEdgeID("customers", "orders") {
		t.Errorf("override edge id = %q, want canonical id", topo.Edges[0].ID)
	}

	// A query set filter ignores the override and derives from the queries.
	derived, err := svc.Build(context.Background(), "reporting")
	if err != nil || derived.Source != "queries+schema" {
		t.Errorf("filtered topology = %+v, %v", derived, err)
	}
}

func TestQuerySetTables(t *testing.T) {
	got := QuerySetTables(ecommerceQuerySets()["checkout"])
	if strings.Join(got, ",") != "categories,customers,order_items,orders,products" {
		t.Errorf("QuerySetTables = %v", got)
	}
}

func demoRules() []models.TransformationConfig {
	return []models.TransformationConfig{
		{Name: "orders", RuleType: "node", TargetType: "Order",
			Source: models.SourceConfig{Type: "query", Value: "SELECT CONCAT('order-', id) AS id FROM orders WHERE id <= 40"}},
		{Name: "customers", RuleType: "node", TargetType: "Customer",
			Source: models.SourceConfig{Type: "query", Value: "select name from `customers` where id in (select customer_id from orders)"}},
		{Name: "products", RuleType: "node", TargetType: "Product",
			Source: models.SourceConfig{Type: "table", Value: "Products"}},
		{Name: "categories", RuleType: "node", TargetType: "Category",
			Source: models.SourceConfig{Type: "query", Value: "SELECT * FROM x", SourceTable: "Categories"}}, // explicit source_table wins
		{Name: "placed", RuleType: "relationship", RelationType: "PLACED_ORDER",
			Source:     models.SourceConfig{Type: "query", Value: "SELECT customer_id, id FROM orders"},
			SourceNode: models.RelationNode{Type: "Customer"}, TargetNode: models.RelationNode{Type: "Order"}},
		{Name: "contains", RuleType: "relationship", RelationType: "CONTAINS_PRODUCT",
			Source:     models.SourceConfig{Type: "query", Value: "SELECT order_id, product_id FROM order_items"},
			SourceNode: models.RelationNode{Type: "Order"}, TargetNode: models.RelationNode{Type: "Product"}},
		{Name: "belongs", RuleType: "relationship", RelationType: "BELONGS_TO_CATEGORY",
			Source:     models.SourceConfig{Type: "query", Value: "SELECT id, category_id FROM products"},
			SourceNode: models.RelationNode{Type: "Product"}, TargetNode: models.RelationNode{Type: "Category"}},
		{Name: "unmapped label", RuleType: "relationship", RelationType: "KNOWS",
			Source:     models.SourceConfig{Type: "query", Value: "SELECT 1 FROM orders"},
			SourceNode: models.RelationNode{Type: "Order"}, TargetNode: models.RelationNode{Type: "Ghost"}},
		{Name: "self", RuleType: "relationship", RelationType: "REPLY_TO",
			Source:     models.SourceConfig{Type: "query", Value: "SELECT 1 FROM orders"},
			SourceNode: models.RelationNode{Type: "Order"}, TargetNode: models.RelationNode{Type: "Order"}},
	}
}

func TestBuildLabelTableMapping(t *testing.T) {
	m := BuildLabelTableMapping(demoRules())

	if m.Source != "transform_rules" {
		t.Errorf("source = %q", m.Source)
	}
	wantLabels := map[string]string{"Order": "orders", "Customer": "customers", "Product": "products", "Category": "categories"}
	for label, table := range wantLabels {
		if got := m.Labels[label].Table; got != table {
			t.Errorf("label %s -> %q, want %q", label, got, table)
		}
	}

	cases := map[string][]string{
		"PLACED_ORDER":        {"customers-orders"},
		"CONTAINS_PRODUCT":    {"order_items-orders", "order_items-products"}, // through the junction table
		"BELONGS_TO_CATEGORY": {"categories-products"},
	}
	for rel, edges := range cases {
		got, ok := m.Relationships[rel]
		if !ok {
			t.Errorf("relationship %s missing", rel)
			continue
		}
		if strings.Join(got.Edges, ",") != strings.Join(edges, ",") || got.Edge != edges[0] {
			t.Errorf("relationship %s edges = %v (edge %q), want %v", rel, got.Edges, got.Edge, edges)
		}
	}
	if got := m.Relationships["PLACED_ORDER"]; got.FromLabel != "Customer" || got.ToLabel != "Order" {
		t.Errorf("end labels = %+v", got)
	}
	if _, ok := m.Relationships["KNOWS"]; ok {
		t.Error("a relationship with an unmapped end label must be skipped")
	}
	if _, ok := m.Relationships["REPLY_TO"]; ok {
		t.Error("a self relationship has no table edge and must be skipped")
	}
}

func TestBuildLabelTableMapping_LegacyNodesAndRelations(t *testing.T) {
	rules := []models.TransformationConfig{{
		Source: models.SourceConfig{Type: "query", Value: "SELECT * FROM users"},
		Nodes:  []models.NodeConfig{{Label: "User"}},
	}, {
		Source: models.SourceConfig{Type: "table", Value: "teams"},
		Nodes:  []models.NodeConfig{{Label: "Team"}},
		Relations: []models.RelationConfig{{
			Type: "MEMBER_OF", From: models.RelationNode{Type: "User"}, To: models.RelationNode{Type: "Team"},
		}},
	}}
	m := BuildLabelTableMapping(rules)
	if m.Labels["User"].Table != "users" || m.Labels["Team"].Table != "teams" {
		t.Errorf("legacy labels = %+v", m.Labels)
	}
	// The rule reads from "teams" which is one end, so it is a direct edge.
	if got := m.Relationships["MEMBER_OF"]; got.Edge != "teams-users" || len(got.Edges) != 1 {
		t.Errorf("legacy relation = %+v", got)
	}
}

func TestRuleSourceTable(t *testing.T) {
	tests := []struct {
		src  models.SourceConfig
		want string
	}{
		{models.SourceConfig{SourceTable: " Orders "}, "orders"},
		{models.SourceConfig{Type: "table", Value: "Customers"}, "customers"},
		{models.SourceConfig{Type: "query", Value: "SELECT\n a\nFROM\n  `Order_Items` oi"}, "order_items"},
		{models.SourceConfig{Type: "query", Value: "SELECT 1"}, ""},
		{models.SourceConfig{}, ""},
	}
	for _, tt := range tests {
		if got := ruleSourceTable(tt.src); got != tt.want {
			t.Errorf("ruleSourceTable(%+v) = %q, want %q", tt.src, got, tt.want)
		}
	}
}

func liveSampleFixture() *ports.LiveSample {
	return &ports.LiveSample{
		BenchmarkID: "run-1",
		Timestamp:   time.Now(),
		Interval:    ports.LiveIntervalStats{Queries: 100, QPS: 100, AvgLatencyMs: 300, Errors: 2},
		Graph: ports.LiveGraph{
			Nodes: []ports.LiveGraphNode{
				{ID: "orders", Table: "orders", QPS: 40, AvgLatencyMs: 300, Heat: 0.9, Queries: 40},
				{ID: "customers", Table: "customers", QPS: 10, AvgLatencyMs: 4, Heat: 0.0, Queries: 10},
				{ID: "order_items", Table: "order_items", QPS: 20, AvgLatencyMs: 40, Heat: 0.4, Queries: 20},
				{ID: "products", Table: "products", QPS: 5, AvgLatencyMs: 3, Heat: 0.0, Queries: 5},
			},
			Edges: []ports.LiveGraphEdge{
				{ID: "customers-orders", QPS: 10, AvgLatencyMs: 300, Load: 0.7, Rank: "SLOW"},
				{ID: "order_items-orders", QPS: 20, AvgLatencyMs: 40, Load: 0.2, Rank: "MEDIUM"},
				{ID: "order_items-products", QPS: 5, AvgLatencyMs: 3, Load: 0.5, Rank: "FAST"},
			},
		},
	}
}

func TestMapLiveToGraph(t *testing.T) {
	mapper := NewGraphPerformanceMapper(newTestLogger(), nil, nil, nil)
	mapping := BuildLabelTableMapping(demoRules())

	base := &models.Graph{
		Nodes: []*models.Node{
			{Label: "Order", Properties: map[string]any{"id": "order-1"}},
			{Label: "Customer", Properties: map[string]any{"id": "customer-1"}},
			{Label: "Product", Properties: map[string]any{"id": "product-1"}},
			{Label: "Unmapped", Properties: map[string]any{"id": "x-1"}},
		},
		Relations: []*models.Relation{
			{Type: "PLACED_ORDER", From: "customer-1", To: "order-1"},
			{Type: "CONTAINS_PRODUCT", From: "order-1", To: "product-1"},
			{Type: "UNKNOWN_REL", From: "x-1", To: "order-1"},
		},
	}

	data, err := mapper.MapLiveToGraph(context.Background(), base, liveSampleFixture(), &mapping)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Nodes) != 4 || len(data.Edges) != 3 {
		t.Fatalf("nodes/edges = %d/%d, want 4/3", len(data.Nodes), len(data.Edges))
	}

	nodes := map[string]PerformanceGraphNode{}
	for _, n := range data.Nodes {
		nodes[n.ID] = n
	}
	order := nodes["order-1"]
	if order.TableName != "orders" || order.Performance.AverageLatency != 300 || order.Visual.Color == heatHex(0) {
		t.Errorf("order node = %+v (a hot node must not be green)", order)
	}
	if order.Visual.Color == nodes["customer-1"].Visual.Color {
		t.Error("hot and cool nodes must get different colours")
	}
	if nodes["customer-1"].Visual.Color != "#22c55e" {
		t.Errorf("cool node colour = %s, want green", nodes["customer-1"].Visual.Color)
	}
	if order.Visual.Size <= nodes["customer-1"].Visual.Size {
		t.Error("the busier node must be bigger")
	}
	if len(order.Issues) != 1 || order.Issues[0].Type != "high_latency" {
		t.Errorf("hot node must carry a latency issue: %+v", order.Issues)
	}
	unmapped := nodes["x-1"]
	if unmapped.Performance.QueriesPerSecond != 0 || unmapped.Visual.Color != "#22c55e" {
		t.Errorf("unmapped node must stay neutral: %+v", unmapped)
	}

	edges := map[string]PerformanceGraphEdge{}
	for _, e := range data.Edges {
		edges[e.RelationType] = e
	}
	placed := edges["PLACED_ORDER"]
	if placed.Performance.PerformanceRank != "poor" || placed.Performance.AverageLatency != 300 || len(placed.Issues) != 1 {
		t.Errorf("PLACED_ORDER edge = %+v", placed)
	}
	// CONTAINS_PRODUCT runs through two joins: worst latency, highest load, summed frequency.
	contains := edges["CONTAINS_PRODUCT"]
	if contains.Performance.AverageLatency != 40 || contains.Performance.PerformanceRank != "fair" ||
		contains.Performance.LoadFactor != 0.5 || contains.Performance.QueryFrequency != 25 {
		t.Errorf("CONTAINS_PRODUCT edge = %+v", contains)
	}
	if got := edges["UNKNOWN_REL"]; got.Performance.PerformanceRank != "excellent" || got.Performance.QueryFrequency != 0 {
		t.Errorf("unmapped relation must stay neutral: %+v", got)
	}

	if len(data.Hotspots) != 1 || data.Hotspots[0].NodeID != "order-1" {
		t.Errorf("hotspots = %+v, want only the orders node", data.Hotspots)
	}
	if len(data.Bottlenecks) != 1 {
		t.Errorf("bottlenecks = %+v, want the slow join", data.Bottlenecks)
	}
	g := data.GlobalMetrics
	if g == nil || g.TotalQueriesPerSec != 100 || g.ErrorRate != 2 || g.HotspotCount != 1 || g.OverallScore != 10 {
		t.Errorf("global metrics = %+v", g)
	}
	if !strings.Contains(data.Metadata.DataSource, "run-1") || data.Metadata.NodeCount != 4 {
		t.Errorf("metadata = %+v", data.Metadata)
	}
}

func TestMapLiveToGraph_Validation(t *testing.T) {
	mapper := NewGraphPerformanceMapper(newTestLogger(), nil, nil, nil)
	if _, err := mapper.MapLiveToGraph(context.Background(), nil, liveSampleFixture(), nil); err == nil {
		t.Error("nil base graph must be rejected")
	}
	if _, err := mapper.MapLiveToGraph(context.Background(), &models.Graph{}, nil, nil); err == nil {
		t.Error("nil sample must be rejected")
	}
	// A nil mapping is tolerated: nothing maps, nothing breaks.
	base := &models.Graph{Nodes: []*models.Node{{Label: "Order"}}}
	data, err := mapper.MapLiveToGraph(context.Background(), base, liveSampleFixture(), nil)
	if err != nil || len(data.Nodes) != 1 || data.Nodes[0].Performance.QueriesPerSecond != 0 {
		t.Errorf("nil mapping result = %+v, %v", data, err)
	}
}

func TestHeatHex(t *testing.T) {
	if heatHex(0) != "#22c55e" || heatHex(0.5) != "#eab308" || heatHex(1) != "#ef4444" {
		t.Errorf("colour stops = %s %s %s", heatHex(0), heatHex(0.5), heatHex(1))
	}
	if heatHex(-3) != heatHex(0) || heatHex(7) != heatHex(1) {
		t.Error("heat must be clamped")
	}
	if rankToLegacy("slow") != "poor" || rankToLegacy("MEDIUM") != "fair" || rankToLegacy("FAST") != "excellent" || rankToLegacy("") != "excellent" {
		t.Error("rank vocabulary conversion is wrong")
	}
}
