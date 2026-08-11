// Package api provides HTTP API handlers for performance monitoring.
package api //nolint:revive // api is a clear and conventional package name

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// PerformanceHandlers contains HTTP handlers for performance-related operations
type PerformanceHandlers struct {
	logger              *logrus.Logger
	benchmarkService    *performance.BenchmarkService
	performanceAnalyzer *performance.PerformanceAnalyzer
	graphMapper         *performance.GraphPerformanceMapper
	realtimeMonitor     *performance.RealtimePerformanceMonitor
	psAdapter           *performance.PerformanceSchemaAdapter
	neo4jRepo           ports.Neo4jPort
}

// Response represents an API response structure.
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *Error      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// Error represents an API error structure.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// BenchmarkRequest represents a benchmark execution request
type BenchmarkRequest struct {
	// BenchmarkType is kept for backward compatibility. Historically the
	// dashboard sends the tool name here (e.g. "sysbench"); it may also carry a
	// sysbench test type (e.g. "oltp_read_write").
	BenchmarkType string `json:"benchmark_type"`
	Tool          string `json:"tool,omitempty"`
	TestType      string `json:"test_type,omitempty"`
	Threads       int    `json:"threads,omitempty"`
	Tables        int    `json:"tables,omitempty"`
	TableSize     int    `json:"table_size,omitempty"`
	WarmupSeconds int    `json:"warmup_seconds,omitempty"`
	DatabaseURL   string `json:"database_url,omitempty"`
	DatabaseType  string `json:"database_type,omitempty"`
	// QuerySet selects a named custom query set when Tool is "custom".
	QuerySet string `json:"query_set,omitempty"`

	Config      map[string]interface{} `json:"config"`
	Duration    int                    `json:"duration_seconds"`
	Description string                 `json:"description,omitempty"`
}

// BenchmarkStatusResponse represents benchmark status
type BenchmarkStatusResponse struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Progress  float64                `json:"progress"`
	Results   interface{}            `json:"results,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// PerformanceDataResponse represents performance data response
type PerformanceDataResponse struct {
	ID              string                            `json:"id"`
	CollectedAt     time.Time                         `json:"collected_at"`
	StatementStats  []performance.StatementStatistic  `json:"statement_stats"`
	TableIOStats    []performance.TableIOStatistic    `json:"table_io_stats"`
	IndexStats      []performance.IndexStatistic      `json:"index_stats"`
	ConnectionStats performance.ConnectionStatistics  `json:"connection_stats"`
	Summary         *PerformanceSummary               `json:"summary"`
	GraphData       *performance.PerformanceGraphData `json:"graph_data,omitempty"`
	AnalysisResults interface{}                       `json:"analysis_results,omitempty"`
}

// PerformanceSummary provides a high-level summary of performance metrics
type PerformanceSummary struct {
	TotalQueries      int64   `json:"total_queries"`
	AverageLatency    float64 `json:"average_latency_ms"`
	QueriesPerSecond  float64 `json:"queries_per_second"`
	SlowQueriesCount  int64   `json:"slow_queries_count"`
	ErrorRate         float64 `json:"error_rate"`
	HotspotCount      int     `json:"hotspot_count"`
	BottleneckCount   int     `json:"bottleneck_count"`
	PerformanceRating string  `json:"performance_rating"`
}

// NewPerformanceHandlers creates new performance API handlers
func NewPerformanceHandlers(
	logger *logrus.Logger,
	benchmarkService *performance.BenchmarkService,
	performanceAnalyzer *performance.PerformanceAnalyzer,
	graphMapper *performance.GraphPerformanceMapper,
	realtimeMonitor *performance.RealtimePerformanceMonitor,
	psAdapter *performance.PerformanceSchemaAdapter,
	neo4jRepo ports.Neo4jPort,
) *PerformanceHandlers {
	return &PerformanceHandlers{
		logger:              logger,
		benchmarkService:    benchmarkService,
		performanceAnalyzer: performanceAnalyzer,
		graphMapper:         graphMapper,
		realtimeMonitor:     realtimeMonitor,
		psAdapter:           psAdapter,
		neo4jRepo:           neo4jRepo,
	}
}

// fetchBaseGraph loads the current domain graph from Neo4j (the same data
// source used by the main visualization's /api/graph endpoint) and converts
// it into the simplified models.Graph shape expected by GraphPerformanceMapper.
func (ph *PerformanceHandlers) fetchBaseGraph() (*models.Graph, error) {
	if ph.neo4jRepo == nil {
		return nil, fmt.Errorf("neo4j repository is not configured")
	}

	graphInterface, err := ph.neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
	if err != nil {
		return nil, fmt.Errorf("failed to export graph from Neo4j: %w", err)
	}
	g, ok := graphInterface.(*graph.GraphAggregate)
	if !ok {
		return nil, fmt.Errorf("unexpected graph type returned from Neo4j export")
	}

	baseGraph := &models.Graph{
		Nodes:     make([]*models.Node, 0, len(g.GetNodes())),
		Relations: make([]*models.Relation, 0, len(g.GetRelationships())),
	}
	for _, node := range g.GetNodes() {
		baseGraph.Nodes = append(baseGraph.Nodes, &models.Node{
			Label:      node.Type,
			Properties: node.Properties,
		})
	}
	for _, rel := range g.GetRelationships() {
		baseGraph.Relations = append(baseGraph.Relations, &models.Relation{
			Type:       rel.Type,
			From:       fmt.Sprintf("%v", rel.SourceNode.ID),
			To:         fmt.Sprintf("%v", rel.TargetNode.ID),
			Properties: rel.Properties,
		})
	}
	return baseGraph, nil
}

// RegisterRoutes registers all performance-related routes
func (ph *PerformanceHandlers) RegisterRoutes(router *mux.Router) {
	// Benchmark control endpoints
	router.HandleFunc("/api/performance/benchmarks", ph.ListBenchmarks).Methods("GET")
	router.HandleFunc("/api/performance/benchmarks", ph.StartBenchmark).Methods("POST")
	router.HandleFunc("/api/performance/benchmarks/{id}", ph.GetBenchmark).Methods("GET")
	router.HandleFunc("/api/performance/benchmarks/{id}/stop", ph.StopBenchmark).Methods("POST")
	router.HandleFunc("/api/performance/benchmarks/{id}/results", ph.GetBenchmarkResults).Methods("GET")

	// Performance data endpoints
	router.HandleFunc("/api/performance/data", ph.GetCurrentPerformanceData).Methods("GET")
	router.HandleFunc("/api/performance/data/history", ph.GetPerformanceHistory).Methods("GET")
	router.HandleFunc("/api/performance/data/analysis", ph.GetPerformanceAnalysis).Methods("GET")
	router.HandleFunc("/api/performance/data/graph", ph.GetPerformanceGraph).Methods("GET")

	// Real-time .monitoring endpoints
	router.HandleFunc("/api/performance/realtime/clients", ph.GetRealtimeClients).Methods("GET")
	router.HandleFunc("/api/performance/realtime/status", ph.GetRealtimeStatus).Methods("GET")
	router.HandleFunc("/ws/performance", ph.HandleWebSocket).Methods("GET")

	// Performance metrics endpoints
	router.HandleFunc("/api/performance/metrics/summary", ph.GetMetricsSummary).Methods("GET")
	router.HandleFunc("/api/performance/metrics/tables", ph.GetTableMetrics).Methods("GET")
	router.HandleFunc("/api/performance/metrics/queries", ph.GetQueryMetrics).Methods("GET")
	router.HandleFunc("/api/performance/metrics/alerts", ph.GetAlerts).Methods("GET")

	// Configuration endpoints
	router.HandleFunc("/api/performance/config", ph.GetPerformanceConfig).Methods("GET")
	router.HandleFunc("/api/performance/config", ph.UpdatePerformanceConfig).Methods("PUT")

	// Reporting and export endpoints
	router.HandleFunc("/api/performance/reports/summary", ph.GetPerformanceReport).Methods("GET")
	router.HandleFunc("/api/performance/export", ph.ExportPerformanceData).Methods("GET")
}

// Benchmark control handlers

// ListBenchmarks handles requests to list all benchmarks.
func (ph *PerformanceHandlers) ListBenchmarks(w http.ResponseWriter, r *http.Request) {
	benchmarks := ph.benchmarkService.ListRunningBenchmarks(r.Context())

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      benchmarks,
		Timestamp: time.Now(),
	})
}

// StartBenchmark handles requests to start a new benchmark.
func (ph *PerformanceHandlers) StartBenchmark(w http.ResponseWriter, r *http.Request) {
	var req BenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body", err.Error())
		return
	}

	// Resolve the tool to run and the sysbench test type. For backward
	// compatibility the dashboard sends the tool name in benchmark_type; it may
	// also carry a test type. An empty test type lets the service apply its
	// configured default.
	tool, testType := resolveBenchmarkRequest(req)

	customParams := req.Config
	if req.QuerySet != "" {
		if customParams == nil {
			customParams = make(map[string]interface{})
		}
		customParams["query_set"] = req.QuerySet
	}

	config := ports.BenchmarkConfig{
		TestType:     testType,
		Duration:     time.Duration(req.Duration) * time.Second,
		Threads:      req.Threads,
		Tables:       req.Tables,
		TableSize:    req.TableSize,
		WarmupTime:   time.Duration(req.WarmupSeconds) * time.Second,
		DatabaseType: req.DatabaseType,
		DatabaseURL:  req.DatabaseURL,
		CustomParams: customParams,
	}

	executionID, err := ph.benchmarkService.ExecuteBenchmark(r.Context(), config, tool)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "benchmark_error", "Failed to start benchmark", err.Error())
		return
	}

	response := BenchmarkStatusResponse{
		ID:        executionID,
		Status:    "started",
		StartTime: time.Now(),
		Progress:  0.0,
		Metadata: map[string]interface{}{
			"tool":      tool,
			"test_type": testType,
			"duration":  req.Duration,
		},
	}

	ph.sendJSONResponse(w, http.StatusCreated, Response{
		Success:   true,
		Data:      response,
		Timestamp: time.Now(),
	})
}

// knownBenchmarkTools enumerates tool selectors accepted in the legacy
// benchmark_type field for backward compatibility with the dashboard.
var knownBenchmarkTools = map[string]bool{"sysbench": true, "custom": true}

// resolveBenchmarkRequest determines the benchmark tool and (optional) sysbench
// test type from a request. It preserves backward compatibility with the
// dashboard, which sends the tool name in benchmark_type. An empty test type is
// returned when none is specified, letting the service apply its default.
func resolveBenchmarkRequest(req BenchmarkRequest) (tool, testType string) {
	tool = req.Tool
	testType = req.TestType

	switch {
	case tool != "":
		// explicit tool wins
	case knownBenchmarkTools[req.BenchmarkType]:
		tool = req.BenchmarkType
	case req.BenchmarkType != "":
		// benchmark_type carried a test type; default the tool to sysbench
		tool = "sysbench"
		if testType == "" {
			testType = req.BenchmarkType
		}
	default:
		tool = "sysbench"
	}

	return tool, testType
}

// GetBenchmark handles requests to get a specific benchmark.
func (ph *PerformanceHandlers) GetBenchmark(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	benchmarkID := vars["id"]

	if benchmarkID == "" {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_id", "Benchmark ID is required", "")
		return
	}

	status := ph.benchmarkService.GetBenchmarkStatus(r.Context(), benchmarkID)
	if status == nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark not found", "")
		return
	}

	var endTime *time.Time
	if status.Result != nil && status.Result.Status == ports.BenchmarkStatusCompleted {
		endTime = &status.Result.EndTime
	}

	var progressFloat float64
	if status.Progress != nil {
		progressFloat = float64(status.Progress.CompletedSteps) / float64(status.Progress.TotalSteps) * 100
	}

	response := BenchmarkStatusResponse{
		ID:        status.ID,
		Status:    string(status.Status),
		StartTime: status.StartTime,
		EndTime:   endTime,
		Progress:  progressFloat,
		Results:   status.Result,
		Error:     "", // No direct error field in execution
		Metadata: map[string]interface{}{
			"config": status.Config,
		},
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      response,
		Timestamp: time.Now(),
	})
}

// StopBenchmark handles requests to stop a running benchmark.
func (ph *PerformanceHandlers) StopBenchmark(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	benchmarkID := vars["id"]

	if benchmarkID == "" {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_id", "Benchmark ID is required", "")
		return
	}

	err := ph.benchmarkService.StopBenchmark(r.Context(), benchmarkID)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "stop_error", "Failed to stop benchmark", err.Error())
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      map[string]string{"status": "stopped"},
		Timestamp: time.Now(),
	})
}

// GetBenchmarkResults handles requests to get benchmark results.
func (ph *PerformanceHandlers) GetBenchmarkResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	benchmarkID := vars["id"]

	if benchmarkID == "" {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_id", "Benchmark ID is required", "")
		return
	}

	results := ph.benchmarkService.GetBenchmarkResults(r.Context(), benchmarkID)
	if results == nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark results not found", "")
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      results,
		Timestamp: time.Now(),
	})
}

// Performance data handlers

// GetCurrentPerformanceData handles requests to get current performance data.
func (ph *PerformanceHandlers) GetCurrentPerformanceData(w http.ResponseWriter, r *http.Request) {
	includeGraph := r.URL.Query().Get("include_graph") == "true"
	includeAnalysis := r.URL.Query().Get("include_analysis") == "true"

	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	response := &PerformanceDataResponse{
		ID:              fmt.Sprintf("perf-data-%d", time.Now().Unix()),
		CollectedAt:     time.Now(),
		StatementStats:  perfData.StatementStats,
		TableIOStats:    perfData.TableIOStats,
		IndexStats:      perfData.IndexStats,
		ConnectionStats: *perfData.ConnectionStats,
		Summary:         ph.generatePerformanceSummary(perfData),
	}

	// Include graph data if requested
	if includeGraph {
		if baseGraph, graphErr := ph.fetchBaseGraph(); graphErr != nil {
			ph.logger.WithError(graphErr).Warn("Failed to load base graph for performance data response")
		} else {
			graphData, mapErr := ph.graphMapper.MapPerformanceToGraph(r.Context(), baseGraph, perfData)
			if mapErr == nil {
				response.GraphData = graphData
			} else {
				ph.logger.WithError(mapErr).Warn("Failed to map performance data to graph")
			}
		}
	}

	// Include analysis if requested
	if includeAnalysis {
		response.AnalysisResults = map[string]interface{}{
			"status":  "analysis_not_available",
			"message": "Performance analysis feature is under development",
		}
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      response,
		Timestamp: time.Now(),
	})
}

// GetPerformanceHistory handles requests to get performance history.
func (ph *PerformanceHandlers) GetPerformanceHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	limitStr := r.URL.Query().Get("limit")

	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_time", "Invalid start_time format", "Use RFC3339 format")
			return
		}
	} else {
		startTime = time.Now().Add(-1 * time.Hour) // Default to last hour
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_time", "Invalid end_time format", "Use RFC3339 format")
			return
		}
	} else {
		endTime = time.Now()
	}

	limit := 100 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Historical performance data currently comes from persisted benchmark
	// results (see BenchmarkResultStorePort); live Performance Schema
	// snapshots are not persisted. Optional tool/test_type filters narrow the
	// result set.
	filter := ports.BenchmarkResultFilter{
		ToolName: r.URL.Query().Get("tool"),
		TestType: r.URL.Query().Get("test_type"),
		Since:    startTime,
		Until:    endTime,
		Limit:    limit,
	}

	results, err := ph.benchmarkService.GetBenchmarkHistory(r.Context(), filter)
	if err != nil {
		ph.sendJSONResponse(w, http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"results": []interface{}{},
				"message": err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"results": results,
			"count":   len(results),
		},
		Timestamp: time.Now(),
	})
}

// GetPerformanceAnalysis handles requests to get performance analysis.
func (ph *PerformanceHandlers) GetPerformanceAnalysis(w http.ResponseWriter, r *http.Request) {
	// Collect current performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	analysisResults := map[string]interface{}{
		"status":  "analysis_not_available",
		"message": "Performance analysis feature is under development",
		"data":    perfData,
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      analysisResults,
		Timestamp: time.Now(),
	})
}

// GetPerformanceGraph handles requests to get performance graph visualization.
func (ph *PerformanceHandlers) GetPerformanceGraph(w http.ResponseWriter, r *http.Request) {
	// Collect performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	baseGraph, err := ph.fetchBaseGraph()
	if err != nil {
		ph.sendErrorResponse(w, http.StatusServiceUnavailable, "graph_unavailable", "Base graph is not available", err.Error())
		return
	}

	graphData, err := ph.graphMapper.MapPerformanceToGraph(r.Context(), baseGraph, perfData)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "mapping_error", "Failed to map performance to graph", err.Error())
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      graphData,
		Timestamp: time.Now(),
	})
}

// PerformanceReport is the response body for GET /api/performance/reports/summary.
type PerformanceReport struct {
	GeneratedAt      time.Time                      `json:"generated_at"`
	TimeRange        PerformanceReportRange         `json:"time_range"`
	RunsAnalyzed     int                            `json:"runs_analyzed"`
	LatestRun        *ports.BenchmarkResult         `json:"latest_run,omitempty"`
	OverallScore     *ports.PerformanceScore        `json:"overall_score,omitempty"`
	Bottlenecks      []ports.PerformanceBottleneck  `json:"bottlenecks"`
	Hotspots         []ports.HotspotNode            `json:"hotspots"`
	QueryPatterns    *ports.QueryPatternAnalysis    `json:"query_patterns,omitempty"`
	Issues           []ports.PerformanceIssue       `json:"issues"`
	Regressions      []ports.PerformanceRegression  `json:"regressions"`
	OptimizationTips []ports.OptimizationSuggestion `json:"optimization_suggestions"`
	Message          string                         `json:"message,omitempty"`
}

// PerformanceReportRange describes the time window a report covers.
type PerformanceReportRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// GetPerformanceReport handles requests for a summarized performance report
// (executive summary, bottlenecks, hotspots, query patterns, optimization
// suggestions, and regression detection) built from persisted benchmark
// history. Requires benchmark result persistence to be configured.
func (ph *PerformanceHandlers) GetPerformanceReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, parseErr := strconv.Atoi(limitStr); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	filter := ports.BenchmarkResultFilter{
		ToolName: r.URL.Query().Get("tool"),
		TestType: r.URL.Query().Get("test_type"),
		Limit:    limit,
	}

	history, err := ph.benchmarkService.GetBenchmarkHistory(ctx, filter)
	if err != nil {
		ph.sendJSONResponse(w, http.StatusOK, Response{
			Success: true,
			Data: PerformanceReport{
				GeneratedAt: time.Now(),
				Message:     err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}
	if len(history) == 0 {
		ph.sendJSONResponse(w, http.StatusOK, Response{
			Success: true,
			Data: PerformanceReport{
				GeneratedAt: time.Now(),
				Message:     "no benchmark runs recorded yet",
			},
			Timestamp: time.Now(),
		})
		return
	}

	latest := history[len(history)-1]
	report := PerformanceReport{
		GeneratedAt: time.Now(),
		TimeRange: PerformanceReportRange{
			StartTime: history[0].StartTime,
			EndTime:   latest.EndTime,
		},
		RunsAnalyzed: len(history),
		LatestRun:    latest,
	}

	if bottlenecks, bErr := ph.performanceAnalyzer.IdentifyBottlenecks(ctx, latest); bErr == nil {
		report.Bottlenecks = bottlenecks
	} else {
		ph.logger.WithError(bErr).Warn("Failed to identify bottlenecks for performance report")
	}

	if queryPatterns, qErr := ph.performanceAnalyzer.AnalyzeQueryPatterns(ctx, latest.QueryResults); qErr == nil {
		report.QueryPatterns = queryPatterns
	} else {
		ph.logger.WithError(qErr).Warn("Failed to analyze query patterns for performance report")
	}

	if issues, iErr := ph.performanceAnalyzer.IdentifyInefficiencies(ctx, latest.QueryResults); iErr == nil {
		report.Issues = issues
	}

	if latest.Metrics != nil {
		if score, sErr := ph.performanceAnalyzer.CalculatePerformanceScore(ctx, latest.Metrics); sErr == nil {
			report.OverallScore = score
		}
	}

	metricsHistory := make([]*ports.PerformanceMetrics, 0, len(history))
	for _, run := range history {
		if run.Metrics != nil {
			metricsHistory = append(metricsHistory, run.Metrics)
		}
	}
	if hotspots, hErr := ph.performanceAnalyzer.DetectHotspots(ctx, metricsHistory); hErr == nil {
		report.Hotspots = hotspots
	}

	if len(history) >= 2 && latest.Metrics != nil {
		previous := history[len(history)-2]
		if previous.Metrics != nil {
			if regressions, rErr := ph.performanceAnalyzer.DetectRegressions(ctx, latest.Metrics, previous.Metrics); rErr == nil {
				report.Regressions = regressions
			}
		}
	}

	analysis := &ports.PerformanceAnalysis{
		OverallScore:  report.OverallScore,
		Bottlenecks:   report.Bottlenecks,
		Hotspots:      report.Hotspots,
		QueryPatterns: report.QueryPatterns,
		Issues:        report.Issues,
		AnalyzedAt:    time.Now(),
	}
	if suggestions, oErr := ph.performanceAnalyzer.GenerateOptimizationSuggestions(ctx, analysis); oErr == nil {
		report.OptimizationTips = suggestions
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      report,
		Timestamp: time.Now(),
	})
}

// ExportPerformanceData handles requests to export persisted benchmark
// history as JSON or CSV, via ?format=json|csv (default json).
func (ph *PerformanceHandlers) ExportPerformanceData(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}

	limit := 1000
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, parseErr := strconv.Atoi(limitStr); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	filter := ports.BenchmarkResultFilter{
		ToolName: r.URL.Query().Get("tool"),
		TestType: r.URL.Query().Get("test_type"),
		Limit:    limit,
	}

	results, err := ph.benchmarkService.GetBenchmarkHistory(r.Context(), filter)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusServiceUnavailable, "persistence_unavailable", "Benchmark result persistence is not configured", err.Error())
		return
	}

	switch format {
	case "csv":
		ph.exportPerformanceCSV(w, results)
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="benchmark_results.json"`)
		if encErr := json.NewEncoder(w).Encode(results); encErr != nil {
			ph.logger.WithError(encErr).Error("Failed to encode performance export as JSON")
		}
	default:
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_format", "Unsupported export format", fmt.Sprintf("format %q is not supported; use json or csv", format))
	}
}

func (ph *PerformanceHandlers) exportPerformanceCSV(w http.ResponseWriter, results []*ports.BenchmarkResult) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="benchmark_results.csv"`)

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{
		"id", "tool_name", "test_type", "status", "start_time", "end_time", "duration_seconds",
		"queries_per_second", "average_latency_ms", "error_rate_percent", "total_errors",
	}
	if err := csvWriter.Write(header); err != nil {
		ph.logger.WithError(err).Error("Failed to write CSV header for performance export")
		return
	}

	for _, result := range results {
		row := []string{
			result.ID,
			result.ToolName,
			result.TestType,
			string(result.Status),
			result.StartTime.Format(time.RFC3339),
			result.EndTime.Format(time.RFC3339),
			strconv.FormatFloat(result.Duration.Seconds(), 'f', 3, 64),
		}
		if result.Metrics != nil {
			row = append(row,
				strconv.FormatFloat(result.Metrics.QueriesPerSecond, 'f', 3, 64),
				strconv.FormatFloat(result.Metrics.AverageLatency, 'f', 3, 64),
				strconv.FormatFloat(result.Metrics.ErrorRate, 'f', 3, 64),
				strconv.Itoa(result.Metrics.TotalErrors),
			)
		} else {
			row = append(row, "", "", "", "")
		}
		if err := csvWriter.Write(row); err != nil {
			ph.logger.WithError(err).Error("Failed to write CSV row for performance export")
			return
		}
	}
}

// Real-time .monitoring handlers

// GetRealtimeClients returns information about realtime clients.
func (ph *PerformanceHandlers) GetRealtimeClients(w http.ResponseWriter, _ *http.Request) {
	clients := ph.realtimeMonitor.GetConnectedClients()

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      clients,
		Timestamp: time.Now(),
	})
}

// GetRealtimeStatus returns realtime monitoring status.
func (ph *PerformanceHandlers) GetRealtimeStatus(w http.ResponseWriter, _ *http.Request) {
	clients := ph.realtimeMonitor.GetConnectedClients()
	lastGraphData := ph.realtimeMonitor.GetLastGraphData()

	status := map[string]interface{}{
		"connected_clients": len(clients),
		"last_update":       nil,
		"monitoring_active": true,
	}

	if lastGraphData != nil {
		status["last_update"] = lastGraphData.GeneratedAt
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      status,
		Timestamp: time.Now(),
	})
}

// HandleWebSocket handles WebSocket connections for real-time performance data.
func (ph *PerformanceHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ph.realtimeMonitor.HandleWebSocket(w, r)
}

// Metrics handlers

// GetMetricsSummary handles requests to get performance metrics summary.
func (ph *PerformanceHandlers) GetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	// Collect current performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	summary := ph.generatePerformanceSummary(perfData)

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      summary,
		Timestamp: time.Now(),
	})
}

// GetTableMetrics handles requests to get table-specific performance metrics.
func (ph *PerformanceHandlers) GetTableMetrics(w http.ResponseWriter, r *http.Request) {
	// Collect current performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      perfData.TableIOStats,
		Timestamp: time.Now(),
	})
}

// GetQueryMetrics handles requests to get query-specific performance metrics.
func (ph *PerformanceHandlers) GetQueryMetrics(w http.ResponseWriter, r *http.Request) {
	limit := 50 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	// Limit results
	queries := perfData.StatementStats
	if len(queries) > limit {
		queries = queries[:limit]
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      queries,
		Timestamp: time.Now(),
	})
}

// GetAlerts returns performance alerts.
func (ph *PerformanceHandlers) GetAlerts(w http.ResponseWriter, _ *http.Request) {
	alerts := []map[string]interface{}{
		{
			"message": "Alerts system not yet implemented",
			"note":    "This endpoint will return active performance alerts",
		},
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      alerts,
		Timestamp: time.Now(),
	})
}

// Configuration handlers

// GetPerformanceConfig returns the current performance configuration.
func (ph *PerformanceHandlers) GetPerformanceConfig(w http.ResponseWriter, _ *http.Request) {
	config := map[string]interface{}{
		"message": "Configuration endpoint not yet implemented",
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      config,
		Timestamp: time.Now(),
	})
}

// UpdatePerformanceConfig updates the performance configuration.
func (ph *PerformanceHandlers) UpdatePerformanceConfig(w http.ResponseWriter, _ *http.Request) {
	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success:   true,
		Data:      map[string]string{"status": "configuration update not yet implemented"},
		Timestamp: time.Now(),
	})
}

// Helper methods

func (ph *PerformanceHandlers) generatePerformanceSummary(perfData *performance.PerformanceSchemaData) *PerformanceSummary {
	var totalQueries int64
	var totalLatency float64
	var slowQueriesCount int64
	var totalErrors int64

	for _, stmt := range perfData.StatementStats {
		totalQueries += stmt.CountStar
		totalLatency += float64(stmt.SumTimerWait) / 1000000 // Convert to milliseconds
		// SumErrors field not available in StatementStatistic - use 0
		// totalErrors += stmt.SumErrors

		avgTime := float64(stmt.AvgTimerWait) / 1000000
		if avgTime > 200.0 { // 200ms threshold
			slowQueriesCount++
		}
	}

	var averageLatency float64
	if totalQueries > 0 {
		averageLatency = totalLatency / float64(totalQueries)
	}

	var errorRate float64
	if totalQueries > 0 {
		errorRate = float64(totalErrors) / float64(totalQueries) * 100
	}

	rating := "good"
	if averageLatency > 500 {
		rating = "poor"
	} else if averageLatency > 200 {
		rating = "fair"
	}

	return &PerformanceSummary{
		TotalQueries:      totalQueries,
		AverageLatency:    averageLatency,
		QueriesPerSecond:  float64(totalQueries) / 300.0, // Assume 5-minute collection period
		SlowQueriesCount:  slowQueriesCount,
		ErrorRate:         errorRate,
		HotspotCount:      0,
		BottleneckCount:   0,
		PerformanceRating: rating,
	}
}

func (ph *PerformanceHandlers) sendJSONResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		ph.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (ph *PerformanceHandlers) sendErrorResponse(w http.ResponseWriter, statusCode int, code, message, details string) {
	response := Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	}

	ph.sendJSONResponse(w, statusCode, response)

	ph.logger.WithFields(logrus.Fields{
		"status_code": statusCode,
		"error_code":  code,
		"message":     message,
		"details":     details,
	}).Warn("API error response sent")
}
