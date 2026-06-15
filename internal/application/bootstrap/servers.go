/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	graphqlserver "sql-graph-visualizer/internal/application/services/graphql"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/infrastructure/middleware"
	"sql-graph-visualizer/internal/interfaces/api"
)

// ServeResult holds the running servers for lifecycle management.
type ServeResult struct {
	VizServer *http.Server
	APIServer *http.Server
}

// Shutdown gracefully shuts down all running servers.
func (s *ServeResult) Shutdown(ctx context.Context) {
	if s.VizServer != nil {
		if err := s.VizServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Error shutting down visualization server: %v", err)
		}
	}
	if s.APIServer != nil {
		if err := s.APIServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Error shutting down API server: %v", err)
		}
	}
}

// StartServers starts the GraphQL, visualization, and API servers.
func (r *Resources) StartServers() (*ServeResult, error) {
	result := &ServeResult{}

	// GraphQL
	graphqlserver.StartGraphQLServer(r.Neo4jRepo, r.Config)
	logrus.Info("GraphQL server started")

	// Visualization server
	if r.DeploymentAdapter.ShouldStartVisualizationServer() {
		result.VizServer = r.startVisualizationServer()
	}

	// API server (with performance routes, debug, health)
	apiServer, err := r.startAPIServer()
	if err != nil {
		return nil, err
	}
	result.APIServer = apiServer

	return result, nil
}

func (r *Resources) startVisualizationServer() *http.Server {
	logrus.Info("Starting visualization server")
	vizMux := http.NewServeMux()

	cfg := r.Config
	neo4jRepo := r.Neo4jRepo

	vizMux.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		configResponse := map[string]any{
			"neo4j": map[string]string{
				"uri":      cfg.Neo4j.URI,
				"username": cfg.Neo4j.User,
				"password": cfg.Neo4j.Password,
			},
		}
		if err := json.NewEncoder(w).Encode(configResponse); err != nil {
			logrus.Errorf("Error encoding config response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	vizMux.HandleFunc("/api/graph", func(w http.ResponseWriter, _ *http.Request) {
		graphInterface, err := neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
		if err != nil {
			logrus.Errorf("Error retrieving data: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		g, ok := graphInterface.(*graph.GraphAggregate)
		if !ok {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		response := struct {
			Nodes         []map[string]any `json:"nodes"`
			Relationships []map[string]any `json:"relationships"`
		}{
			Nodes:         make([]map[string]any, 0),
			Relationships: make([]map[string]any, 0),
		}
		for _, node := range g.GetNodes() {
			response.Nodes = append(response.Nodes, map[string]any{
				"id": node.ID, "label": node.Type, "properties": node.Properties,
			})
		}
		for _, rel := range g.GetRelationships() {
			response.Relationships = append(response.Relationships, map[string]any{
				"from": rel.SourceNode.ID, "to": rel.TargetNode.ID,
				"type": rel.Type, "properties": rel.Properties,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error serializing response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	webRoot := filepath.Join(FindProjectRoot(), "internal", "interfaces", "web")
	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	vizMux.Handle("/static/", http.StripPrefix("/static/", fs))

	vizMux.HandleFunc("/performance", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "performance_dashboard.html"))
	})
	vizMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	vizPort := r.DeploymentAdapter.GetVisualizationPort()
	vizAddr := ":" + vizPort
	server := &http.Server{
		Handler:           vizMux,
		Addr:              vizAddr,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logrus.Infof("Starting visualization server on %s", vizAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Visualization server terminated with error: %v", err)
		}
	}()

	logrus.Infof("Visualization is available at http://localhost:%s", vizPort)
	return server
}

func (r *Resources) startAPIServer() (*http.Server, error) {
	router := mux.NewRouter()

	// Performance API routes
	if r.PerformanceServices != nil {
		logrus.Info("Registering performance API routes...")
		performanceHandlers := api.NewPerformanceHandlers(
			logrus.StandardLogger(),
			r.PerformanceServices.BenchmarkService,
			r.PerformanceServices.PerformanceAnalyzer,
			r.PerformanceServices.GraphMapper,
			r.PerformanceServices.RealtimeMonitor,
			r.PerformanceServices.PSAdapter,
		)
		performanceHandlers.RegisterRoutes(router)
		logrus.Info("Performance API routes registered")
	}

	// Debug endpoint
	router.HandleFunc("/api/debug", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		debugInfo := map[string]interface{}{
			"platform":            r.DeploymentAdapter.GetPlatformName(),
			"environment":         r.DeploymentAdapter.GetEnvironmentInfo(),
			"FORCE_FULL_MODE":     os.Getenv("FORCE_FULL_MODE"),
			"neo4j_repo_type":     fmt.Sprintf("%T", r.Neo4jRepo),
			"real_neo4j_repo_nil": r.RealNeo4jRepo == nil,
			"timestamp":           time.Now().Format(time.RFC3339),
		}
		if err := json.NewEncoder(w).Encode(debugInfo); err != nil {
			logrus.Errorf("Error encoding debug response: %v", err)
		}
	}).Methods("GET")

	// Health endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		dbStatus := "not_initialized"
		if r.DB != nil {
			if err := r.DB.Ping(); err == nil {
				dbStatus = "connected"
			} else {
				dbStatus = "error: " + err.Error()
			}
		}
		response := map[string]interface{}{
			"status":      "healthy",
			"timestamp":   time.Now().Format(time.RFC3339),
			"version":     Version,
			"platform":    r.DeploymentAdapter.GetPlatformName(),
			"database":    dbStatus,
			"neo4j":       "connected",
			"environment": r.DeploymentAdapter.GetEnvironmentInfo(),
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error encoding health response: %v", err)
		}
	})

	// Config endpoint
	router.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(r.Config); err != nil {
			logrus.Errorf("Error encoding config: %v", err)
		}
	})

	if !r.DeploymentAdapter.ShouldStartVisualizationServer() {
		router.HandleFunc("/", r.DeploymentAdapter.GetHomepageHandler())
	}

	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(router)

	apiPort := r.DeploymentAdapter.GetAPIPort()
	apiAddr := ":" + apiPort

	server := &http.Server{
		Handler:           handler,
		Addr:              apiAddr,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	server = r.DeploymentAdapter.ConfigureServer(server)

	go func() {
		logrus.Infof("Starting API server on %s", apiAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("API server terminated with error: %v", err)
		}
	}()

	return server, nil
}

// FindProjectRoot locates the project root by looking for go.mod.
func FindProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Cannot get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			logrus.Error("Cannot find project root directory; falling back to current working directory")
			return wd
		}
		wd = parent
	}
}

// Version is set at build time via ldflags.
var Version = "dev"

func initPerformanceServices(cfg *models.Config, db *sql.DB) *PerformanceServiceContainer {
	logger := logrus.StandardLogger()

	updateInterval, err := time.ParseDuration(cfg.Performance.Monitoring.UpdateInterval)
	if err != nil {
		logrus.Warnf("Invalid update_interval, using default 5s: %v", err)
		updateInterval = 5 * time.Second
	}

	maxStatements := 100
	maxTables := 50
	if cfg.Performance.Monitoring.PerformanceSchema != nil {
		maxStatements = cfg.Performance.Monitoring.PerformanceSchema.StatementLimit
		maxTables = cfg.Performance.Monitoring.PerformanceSchema.TableIOLimit
	}

	psConfig := &performance.PerformanceSchemaConfig{
		CollectionInterval:  updateInterval,
		SlowQueryThreshold:  1 * time.Second,
		MaxHistoryRetention: 1 * time.Hour,
		CollectStatements:   true,
		CollectTableIO:      true,
		CollectIndexUsage:   true,
		CollectWaitEvents:   true,
		CollectConnections:  true,
		CollectReplication:  false,
		MaxStatements:       maxStatements,
		MaxTables:           maxTables,
		IgnoredSchemas:      []string{"mysql", "information_schema", "performance_schema", "sys"},
		IgnoredUsers:        []string{"root", "mysql.sys", "mysql.session"},
		EnableDigestText:    true,
		MinExecutionCount:   10,
		MinAvgLatency:       10.0,
	}
	psAdapter := performance.NewPerformanceSchemaAdapter(db, logger, psConfig)

	slowQueryThreshold := 200.0
	if cfg.Performance.Monitoring.Analysis != nil {
		slowQueryThreshold = cfg.Performance.Monitoring.Analysis.SlowQueryThreshold
	}

	analyzerConfig := &performance.PerformanceAnalyzerConfig{
		HighLatencyThreshold:      time.Duration(slowQueryThreshold) * time.Millisecond,
		LowThroughputThreshold:    10.0,
		HighErrorRateThreshold:    1.0,
		HotspotLatencyWeight:      0.4,
		HotspotFrequencyWeight:    0.4,
		HotspotResourceWeight:     0.2,
		MaxCriticalPaths:          10,
		MinPathImpactScore:        50.0,
		MinPatternFrequency:       100,
		SimilarityThreshold:       0.8,
		IndexSuggestionMinGain:    20.0,
		QueryRewriteMinComplexity: 3,
		MinDataPoints:             5,
		TrendSignificanceLevel:    0.05,
	}
	performanceAnalyzer := performance.NewPerformanceAnalyzer(logger, analyzerConfig)

	graphMapperConfig := createGraphMapperConfig(cfg)
	graphMapper := performance.NewGraphPerformanceMapper(logger, graphMapperConfig, psAdapter, performanceAnalyzer)

	realtimeConfig := createRealtimeConfig(cfg)
	realtimeMonitor := performance.NewRealtimePerformanceMonitor(logger, realtimeConfig, psAdapter, performanceAnalyzer, graphMapper)

	benchmarkConfig := createBenchmarkConfig(cfg)
	// The source/graph repositories are intentionally nil here: the current
	// benchmark execution path runs sysbench, which connects to the database
	// directly via the configured DatabaseURL. Repositories will be wired in when
	// benchmark-result persistence is added.
	benchmarkService := performance.NewBenchmarkService(nil, nil, nil, performanceAnalyzer, logger, benchmarkConfig)
	registerBenchmarkTools(benchmarkService, cfg, logger)

	if cfg.Performance.Realtime != nil && cfg.Performance.Realtime.Enabled {
		ctx := context.Background()
		if err := realtimeMonitor.Start(ctx); err != nil {
			logrus.Errorf("Failed to start real-time monitor: %v", err)
		} else {
			logrus.Info("Real-time performance monitoring started")
		}
	}

	return &PerformanceServiceContainer{
		BenchmarkService:    benchmarkService,
		PerformanceAnalyzer: performanceAnalyzer,
		PSAdapter:           psAdapter,
		GraphMapper:         graphMapper,
		RealtimeMonitor:     realtimeMonitor,
	}
}

func createGraphMapperConfig(cfg *models.Config) *performance.GraphPerformanceMapperConfig {
	config := &performance.GraphPerformanceMapperConfig{}
	if cfg.Performance.Visualization != nil {
		updateInterval, _ := time.ParseDuration(cfg.Performance.Visualization.UpdateInterval)
		historyRetention, _ := time.ParseDuration(cfg.Performance.Visualization.HistoryRetention)
		config.UpdateInterval = updateInterval
		config.HistoryRetention = historyRetention
		config.MaxConcurrentUpdates = cfg.Performance.Visualization.MaxConcurrentUpdates
		if cfg.Performance.Visualization.EdgeThickness != nil {
			config.EdgeThickness = performance.EdgeThicknessConfig{
				Metric:       cfg.Performance.Visualization.EdgeThickness.Metric,
				Scale:        cfg.Performance.Visualization.EdgeThickness.Scale,
				MinThickness: cfg.Performance.Visualization.EdgeThickness.MinThickness,
				MaxThickness: cfg.Performance.Visualization.EdgeThickness.MaxThickness,
				Multiplier:   cfg.Performance.Visualization.EdgeThickness.Multiplier,
			}
		}
	}
	return config
}

func createRealtimeConfig(cfg *models.Config) *performance.RealtimeMonitorConfig {
	config := &performance.RealtimeMonitorConfig{}
	if cfg.Performance.Realtime != nil {
		updateInterval, _ := time.ParseDuration(cfg.Performance.Realtime.UpdateInterval)
		heartbeatInterval, _ := time.ParseDuration(cfg.Performance.Realtime.HeartbeatInterval)
		writeTimeout, _ := time.ParseDuration(cfg.Performance.Realtime.WriteTimeout)
		readTimeout, _ := time.ParseDuration(cfg.Performance.Realtime.ReadTimeout)
		pingTimeout, _ := time.ParseDuration(cfg.Performance.Realtime.PingTimeout)
		config.DataUpdateInterval = updateInterval
		config.HeartbeatInterval = heartbeatInterval
		config.MaxConnections = cfg.Performance.Realtime.MaxConnections
		config.WriteTimeout = writeTimeout
		config.ReadTimeout = readTimeout
		config.PingTimeout = pingTimeout
		config.MaxMessageSize = cfg.Performance.Realtime.MaxMessageSize
		config.CompressionEnabled = cfg.Performance.Realtime.CompressionEnabled
		if cfg.Performance.Realtime.Alerts != nil {
			config.AlertThresholds = performance.AlertThresholds{
				HighLatency:        cfg.Performance.Realtime.Alerts.HighLatency,
				HighErrorRate:      cfg.Performance.Realtime.Alerts.HighErrorRate,
				HighCPUUsage:       cfg.Performance.Realtime.Alerts.HighCPUUsage,
				HighMemoryUsage:    cfg.Performance.Realtime.Alerts.HighMemoryUsage,
				SlowQueryThreshold: cfg.Performance.Realtime.Alerts.SlowQueryThreshold,
				DeadlockThreshold:  cfg.Performance.Realtime.Alerts.DeadlockThreshold,
			}
		}
	}
	return config
}

func createBenchmarkConfig(cfg *models.Config) *performance.BenchmarkServiceConfig {
	// Start from sane defaults so safety limits and execution defaults are always
	// populated, then override with user configuration where provided.
	config := performance.DefaultBenchmarkServiceConfig()

	// Default the benchmark target to the configured source database so that
	// benchmark requests work without explicitly specifying a connection.
	if url, dbType := benchmarkDatabaseURL(cfg); url != "" {
		config.DefaultDatabaseURL = url
		config.DefaultDatabaseType = dbType
	}

	if b := cfg.Performance.Benchmarks; b != nil {
		if d, err := time.ParseDuration(b.DefaultDuration); err == nil && d > 0 {
			config.DefaultTestDuration = d
		}
		if d, err := time.ParseDuration(b.MaxDuration); err == nil && d > 0 {
			config.MaxDuration = d
		}
		if d, err := time.ParseDuration(b.ResultsRetention); err == nil && d > 0 {
			config.RetainResults = d
		}
		if b.Limits != nil {
			if b.Limits.MaxConcurrentBenchmarks > 0 {
				config.MaxConcurrentRuns = b.Limits.MaxConcurrentBenchmarks
			}
			if b.Limits.MemoryLimitMB > 0 {
				config.MaxResultsInMemory = b.Limits.MemoryLimitMB
			}
		}
		if b.Sysbench != nil && b.Sysbench.Defaults != nil {
			if b.Sysbench.Defaults.Threads > 0 {
				config.DefaultThreads = b.Sysbench.Defaults.Threads
			}
			if b.Sysbench.Defaults.TableSize > 0 {
				config.DefaultTableSize = b.Sysbench.Defaults.TableSize
			}
			if b.Sysbench.Defaults.Time > 0 {
				config.DefaultTestDuration = time.Duration(b.Sysbench.Defaults.Time) * time.Second
			}
		}
	}

	// The execution context must comfortably outlast a default-length run.
	if config.DefaultTimeout <= config.DefaultTestDuration {
		config.DefaultTimeout = config.DefaultTestDuration + 5*time.Minute
	}
	if config.MaxDuration < config.DefaultTestDuration {
		config.MaxDuration = config.DefaultTestDuration
	}

	return config
}

// registerBenchmarkTools wires the available benchmark tools (currently
// sysbench) into the benchmark service. Unavailable tools are logged and
// skipped so the application keeps running without benchmarking support.
func registerBenchmarkTools(svc *performance.BenchmarkService, cfg *models.Config, logger *logrus.Logger) {
	if b := cfg.Performance.Benchmarks; b != nil && !b.Enabled {
		logrus.Info("Benchmarking disabled in configuration; skipping benchmark tool registration")
		return
	}

	sysbenchAdapter := performance.NewSysbenchAdapter(logger, createSysbenchConfig(cfg))
	if err := svc.RegisterBenchmarkTool("sysbench", sysbenchAdapter); err != nil {
		logrus.Warnf("Sysbench benchmark tool unavailable; sysbench benchmarks disabled: %v", err)
		return
	}
	logrus.Info("Registered sysbench benchmark tool")
}

// createSysbenchConfig maps user configuration onto the sysbench adapter config,
// starting from the adapter defaults.
func createSysbenchConfig(cfg *models.Config) *performance.SysbenchConfig {
	sbConfig := performance.DefaultSysbenchConfig()

	if b := cfg.Performance.Benchmarks; b != nil && b.Sysbench != nil {
		if b.Sysbench.ExecutablePath != "" {
			sbConfig.BinaryPath = b.Sysbench.ExecutablePath
		}
		if d := b.Sysbench.Defaults; d != nil && d.TableSize > 0 {
			sbConfig.DefaultTableSize = d.TableSize
		}
	}

	return sbConfig
}

// benchmarkDatabaseURL builds a sysbench-compatible database URL and driver type
// from the active source database configuration.
func benchmarkDatabaseURL(cfg *models.Config) (string, string) {
	dbCfg := cfg.GetDatabaseConfig()
	switch dbCfg.Type {
	case models.DatabaseTypePostgreSQL:
		if pg := dbCfg.PostgreSQL; pg != nil {
			return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
				pg.GetUsername(), pg.GetPassword(), pg.GetHost(), pg.GetPort(), pg.GetDatabase()), "postgresql"
		}
	case models.DatabaseTypeMySQL:
		if my := dbCfg.MySQL; my != nil {
			return fmt.Sprintf("mysql://%s:%s@%s:%d/%s",
				my.GetUsername(), my.GetPassword(), my.GetHost(), my.GetPort(), my.GetDatabase()), "mysql"
		}
	}
	return "", ""
}
