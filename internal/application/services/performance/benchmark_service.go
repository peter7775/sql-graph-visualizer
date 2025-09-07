package performance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// BenchmarkService provides performance benchmarking capabilities
type BenchmarkService struct {
	// Dependencies
	mysqlRepo      ports.MySQLPort
	postgresqlRepo ports.PostgreSQLPort
	neo4jRepo      ports.Neo4jPort
	analyzer       ports.PerformanceAnalyzerPort
	logger         *logrus.Logger

	// Benchmark tools
	tools      map[string]ports.BenchmarkToolPort
	toolsMutex sync.RWMutex

	// State management
	activeRuns map[string]*BenchmarkExecution
	runsMutex  sync.RWMutex

	// Configuration
	config *BenchmarkServiceConfig
}

// BenchmarkServiceConfig contains service configuration
type BenchmarkServiceConfig struct {
	// Execution limits
	MaxConcurrentRuns int           `yaml:"max_concurrent_runs" json:"max_concurrent_runs"`
	DefaultTimeout    time.Duration `yaml:"default_timeout" json:"default_timeout"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`

	// Storage settings
	RetainResults      time.Duration `yaml:"retain_results" json:"retain_results"`
	MaxResultsInMemory int           `yaml:"max_results_in_memory" json:"max_results_in_memory"`

	// Safety limits
	MaxTableSize int           `yaml:"max_table_size" json:"max_table_size"`
	MaxDuration  time.Duration `yaml:"max_duration" json:"max_duration"`
	MaxThreads   int           `yaml:"max_threads" json:"max_threads"`

	// Tool configurations
	EnabledTools       []string               `yaml:"enabled_tools" json:"enabled_tools"`
	ToolConfigurations map[string]interface{} `yaml:"tool_configurations" json:"tool_configurations"`
}

// BenchmarkExecution tracks a running benchmark
type BenchmarkExecution struct {
	ID         string
	Config     ports.BenchmarkConfig
	StartTime  time.Time
	Status     ports.BenchmarkStatus
	Tool       ports.BenchmarkToolPort
	Context    context.Context
	CancelFunc context.CancelFunc
	Result     *ports.BenchmarkResult
	Progress   *BenchmarkProgress
	mutex      sync.RWMutex
}

// BenchmarkProgress tracks execution progress
type BenchmarkProgress struct {
	CurrentPhase       string        `json:"current_phase"`
	CompletedSteps     int           `json:"completed_steps"`
	TotalSteps         int           `json:"total_steps"`
	ElapsedTime        time.Duration `json:"elapsed_time"`
	EstimatedRemaining time.Duration `json:"estimated_remaining"`
	LastUpdate         time.Time     `json:"last_update"`
	Messages           []string      `json:"messages"`
}

// NewBenchmarkService creates a new benchmark service instance
func NewBenchmarkService(
	mysqlRepo ports.MySQLPort,
	postgresqlRepo ports.PostgreSQLPort,
	neo4jRepo ports.Neo4jPort,
	analyzer ports.PerformanceAnalyzerPort,
	logger *logrus.Logger,
	config *BenchmarkServiceConfig,
) *BenchmarkService {
	if config == nil {
		config = defaultBenchmarkServiceConfig()
	}

	service := &BenchmarkService{
		mysqlRepo:      mysqlRepo,
		postgresqlRepo: postgresqlRepo,
		neo4jRepo:      neo4jRepo,
		analyzer:       analyzer,
		logger:         logger,
		config:         config,
		tools:          make(map[string]ports.BenchmarkToolPort),
		activeRuns:     make(map[string]*BenchmarkExecution),
	}

	// Start cleanup routine
	go service.cleanupRoutine()

	return service
}

// RegisterBenchmarkTool registers a benchmark tool implementation
func (s *BenchmarkService) RegisterBenchmarkTool(name string, tool ports.BenchmarkToolPort) error {
	s.toolsMutex.Lock()
	defer s.toolsMutex.Unlock()

	if !tool.IsAvailable() {
		return fmt.Errorf("benchmark tool %s is not available", name)
	}

	s.tools[name] = tool
	s.logger.WithField("tool", name).Info("Registered benchmark tool")

	return nil
}

// GetAvailableTools returns list of available benchmark tools
func (s *BenchmarkService) GetAvailableTools() []string {
	s.toolsMutex.RLock()
	defer s.toolsMutex.RUnlock()

	tools := make([]string, 0, len(s.tools))
	for name, tool := range s.tools {
		if tool.IsAvailable() {
			tools = append(tools, name)
		}
	}

	return tools
}

// ExecuteBenchmark runs a benchmark with the specified configuration
func (s *BenchmarkService) ExecuteBenchmark(ctx context.Context, config ports.BenchmarkConfig, toolName string) (string, error) {
	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return "", fmt.Errorf("invalid configuration: %w", err)
	}

	// Check concurrent run limits
	if s.getActiveRunCount() >= s.config.MaxConcurrentRuns {
		return "", fmt.Errorf("maximum concurrent runs (%d) exceeded", s.config.MaxConcurrentRuns)
	}

	// Get benchmark tool
	tool, err := s.getBenchmarkTool(toolName)
	if err != nil {
		return "", fmt.Errorf("failed to get benchmark tool: %w", err)
	}

	// Validate tool configuration
	if err := tool.Validate(config); err != nil {
		return "", fmt.Errorf("tool validation failed: %w", err)
	}

	// Create execution context
	executionID := uuid.New().String()
	executionCtx, cancel := context.WithTimeout(ctx, s.config.DefaultTimeout)

	execution := &BenchmarkExecution{
		ID:         executionID,
		Config:     config,
		StartTime:  time.Now(),
		Status:     ports.BenchmarkStatusPending,
		Tool:       tool,
		Context:    executionCtx,
		CancelFunc: cancel,
		Progress: &BenchmarkProgress{
			CurrentPhase: "initializing",
			TotalSteps:   4, // prepare, warmup, execute, analyze
			LastUpdate:   time.Now(),
		},
	}

	// Register execution
	s.registerExecution(execution)

	// Start execution asynchronously
	go s.executeAsync(execution)

	s.logger.WithFields(logrus.Fields{
		"execution_id": executionID,
		"tool":         toolName,
		"test_type":    config.TestType,
		"duration":     config.Duration,
	}).Info("Started benchmark execution")

	return executionID, nil
}

// executeAsync runs the benchmark asynchronously
func (s *BenchmarkService) executeAsync(execution *BenchmarkExecution) {
	defer s.cleanupExecution(execution.ID)
	defer execution.CancelFunc()

	// Update status to running
	s.updateExecutionStatus(execution.ID, ports.BenchmarkStatusRunning, "executing benchmark")

	// Execute the benchmark
	result, err := execution.Tool.Execute(execution.Context, execution.Config)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"execution_id": execution.ID,
			"error":        err.Error(),
		}).Error("Benchmark execution failed")

		result = &ports.BenchmarkResult{
			ID:        execution.ID,
			ToolName:  s.getToolName(execution.Tool),
			TestType:  execution.Config.TestType,
			StartTime: execution.StartTime,
			EndTime:   time.Now(),
			Duration:  time.Since(execution.StartTime),
			Status:    ports.BenchmarkStatusFailed,
			Error:     err.Error(),
		}
	} else {
		result.ID = execution.ID
		result.Status = ports.BenchmarkStatusCompleted
	}

	// Store result
	execution.mutex.Lock()
	execution.Result = result
	execution.Status = result.Status
	execution.mutex.Unlock()

	// Log completion
	s.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"status":       result.Status,
		"duration":     result.Duration,
		"queries_per_sec": func() float64 {
			if result.Metrics != nil {
				return result.Metrics.QueriesPerSecond
			}
			return 0
		}(),
	}).Info("Benchmark execution completed")
}

// GetBenchmarkResult returns the result of a benchmark execution
func (s *BenchmarkService) GetBenchmarkResult(executionID string) (*ports.BenchmarkResult, error) {
	s.runsMutex.RLock()
	execution, exists := s.activeRuns[executionID]
	s.runsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}

	execution.mutex.RLock()
	defer execution.mutex.RUnlock()

	if execution.Result == nil {
		// Return progress information
		return &ports.BenchmarkResult{
			ID:       executionID,
			Status:   execution.Status,
			Duration: time.Since(execution.StartTime),
		}, nil
	}

	return execution.Result, nil
}

// GetBenchmarkProgress returns the current progress of a benchmark execution
func (s *BenchmarkService) GetBenchmarkProgress(executionID string) (*BenchmarkProgress, error) {
	s.runsMutex.RLock()
	execution, exists := s.activeRuns[executionID]
	s.runsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}

	execution.mutex.RLock()
	defer execution.mutex.RUnlock()

	// Update elapsed time
	execution.Progress.ElapsedTime = time.Since(execution.StartTime)

	return execution.Progress, nil
}

// CancelBenchmark cancels a running benchmark
func (s *BenchmarkService) CancelBenchmark(executionID string) error {
	s.runsMutex.RLock()
	execution, exists := s.activeRuns[executionID]
	s.runsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	execution.mutex.Lock()
	if execution.Status == ports.BenchmarkStatusRunning || execution.Status == ports.BenchmarkStatusPending {
		execution.Status = ports.BenchmarkStatusCancelled
		execution.CancelFunc()
	}
	execution.mutex.Unlock()

	s.logger.WithField("execution_id", executionID).Info("Cancelled benchmark execution")
	return nil
}

// ListActiveRuns returns list of currently active benchmark runs
func (s *BenchmarkService) ListActiveRuns() []BenchmarkExecutionInfo {
	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	runs := make([]BenchmarkExecutionInfo, 0, len(s.activeRuns))
	for _, execution := range s.activeRuns {
		execution.mutex.RLock()
		runs = append(runs, BenchmarkExecutionInfo{
			ID:        execution.ID,
			ToolName:  s.getToolName(execution.Tool),
			TestType:  execution.Config.TestType,
			Status:    execution.Status,
			StartTime: execution.StartTime,
			Duration:  time.Since(execution.StartTime),
		})
		execution.mutex.RUnlock()
	}

	return runs
}

// BenchmarkExecutionInfo provides summary information about a benchmark execution
type BenchmarkExecutionInfo struct {
	ID        string                `json:"id"`
	ToolName  string                `json:"tool_name"`
	TestType  string                `json:"test_type"`
	Status    ports.BenchmarkStatus `json:"status"`
	StartTime time.Time             `json:"start_time"`
	Duration  time.Duration         `json:"duration"`
}

// Private helper methods

func (s *BenchmarkService) validateConfig(config ports.BenchmarkConfig) error {
	if config.Duration > s.config.MaxDuration {
		return fmt.Errorf("duration %v exceeds maximum allowed %v", config.Duration, s.config.MaxDuration)
	}

	if config.Threads > s.config.MaxThreads {
		return fmt.Errorf("threads %d exceeds maximum allowed %d", config.Threads, s.config.MaxThreads)
	}

	if config.TableSize > s.config.MaxTableSize {
		return fmt.Errorf("table size %d exceeds maximum allowed %d", config.TableSize, s.config.MaxTableSize)
	}

	return nil
}

func (s *BenchmarkService) getBenchmarkTool(name string) (ports.BenchmarkToolPort, error) {
	s.toolsMutex.RLock()
	defer s.toolsMutex.RUnlock()

	tool, exists := s.tools[name]
	if !exists {
		return nil, fmt.Errorf("benchmark tool '%s' not found", name)
	}

	if !tool.IsAvailable() {
		return nil, fmt.Errorf("benchmark tool '%s' is not available", name)
	}

	return tool, nil
}

func (s *BenchmarkService) getToolName(tool ports.BenchmarkToolPort) string {
	s.toolsMutex.RLock()
	defer s.toolsMutex.RUnlock()

	for name, t := range s.tools {
		if t == tool {
			return name
		}
	}
	return "unknown"
}

func (s *BenchmarkService) registerExecution(execution *BenchmarkExecution) {
	s.runsMutex.Lock()
	defer s.runsMutex.Unlock()
	s.activeRuns[execution.ID] = execution
}

func (s *BenchmarkService) updateExecutionStatus(executionID string, status ports.BenchmarkStatus, message string) {
	s.runsMutex.RLock()
	execution, exists := s.activeRuns[executionID]
	s.runsMutex.RUnlock()

	if exists {
		execution.mutex.Lock()
		execution.Status = status
		if execution.Progress != nil {
			execution.Progress.CurrentPhase = message
			execution.Progress.LastUpdate = time.Now()
			execution.Progress.Messages = append(execution.Progress.Messages, message)
			// Keep only last 10 messages
			if len(execution.Progress.Messages) > 10 {
				execution.Progress.Messages = execution.Progress.Messages[len(execution.Progress.Messages)-10:]
			}
		}
		execution.mutex.Unlock()
	}
}

func (s *BenchmarkService) cleanupExecution(executionID string) {
	// Note: We don't immediately remove executions to allow result retrieval
	// The cleanup routine will remove them after the retention period
}

func (s *BenchmarkService) getActiveRunCount() int {
	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	count := 0
	for _, execution := range s.activeRuns {
		execution.mutex.RLock()
		if execution.Status == ports.BenchmarkStatusRunning || execution.Status == ports.BenchmarkStatusPending {
			count++
		}
		execution.mutex.RUnlock()
	}

	return count
}

func (s *BenchmarkService) cleanupRoutine() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupOldExecutions()
	}
}

func (s *BenchmarkService) cleanupOldExecutions() {
	s.runsMutex.Lock()
	defer s.runsMutex.Unlock()

	cutoff := time.Now().Add(-s.config.RetainResults)
	toDelete := make([]string, 0)

	for id, execution := range s.activeRuns {
		execution.mutex.RLock()
		isOld := execution.StartTime.Before(cutoff)
		isCompleted := execution.Status == ports.BenchmarkStatusCompleted ||
			execution.Status == ports.BenchmarkStatusFailed ||
			execution.Status == ports.BenchmarkStatusCancelled
		execution.mutex.RUnlock()

		if isOld && isCompleted {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(s.activeRuns, id)
		s.logger.WithField("execution_id", id).Debug("Cleaned up old execution")
	}

	if len(toDelete) > 0 {
		s.logger.WithField("cleaned_count", len(toDelete)).Info("Cleaned up old benchmark executions")
	}
}

// defaultBenchmarkServiceConfig returns default configuration
func defaultBenchmarkServiceConfig() *BenchmarkServiceConfig {
	return &BenchmarkServiceConfig{
		MaxConcurrentRuns:  5,
		DefaultTimeout:     30 * time.Minute,
		CleanupInterval:    15 * time.Minute,
		RetainResults:      2 * time.Hour,
		MaxResultsInMemory: 100,
		MaxTableSize:       1000000, // 1M rows
		MaxDuration:        1 * time.Hour,
		MaxThreads:         64,
		EnabledTools:       []string{"sysbench", "custom"},
		ToolConfigurations: make(map[string]interface{}),
	}
}

// Additional methods for integration with existing graph services

// CreatePerformanceGraph creates a graph enhanced with performance data
func (s *BenchmarkService) CreatePerformanceGraph(ctx context.Context, benchmarkResult *ports.BenchmarkResult) (*PerformanceEnhancedGraph, error) {
	if benchmarkResult == nil || benchmarkResult.Metrics == nil {
		return nil, fmt.Errorf("benchmark result and metrics are required")
	}

	// Analyze query patterns to extract table relationships
	_, err := s.extractTableRelationships(benchmarkResult.QueryResults)
	if err != nil {
		return nil, fmt.Errorf("failed to extract relationships: %w", err)
	}

	// Create performance-enhanced graph
	graph := &PerformanceEnhancedGraph{
		ID:            fmt.Sprintf("perf-%s", benchmarkResult.ID),
		BenchmarkID:   benchmarkResult.ID,
		GeneratedAt:   time.Now(),
		GlobalMetrics: benchmarkResult.Metrics,
		Nodes:         make([]PerformanceNode, 0),
		Edges:         make([]PerformanceEdge, 0),
	}

	// Build nodes and edges with performance data
	nodeMap := make(map[string]*PerformanceNode)

	for _, queryResult := range benchmarkResult.QueryResults {
		// Create nodes for source tables
		for _, tableName := range queryResult.SourceTables {
			if node, exists := nodeMap[tableName]; exists {
				s.updateNodeMetrics(node, &queryResult)
			} else {
				node := s.createPerformanceNode(tableName, &queryResult)
				nodeMap[tableName] = node
				graph.Nodes = append(graph.Nodes, *node)
			}
		}

		// Create edges for joined tables
		s.createPerformanceEdges(graph, &queryResult, nodeMap)
	}

	return graph, nil
}

// Performance graph data structures
type PerformanceEnhancedGraph struct {
	ID            string                    `json:"id"`
	BenchmarkID   string                    `json:"benchmark_id"`
	GeneratedAt   time.Time                 `json:"generated_at"`
	GlobalMetrics *ports.PerformanceMetrics `json:"global_metrics"`
	Nodes         []PerformanceNode         `json:"nodes"`
	Edges         []PerformanceEdge         `json:"edges"`
}

type PerformanceNode struct {
	ID              string  `json:"id"`
	TableName       string  `json:"table_name"`
	QueriesPerSec   float64 `json:"queries_per_sec"`
	AvgLatency      float64 `json:"avg_latency_ms"`
	HotspotScore    float64 `json:"hotspot_score"`
	TotalQueries    int64   `json:"total_queries"`
	RowsProcessed   int64   `json:"rows_processed"`
	IndexEfficiency float64 `json:"index_efficiency"`
}

type PerformanceEdge struct {
	ID              string  `json:"id"`
	SourceTable     string  `json:"source_table"`
	TargetTable     string  `json:"target_table"`
	RelationType    string  `json:"relation_type"`
	QueryFrequency  float64 `json:"query_frequency"`
	AvgLatency      float64 `json:"avg_latency_ms"`
	LoadFactor      float64 `json:"load_factor"`
	PerformanceRank string  `json:"performance_rank"` // FAST, MEDIUM, SLOW
}

// Helper methods for graph creation
func (s *BenchmarkService) extractTableRelationships(queryResults []ports.QueryPerformance) ([]TableRelationship, error) {
	relationships := make([]TableRelationship, 0)

	for _, query := range queryResults {
		if len(query.JoinedTables) > 0 {
			for _, sourceTable := range query.SourceTables {
				for _, joinedTable := range query.JoinedTables {
					relationships = append(relationships, TableRelationship{
						SourceTable: sourceTable,
						TargetTable: joinedTable,
						Type:        "JOIN",
						Frequency:   query.ExecutionCount,
						AvgLatency:  query.AverageTime,
					})
				}
			}
		}
	}

	return relationships, nil
}

type TableRelationship struct {
	SourceTable string
	TargetTable string
	Type        string
	Frequency   int64
	AvgLatency  time.Duration
}

// Missing methods for API compatibility

// ListRunningBenchmarks returns all running benchmarks
func (s *BenchmarkService) ListRunningBenchmarks(ctx context.Context) []*BenchmarkExecution {
	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	running := make([]*BenchmarkExecution, 0)
	for _, execution := range s.activeRuns {
		execution.mutex.RLock()
		if execution.Status == ports.BenchmarkStatusRunning || execution.Status == ports.BenchmarkStatusPending {
			running = append(running, execution)
		}
		execution.mutex.RUnlock()
	}

	return running
}

// GetBenchmarkStatus returns the status of a benchmark
func (s *BenchmarkService) GetBenchmarkStatus(ctx context.Context, executionID string) *BenchmarkExecution {
	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	if execution, exists := s.activeRuns[executionID]; exists {
		return execution
	}
	return nil
}

// StopBenchmark stops a running benchmark
func (s *BenchmarkService) StopBenchmark(ctx context.Context, executionID string) error {
	s.runsMutex.RLock()
	execution, exists := s.activeRuns[executionID]
	s.runsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("benchmark execution not found: %s", executionID)
	}

	execution.mutex.Lock()
	defer execution.mutex.Unlock()

	if execution.Status != ports.BenchmarkStatusRunning && execution.Status != ports.BenchmarkStatusPending {
		return fmt.Errorf("benchmark is not running: %s", execution.Status)
	}

	execution.CancelFunc()
	execution.Status = ports.BenchmarkStatusCancelled

	s.logger.WithField("execution_id", executionID).Info("Benchmark stopped")
	return nil
}

// GetBenchmarkResults returns the results of a benchmark
func (s *BenchmarkService) GetBenchmarkResults(ctx context.Context, executionID string) *ports.BenchmarkResult {
	s.runsMutex.RLock()
	defer s.runsMutex.RUnlock()

	if execution, exists := s.activeRuns[executionID]; exists {
		execution.mutex.RLock()
		defer execution.mutex.RUnlock()
		return execution.Result
	}
	return nil
}

func (s *BenchmarkService) createPerformanceNode(tableName string, query *ports.QueryPerformance) *PerformanceNode {
	return &PerformanceNode{
		ID:              fmt.Sprintf("node-%s", tableName),
		TableName:       tableName,
		QueriesPerSec:   float64(query.ExecutionCount) / query.TotalTime.Seconds(),
		AvgLatency:      float64(query.AverageTime.Milliseconds()),
		TotalQueries:    query.ExecutionCount,
		RowsProcessed:   query.RowsExamined,
		IndexEfficiency: s.calculateIndexEfficiency(query),
		HotspotScore:    s.calculateHotspotScore(query),
	}
}

func (s *BenchmarkService) updateNodeMetrics(node *PerformanceNode, query *ports.QueryPerformance) {
	// Update aggregated metrics
	node.TotalQueries += query.ExecutionCount
	node.RowsProcessed += query.RowsExamined

	// Recalculate averages
	totalLatency := (node.AvgLatency * float64(node.TotalQueries-query.ExecutionCount)) +
		float64(query.TotalTime.Milliseconds())
	node.AvgLatency = totalLatency / float64(node.TotalQueries)

	// Update derived metrics
	node.HotspotScore = s.calculateHotspotScore(query)
	node.IndexEfficiency = s.calculateIndexEfficiency(query)
}

func (s *BenchmarkService) createPerformanceEdges(graph *PerformanceEnhancedGraph, query *ports.QueryPerformance, nodeMap map[string]*PerformanceNode) {
	if len(query.JoinedTables) == 0 {
		return
	}

	for _, sourceTable := range query.SourceTables {
		for _, joinedTable := range query.JoinedTables {
			edge := PerformanceEdge{
				ID:              fmt.Sprintf("edge-%s-%s", sourceTable, joinedTable),
				SourceTable:     sourceTable,
				TargetTable:     joinedTable,
				RelationType:    "JOIN",
				QueryFrequency:  float64(query.ExecutionCount),
				AvgLatency:      float64(query.AverageTime.Milliseconds()),
				LoadFactor:      s.calculateLoadFactor(query),
				PerformanceRank: s.calculatePerformanceRank(query),
			}
			graph.Edges = append(graph.Edges, edge)
		}
	}
}

func (s *BenchmarkService) calculateIndexEfficiency(query *ports.QueryPerformance) float64 {
	if query.RowsExamined == 0 {
		return 1.0
	}
	if query.RowsReturned == 0 {
		return 0.0
	}
	return float64(query.RowsReturned) / float64(query.RowsExamined)
}

func (s *BenchmarkService) calculateHotspotScore(query *ports.QueryPerformance) float64 {
	// Simple hotspot calculation based on frequency and latency
	frequencyScore := float64(query.ExecutionCount) / 1000.0          // normalize
	latencyScore := float64(query.AverageTime.Milliseconds()) / 100.0 // normalize

	score := (frequencyScore * 0.6) + (latencyScore * 0.4)
	if score > 100.0 {
		return 100.0
	}
	return score
}

func (s *BenchmarkService) calculateLoadFactor(query *ports.QueryPerformance) float64 {
	// Calculate load based on frequency and resource usage
	return float64(query.ExecutionCount*query.RowsExamined) / 10000.0
}

func (s *BenchmarkService) calculatePerformanceRank(query *ports.QueryPerformance) string {
	avgLatencyMs := float64(query.AverageTime.Milliseconds())

	if avgLatencyMs < 50 {
		return "FAST"
	} else if avgLatencyMs < 200 {
		return "MEDIUM"
	}
	return "SLOW"
}
