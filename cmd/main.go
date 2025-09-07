/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	neo4jDriver "github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/sirupsen/logrus"

	"sql-graph-visualizer/internal/application/ports"
	graphqlserver "sql-graph-visualizer/internal/application/services/graphql"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/application/services/transform"
	"sql-graph-visualizer/internal/domain/aggregates/graph"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repositories/config"
	"sql-graph-visualizer/internal/domain/repositories/configrule"
	"sql-graph-visualizer/internal/infrastructure/middleware"
	mysqlrepo "sql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"sql-graph-visualizer/internal/infrastructure/persistence/neo4j"
	postgresqlrepo "sql-graph-visualizer/internal/infrastructure/persistence/postgresql"
	"sql-graph-visualizer/internal/interfaces/api"

	// Import database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var addr = "127.0.0.1:3000"

func main() {
	ctx := context.Background()

	// Check for Railway environment or explicit demo mode
	if os.Getenv("DEMO_MODE") == "railway_demo" || (os.Getenv("RAILWAY_ENVIRONMENT") != "" && os.Getenv("DEMO_MODE") != "false") {
		logrus.Info("Railway demo mode requested - starting in demo mode")
		logrus.Infof("Railway environment: %s", os.Getenv("RAILWAY_ENVIRONMENT"))
		logrus.Infof("PORT env var: %s", os.Getenv("PORT"))

		// Start simplified Railway server without database dependencies
		startRailwayDemoServer()
		return
	}

	logrus.Infof("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logrus.Errorf("Failed to load configuration: %v", err)
		// On Railway, try to continue with minimal config
		if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
			logrus.Warn("Railway deployment - creating minimal config...")
			cfg = createMinimalRailwayConfig()
		} else {
			logrus.Fatalf("Failed to load configuration and not on Railway: %v", err)
		}
	}

	// Initialize database connection based on configuration
	var dbPort ports.DatabasePort
	var db *sql.DB

	// Check if we have new multi-database configuration or legacy MySQL
	if cfg.Database != nil && cfg.Database.Type != "" {
		logrus.Infof("Using new multi-database configuration: %s", cfg.Database.Type)

		// For now, PostgreSQL will use the existing MySQL port interface
		// This is a temporary workaround until we have a unified database interface
		switch cfg.Database.Type {
		case models.DatabaseTypePostgreSQL:
			pgConfig := cfg.Database.PostgreSQL
			logrus.Infof("Connecting to PostgreSQL: %s@%s:%d/%s", pgConfig.GetUsername(), pgConfig.GetHost(), pgConfig.GetPort(), pgConfig.GetDatabase())

			// Create PostgreSQL repository
			postgresRepo := postgresqlrepo.NewPostgreSQLRepository(nil)
			db, err = postgresRepo.ConnectToExisting(ctx, pgConfig)
			if err != nil {
				logrus.Fatalf("Failed to connect to PostgreSQL: %v", err)
			}

			// Use PostgreSQL repository as DatabasePort
			dbPort = postgresqlrepo.NewPostgreSQLDatabasePort(db)
			logrus.Infof("Successfully connected to PostgreSQL database")

		case models.DatabaseTypeMySQL:
			mysqlConfig := cfg.Database.MySQL
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				mysqlConfig.GetUsername(),
				mysqlConfig.GetPassword(),
				mysqlConfig.GetHost(),
				mysqlConfig.GetPort(),
				mysqlConfig.GetDatabase(),
			)

			db, err = sql.Open("mysql", dsn)
			if err != nil {
				logrus.Fatalf("Failed to connect to MySQL: %v", err)
			}

			dbPort = mysqlrepo.NewMySQLDatabasePort(db)
			logrus.Infof("Successfully connected to MySQL database")

		default:
			logrus.Fatalf("Unsupported database type: %s", cfg.Database.Type)
		}

	} else {
		// Legacy MySQL configuration
		logrus.Infof("Using legacy MySQL configuration")

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			cfg.MySQL.User,
			cfg.MySQL.Password,
			cfg.MySQL.Host,
			cfg.MySQL.Port,
			cfg.MySQL.Database,
		)

		logrus.Infof("DSN: %s", dsn)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			logrus.Fatalf("Failed to connect to MySQL: %v", err)
		}

		dbPort = mysqlrepo.NewMySQLDatabasePort(db)
		logrus.Infof("MySQL connection successful")
	}

	defer func() {
		if err := db.Close(); err != nil {
			logrus.Errorf("Error closing database connection: %v", err)
		}
	}()

	logrus.Infof("Initializing Neo4j connection...")
	neo4jRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		logrus.Fatalf("Failed to create Neo4j repository: %v", err)
	}
	logrus.Infof("Neo4j connection successful")
	defer func() {
		if err := neo4jRepo.Close(); err != nil {
			logrus.Errorf("Error closing Neo4j repository: %v", err)
		}
	}()

	logrus.Infof("Deleting all data in Neo4j...")
	session := neo4jRepo.NewSession(neo4jDriver.SessionConfig{})
	defer func() {
		if err := session.Close(); err != nil {
			logrus.Errorf("Error closing session: %v", err)
		}
	}()

	_, err = session.Run("MATCH (n) DETACH DELETE n", nil)
	if err != nil {
		logrus.Fatalf("Error deleting data in Neo4j: %v", err)
	}
	logrus.Infof("All data in Neo4j deleted")

	logrus.Infof("Initializing services...")
	transformService := transform.NewTransformService(dbPort, neo4jRepo, configrule.NewRuleRepository())

	// Initialize performance services if enabled
	var performanceServices *PerformanceServiceContainer
	if cfg.Performance != nil && cfg.Performance.Monitoring != nil && cfg.Performance.Monitoring.Enabled {
		logrus.Info("Initializing performance .monitoring services...")
		performanceServices = initializePerformanceServices(cfg, db)
		logrus.Info("Performance services initialized")
	} else {
		logrus.Info("Performance .monitoring is disabled")
	}

	// Initialize SimpleMetricsInjector for demo visualization (always enabled)
	logrus.Info("Initializing performance metrics visualization...")
	metricsInjectorConfig := &performance.SimpleMetricsConfig{
		UpdateInterval:   5 * time.Second,
		MetricsRetention: 1 * time.Hour,
		SimulationMode:   true,
	}

	metricsInjector := performance.NewSimpleMetricsInjector(neo4jRepo, logrus.StandardLogger(), metricsInjectorConfig)

	// Start MetricsInjector for live performance visualization
	if err := metricsInjector.Start(ctx); err != nil {
		logrus.Errorf("Failed to start metrics injector: %v", err)
	} else {
		logrus.Info("🚀 Performance metrics visualization started!")
	}

	logrus.Infof("Services initialized")

	// Start GraphQL server
	graphqlserver.StartGraphQLServer(neo4jRepo, cfg)
	logrus.Info("GraphQL server started")

	logrus.Infof("Starting data transformation...")
	if err := transformService.TransformAndStore(ctx); err != nil {
		logrus.Fatalf("Failed to transform and store data: %v", err)
	}
	logrus.Infof("Data transformation successful")

	logrus.Infof("Starting server...")
	startVisualizationServer(neo4jRepo, cfg)

	router := mux.NewRouter()

	// Register performance routes if services are initialized
	if performanceServices != nil {
		logrus.Info("Registering performance API routes...")
		performanceHandlers := api.NewPerformanceHandlers(
			logrus.StandardLogger(),
			performanceServices.BenchmarkService,
			performanceServices.PerformanceAnalyzer,
			performanceServices.GraphMapper,
			performanceServices.RealtimeMonitor,
			performanceServices.PSAdapter,
		)
		performanceHandlers.RegisterRoutes(router)
		logrus.Info("Performance API routes registered")
	}

	// Health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Health check requested")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Test database connectivity
		dbStatus := "unknown"
		if db != nil {
			if err := db.Ping(); err == nil {
				dbStatus = "connected"
			} else {
				dbStatus = "error: " + err.Error()
			}
		} else {
			dbStatus = "not_initialized"
		}

		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "v1.1.0",
			"database":  dbStatus,
			"neo4j":     "connected", // TODO: Add real Neo4j health check
			"environment": map[string]string{
				"railway":    getEnvOrDefault("RAILWAY_ENVIRONMENT", "not_set"),
				"port":       getEnvOrDefault("PORT", "not_set"),
				"mysql_host": getEnvOrDefault("MYSQL_HOST", "not_set"),
				"neo4j_uri": func() string {
					if uri := os.Getenv("NEO4J_URI"); uri != "" {
						return "set"
					}
					return "not_set"
				}(),
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error encoding health response: %v", err)
			http.Error(w, "Health check failed", http.StatusInternalServerError)
		}
	})

	router.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(cfg); err != nil {
			logrus.Errorf("Error encoding config: %v", err)
		}
	})

	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(router)

	// Use PORT environment variable if available (for Railway deployment)
	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	apiAddr := ":" + apiPort // Listen on all interfaces

	server := &http.Server{
		Handler:           handler,
		Addr:              apiAddr,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logrus.Infof("Starting API server on %s", apiAddr)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

func startVisualizationServer(neo4jRepo ports.Neo4jPort, cfg *models.Config) *http.Server {
	logrus.Infof("Starting visualization server")
	mux := http.NewServeMux()

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to /config endpoint")
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
			return
		}
		logrus.Infof("Config response sent successfully")
	})

	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to API endpoint /api/graph")

		graphInterface, err := neo4jRepo.ExportGraph("MATCH (n)-[r]->(m) RETURN n, r, m")
		if err != nil {
			logrus.Errorf("Error retrieving data: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		g, ok := graphInterface.(*graph.GraphAggregate)
		if !ok {
			logrus.Warnf("Invalid graph type")
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
			nodeData := map[string]any{
				"id":         node.ID,
				"label":      node.Type,
				"properties": node.Properties,
			}
			response.Nodes = append(response.Nodes, nodeData)
			logrus.Infof("Adding node: %v", nodeData)
		}

		for _, rel := range g.GetRelationships() {
			relData := map[string]any{
				"from":       rel.SourceNode.ID,
				"to":         rel.TargetNode.ID,
				"type":       rel.Type,
				"properties": rel.Properties,
			}
			response.Relationships = append(response.Relationships, relData)
			logrus.Infof("Adding relationship: %v", relData)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		logrus.Infof("Sending response: %d nodes, %d relationships", len(response.Nodes), len(response.Relationships))

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error serializing response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	webRoot := filepath.Join(findProjectRoot(), "internal", "interfaces", "web")
	logrus.Infof("Using web root: %s", webRoot)

	fs := http.FileServer(http.Dir(filepath.Join(webRoot, "static")))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Performance dashboard route
	mux.HandleFunc("/performance", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to performance dashboard")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "performance_dashboard.html"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to main page")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.Warnf("Port %s is occupied: %v", addr, err)
		if err := exec.Command("fuser", "-k", "3000/tcp").Run(); err != nil {
			logrus.Warnf("Error killing processes on port 3000: %v", err)
		}
		time.Sleep(time.Second)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			logrus.Fatalf("Cannot create listener: %v", err)
		}
	}
	logrus.Infof("Listener created on %s", addr)

	server := &http.Server{
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logrus.Warnf("Starting server on %s", addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Server terminated with error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logrus.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Error shutting down server: %v", err)
	}
	logrus.Println("Server successfully shut down")

	logrus.Infof("Visualization is available at http://localhost:3000")
	return server
}

func findProjectRoot() string {
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
			logrus.Fatalf("Cannot find project root directory")
			return ""
		}
		wd = parent
	}
}

// startRailwayDemoServer starts a simplified server for Railway deployment without database dependencies
func startRailwayDemoServer() {
	logrus.Info("Starting Railway demo server...")

	router := mux.NewRouter()

	// Health check endpoint - essential for Railway deployment
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Health check requested")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "v1.1.0-railway",
			"mode":      "demo",
			"database":  "demo_mode",
			"neo4j":     "demo_mode",
			"environment": map[string]string{
				"railway":    getEnvOrDefault("RAILWAY_ENVIRONMENT", "not_set"),
				"port":       getEnvOrDefault("PORT", "not_set"),
				"mysql_host": getEnvOrDefault("MYSQL_HOST", "not_set"),
				"neo4j_uri":  "demo_mode",
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error encoding health response: %v", err)
			http.Error(w, "Health check failed", http.StatusInternalServerError)
		}
	}).Methods("GET")

	// Use PORT environment variable if available (for Railway deployment)
	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	apiAddr := ":" + apiPort // Listen on all interfaces

	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(router)

	server := &http.Server{
		Handler:           handler,
		Addr:              apiAddr,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logrus.Infof("Starting Railway demo server on %s", apiAddr)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Failed to start Railway demo server: %v", err)
	}
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func init() {
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
}

// PerformanceServiceContainer holds all performance-related services
type PerformanceServiceContainer struct {
	BenchmarkService    *performance.BenchmarkService
	PerformanceAnalyzer *performance.PerformanceAnalyzer
	PSAdapter           *performance.PerformanceSchemaAdapter
	GraphMapper         *performance.GraphPerformanceMapper
	RealtimeMonitor     *performance.RealtimePerformanceMonitor
	MetricsInjector     *performance.SimpleMetricsInjector
}

// initializePerformanceServices creates and configures all performance services
func initializePerformanceServices(cfg *models.Config, db *sql.DB) *PerformanceServiceContainer {
	logger := logrus.StandardLogger()

	// Parse configuration durations
	updateInterval, err := time.ParseDuration(cfg.Performance.Monitoring.UpdateInterval)
	if err != nil {
		logrus.Warnf("Invalid update_interval, using default 5s: %v", err)
		updateInterval = 5 * time.Second
	}

	// Cache duration is handled internally by the performance schema adapter

	// Create Performance Schema Adapter configuration with safe defaults
	maxStatements := 100
	maxTables := 50
	if cfg.Performance != nil && cfg.Performance.Monitoring != nil && cfg.Performance.Monitoring.PerformanceSchema != nil {
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

	// Initialize Performance Schema Adapter
	psAdapter := performance.NewPerformanceSchemaAdapter(db, logger, psConfig)

	// Create Performance Analyzer configuration with safe defaults
	slowQueryThreshold := 200.0 // Default 200ms
	if cfg.Performance != nil && cfg.Performance.Monitoring != nil && cfg.Performance.Monitoring.Analysis != nil {
		slowQueryThreshold = cfg.Performance.Monitoring.Analysis.SlowQueryThreshold
	}

	analyzerConfig := &performance.PerformanceAnalyzerConfig{
		HighLatencyThreshold:      time.Duration(slowQueryThreshold) * time.Millisecond,
		LowThroughputThreshold:    10.0, // Default value
		HighErrorRateThreshold:    1.0,  // Default value
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

	// Initialize Performance Analyzer
	performanceAnalyzer := performance.NewPerformanceAnalyzer(logger, analyzerConfig)

	// Create Graph Performance Mapper configuration
	graphMapperConfig := createGraphMapperConfig(cfg)

	// Initialize Graph Performance Mapper
	graphMapper := performance.NewGraphPerformanceMapper(logger, graphMapperConfig, psAdapter, performanceAnalyzer)

	// Create Real-time Monitor configuration
	realtimeConfig := createRealtimeConfig(cfg)

	// Initialize Real-time Performance Monitor
	realtimeMonitor := performance.NewRealtimePerformanceMonitor(logger, realtimeConfig, psAdapter, performanceAnalyzer, graphMapper)

	// Create Benchmark Service configuration
	benchmarkConfig := createBenchmarkConfig(cfg)

	// TODO: Initialize benchmark tools when implemented
	// For now, create benchmark service with minimal configuration
	benchmarkService := performance.NewBenchmarkService(nil, nil, nil, performanceAnalyzer, logger, benchmarkConfig)

	// Start real-time .monitoring if enabled
	if cfg.Performance != nil && cfg.Performance.Realtime != nil && cfg.Performance.Realtime.Enabled {
		ctx := context.Background()
		if err := realtimeMonitor.Start(ctx); err != nil {
			logrus.Errorf("Failed to start real-time monitor: %v", err)
		} else {
			logrus.Info("Real-time performance .monitoring started")
		}
	}

	return &PerformanceServiceContainer{
		BenchmarkService:    benchmarkService,
		PerformanceAnalyzer: performanceAnalyzer,
		PSAdapter:           psAdapter,
		GraphMapper:         graphMapper,
		RealtimeMonitor:     realtimeMonitor,
		MetricsInjector:     nil, // Handled separately in main function
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

		// Set other visualization configs similarly...
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

// createMinimalRailwayConfig creates a basic config when YAML loading fails on Railway
func createMinimalRailwayConfig() *models.Config {
	logrus.Info("Creating minimal Railway configuration from environment variables...")

	return &models.Config{
		MySQL: models.MySQLConfig{
			Host:     os.Getenv("MYSQL_HOST"),
			Port:     3306,
			User:     os.Getenv("MYSQL_USER"),
			Password: os.Getenv("MYSQL_PASSWORD"),
			Database: os.Getenv("MYSQL_DATABASE"),
		},
		Neo4j: models.Neo4jConfig{
			URI:      os.Getenv("NEO4J_URI"),
			User:     getEnvOrDefault("NEO4J_USER", "neo4j"),
			Password: os.Getenv("NEO4J_PASSWORD"),
		},
		TransformRules: []models.TransformationConfig{
			{
				Name:     "demo_rule",
				RuleType: "node",
				Source: models.SourceConfig{
					Type:  "query",
					Value: "SELECT 'Railway Demo' as name, 'demo' as type",
				},
				TargetType: "DemoNode",
				FieldMappings: map[string]string{
					"name": "name",
					"type": "type",
				},
			},
		},
	}
}

func createBenchmarkConfig(cfg *models.Config) *performance.BenchmarkServiceConfig {
	config := &performance.BenchmarkServiceConfig{}

	if cfg.Performance.Benchmarks != nil {
		defaultDuration, _ := time.ParseDuration(cfg.Performance.Benchmarks.DefaultDuration)
		maxDuration, _ := time.ParseDuration(cfg.Performance.Benchmarks.MaxDuration)
		resultsRetention, _ := time.ParseDuration(cfg.Performance.Benchmarks.ResultsRetention)
		cleanupInterval := 15 * time.Minute // Default cleanup interval

		config.DefaultTimeout = defaultDuration
		config.MaxDuration = maxDuration
		config.RetainResults = resultsRetention
		config.CleanupInterval = cleanupInterval

		if cfg.Performance.Benchmarks.Limits != nil {
			config.MaxConcurrentRuns = cfg.Performance.Benchmarks.Limits.MaxConcurrentBenchmarks
			config.MaxResultsInMemory = cfg.Performance.Benchmarks.Limits.MemoryLimitMB
			// CPUThreshold not available in BenchmarkServiceConfig
		}
	}

	return config
}
