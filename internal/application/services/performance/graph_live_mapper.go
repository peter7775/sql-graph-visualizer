package performance

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
)

const (
	liveGraphVersion = "2.1"

	// hotspotHeat is the node heat (0..1) from which a node counts as hotspot.
	hotspotHeat = 0.7

	// Heat colour stops, the same green-yellow-red scale as the live UI.
	heatGreen  = "#22c55e"
	heatYellow = "#eab308"
	heatRed    = "#ef4444"
)

// MapLiveToGraph maps a live benchmark sample onto the instance-level base graph
// (the Neo4j graph). Every node receives the performance of its table (resolved
// from the node's table_name property or the label mapping) and every relation
// the performance of the edge(s) its type maps to. Elements without a mapping
// keep zero performance; there is no heuristic matching.
func (gpm *GraphPerformanceMapper) MapLiveToGraph(
	_ context.Context,
	baseGraph *models.Graph,
	sample *ports.LiveSample,
	mapping *ports.LabelTableMapping,
) (*PerformanceGraphData, error) {
	if baseGraph == nil {
		return nil, fmt.Errorf("base graph is required")
	}
	if sample == nil {
		return nil, fmt.Errorf("live sample is required")
	}
	if mapping == nil {
		mapping = &ports.LabelTableMapping{Labels: map[string]ports.LabelMapping{}, Relationships: map[string]ports.RelationshipMapping{}}
	}
	started := time.Now()

	tableNodes := make(map[string]ports.LiveGraphNode, len(sample.Graph.Nodes))
	maxQPS := 0.0
	for _, n := range sample.Graph.Nodes {
		tableNodes[strings.ToLower(n.ID)] = n
		maxQPS = math.Max(maxQPS, n.QPS)
	}
	sampleEdges := make(map[string]ports.LiveGraphEdge, len(sample.Graph.Edges))
	for _, e := range sample.Graph.Edges {
		sampleEdges[e.ID] = e
	}

	data := &PerformanceGraphData{
		ID:          fmt.Sprintf("perf-graph-%d", time.Now().UnixNano()),
		GeneratedAt: time.Now(),
		Nodes:       make([]PerformanceGraphNode, 0, len(baseGraph.Nodes)),
		Edges:       make([]PerformanceGraphEdge, 0, len(baseGraph.Relations)),
		Hotspots:    make([]HotspotInfo, 0),
		Bottlenecks: make([]BottleneckInfo, 0),
	}

	maxHeat := 0.0
	for _, node := range baseGraph.Nodes {
		perfNode := gpm.liveNode(node, tableNodes, mapping, maxQPS)
		if perfNode.Performance.HotspotScore/100 >= hotspotHeat {
			data.Hotspots = append(data.Hotspots, HotspotInfo{
				NodeID:      perfNode.ID,
				Score:       perfNode.Performance.HotspotScore,
				Type:        "latency",
				Description: fmt.Sprintf("%s (%s) averages %.1f ms", perfNode.Label, perfNode.TableName, perfNode.Performance.AverageLatency),
			})
		}
		maxHeat = math.Max(maxHeat, perfNode.Performance.HotspotScore/100)
		data.Nodes = append(data.Nodes, perfNode)
	}

	for _, rel := range baseGraph.Relations {
		perfEdge := liveEdge(rel, sampleEdges, mapping)
		if perfEdge.Performance.PerformanceRank == "poor" {
			data.Bottlenecks = append(data.Bottlenecks, BottleneckInfo{
				ID:          perfEdge.ID,
				Location:    fmt.Sprintf("%s -> %s", perfEdge.SourceID, perfEdge.TargetID),
				Type:        "join",
				Severity:    "high",
				Impact:      perfEdge.Performance.LoadFactor * 100,
				Description: fmt.Sprintf("%s joins average %.1f ms", perfEdge.RelationType, perfEdge.Performance.AverageLatency),
			})
		}
		data.Edges = append(data.Edges, perfEdge)
	}

	iv := sample.Interval
	errorRate := 0.0
	if iv.Queries > 0 {
		errorRate = float64(iv.Errors) / float64(iv.Queries) * 100
	}
	data.GlobalMetrics = &PerformanceGlobalMetrics{
		OverallScore:       math.Round((1 - maxHeat) * 100),
		TotalQueriesPerSec: iv.QPS,
		AverageLatency:     iv.AvgLatencyMs,
		ErrorRate:          errorRate,
		HotspotCount:       len(data.Hotspots),
		BottleneckCount:    len(data.Bottlenecks),
		PerformanceRating:  ratingForLatency(iv.AvgLatencyMs),
		LastUpdated:        sample.Timestamp,
	}
	data.Metadata = GraphMetadata{
		NodeCount:        len(data.Nodes),
		EdgeCount:        len(data.Edges),
		DataSource:       "Live benchmark " + sample.BenchmarkID,
		CollectionPeriod: "1s",
		GenerationTime:   float64(time.Since(started).Milliseconds()),
		Version:          liveGraphVersion,
	}
	return data, nil
}

func (gpm *GraphPerformanceMapper) liveNode(
	node *models.Node,
	tableNodes map[string]ports.LiveGraphNode,
	mapping *ports.LabelTableMapping,
	maxQPS float64,
) PerformanceGraphNode {
	table := ""
	if prop, ok := node.Properties["table_name"].(string); ok && prop != "" {
		table = strings.ToLower(prop)
	} else if lm, ok := mapping.Labels[node.Label]; ok {
		table = lm.Table
	}

	// The node ID convention is unchanged from the Performance Schema path.
	nodeID := fmt.Sprintf("%s_%s", node.Label, table)
	if idProp, ok := node.Properties["id"]; ok {
		nodeID = fmt.Sprintf("%v", idProp)
	}

	perfNode := PerformanceGraphNode{
		ID:        nodeID,
		TableName: table,
		Label:     node.Label,
		Visual: NodeVisualProperties{
			Size:        20,
			Color:       heatHex(0),
			BorderColor: heatHex(0),
			BorderWidth: 2,
			Shape:       "circle",
			Label:       NodeLabelStyle{Text: node.Label, Size: 12, Position: "bottom", Visible: true},
		},
		Issues:          []NodeIssue{},
		Recommendations: []string{},
	}

	sampleNode, ok := tableNodes[table]
	if !ok {
		return perfNode
	}

	heat := sampleNode.Heat
	size := 20.0
	if maxQPS > 0 {
		size = 20 + 60*math.Sqrt(math.Min(1, sampleNode.QPS/maxQPS))
	}
	perfNode.Visual.Size = size
	perfNode.Visual.Color = heatHex(heat)
	perfNode.Visual.ColorIntensity = heat
	perfNode.Visual.BorderColor = heatHex(heat)
	perfNode.Performance = NodePerformanceData{
		QueriesPerSecond: sampleNode.QPS,
		AverageLatency:   sampleNode.AvgLatencyMs,
		TotalQueries:     sampleNode.Queries,
		HotspotScore:     heat * 100,
		LoadScore:        heat * 100,
		PerformanceScore: (1 - heat) * 100,
		TrendDirection:   "stable",
	}
	if heat >= hotspotHeat {
		perfNode.Issues = append(perfNode.Issues, NodeIssue{
			Type:        "high_latency",
			Severity:    "high",
			Description: fmt.Sprintf("Average latency %.1f ms on %s", sampleNode.AvgLatencyMs, table),
			Impact:      heat * 100,
		})
	}
	return perfNode
}

func liveEdge(
	rel *models.Relation,
	sampleEdges map[string]ports.LiveGraphEdge,
	mapping *ports.LabelTableMapping,
) PerformanceGraphEdge {
	edgeID := fmt.Sprintf("%s_%s_%s", rel.Type, rel.From, rel.To)
	if idProp, ok := rel.Properties["id"]; ok {
		edgeID = fmt.Sprintf("%v", idProp)
	}

	perfEdge := PerformanceGraphEdge{
		ID:           edgeID,
		SourceID:     rel.From,
		TargetID:     rel.To,
		RelationType: rel.Type,
		Label:        rel.Type,
		Visual:       EdgeVisualProperties{Thickness: 1, Color: heatHex(0), Style: "solid"},
		Performance:  EdgePerformanceData{PerformanceRank: "excellent"},
		Issues:       []EdgeIssue{},
	}

	rm, ok := mapping.Relationships[rel.Type]
	if !ok {
		return perfEdge
	}
	ids := rm.Edges
	if len(ids) == 0 && rm.Edge != "" {
		ids = []string{rm.Edge}
	}

	// Aggregate over the edges the relationship runs through: worst latency,
	// highest load, summed frequency.
	var worst *ports.LiveGraphEdge
	var frequency, load float64
	found := false
	for _, id := range ids {
		e, exists := sampleEdges[id]
		if !exists {
			continue
		}
		found = true
		frequency += e.QPS
		load = math.Max(load, e.Load)
		if worst == nil || e.AvgLatencyMs > worst.AvgLatencyMs {
			copied := e
			worst = &copied
		}
	}
	if !found || worst == nil {
		return perfEdge
	}

	heat := heatFor(worst.AvgLatencyMs)
	perfEdge.Visual = EdgeVisualProperties{
		Thickness:      1 + 9*math.Min(1, load),
		Color:          heatHex(heat),
		ColorIntensity: heat,
		Style:          "solid",
	}
	perfEdge.Performance = EdgePerformanceData{
		QueryFrequency:  frequency,
		AverageLatency:  worst.AvgLatencyMs,
		LoadFactor:      load,
		PerformanceRank: rankToLegacy(worst.Rank),
	}
	if worst.Rank == "SLOW" {
		perfEdge.Issues = append(perfEdge.Issues, EdgeIssue{
			Type:        "slow_join",
			Severity:    "high",
			Description: fmt.Sprintf("Join %s averages %.1f ms", worst.ID, worst.AvgLatencyMs),
			Impact:      load * 100,
		})
	}
	return perfEdge
}

// rankToLegacy converts the live FAST/MEDIUM/SLOW rank to the rank vocabulary
// the existing overlay understands.
func rankToLegacy(rank string) string {
	switch strings.ToUpper(rank) {
	case "SLOW":
		return "poor"
	case "MEDIUM":
		return "fair"
	default:
		return "excellent"
	}
}

func ratingForLatency(avgMs float64) string {
	switch {
	case avgMs > 500:
		return "poor"
	case avgMs > 100:
		return "fair"
	default:
		return "good"
	}
}

// heatHex maps heat 0..1 to the green-yellow-red colour scale.
func heatHex(heat float64) string {
	heat = math.Max(0, math.Min(1, heat))
	var from, to [3]float64
	var t float64
	if heat < 0.5 {
		from, to, t = hexToRGB(heatGreen), hexToRGB(heatYellow), heat/0.5
	} else {
		from, to, t = hexToRGB(heatYellow), hexToRGB(heatRed), (heat-0.5)/0.5
	}
	return fmt.Sprintf("#%02x%02x%02x",
		int(math.Round(from[0]+(to[0]-from[0])*t)),
		int(math.Round(from[1]+(to[1]-from[1])*t)),
		int(math.Round(from[2]+(to[2]-from[2])*t)))
}

func hexToRGB(hex string) [3]float64 {
	var r, g, b int
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return [3]float64{float64(r), float64(g), float64(b)}
}
