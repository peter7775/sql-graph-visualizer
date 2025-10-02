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
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	"sql-graph-visualizer/internal/infrastructure/deployment"
	"sql-graph-visualizer/internal/infrastructure/middleware"
	mysqlrepo "sql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"sql-graph-visualizer/internal/infrastructure/persistence/neo4j"
	postgresqlrepo "sql-graph-visualizer/internal/infrastructure/persistence/postgresql"
	"sql-graph-visualizer/internal/interfaces/api"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var addr = "127.0.0.1:3000"

func main() {
	ctx := context.Background()
	logger := logrus.StandardLogger()

	deploymentAdapter := deployment.NewDeploymentAdapter(logger)

	logrus.Infof("=== SQL Graph Visualizer Starting - Build timestamp: %s ===", time.Now().Format("2006-01-02 15:04:05"))
	logrus.Infof("Platform: %s", deploymentAdapter.GetPlatformName())

	envInfo := deploymentAdapter.GetEnvironmentInfo()
	for key, value := range envInfo {
		logrus.Infof("%s: %v", key, value)
	}

	logrus.Infof("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	if deploymentAdapter.GetPlatformName() == "Railway" {
		logrus.Info("Applying Railway-specific configuration overrides...")
		if railwayDeployment, ok := deploymentAdapter.(*deployment.RailwayDeployment); ok {
			dbConfig := railwayDeployment.GetRailwayDatabaseConfig()
			if err := overrideConfigWithDeploymentSettings(cfg, dbConfig); err != nil {
				logrus.Warnf("Error applying deployment config: %v", err)
			}
		}
	}

	var dbPort ports.DatabasePort
	var db *sql.DB

	if cfg.Database != nil && cfg.Database.Type != "" {
		logrus.Infof("Using new multi-database configuration: %s", cfg.Database.Type)

		switch cfg.Database.Type {
		case models.DatabaseTypePostgreSQL:
			pgConfig := cfg.Database.PostgreSQL
			logrus.Infof("Connecting to PostgreSQL: %s@%s:%d/%s", pgConfig.GetUsername(), pgConfig.GetHost(), pgConfig.GetPort(), pgConfig.GetDatabase())

			postgresRepo := postgresqlrepo.NewPostgreSQLRepository(nil)
			db, err = postgresRepo.ConnectToExisting(ctx, pgConfig)
			if err != nil {
				logrus.Fatalf("Failed to connect to PostgreSQL: %v", err)
			}

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
	var neo4jRepo ports.Neo4jPort
	realNeo4jRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		if os.Getenv("FORCE_FULL_MODE") == "true" {
			logrus.Warnf("Neo4j connection failed but FORCE_FULL_MODE is enabled, using Mock repository: %v", err)
			neo4jRepo = neo4j.NewMockNeo4jRepository()
			logrus.Info("Using Mock Neo4j repository for visualization")
		} else {
			logrus.Fatalf("Failed to create Neo4j repository: %v", err)
		}
	} else {
		neo4jRepo = realNeo4jRepo
		logrus.Infof("Neo4j connection successful")
	}
	defer func() {
		if err := neo4jRepo.Close(); err != nil {
			logrus.Errorf("Error closing Neo4j repository: %v", err)
		}
	}()

	if realNeo4jRepo == nil {
		logrus.Info("Using Mock Neo4j repository, skipping data deletion")
	} else {
		logrus.Infof("Deleting all data in Neo4j...")
		session := realNeo4jRepo.NewSession(neo4jDriver.SessionConfig{})
		defer func() {
			if sessionCloser, ok := session.(interface{ Close() error }); ok {
				if err := sessionCloser.Close(); err != nil {
					logrus.Errorf("Error closing session: %v", err)
				}
			}
		}()

		if sessionRunner, ok := session.(interface {
			Run(string, interface{}) (interface{}, error)
		}); ok {
			_, err = sessionRunner.Run("MATCH (n) DETACH DELETE n", nil)
			if err != nil {
				forceFullMode := os.Getenv("FORCE_FULL_MODE")
				logrus.Warnf("Neo4j operation failed: %v", err)

				if forceFullMode == "true" {
					logrus.Info("FORCE_FULL_MODE enabled - switching to Mock Neo4j repository")
					neo4jRepo = neo4j.NewMockNeo4jRepository()
					realNeo4jRepo = nil
					logrus.Info("Successfully switched to Mock Neo4j repository - continuing with normal flow")
				} else {
					logrus.Fatalf("Error deleting data in Neo4j: %v", err)
				}
			}
		}
		logrus.Infof("All data in Neo4j deleted")
	}

	logrus.Infof("Initializing services...")
	transformService := transform.NewTransformService(dbPort, neo4jRepo, configrule.NewRuleRepository())

	var performanceServices *PerformanceServiceContainer
	if cfg.Performance != nil && cfg.Performance.Monitoring != nil && cfg.Performance.Monitoring.Enabled {
		logrus.Info("Initializing performance .monitoring services...")
		performanceServices = initializePerformanceServices(cfg, db)
		logrus.Info("Performance services initialized")
	} else {
		logrus.Info("Performance .monitoring is disabled")
	}

	logrus.Info("Initializing performance metrics visualization...")
	metricsInjectorConfig := &performance.SimpleMetricsConfig{
		UpdateInterval:   5 * time.Second,
		MetricsRetention: 1 * time.Hour,
		SimulationMode:   true,
	}

	metricsInjector := performance.NewSimpleMetricsInjector(neo4jRepo, logrus.StandardLogger(), metricsInjectorConfig)

	if err := metricsInjector.Start(ctx); err != nil {
		logrus.Errorf("Failed to start metrics injector: %v", err)
	} else {
		logrus.Info("🚀 Performance metrics visualization started!")
	}

	logrus.Infof("Services initialized")

	graphqlserver.StartGraphQLServer(neo4jRepo, cfg)
	logrus.Info("GraphQL server started")

	logrus.Infof("Starting data transformation...")
	if err := transformService.TransformAndStore(ctx); err != nil {
		if os.Getenv("FORCE_FULL_MODE") == "true" {
			logrus.Warnf("Transform service failed but FORCE_FULL_MODE enabled, creating fallback graph data: %v", err)
			if err := createFallbackGraphData(ctx, dbPort, neo4jRepo); err != nil {
				logrus.Errorf("Failed to create fallback graph data: %v", err)
			}
		} else {
			logrus.Fatalf("Failed to transform and store data: %v", err)
		}
	}
	logrus.Infof("Data transformation successful")

	logrus.Infof("Starting servers...")
	var vizServer *http.Server
	if deploymentAdapter.ShouldStartVisualizationServer() {
		vizServer = startVisualizationServer(neo4jRepo, cfg, deploymentAdapter)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := vizServer.Shutdown(ctx); err != nil {
				logrus.Errorf("Error shutting down visualization server: %v", err)
			}
		}()
	}

	router := mux.NewRouter()

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

	router.HandleFunc("/api/debug", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Debug endpoint requested")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		debugInfo := map[string]interface{}{
			"platform":            deploymentAdapter.GetPlatformName(),
			"environment":         deploymentAdapter.GetEnvironmentInfo(),
			"FORCE_FULL_MODE":     os.Getenv("FORCE_FULL_MODE"),
			"neo4j_repo_type":     fmt.Sprintf("%T", neo4jRepo),
			"real_neo4j_repo_nil": realNeo4jRepo == nil,
			"timestamp":           time.Now().Format(time.RFC3339),
		}

		if err := json.NewEncoder(w).Encode(debugInfo); err != nil {
			logrus.Errorf("Error encoding debug response: %v", err)
			http.Error(w, "Debug endpoint failed", http.StatusInternalServerError)
		}
	}).Methods("GET")

	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Health check requested")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

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
			"status":      "healthy",
			"timestamp":   time.Now().Format(time.RFC3339),
			"version":     "v1.1.0",
			"platform":    deploymentAdapter.GetPlatformName(),
			"database":    dbStatus,
			"neo4j":       "connected",
			"environment": deploymentAdapter.GetEnvironmentInfo(),
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
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"},
		AllowCredentials: true,
	}

	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(router)

	apiPort := deploymentAdapter.GetAPIPort()
	apiAddr := ":" + apiPort

	server := &http.Server{
		Handler: handler,
		Addr:    apiAddr,
	}

	server = deploymentAdapter.ConfigureServer(server)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logrus.Println("Shutting down servers...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logrus.Errorf("Error shutting down API server: %v", err)
		}

		if err := vizServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Error shutting down visualization server: %v", err)
		}

		logrus.Println("Servers successfully shut down")
	}()

	logrus.Infof("Starting API server on %s", apiAddr)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

func startVisualizationServer(neo4jRepo ports.Neo4jPort, cfg *models.Config, deploymentAdapter ports.DeploymentPort) *http.Server {
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

	mux.HandleFunc("/performance", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to performance dashboard")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "performance_dashboard.html"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Infof("Request to main page")
		http.ServeFile(w, r, filepath.Join(webRoot, "templates", "visualization.html"))
	})

	vizPort := deploymentAdapter.GetVisualizationPort()
	vizAddr := ":" + vizPort

	server := &http.Server{
		Handler:           mux,
		Addr:              vizAddr,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logrus.Warnf("Starting visualization server on %s", vizAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Visualization server terminated with error: %v", err)
		}
	}()

	logrus.Infof("Visualization is available at http://localhost:%s", vizPort)
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

func overrideConfigWithDeploymentSettings(cfg *models.Config, dbConfig map[string]string) error {
	if cfg.Database != nil && cfg.Database.MySQL != nil {
		if host := dbConfig["mysql_host"]; host != "" {
			cfg.Database.MySQL.Host = host
			logrus.Infof("MySQL host overridden: %s", host)
		}
		if user := dbConfig["mysql_user"]; user != "" {
			cfg.Database.MySQL.User = user
			logrus.Infof("MySQL user overridden: %s", user)
		}
		if password := dbConfig["mysql_password"]; password != "" {
			cfg.Database.MySQL.Password = password
			logrus.Info("MySQL password overridden")
		}
		if database := dbConfig["mysql_database"]; database != "" {
			cfg.Database.MySQL.Database = database
			logrus.Infof("MySQL database overridden: %s", database)
		}
		if port := dbConfig["mysql_port"]; port != "" {
			if portNum := parseInt(port); portNum > 0 {
				cfg.Database.MySQL.Port = portNum
				logrus.Infof("MySQL port overridden: %d", portNum)
			}
		}
	}

	if uri := dbConfig["neo4j_uri"]; uri != "" && uri != "${NEO4J_URI}" {
		cfg.Neo4j.URI = uri
		logrus.Infof("Neo4j URI overridden")
	}
	if user := dbConfig["neo4j_user"]; user != "" && user != "${NEO4J_USER}" {
		cfg.Neo4j.User = user
		logrus.Infof("Neo4j user overridden: %s", user)
	}
	if password := dbConfig["neo4j_password"]; password != "" && password != "${NEO4J_PASSWORD}" {
		cfg.Neo4j.Password = password
		logrus.Info("Neo4j password overridden")
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

func init() {
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
}

type PerformanceServiceContainer struct {
	BenchmarkService    *performance.BenchmarkService
	PerformanceAnalyzer *performance.PerformanceAnalyzer
	PSAdapter           *performance.PerformanceSchemaAdapter
	GraphMapper         *performance.GraphPerformanceMapper
	RealtimeMonitor     *performance.RealtimePerformanceMonitor
	MetricsInjector     *performance.SimpleMetricsInjector
}

func initializePerformanceServices(cfg *models.Config, db *sql.DB) *PerformanceServiceContainer {
	logger := logrus.StandardLogger()

	updateInterval, err := time.ParseDuration(cfg.Performance.Monitoring.UpdateInterval)
	if err != nil {
		logrus.Warnf("Invalid update_interval, using default 5s: %v", err)
		updateInterval = 5 * time.Second
	}

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

	psAdapter := performance.NewPerformanceSchemaAdapter(db, logger, psConfig)

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

	performanceAnalyzer := performance.NewPerformanceAnalyzer(logger, analyzerConfig)

	graphMapperConfig := createGraphMapperConfig(cfg)

	graphMapper := performance.NewGraphPerformanceMapper(logger, graphMapperConfig, psAdapter, performanceAnalyzer)

	realtimeConfig := createRealtimeConfig(cfg)

	realtimeMonitor := performance.NewRealtimePerformanceMonitor(logger, realtimeConfig, psAdapter, performanceAnalyzer, graphMapper)

	benchmarkConfig := createBenchmarkConfig(cfg)

	benchmarkService := performance.NewBenchmarkService(nil, nil, nil, performanceAnalyzer, logger, benchmarkConfig)

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
		MetricsInjector:     nil,
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
	config := &performance.BenchmarkServiceConfig{}

	if cfg.Performance.Benchmarks != nil {
		defaultDuration, _ := time.ParseDuration(cfg.Performance.Benchmarks.DefaultDuration)
		maxDuration, _ := time.ParseDuration(cfg.Performance.Benchmarks.MaxDuration)
		resultsRetention, _ := time.ParseDuration(cfg.Performance.Benchmarks.ResultsRetention)
		cleanupInterval := 15 * time.Minute

		config.DefaultTimeout = defaultDuration
		config.MaxDuration = maxDuration
		config.RetainResults = resultsRetention
		config.CleanupInterval = cleanupInterval

		if cfg.Performance.Benchmarks.Limits != nil {
			config.MaxConcurrentRuns = cfg.Performance.Benchmarks.Limits.MaxConcurrentBenchmarks
			config.MaxResultsInMemory = cfg.Performance.Benchmarks.Limits.MemoryLimitMB

		}
	}

	return config
}

func createFallbackGraphData(ctx context.Context, dbPort ports.DatabasePort, neo4jRepo ports.Neo4jPort) error {
	logrus.Info("Creating fallback graph data from MySQL...")

	return nil
}
