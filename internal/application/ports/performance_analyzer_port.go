package ports

import (
	"context"
	"time"
)

// PerformanceAnalyzerPort defines the interface for performance analysis services
type PerformanceAnalyzerPort interface {
	IdentifyBottlenecks(ctx context.Context, results *BenchmarkResult) ([]PerformanceBottleneck, error)
	AnalyzeCriticalPath(ctx context.Context, graphData *GraphPerformanceData) (*CriticalPathAnalysis, error)
	DetectHotspots(ctx context.Context, metrics []*PerformanceMetrics) ([]HotspotNode, error)

	AnalyzeQueryPatterns(ctx context.Context, queryResults []QueryPerformance) (*QueryPatternAnalysis, error)
	IdentifyInefficiencies(ctx context.Context, queryResults []QueryPerformance) ([]PerformanceIssue, error)

	GenerateOptimizationSuggestions(ctx context.Context, analysis *PerformanceAnalysis) ([]OptimizationSuggestion, error)
	ValidateOptimization(ctx context.Context, suggestion *OptimizationSuggestion) (*OptimizationValidation, error)

	AnalyzeTrends(ctx context.Context, historicalData []PerformanceSnapshot) (*TrendAnalysis, error)
	DetectRegressions(ctx context.Context, current, previous *PerformanceMetrics) ([]PerformanceRegression, error)

	CalculatePerformanceScore(ctx context.Context, metrics *PerformanceMetrics) (*PerformanceScore, error)
	ComparePerformance(ctx context.Context, baseline, current *PerformanceMetrics) (*PerformanceComparison, error)
}

// PerformanceBottleneck represents identified performance bottlenecks
type PerformanceBottleneck struct {
	ID              string             `json:"id"`
	Type            BottleneckType     `json:"type"`
	Severity        SeverityLevel      `json:"severity"`
	Location        BottleneckLocation `json:"location"`
	Description     string             `json:"description"`
	Impact          PerformanceImpact  `json:"impact"`
	Recommendations []string           `json:"recommendations"`
	Confidence      float64            `json:"confidence"` // 0-1
	DetectedAt      time.Time          `json:"detected_at"`
}

// BottleneckType categorizes the type of performance bottleneck
type BottleneckType string

const (
	BottleneckTypeQuery   BottleneckType = "query"
	BottleneckTypeIndex   BottleneckType = "index"
	BottleneckTypeJoin    BottleneckType = "join"
	BottleneckTypeLock    BottleneckType = "lock"
	BottleneckTypeIO      BottleneckType = "io"
	BottleneckTypeCPU     BottleneckType = "cpu"
	BottleneckTypeMemory  BottleneckType = "memory"
	BottleneckTypeNetwork BottleneckType = "network"
	BottleneckTypeSchema  BottleneckType = "schema"
)

// SeverityLevel indicates the severity of a performance issue
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// BottleneckLocation identifies where the bottleneck occurs
type BottleneckLocation struct {
	TableName    string   `json:"table_name,omitempty"`
	QueryPattern string   `json:"query_pattern,omitempty"`
	Relationship string   `json:"relationship,omitempty"`
	IndexName    string   `json:"index_name,omitempty"`
	JoinTables   []string `json:"join_tables,omitempty"`
}

// PerformanceImpact quantifies the impact of a performance issue
type PerformanceImpact struct {
	LatencyIncrease    float64 `json:"latency_increase"`    // Percentage increase
	ThroughputDecrease float64 `json:"throughput_decrease"` // Percentage decrease
	ResourceUsage      float64 `json:"resource_usage"`      // Additional resource consumption
	AffectedQueries    int     `json:"affected_queries"`    // Number of queries affected
	BusinessImpact     string  `json:"business_impact"`     // HIGH, MEDIUM, LOW
}

// CriticalPathAnalysis identifies the slowest execution paths
type CriticalPathAnalysis struct {
	CriticalPaths  []CriticalPath `json:"critical_paths"`
	OverallScore   float64        `json:"overall_score"`
	MaxPathLatency time.Duration  `json:"max_path_latency"`
	AverageLatency time.Duration  `json:"average_latency"`
	AnalyzedAt     time.Time      `json:"analyzed_at"`
}

// CriticalPath represents a slow execution path through the database
type CriticalPath struct {
	ID           string                  `json:"id"`
	Path         []PathNode              `json:"path"`
	TotalLatency time.Duration           `json:"total_latency"`
	Frequency    int64                   `json:"frequency"`
	Impact       float64                 `json:"impact"` // Combined latency * frequency
	Bottlenecks  []PerformanceBottleneck `json:"bottlenecks"`
}

// PathNode represents a node in a critical path
type PathNode struct {
	TableName     string        `json:"table_name"`
	Operation     string        `json:"operation"`
	Latency       time.Duration `json:"latency"`
	RowsProcessed int64         `json:"rows_processed"`
	IndexUsed     string        `json:"index_used,omitempty"`
}

// HotspotNode represents a table or relationship under heavy load
type HotspotNode struct {
	NodeID          string             `json:"node_id"`
	NodeType        string             `json:"node_type"` // table, relationship
	TableName       string             `json:"table_name"`
	HotspotScore    float64            `json:"hotspot_score"` // 0-100
	LoadMetrics     HotspotLoadMetrics `json:"load_metrics"`
	Issues          []PerformanceIssue `json:"issues"`
	Recommendations []string           `json:"recommendations"`
	TrendDirection  TrendDirection     `json:"trend_direction"`
}

// HotspotLoadMetrics contains load metrics for hotspot analysis
type HotspotLoadMetrics struct {
	QueriesPerSecond float64 `json:"queries_per_second"`
	AverageLatency   float64 `json:"average_latency"`
	CPUUtilization   float64 `json:"cpu_utilization"`
	IOUtilization    float64 `json:"io_utilization"`
	LockContention   float64 `json:"lock_contention"`
	CacheHitRatio    float64 `json:"cache_hit_ratio"`
}

// QueryPatternAnalysis provides insights into query access patterns
type QueryPatternAnalysis struct {
	PatternGroups   []QueryPatternGroup `json:"pattern_groups"`
	CommonPatterns  []QueryPattern      `json:"common_patterns"`
	AntiPatterns    []QueryAntiPattern  `json:"anti_patterns"`
	Recommendations []string            `json:"recommendations"`
	AnalyzedQueries int                 `json:"analyzed_queries"`
	AnalyzedAt      time.Time           `json:"analyzed_at"`
}

// QueryPatternGroup groups similar query patterns together
type QueryPatternGroup struct {
	PatternID       string            `json:"pattern_id"`
	PatternTemplate string            `json:"pattern_template"`
	QueryCount      int64             `json:"query_count"`
	TotalLatency    time.Duration     `json:"total_latency"`
	AverageLatency  time.Duration     `json:"average_latency"`
	TablesInvolved  []string          `json:"tables_involved"`
	Performance     PerformanceRating `json:"performance"`
	Examples        []string          `json:"examples,omitempty"`
}

// QueryPattern represents a common query access pattern
type QueryPattern struct {
	Type         string            `json:"type"`
	Description  string            `json:"description"`
	Frequency    int64             `json:"frequency"`
	Performance  PerformanceRating `json:"performance"`
	Optimization []string          `json:"optimization,omitempty"`
}

// QueryAntiPattern represents inefficient query patterns
type QueryAntiPattern struct {
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Impact      SeverityLevel `json:"impact"`
	Examples    []string      `json:"examples"`
	Solutions   []string      `json:"solutions"`
	Frequency   int64         `json:"frequency"`
}

// PerformanceIssue represents a specific performance problem
type PerformanceIssue struct {
	ID          string             `json:"id"`
	Type        IssueType          `json:"type"`
	Severity    SeverityLevel      `json:"severity"`
	Description string             `json:"description"`
	Location    BottleneckLocation `json:"location"`
	Impact      PerformanceImpact  `json:"impact"`
	Solution    string             `json:"solution"`
	Priority    int                `json:"priority"` // 1-10
	Effort      EffortLevel        `json:"effort"`
}

// IssueType categorizes performance issues
type IssueType string

const (
	IssueTypeMissingIndex     IssueType = "missing_index"
	IssueTypeInefficiencyJoin IssueType = "inefficient_join"
	IssueTypeFullTableScan    IssueType = "full_table_scan"
	IssueTypeSlowQuery        IssueType = "slow_query"
	IssueTypeLockContention   IssueType = "lock_contention"
	IssueTypeDeadlock         IssueType = "deadlock"
	IssueTypeResourceHog      IssueType = "resource_hog"
	IssueTypeSuboptimalSchema IssueType = "suboptimal_schema"
)

// EffortLevel indicates implementation effort for fixes
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
)

// OptimizationSuggestion provides actionable optimization recommendations
type OptimizationSuggestion struct {
	ID             string                     `json:"id"`
	Type           OptimizationType           `json:"type"`
	Title          string                     `json:"title"`
	Description    string                     `json:"description"`
	Impact         OptimizationImpact         `json:"impact"`
	Implementation OptimizationImplementation `json:"implementation"`
	Prerequisites  []string                   `json:"prerequisites,omitempty"`
	Risks          []string                   `json:"risks,omitempty"`
	Priority       int                        `json:"priority"`   // 1-10
	Confidence     float64                    `json:"confidence"` // 0-1
}

// OptimizationType categorizes optimization suggestions
type OptimizationType string

const (
	OptimizationTypeIndex           OptimizationType = "index"
	OptimizationTypeQuery           OptimizationType = "query"
	OptimizationTypeSchema          OptimizationType = "schema"
	OptimizationTypeConfiguration   OptimizationType = "configuration"
	OptimizationTypePartitioning    OptimizationType = "partitioning"
	OptimizationTypeDenormalization OptimizationType = "denormalization"
	OptimizationTypeCaching         OptimizationType = "caching"
)

// OptimizationImpact quantifies expected improvement
type OptimizationImpact struct {
	LatencyImprovement    float64 `json:"latency_improvement"`    // Expected % improvement
	ThroughputImprovement float64 `json:"throughput_improvement"` // Expected % improvement
	ResourceSavings       float64 `json:"resource_savings"`       // Expected % resource reduction
	AffectedQueries       int     `json:"affected_queries"`
	EstimatedBenefit      string  `json:"estimated_benefit"` // HIGH, MEDIUM, LOW
}

// OptimizationImplementation provides implementation details
type OptimizationImplementation struct {
	Steps           []string      `json:"steps"`
	Code            string        `json:"code,omitempty"` // SQL, configuration, etc.
	EstimatedTime   time.Duration `json:"estimated_time"`
	DifficultyLevel EffortLevel   `json:"difficulty_level"`
	Rollback        string        `json:"rollback,omitempty"`
}

// OptimizationValidation contains validation results for optimization
type OptimizationValidation struct {
	IsValid          bool               `json:"is_valid"`
	ValidationErrors []string           `json:"validation_errors,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
	Prerequisites    []string           `json:"prerequisites,omitempty"`
	Impact           OptimizationImpact `json:"impact"`
}

// TrendAnalysis provides insights into performance trends
type TrendAnalysis struct {
	TimeRange    TimeRange               `json:"time_range"`
	OverallTrend TrendDirection          `json:"overall_trend"`
	TrendMetrics []TrendMetric           `json:"trend_metrics"`
	Anomalies    []PerformanceAnomaly    `json:"anomalies"`
	Predictions  []PerformancePrediction `json:"predictions,omitempty"`
	AnalyzedAt   time.Time               `json:"analyzed_at"`
}

// TrendDirection indicates the direction of a performance trend
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendStable    TrendDirection = "stable"
	TrendDegrading TrendDirection = "degrading"
	TrendVolatile  TrendDirection = "volatile"
)

// TrendMetric represents trend data for a specific metric
type TrendMetric struct {
	MetricName   string         `json:"metric_name"`
	Trend        TrendDirection `json:"trend"`
	ChangeRate   float64        `json:"change_rate"`  // % change over time period
	Confidence   float64        `json:"confidence"`   // 0-1
	Significance string         `json:"significance"` // SIGNIFICANT, MINOR, NEGLIGIBLE
}

// PerformanceAnomaly represents unusual performance behavior
type PerformanceAnomaly struct {
	ID              string        `json:"id"`
	Type            AnomalyType   `json:"type"`
	DetectedAt      time.Time     `json:"detected_at"`
	Duration        time.Duration `json:"duration"`
	Severity        SeverityLevel `json:"severity"`
	Description     string        `json:"description"`
	MetricsAffected []string      `json:"metrics_affected"`
	PossibleCause   string        `json:"possible_cause,omitempty"`
}

// AnomalyType categorizes performance anomalies
type AnomalyType string

const (
	AnomalyTypeSpike       AnomalyType = "spike"
	AnomalyTypeDrop        AnomalyType = "drop"
	AnomalyTypeOscillation AnomalyType = "oscillation"
	AnomalyTypeDrift       AnomalyType = "drift"
)

// PerformancePrediction provides future performance predictions
type PerformancePrediction struct {
	MetricName     string        `json:"metric_name"`
	PredictedValue float64       `json:"predicted_value"`
	Confidence     float64       `json:"confidence"`
	TimeHorizon    time.Duration `json:"time_horizon"`
	Scenario       string        `json:"scenario"` // BEST, WORST, LIKELY
}

// PerformanceRegression identifies performance regressions
type PerformanceRegression struct {
	MetricName       string        `json:"metric_name"`
	BaselineValue    float64       `json:"baseline_value"`
	CurrentValue     float64       `json:"current_value"`
	RegressionAmount float64       `json:"regression_amount"` // % degradation
	Severity         SeverityLevel `json:"severity"`
	PossibleCause    string        `json:"possible_cause,omitempty"`
	DetectedAt       time.Time     `json:"detected_at"`
}

// Additional supporting types

// PerformanceRating categorizes performance levels
type PerformanceRating string

const (
	PerformanceExcellent PerformanceRating = "excellent"
	PerformanceGood      PerformanceRating = "good"
	PerformanceFair      PerformanceRating = "fair"
	PerformancePoor      PerformanceRating = "poor"
	PerformanceCritical  PerformanceRating = "critical"
)

// TimeRange represents a time range for analysis
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// PerformanceSnapshot represents performance state at a point in time
type PerformanceSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   *PerformanceMetrics    `json:"metrics"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// PerformanceScore provides an overall performance rating
type PerformanceScore struct {
	OverallScore    float64                  `json:"overall_score"` // 0-100
	ComponentScores map[string]float64       `json:"component_scores"`
	Rating          PerformanceRating        `json:"rating"`
	Factors         []PerformanceScoreFactor `json:"factors"`
	CalculatedAt    time.Time                `json:"calculated_at"`
}

// PerformanceScoreFactor explains what contributes to the score
type PerformanceScoreFactor struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Value       float64 `json:"value"`
	Impact      string  `json:"impact"` // POSITIVE, NEGATIVE, NEUTRAL
	Description string  `json:"description"`
}

// PerformanceComparison compares two performance states
type PerformanceComparison struct {
	BaselineScore float64             `json:"baseline_score"`
	CurrentScore  float64             `json:"current_score"`
	Improvement   float64             `json:"improvement"` // % change
	Changes       []PerformanceChange `json:"changes"`
	Summary       string              `json:"summary"`
	ComparedAt    time.Time           `json:"compared_at"`
}

// PerformanceChange represents a change in performance metrics
type PerformanceChange struct {
	MetricName      string         `json:"metric_name"`
	BaselineValue   float64        `json:"baseline_value"`
	CurrentValue    float64        `json:"current_value"`
	ChangeAmount    float64        `json:"change_amount"` // % change
	ChangeDirection TrendDirection `json:"change_direction"`
	Significance    string         `json:"significance"`
}

// GraphPerformanceData contains performance data for graph analysis
type GraphPerformanceData struct {
	Nodes         []NodePerformanceData `json:"nodes"`
	Edges         []EdgePerformanceData `json:"edges"`
	GlobalMetrics *PerformanceMetrics   `json:"global_metrics"`
	Timestamp     time.Time             `json:"timestamp"`
}

// NodePerformanceData contains performance data for a graph node (table)
type NodePerformanceData struct {
	NodeID       string              `json:"node_id"`
	TableName    string              `json:"table_name"`
	Metrics      *PerformanceMetrics `json:"metrics"`
	HotspotScore float64             `json:"hotspot_score"`
}

// EdgePerformanceData contains performance data for graph edges (relationships)
type EdgePerformanceData struct {
	EdgeID       string              `json:"edge_id"`
	SourceTable  string              `json:"source_table"`
	TargetTable  string              `json:"target_table"`
	RelationType string              `json:"relation_type"`
	Metrics      *PerformanceMetrics `json:"metrics"`
	LoadFactor   float64             `json:"load_factor"`
}

// PerformanceAnalysis contains comprehensive performance analysis results
type PerformanceAnalysis struct {
	OverallScore  *PerformanceScore       `json:"overall_score"`
	Bottlenecks   []PerformanceBottleneck `json:"bottlenecks"`
	CriticalPath  *CriticalPathAnalysis   `json:"critical_path"`
	Hotspots      []HotspotNode           `json:"hotspots"`
	QueryPatterns *QueryPatternAnalysis   `json:"query_patterns"`
	Issues        []PerformanceIssue      `json:"issues"`
	TrendAnalysis *TrendAnalysis          `json:"trend_analysis,omitempty"`
	AnalyzedAt    time.Time               `json:"analyzed_at"`
}
