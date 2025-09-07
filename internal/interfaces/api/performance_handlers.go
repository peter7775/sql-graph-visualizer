package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
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
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// BenchmarkRequest represents a benchmark execution request
type BenchmarkRequest struct {
	BenchmarkType string                 `json:"benchmark_type"`
	Config        map[string]interface{} `json:"config"`
	Duration      int                    `json:"duration_seconds"`
	Description   string                 `json:"description,omitempty"`
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
) *PerformanceHandlers {
	return &PerformanceHandlers{
		logger:              logger,
		benchmarkService:    benchmarkService,
		performanceAnalyzer: performanceAnalyzer,
		graphMapper:         graphMapper,
		realtimeMonitor:     realtimeMonitor,
		psAdapter:           psAdapter,
	}
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
}

// Benchmark control handlers

func (ph *PerformanceHandlers) ListBenchmarks(w http.ResponseWriter, r *http.Request) {
	benchmarks := ph.benchmarkService.ListRunningBenchmarks(r.Context())

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      benchmarks,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) StartBenchmark(w http.ResponseWriter, r *http.Request) {
	var req BenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body", err.Error())
		return
	}

	// Validate request
	if req.BenchmarkType == "" {
		ph.sendErrorResponse(w, http.StatusBadRequest, "validation_error", "benchmark_type is required", "")
		return
	}

	// Create benchmark configuration from ports
	config := ports.BenchmarkConfig{
		TestType:     req.BenchmarkType,
		Duration:     time.Duration(req.Duration) * time.Second,
		CustomParams: req.Config,
	}

	// Start benchmark
	executionID, err := ph.benchmarkService.ExecuteBenchmark(r.Context(), config, req.BenchmarkType)
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
			"benchmark_type": req.BenchmarkType,
			"duration":       req.Duration,
		},
	}

	ph.sendJSONResponse(w, http.StatusCreated, APIResponse{
		Success:   true,
		Data:      response,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetBenchmark(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	benchmarkID := vars["id"]

	if benchmarkID == "" {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_id", "Benchmark ID is required", "")
		return
	}

	// Get benchmark status
	status := ph.benchmarkService.GetBenchmarkStatus(r.Context(), benchmarkID)
	if status == nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark not found", "")
		return
	}

	var endTime *time.Time
	if status.Result != nil && status.Result.Status == ports.BenchmarkStatusCompleted {
		endTime = &status.Result.EndTime
	}

	var progressFloat float64 = 0.0
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

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      response,
		Timestamp: time.Now(),
	})
}

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

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      map[string]string{"status": "stopped"},
		Timestamp: time.Now(),
	})
}

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

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      results,
		Timestamp: time.Now(),
	})
}

// Performance data handlers

func (ph *PerformanceHandlers) GetCurrentPerformanceData(w http.ResponseWriter, r *http.Request) {
	includeGraph := r.URL.Query().Get("include_graph") == "true"
	includeAnalysis := r.URL.Query().Get("include_analysis") == "true"

	// Collect current performance data
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
		// TODO: Get base graph from graph service
		var baseGraph *models.Graph
		if baseGraph != nil {
			graphData, err := ph.graphMapper.MapPerformanceToGraph(r.Context(), baseGraph, perfData)
			if err == nil {
				response.GraphData = graphData
			}
		}
	}

	// Include analysis if requested
	if includeAnalysis {
		// TODO: Implement performance analysis when method is available
		response.AnalysisResults = map[string]interface{}{
			"status":  "analysis_not_available",
			"message": "Performance analysis feature is under development",
		}
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      response,
		Timestamp: time.Now(),
	})
}

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

	// TODO: Implement historical data collection
	// This would typically query a time-series database or historical storage
	historyData := []interface{}{
		map[string]interface{}{
			"message": "Historical data collection not yet implemented",
			"params": map[string]interface{}{
				"start_time": startTime,
				"end_time":   endTime,
				"limit":      limit,
			},
		},
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      historyData,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetPerformanceAnalysis(w http.ResponseWriter, r *http.Request) {
	// Collect current performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	// TODO: Implement performance analysis
	analysisResults := map[string]interface{}{
		"status":  "analysis_not_available",
		"message": "Performance analysis feature is under development",
		"data":    perfData,
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      analysisResults,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetPerformanceGraph(w http.ResponseWriter, r *http.Request) {
	// Collect performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	// TODO: Get base graph from graph service
	var baseGraph *models.Graph
	if baseGraph == nil {
		ph.sendErrorResponse(w, http.StatusServiceUnavailable, "graph_unavailable", "Base graph is not available", "")
		return
	}

	// Map performance data to graph
	graphData, err := ph.graphMapper.MapPerformanceToGraph(r.Context(), baseGraph, perfData)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "mapping_error", "Failed to map performance to graph", err.Error())
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      graphData,
		Timestamp: time.Now(),
	})
}

// Real-time .monitoring handlers

func (ph *PerformanceHandlers) GetRealtimeClients(w http.ResponseWriter, r *http.Request) {
	clients := ph.realtimeMonitor.GetConnectedClients()

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      clients,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetRealtimeStatus(w http.ResponseWriter, r *http.Request) {
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

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      status,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ph.realtimeMonitor.HandleWebSocket(w, r)
}

// Metrics handlers

func (ph *PerformanceHandlers) GetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	// Collect current performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	summary := ph.generatePerformanceSummary(perfData)

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      summary,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetTableMetrics(w http.ResponseWriter, r *http.Request) {
	// Collect current performance data
	perfData, err := ph.psAdapter.CollectPerformanceData(r.Context())
	if err != nil {
		ph.sendErrorResponse(w, http.StatusInternalServerError, "collection_error", "Failed to collect performance data", err.Error())
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      perfData.TableIOStats,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetQueryMetrics(w http.ResponseWriter, r *http.Request) {
	limit := 50 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Collect current performance data
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

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      queries,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement alerts retrieval
	// This would typically query an alerts storage system

	alerts := []map[string]interface{}{
		{
			"message": "Alerts system not yet implemented",
			"note":    "This endpoint will return active performance alerts",
		},
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      alerts,
		Timestamp: time.Now(),
	})
}

// Configuration handlers

func (ph *PerformanceHandlers) GetPerformanceConfig(w http.ResponseWriter, r *http.Request) {
	// TODO: Return current performance .monitoring configuration
	config := map[string]interface{}{
		"message": "Configuration endpoint not yet implemented",
	}

	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
		Success:   true,
		Data:      config,
		Timestamp: time.Now(),
	})
}

func (ph *PerformanceHandlers) UpdatePerformanceConfig(w http.ResponseWriter, r *http.Request) {
	// TODO: Update performance .monitoring configuration
	ph.sendJSONResponse(w, http.StatusOK, APIResponse{
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
		HotspotCount:      0, // TODO: Calculate from analysis
		BottleneckCount:   0, // TODO: Calculate from analysis
		PerformanceRating: rating,
	}
}

func (ph *PerformanceHandlers) sendJSONResponse(w http.ResponseWriter, statusCode int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		ph.logger.WithError(err).Error("Failed to encode JSON response")
	}
}

func (ph *PerformanceHandlers) sendErrorResponse(w http.ResponseWriter, statusCode int, code, message, details string) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
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
