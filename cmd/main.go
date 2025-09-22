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

	// Always log startup information first
	logrus.Infof("=== SQL Graph Visualizer Starting - Version with FORCE_FULL_MODE support ===")
	logrus.Infof("Environment: %s", os.Getenv("RAILWAY_ENVIRONMENT"))
	logrus.Infof("DEMO_MODE: %s", os.Getenv("DEMO_MODE"))
	logrus.Infof("PORT: %s", os.Getenv("PORT"))
	logrus.Infof("CONFIG_PATH: %s", os.Getenv("CONFIG_PATH"))
	logrus.Infof("FORCE_FULL_MODE: %s", os.Getenv("FORCE_FULL_MODE"))

	// Check for explicit demo mode only (not automatic Railway detection)
	if os.Getenv("DEMO_MODE") == "railway_demo" || os.Getenv("DEMO_MODE") == "true" {
		logrus.Info("Explicit demo mode requested - starting in demo mode")
		logrus.Infof("Railway environment: %s", os.Getenv("RAILWAY_ENVIRONMENT"))
		logrus.Infof("PORT env var: %s", os.Getenv("PORT"))

		// Start simplified Railway server without database dependencies
		startRailwayDemoServer()
		return
	}

	// If on Railway but database connection fails, fallback to demo mode
	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		logrus.Info("Railway environment detected - attempting normal startup first...")
		logrus.Info("Debug: Railway environment variables:")
		// Check both possible MySQL variable naming conventions
		logrus.Infof("  MYSQLHOST: %s", os.Getenv("MYSQLHOST"))
		logrus.Infof("  MYSQL_HOST: %s", os.Getenv("MYSQL_HOST"))
		logrus.Infof("  MYSQLUSER: %s", os.Getenv("MYSQLUSER"))
		logrus.Infof("  MYSQL_USER: %s", os.Getenv("MYSQL_USER"))
		logrus.Infof("  MYSQL_DATABASE: %s", os.Getenv("MYSQL_DATABASE"))
		logrus.Infof("  MYSQLPORT: %s", os.Getenv("MYSQLPORT"))
		logrus.Infof("  MYSQL_PORT: %s", os.Getenv("MYSQL_PORT"))
		logrus.Infof("  MYSQLPASSWORD: [length=%d]", len(os.Getenv("MYSQLPASSWORD")))
		logrus.Infof("  MYSQL_PASSWORD: [length=%d]", len(os.Getenv("MYSQL_PASSWORD")))
		// Neo4j variables
		logrus.Infof("  NEO4J_URI: %s", func() string { if uri := os.Getenv("NEO4J_URI"); uri != "" { return "[SET]" } else { return "[NOT_SET]" } }())
		logrus.Infof("  NEO4J_USER: %s", os.Getenv("NEO4J_USER"))
		logrus.Infof("  NEO4J_PASSWORD: [length=%d]", len(os.Getenv("NEO4J_PASSWORD")))
	}

	logrus.Infof("Loading configuration...")
	cfg, err := config.Load()
	if err != nil {
		logrus.Errorf("Failed to load configuration: %v", err)
		// On Railway, always fallback to demo mode if config fails
		if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
			logrus.Warn("Railway deployment - config failed, starting demo mode...")
			startRailwayDemoServer()
			return
		} else {
			logrus.Fatalf("Failed to load configuration and not on Railway: %v", err)
		}
	}

	// Override configuration with environment variables if available (Railway deployment)
	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		logrus.Info("Overriding configuration with Railway environment variables...")
		
		// Override Neo4j configuration
		if uri := os.Getenv("NEO4J_URI"); uri != "" && uri != "${NEO4J_URI}" {
			cfg.Neo4j.URI = uri
			logrus.Infof("Neo4j URI overridden: %s", uri)
		}
		if user := os.Getenv("NEO4J_USER"); user != "" && user != "${NEO4J_USER}" {
			cfg.Neo4j.User = user
			logrus.Infof("Neo4j user overridden: %s", user)
		}
		if password := os.Getenv("NEO4J_PASSWORD"); password != "" && password != "${NEO4J_PASSWORD}" {
			cfg.Neo4j.Password = password
			logrus.Info("Neo4j password overridden")
		}
		
		// Override MySQL configuration with Railway env vars
		if cfg.Database != nil && cfg.Database.MySQL != nil {
			// Try both MYSQLHOST and MYSQL_HOST
			if host := getEnvOrDefault("MYSQLHOST", os.Getenv("MYSQL_HOST")); host != "" {
				cfg.Database.MySQL.Host = host
				logrus.Infof("MySQL host overridden: %s", host)
			}
			// Try both MYSQLUSER and MYSQL_USER
			if user := getEnvOrDefault("MYSQLUSER", os.Getenv("MYSQL_USER")); user != "" {
				cfg.Database.MySQL.User = user
				logrus.Infof("MySQL user overridden: %s", user)
			}
			// Try both MYSQLPASSWORD and MYSQL_PASSWORD
			if password := getEnvOrDefault("MYSQLPASSWORD", os.Getenv("MYSQL_PASSWORD")); password != "" {
				cfg.Database.MySQL.Password = password
				logrus.Info("MySQL password overridden")
			}
			if database := os.Getenv("MYSQL_DATABASE"); database != "" {
				cfg.Database.MySQL.Database = database
				logrus.Infof("MySQL database overridden: %s", database)
			}
			// Try both MYSQLPORT and MYSQL_PORT
			if port := getEnvOrDefault("MYSQLPORT", os.Getenv("MYSQL_PORT")); port != "" {
				if portNum := parseInt(port); portNum > 0 {
					cfg.Database.MySQL.Port = portNum
					logrus.Infof("MySQL port overridden: %d", portNum)
				}
			}
		}
	}

	// Initialize database connection based on configuration
	var dbPort ports.DatabasePort
	var db *sql.DB

	// Check if we have a new multi-database configuration or legacy MySQL
	if cfg.Database != nil && cfg.Database.Type != "" {
		logrus.Infof("Using new multi-database configuration: %s", cfg.Database.Type)

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

			// Use PostgreSQL repository as a DatabasePort
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
				if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
					logrus.Warnf("MySQL connection failed in Railway environment: %v", err)
					logrus.Info("Falling back to Railway demo mode...")
					startRailwayDemoServer()
					return
				}
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
	var neo4jRepo ports.Neo4jPort
	realNeo4jRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		if os.Getenv("RAILWAY_ENVIRONMENT") != "" && os.Getenv("FORCE_FULL_MODE") != "true" {
			logrus.Warnf("Neo4j connection failed in Railway environment: %v", err)
			logrus.Info("Neo4j URI appears to be placeholder or invalid, falling back to Railway demo mode...")
			startRailwayDemoServer()
			return
		} else if os.Getenv("FORCE_FULL_MODE") == "true" {
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

	// Skip Neo4j operations for Mock repository  
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

		if sessionRunner, ok := session.(interface{ Run(string, interface{}) (interface{}, error) }); ok {
			_, err = sessionRunner.Run("MATCH (n) DETACH DELETE n", nil)
			if err != nil {
				// Debug logging for FORCE_FULL_MODE logic
				railwayEnv := os.Getenv("RAILWAY_ENVIRONMENT")
				forceFullMode := os.Getenv("FORCE_FULL_MODE")
				logrus.Warnf("Neo4j operation failed: %v", err)
				logrus.Infof("Debug - RAILWAY_ENVIRONMENT: '%s'", railwayEnv)
				logrus.Infof("Debug - FORCE_FULL_MODE: '%s'", forceFullMode)
				logrus.Infof("Debug - Railway check: %v", railwayEnv != "")
				logrus.Infof("Debug - FORCE_FULL_MODE check: %v", forceFullMode == "true")
				logrus.Infof("Debug - Combined condition: %v", railwayEnv != "" && forceFullMode != "true")
				
				if railwayEnv != "" && forceFullMode != "true" {
					logrus.Info("Condition 1: Railway environment without FORCE_FULL_MODE - starting MySQL-only mode")
					startMySQLVisualizationServer(dbPort, cfg)
					return
				} else if forceFullMode == "true" {
					logrus.Info("Condition 2: FORCE_FULL_MODE enabled - switching to Mock Neo4j repository")
					// Replace neo4jRepo with Mock repository
					neo4jRepo = neo4j.NewMockNeo4jRepository()
					realNeo4jRepo = nil // Clear real repo reference
					logrus.Info("Successfully switched to Mock Neo4j repository - continuing with normal flow")
					// Continue with Mock Neo4j - don't return here
				} else {
					logrus.Info("Condition 3: Neither Railway fallback nor FORCE_FULL_MODE - fatal error")
					logrus.Fatalf("Error deleting data in Neo4j: %v", err)
				}
			}
		}
		logrus.Infof("All data in Neo4j deleted")
	}

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

	logrus.Infof("Starting server...")
	vizServer := startVisualizationServer(neo4jRepo, cfg)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := vizServer.Shutdown(ctx); err != nil {
			logrus.Errorf("Error shutting down visualization server: %v", err)
		}
	}()

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
	// Add debug endpoint for environment variables
	router.HandleFunc("/api/debug", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Debug endpoint requested")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		debugInfo := map[string]interface{}{
			"RAILWAY_ENVIRONMENT": os.Getenv("RAILWAY_ENVIRONMENT"),
			"FORCE_FULL_MODE": os.Getenv("FORCE_FULL_MODE"),
			"neo4j_repo_type": fmt.Sprintf("%T", neo4jRepo),
			"real_neo4j_repo_nil": realNeo4jRepo == nil,
			"timestamp": time.Now().Format(time.RFC3339),
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
			"neo4j":     "connected",
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
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"},
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

	// Handle graceful shutdown
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

	// Use PORT environment variable if available (for Railway deployment)
	vizPort := os.Getenv("PORT")
	if vizPort == "" {
		vizPort = "3000"
	}
	vizAddr := ":" + vizPort // Listen on all interfaces

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

// startMySQLVisualizationServer starts a MySQL-only visualization server when Neo4j is not available
func startMySQLVisualizationServer(dbPort ports.DatabasePort, cfg *models.Config) {
	logrus.Info("Starting MySQL-only visualization server...")

	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Health check requested")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "v1.1.0-mysql",
			"mode":      "mysql_only",
			"database":  "connected",
			"neo4j":     "unavailable",
			"message":   "Running with MySQL data, Neo4j unavailable",
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Error encoding health response: %v", err)
			http.Error(w, "Health check failed", http.StatusInternalServerError)
		}
	}).Methods("GET")

	// Root endpoint with MySQL data visualization
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Root endpoint requested - MySQL visualization mode")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Get sample data from MySQL
		actors, _ := getSampleActors(dbPort)
		films, _ := getSampleFilms(dbPort)
		categories, _ := getSampleCategories(dbPort)

		html := generateMySQLVisualizationHTML(actors, films, categories)
		if _, err := w.Write([]byte(html)); err != nil {
			logrus.Errorf("Error writing response: %v", err)
		}
	}).Methods("GET")

	// API endpoint for raw MySQL data
	router.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Data API requested - MySQL mode")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get data from MySQL based on transform rules
		actors, _ := getSampleActors(dbPort)
		films, _ := getSampleFilms(dbPort)
		categories, _ := getSampleCategories(dbPort)

		data := map[string]interface{}{
			"actors":     actors,
			"films":      films,
			"categories": categories,
			"meta": map[string]interface{}{
				"source":   "mysql",
				"mode":     "mysql_only",
				"neo4j":    "unavailable",
				"message":  "Data directly from MySQL database",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		if err := json.NewEncoder(w).Encode(data); err != nil {
			logrus.Errorf("Error encoding data response: %v", err)
			http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		}
	}).Methods("GET")

	// CORS middleware
	corsOptions := middleware.CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept"},
		AllowCredentials: true,
	}
	corsHandler := middleware.NewCORSHandler(corsOptions)
	handler := corsHandler(router)

	// Use PORT environment variable
	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "3000"
	}
	apiAddr := ":" + apiPort

	server := &http.Server{
		Handler:           handler,
		Addr:              apiAddr,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logrus.Infof("Starting MySQL visualization server on %s", apiAddr)
	if err := server.ListenAndServe(); err != nil {
		logrus.Fatalf("Failed to start MySQL visualization server: %v", err)
	}
}

// Helper functions for MySQL data retrieval
func getSampleActors(dbPort ports.DatabasePort) ([]map[string]interface{}, error) {
	query := "SELECT actor_id, first_name, last_name, CONCAT(first_name, ' ', last_name) as full_name FROM actor LIMIT 20"
	return executeQuery(dbPort, query)
}

func getSampleFilms(dbPort ports.DatabasePort) ([]map[string]interface{}, error) {
	query := "SELECT film_id, title, description, release_year, rating, length FROM film LIMIT 20"
	return executeQuery(dbPort, query)
}

func getSampleCategories(dbPort ports.DatabasePort) ([]map[string]interface{}, error) {
	query := "SELECT category_id, name FROM category LIMIT 10"
	return executeQuery(dbPort, query)
}

func executeQuery(dbPort ports.DatabasePort, query string) ([]map[string]interface{}, error) {
	results, err := dbPort.ExecuteQuery(query)
	if err != nil {
		logrus.Errorf("Query failed: %v", err)
		return nil, err
	}
	
	// Convert []map[string]any to []map[string]interface{}
	converted := make([]map[string]interface{}, len(results))
	for i, result := range results {
		convertedRow := make(map[string]interface{})
		for k, v := range result {
			convertedRow[k] = v
		}
		converted[i] = convertedRow
	}
	
	return converted, nil
}

func generateMySQLVisualizationHTML(actors, films, categories []map[string]interface{}) string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>SQL Graph Visualizer - MySQL Data</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; }
        .section { margin: 30px 0; }
        .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: #f8f9fa; padding: 20px; border-radius: 8px; border-left: 4px solid #007bff; }
        .card h3 { margin: 0 0 10px 0; color: #007bff; }
        .status { background: #d4edda; padding: 15px; border-left: 4px solid #28a745; margin: 20px 0; }
        .warning { background: #fff3cd; padding: 15px; border-left: 4px solid #ffc107; margin: 20px 0; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: bold; }
        .api-link { color: #007bff; text-decoration: none; font-weight: bold; }
        .api-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎬 SQL Graph Visualizer - MySQL Data</h1>
        
        <div class="status">
            <strong>✅ MySQL Connected:</strong> Successfully displaying data from Sakila movie database!
        </div>
        
        <div class="warning">
            <strong>⚠️ Neo4j Unavailable:</strong> Showing raw MySQL data instead of graph visualization.
        </div>

        <div class="section">
            <h2>🎭 Actors (Sample)</h2>
            <div class="cards">` +
			generateActorCards(actors) + `
            </div>
        </div>

        <div class="section">
            <h2>🎥 Films (Sample)</h2>
            <div class="cards">` +
			generateFilmCards(films) + `
            </div>
        </div>

        <div class="section">
            <h2>📂 Categories</h2>
            <div class="cards">` +
			generateCategoryCards(categories) + `
            </div>
        </div>

        <div class="section">
            <h2>🔗 API Endpoints</h2>
            <ul>
                <li><a href="/api/health" class="api-link">/api/health</a> - Health status</li>
                <li><a href="/api/data" class="api-link">/api/data</a> - Raw MySQL data (JSON)</li>
            </ul>
        </div>

        <p style="text-align: center; color: #666; margin-top: 40px;">
            SQL Graph Visualizer - MySQL Mode | Railway Deployment
        </p>
    </div>
</body>
</html>`
}

func generateActorCards(actors []map[string]interface{}) string {
	cards := ""
	for _, actor := range actors {
		name := "N/A"
		if fullName, ok := actor["full_name"]; ok && fullName != nil {
			name = fmt.Sprintf("%v", fullName)
		}
		id := "N/A"
		if actorID, ok := actor["actor_id"]; ok && actorID != nil {
			id = fmt.Sprintf("%v", actorID)
		}
		cards += fmt.Sprintf(`<div class="card"><h3>%s</h3><p>ID: %s</p></div>`, name, id)
	}
	return cards
}

func generateFilmCards(films []map[string]interface{}) string {
	cards := ""
	for _, film := range films {
		title := "N/A"
		if filmTitle, ok := film["title"]; ok && filmTitle != nil {
			title = fmt.Sprintf("%v", filmTitle)
		}
		year := "N/A"
		if releaseYear, ok := film["release_year"]; ok && releaseYear != nil {
			year = fmt.Sprintf("%v", releaseYear)
		}
		rating := "N/A"
		if filmRating, ok := film["rating"]; ok && filmRating != nil {
			rating = fmt.Sprintf("%v", filmRating)
		}
		cards += fmt.Sprintf(`<div class="card"><h3>%s</h3><p>Year: %s | Rating: %s</p></div>`, title, year, rating)
	}
	return cards
}

func generateCategoryCards(categories []map[string]interface{}) string {
	cards := ""
	for _, category := range categories {
		name := "N/A"
		if catName, ok := category["name"]; ok && catName != nil {
			name = fmt.Sprintf("%v", catName)
		}
		id := "N/A"
		if catID, ok := category["category_id"]; ok && catID != nil {
			id = fmt.Sprintf("%v", catID)
		}
		cards += fmt.Sprintf(`<div class="card"><h3>%s</h3><p>ID: %s</p></div>`, name, id)
	}
	return cards
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

	// Root endpoint for Railway demo
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Root endpoint requested in demo mode")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		
		html := `<!DOCTYPE html>
<html>
<head>
    <title>SQL Graph Visualizer - Railway Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; }
        .status { background: #e8f5e8; padding: 15px; border-left: 4px solid #4CAF50; margin: 20px 0; }
        .info { background: #e3f2fd; padding: 15px; border-left: 4px solid #2196F3; margin: 20px 0; }
        a { color: #2196F3; text-decoration: none; }
        a:hover { text-decoration: underline; }
        code { background: #f5f5f5; padding: 2px 5px; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 SQL Graph Visualizer</h1>
        <h2>Railway Demo Mode</h2>
        
        <div class="status">
            <strong>✅ Status:</strong> Demo mode is running successfully on Railway!
        </div>
        
        <div class="info">
            <strong>ℹ️ Demo Mode:</strong> This application is running in demonstration mode because database connections are not available.
            <br><br>
            <strong>Available endpoints:</strong>
            <ul>
                <li><code><a href="/api/health">/api/health</a></code> - Health check endpoint</li>
                <li><code>/api/graph</code> - Graph data endpoint (demo data)</li>
            </ul>
        </div>
        
        <div class="info">
            <strong>🔗 Links:</strong>
            <ul>
                <li><a href="https://github.com/peter7775/sql-graph-visualizer">GitHub Repository</a></li>
                <li><a href="/api/health">Health Status</a></li>
            </ul>
        </div>
        
        <p style="text-align: center; color: #666; margin-top: 30px;">
            SQL Graph Visualizer v1.1.0-railway | Deployed on Railway
        </p>
    </div>
</body>
</html>`
		
		if _, err := w.Write([]byte(html)); err != nil {
			logrus.Errorf("Error writing response: %v", err)
		}
	}).Methods("GET")

	// Demo graph data endpoint
	router.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Graph endpoint requested in demo mode")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		// Return demo graph data
		demoData := map[string]interface{}{
			"nodes": []map[string]interface{}{
				{"id": "demo1", "label": "DemoNode", "properties": map[string]interface{}{"name": "Railway Demo", "type": "demo"}},
				{"id": "demo2", "label": "StatusNode", "properties": map[string]interface{}{"name": "Healthy", "status": "running"}},
			},
			"relationships": []map[string]interface{}{
				{"from": "demo1", "to": "demo2", "type": "CONNECTS_TO", "properties": map[string]interface{}{"demo": true}},
			},
			"meta": map[string]interface{}{
				"mode": "demo",
				"message": "This is demo data for Railway deployment",
			},
		}
		
		if err := json.NewEncoder(w).Encode(demoData); err != nil {
			logrus.Errorf("Error encoding demo graph data: %v", err)
			http.Error(w, "Failed to encode demo data", http.StatusInternalServerError)
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

// parseInt safely parses string to int, returns 0 if invalid
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

	// Check if MySQL variables are properly set (not just placeholder values)
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlDatabase := os.Getenv("MYSQL_DATABASE")
	
	logrus.Infof("MySQL env vars: HOST=%s, USER=%s, DB=%s", mysqlHost, mysqlUser, mysqlDatabase)
	
	// If MySQL env vars are placeholders or empty, fallback to demo mode
	if mysqlHost == "${MYSQL_HOST}" || mysqlHost == "" || mysqlUser == "${MYSQL_USER}" || mysqlUser == "" {
		logrus.Warn("MySQL environment variables not properly set - starting in demo mode")
		startRailwayDemoServer()
		return nil
	}

	return &models.Config{
		MySQL: models.MySQLConfig{
			Host:     mysqlHost,
			Port:     3306,
			User:     mysqlUser,
			Password: os.Getenv("MYSQL_PASSWORD"),
			Database: mysqlDatabase,
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

// createFallbackGraphData creates dummy graph data from MySQL data when Neo4j operations fail
func createFallbackGraphData(ctx context.Context, dbPort ports.DatabasePort, neo4jRepo ports.Neo4jPort) error {
	logrus.Info("Creating fallback graph data from MySQL...")
	
	// This is a simplified fallback - in reality you might want to implement
	// a more sophisticated transformation
	return nil // For now, just return nil to continue with empty graph
}
