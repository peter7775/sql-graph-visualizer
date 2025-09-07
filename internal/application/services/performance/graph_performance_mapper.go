package performance

import (
	"context"
	"fmt"
	"time"

	"sql-graph-visualizer/internal/domain/models"

	"github.com/sirupsen/logrus"
)

// GraphPerformanceMapper maps performance data to graph visualization elements
type GraphPerformanceMapper struct {
	logger    *logrus.Logger
	config    *GraphPerformanceMapperConfig
	psAdapter *PerformanceSchemaAdapter
	analyzer  *PerformanceAnalyzer
}

// GraphPerformanceMapperConfig contains configuration for graph performance mapping
type GraphPerformanceMapperConfig struct {
	// Visual encoding settings
	EdgeThickness EdgeThicknessConfig `yaml:"edge_thickness" json:"edge_thickness"`
	EdgeColor     EdgeColorConfig     `yaml:"edge_color" json:"edge_color"`
	NodeSize      NodeSizeConfig      `yaml:"node_size" json:"node_size"`
	NodeColor     NodeColorConfig     `yaml:"node_color" json:"node_color"`
	Animation     AnimationConfig     `yaml:"animation" json:"animation"`

	// Performance thresholds
	HotspotThreshold   float64 `yaml:"hotspot_threshold" json:"hotspot_threshold"`
	SlowQueryThreshold float64 `yaml:"slow_query_threshold" json:"slow_query_threshold"`
	HighLoadThreshold  float64 `yaml:"high_load_threshold" json:"high_load_threshold"`

	// Update settings
	UpdateInterval       time.Duration `yaml:"update_interval" json:"update_interval"`
	HistoryRetention     time.Duration `yaml:"history_retention" json:"history_retention"`
	MaxConcurrentUpdates int           `yaml:"max_concurrent_updates" json:"max_concurrent_updates"`
}

// EdgeThicknessConfig controls edge thickness mapping
type EdgeThicknessConfig struct {
	Metric       string  `yaml:"metric" json:"metric"`
	Scale        string  `yaml:"scale" json:"scale"`
	MinThickness float64 `yaml:"min_thickness" json:"min_thickness"`
	MaxThickness float64 `yaml:"max_thickness" json:"max_thickness"`
	Multiplier   float64 `yaml:"multiplier" json:"multiplier"`
}

// EdgeColorConfig controls edge color mapping
type EdgeColorConfig struct {
	Metric     string                 `yaml:"metric" json:"metric"`
	ColorScale string                 `yaml:"color_scale" json:"color_scale"`
	Thresholds map[string]interface{} `yaml:"thresholds" json:"thresholds"`
}

// NodeSizeConfig controls node size mapping
type NodeSizeConfig struct {
	Metric  string  `yaml:"metric" json:"metric"`
	MinSize float64 `yaml:"min_size" json:"min_size"`
	MaxSize float64 `yaml:"max_size" json:"max_size"`
	Scale   string  `yaml:"scale" json:"scale"`
}

// NodeColorConfig controls node color mapping
type NodeColorConfig struct {
	Metric     string                 `yaml:"metric" json:"metric"`
	ColorScale string                 `yaml:"color_scale" json:"color_scale"`
	Thresholds map[string]interface{} `yaml:"thresholds" json:"thresholds"`
}

// AnimationConfig controls animation settings
type AnimationConfig struct {
	ShowDataFlow   bool    `yaml:"show_data_flow" json:"show_data_flow"`
	SpeedBasedOn   string  `yaml:"speed_based_on" json:"speed_based_on"`
	AnimationSpeed float64 `yaml:"animation_speed" json:"animation_speed"`
	ParticleCount  int     `yaml:"particle_count" json:"particle_count"`
}

// PerformanceGraphData contains performance-enhanced graph data
type PerformanceGraphData struct {
	ID            string                    `json:"id"`
	GeneratedAt   time.Time                 `json:"generated_at"`
	Nodes         []PerformanceGraphNode    `json:"nodes"`
	Edges         []PerformanceGraphEdge    `json:"edges"`
	GlobalMetrics *PerformanceGlobalMetrics `json:"global_metrics"`
	Hotspots      []HotspotInfo             `json:"hotspots"`
	Bottlenecks   []BottleneckInfo          `json:"bottlenecks"`
	Metadata      GraphMetadata             `json:"metadata"`
}

// PerformanceGraphNode represents a node with performance data
type PerformanceGraphNode struct {
	ID              string               `json:"id"`
	TableName       string               `json:"table_name"`
	Label           string               `json:"label"`
	Position        NodePosition         `json:"position"`
	Visual          NodeVisualProperties `json:"visual"`
	Performance     NodePerformanceData  `json:"performance"`
	Issues          []NodeIssue          `json:"issues"`
	Recommendations []string             `json:"recommendations"`
}

// PerformanceGraphEdge represents an edge with performance data
type PerformanceGraphEdge struct {
	ID           string               `json:"id"`
	SourceID     string               `json:"source_id"`
	TargetID     string               `json:"target_id"`
	RelationType string               `json:"relation_type"`
	Label        string               `json:"label"`
	Visual       EdgeVisualProperties `json:"visual"`
	Performance  EdgePerformanceData  `json:"performance"`
	QueryPattern string               `json:"query_pattern"`
	Issues       []EdgeIssue          `json:"issues"`
}

// NodePosition represents node coordinates
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

// NodeVisualProperties controls node appearance
type NodeVisualProperties struct {
	Size           float64        `json:"size"`
	Color          string         `json:"color"`
	ColorIntensity float64        `json:"color_intensity"`
	BorderColor    string         `json:"border_color"`
	BorderWidth    float64        `json:"border_width"`
	Shape          string         `json:"shape"`
	Icon           string         `json:"icon,omitempty"`
	Label          NodeLabelStyle `json:"label"`
	Effects        []VisualEffect `json:"effects,omitempty"`
}

// EdgeVisualProperties controls edge appearance
type EdgeVisualProperties struct {
	Thickness      float64        `json:"thickness"`
	Color          string         `json:"color"`
	ColorIntensity float64        `json:"color_intensity"`
	Style          string         `json:"style"` // solid, dashed, dotted
	Arrow          ArrowStyle     `json:"arrow"`
	Animation      EdgeAnimation  `json:"animation,omitempty"`
	Effects        []VisualEffect `json:"effects,omitempty"`
}

// NodeLabelStyle controls label appearance
type NodeLabelStyle struct {
	Text     string  `json:"text"`
	Size     float64 `json:"size"`
	Color    string  `json:"color"`
	Position string  `json:"position"`
	Visible  bool    `json:"visible"`
}

// ArrowStyle controls arrow appearance
type ArrowStyle struct {
	Size  float64 `json:"size"`
	Style string  `json:"style"`
	Color string  `json:"color"`
}

// EdgeAnimation controls edge animations
type EdgeAnimation struct {
	Type         string  `json:"type"`
	Speed        float64 `json:"speed"`
	Direction    string  `json:"direction"`
	ParticleSize float64 `json:"particle_size"`
	Enabled      bool    `json:"enabled"`
}

// VisualEffect represents visual effects
type VisualEffect struct {
	Type       string                 `json:"type"`
	Intensity  float64                `json:"intensity"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// NodePerformanceData contains performance metrics for nodes
type NodePerformanceData struct {
	QueriesPerSecond float64           `json:"queries_per_second"`
	AverageLatency   float64           `json:"average_latency"`
	TotalQueries     int64             `json:"total_queries"`
	ErrorRate        float64           `json:"error_rate"`
	HotspotScore     float64           `json:"hotspot_score"`
	LoadScore        float64           `json:"load_score"`
	PerformanceScore float64           `json:"performance_score"`
	IndexEfficiency  float64           `json:"index_efficiency"`
	ResourceUsage    ResourceUsageData `json:"resource_usage"`
	TrendDirection   string            `json:"trend_direction"`
}

// EdgePerformanceData contains performance metrics for edges
type EdgePerformanceData struct {
	QueryFrequency  float64 `json:"query_frequency"`
	AverageLatency  float64 `json:"average_latency"`
	TotalExecutions int64   `json:"total_executions"`
	ErrorRate       float64 `json:"error_rate"`
	LoadFactor      float64 `json:"load_factor"`
	PerformanceRank string  `json:"performance_rank"`
	IndexUsage      bool    `json:"index_usage"`
	JoinEfficiency  float64 `json:"join_efficiency"`
}

// ResourceUsageData contains resource utilization data
type ResourceUsageData struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryPercent  float64 `json:"memory_percent"`
	IOUtilization  float64 `json:"io_utilization"`
	LockContention float64 `json:"lock_contention"`
}

// HotspotInfo contains hotspot information
type HotspotInfo struct {
	NodeID      string  `json:"node_id"`
	Score       float64 `json:"score"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
}

// BottleneckInfo contains bottleneck information
type BottleneckInfo struct {
	ID          string  `json:"id"`
	Location    string  `json:"location"`
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
}

// NodeIssue represents a performance issue with a node
type NodeIssue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`
}

// EdgeIssue represents a performance issue with an edge
type EdgeIssue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Impact      float64 `json:"impact"`
}

// PerformanceGlobalMetrics contains global performance metrics
type PerformanceGlobalMetrics struct {
	OverallScore       float64   `json:"overall_score"`
	TotalQueriesPerSec float64   `json:"total_queries_per_sec"`
	AverageLatency     float64   `json:"average_latency"`
	ErrorRate          float64   `json:"error_rate"`
	HotspotCount       int       `json:"hotspot_count"`
	BottleneckCount    int       `json:"bottleneck_count"`
	PerformanceRating  string    `json:"performance_rating"`
	LastUpdated        time.Time `json:"last_updated"`
}

// GraphMetadata contains graph metadata
type GraphMetadata struct {
	NodeCount        int     `json:"node_count"`
	EdgeCount        int     `json:"edge_count"`
	DataSource       string  `json:"data_source"`
	CollectionPeriod string  `json:"collection_period"`
	GenerationTime   float64 `json:"generation_time_ms"`
	Version          string  `json:"version"`
}

// NewGraphPerformanceMapper creates a new graph performance mapper
func NewGraphPerformanceMapper(
	logger *logrus.Logger,
	config *GraphPerformanceMapperConfig,
	psAdapter *PerformanceSchemaAdapter,
	analyzer *PerformanceAnalyzer,
) *GraphPerformanceMapper {
	if config == nil {
		config = defaultGraphPerformanceMapperConfig()
	}

	return &GraphPerformanceMapper{
		logger:    logger,
		config:    config,
		psAdapter: psAdapter,
		analyzer:  analyzer,
	}
}

// MapPerformanceToGraph maps performance data to graph visualization
func (gpm *GraphPerformanceMapper) MapPerformanceToGraph(
	ctx context.Context,
	baseGraph *models.Graph,
	performanceData *PerformanceSchemaData,
) (*PerformanceGraphData, error) {
	startTime := time.Now()

	if baseGraph == nil {
		return nil, fmt.Errorf("base graph is required")
	}

	if performanceData == nil {
		return nil, fmt.Errorf("performance data is required")
	}

	// Create performance graph data structure
	perfGraph := &PerformanceGraphData{
		ID:          fmt.Sprintf("perf-graph-%d", time.Now().Unix()),
		GeneratedAt: time.Now(),
		Nodes:       make([]PerformanceGraphNode, 0),
		Edges:       make([]PerformanceGraphEdge, 0),
		Hotspots:    make([]HotspotInfo, 0),
		Bottlenecks: make([]BottleneckInfo, 0),
	}

	// Build table performance map
	tablePerformanceMap := gpm.buildTablePerformanceMap(performanceData)

	// Map nodes with performance data
	if err := gpm.mapNodesToPerformance(ctx, baseGraph, tablePerformanceMap, perfGraph); err != nil {
		return nil, fmt.Errorf("failed to map nodes: %w", err)
	}

	// Map edges with performance data
	if err := gpm.mapEdgesToPerformance(ctx, baseGraph, performanceData, perfGraph); err != nil {
		return nil, fmt.Errorf("failed to map edges: %w", err)
	}

	// Calculate global metrics
	gpm.calculateGlobalMetrics(perfGraph)

	// Identify hotspots and bottlenecks
	gpm.identifyHotspotsAndBottlenecks(perfGraph)

	// Set metadata
	perfGraph.Metadata = GraphMetadata{
		NodeCount:        len(perfGraph.Nodes),
		EdgeCount:        len(perfGraph.Edges),
		DataSource:       "MySQL Performance Schema",
		CollectionPeriod: gpm.config.UpdateInterval.String(),
		GenerationTime:   float64(time.Since(startTime).Milliseconds()),
		Version:          "2.0",
	}

	gpm.logger.WithFields(logrus.Fields{
		"nodes":           len(perfGraph.Nodes),
		"edges":           len(perfGraph.Edges),
		"hotspots":        len(perfGraph.Hotspots),
		"bottlenecks":     len(perfGraph.Bottlenecks),
		"generation_time": perfGraph.Metadata.GenerationTime,
	}).Info("Performance graph mapping completed")

	return perfGraph, nil
}

// BuildTablePerformanceMap creates a map of table performance data
func (gpm *GraphPerformanceMapper) buildTablePerformanceMap(data *PerformanceSchemaData) map[string]*TablePerformanceInfo {
	tableMap := make(map[string]*TablePerformanceInfo)

	// Process statement statistics
	for _, stmt := range data.StatementStats {
		for _, tableName := range gpm.extractTableNames(stmt.DigestText) {
			if info, exists := tableMap[tableName]; exists {
				gpm.aggregateTablePerformance(info, &stmt)
			} else {
				tableMap[tableName] = gpm.createTablePerformanceInfo(tableName, &stmt)
			}
		}
	}

	// Process table I/O statistics
	for _, tableIO := range data.TableIOStats {
		tableName := fmt.Sprintf("%s.%s", tableIO.SchemaName, tableIO.TableName)
		if info, exists := tableMap[tableName]; exists {
			gpm.enhanceWithIOStats(info, &tableIO)
		}
	}

	return tableMap
}

// TablePerformanceInfo aggregates performance data for a table
type TablePerformanceInfo struct {
	TableName        string
	QueriesPerSecond float64
	AverageLatency   float64
	TotalQueries     int64
	ErrorRate        float64
	IndexEfficiency  float64
	ResourceUsage    ResourceUsageData
	Issues           []NodeIssue
}

// Private helper methods

func (gpm *GraphPerformanceMapper) mapNodesToPerformance(
	ctx context.Context,
	baseGraph *models.Graph,
	tableMap map[string]*TablePerformanceInfo,
	perfGraph *PerformanceGraphData,
) error {
	for _, node := range baseGraph.Nodes {
		perfNode := gpm.createPerformanceNode(node, tableMap)
		perfGraph.Nodes = append(perfGraph.Nodes, perfNode)
	}
	return nil
}

func (gpm *GraphPerformanceMapper) mapEdgesToPerformance(
	ctx context.Context,
	baseGraph *models.Graph,
	performanceData *PerformanceSchemaData,
	perfGraph *PerformanceGraphData,
) error {
	for _, edge := range baseGraph.Relations {
		perfEdge := gpm.createPerformanceEdge(edge, performanceData)
		perfGraph.Edges = append(perfGraph.Edges, perfEdge)
	}
	return nil
}

func (gpm *GraphPerformanceMapper) createPerformanceNode(
	node *models.Node,
	tableMap map[string]*TablePerformanceInfo,
) PerformanceGraphNode {
	// Extract table name from node properties or label
	tableName := node.Label
	if tableNameProp, exists := node.Properties["table_name"]; exists {
		if tn, ok := tableNameProp.(string); ok {
			tableName = tn
		}
	}

	// Generate ID from label if not directly available
	nodeID := fmt.Sprintf("%s_%s", node.Label, tableName)
	if idProp, exists := node.Properties["id"]; exists {
		nodeID = fmt.Sprintf("%v", idProp)
	}

	// Get performance data for this table
	var perfInfo *TablePerformanceInfo
	if info, exists := tableMap[tableName]; exists {
		perfInfo = info
	} else {
		// Create default performance info
		perfInfo = &TablePerformanceInfo{
			TableName: tableName,
		}
	}

	// Calculate visual properties
	visual := gpm.calculateNodeVisualProperties(perfInfo)
	performance := gpm.mapNodePerformanceData(perfInfo)

	// Extract position from properties with defaults
	var x, y float64 = 0, 0
	if xProp, exists := node.Properties["x"]; exists {
		if xVal, ok := xProp.(float64); ok {
			x = xVal
		} else if xInt, ok := xProp.(int); ok {
			x = float64(xInt)
		}
	}
	if yProp, exists := node.Properties["y"]; exists {
		if yVal, ok := yProp.(float64); ok {
			y = yVal
		} else if yInt, ok := yProp.(int); ok {
			y = float64(yInt)
		}
	}

	return PerformanceGraphNode{
		ID:        nodeID,
		TableName: tableName,
		Label:     node.Label,
		Position: NodePosition{
			X: x,
			Y: y,
		},
		Visual:          visual,
		Performance:     performance,
		Issues:          perfInfo.Issues,
		Recommendations: gpm.generateNodeRecommendations(perfInfo),
	}
}

func (gpm *GraphPerformanceMapper) createPerformanceEdge(
	edge *models.Relation,
	performanceData *PerformanceSchemaData,
) PerformanceGraphEdge {
	// Find relevant query performance for this relationship
	edgePerf := gpm.findEdgePerformanceData(edge, performanceData)

	// Calculate visual properties
	visual := gpm.calculateEdgeVisualProperties(edgePerf)

	// Generate edge ID from type and endpoints
	edgeID := fmt.Sprintf("%s_%s_%s", edge.Type, edge.From, edge.To)
	if idProp, exists := edge.Properties["id"]; exists {
		edgeID = fmt.Sprintf("%v", idProp)
	}

	return PerformanceGraphEdge{
		ID:           edgeID,
		SourceID:     edge.From,
		TargetID:     edge.To,
		RelationType: edge.Type,
		Label:        edge.Type,
		Visual:       visual,
		Performance:  edgePerf,
		QueryPattern: gpm.generateQueryPattern(edge),
		Issues:       gpm.identifyEdgeIssues(edgePerf),
	}
}

// Stub implementations for helper methods
func (gpm *GraphPerformanceMapper) extractTableNames(digestText string) []string { return []string{} }
func (gpm *GraphPerformanceMapper) aggregateTablePerformance(info *TablePerformanceInfo, stmt *StatementStatistic) {
}
func (gpm *GraphPerformanceMapper) createTablePerformanceInfo(tableName string, stmt *StatementStatistic) *TablePerformanceInfo {
	return nil
}
func (gpm *GraphPerformanceMapper) generateNodeRecommendations(perfInfo *TablePerformanceInfo) []string {
	return []string{}
}
func (gpm *GraphPerformanceMapper) enhanceWithIOStats(info *TablePerformanceInfo, tableIO *TableIOStatistic) {
}
func (gpm *GraphPerformanceMapper) calculateNodeVisualProperties(info *TablePerformanceInfo) NodeVisualProperties {
	return NodeVisualProperties{}
}
func (gpm *GraphPerformanceMapper) mapNodePerformanceData(info *TablePerformanceInfo) NodePerformanceData {
	return NodePerformanceData{}
}
func (gpm *GraphPerformanceMapper) findEdgePerformanceData(edge *models.Relation, data *PerformanceSchemaData) EdgePerformanceData {
	return EdgePerformanceData{}
}
func (gpm *GraphPerformanceMapper) calculateEdgeVisualProperties(edgePerf EdgePerformanceData) EdgeVisualProperties {
	return EdgeVisualProperties{}
}
func (gpm *GraphPerformanceMapper) generateQueryPattern(edge *models.Relation) string { return "" }
func (gpm *GraphPerformanceMapper) identifyEdgeIssues(edgePerf EdgePerformanceData) []EdgeIssue {
	return []EdgeIssue{}
}
func (gpm *GraphPerformanceMapper) calculateGlobalMetrics(perfGraph *PerformanceGraphData)         {}
func (gpm *GraphPerformanceMapper) identifyHotspotsAndBottlenecks(perfGraph *PerformanceGraphData) {}

// Default configuration
func defaultGraphPerformanceMapperConfig() *GraphPerformanceMapperConfig {
	return &GraphPerformanceMapperConfig{
		EdgeThickness: EdgeThicknessConfig{
			Metric:       "query_frequency",
			Scale:        "linear",
			MinThickness: 1.0,
			MaxThickness: 10.0,
			Multiplier:   1.0,
		},
		EdgeColor: EdgeColorConfig{
			Metric:     "avg_execution_time",
			ColorScale: "green_yellow_red",
			Thresholds: map[string]interface{}{
				"fast":   "< 100ms",
				"medium": "100ms - 1s",
				"slow":   "> 1s",
			},
		},
		NodeSize: NodeSizeConfig{
			Metric:  "total_queries_involved",
			MinSize: 20.0,
			MaxSize: 100.0,
			Scale:   "sqrt",
		},
		NodeColor: NodeColorConfig{
			Metric:     "avg_response_time",
			ColorScale: "green_yellow_red",
			Thresholds: map[string]interface{}{
				"good": "< 50ms",
				"ok":   "50ms - 200ms",
				"slow": "> 200ms",
			},
		},
		Animation: AnimationConfig{
			ShowDataFlow:   true,
			SpeedBasedOn:   "query_frequency",
			AnimationSpeed: 1.0,
			ParticleCount:  10,
		},
		HotspotThreshold:     70.0,
		SlowQueryThreshold:   200.0,
		HighLoadThreshold:    80.0,
		UpdateInterval:       5 * time.Second,
		HistoryRetention:     1 * time.Hour,
		MaxConcurrentUpdates: 3,
	}
}
