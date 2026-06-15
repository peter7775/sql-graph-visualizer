/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

// Package bootstrap provides application initialization and lifecycle management.
// It extracts the reusable startup logic from the monolithic cmd/main.go into
// composable functions used by CLI commands (transform, serve, etc.).
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	neo4jDriver "github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/sirupsen/logrus"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/application/services/transform"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repositories/config"
	"sql-graph-visualizer/internal/domain/repositories/configrule"
	"sql-graph-visualizer/internal/infrastructure/deployment"
	mysqlrepo "sql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"sql-graph-visualizer/internal/infrastructure/persistence/neo4j"
	postgresqlrepo "sql-graph-visualizer/internal/infrastructure/persistence/postgresql"

	_ "github.com/go-sql-driver/mysql"  // MySQL driver registration
	_ "github.com/lib/pq"               // PostgreSQL driver registration
	_ "github.com/microsoft/go-mssqldb" // SQL Server driver registration
)

// Resources holds all initialized application resources.
type Resources struct {
	Config              *models.Config
	DB                  *sql.DB
	DBPort              ports.DatabasePort
	Neo4jRepo           ports.Neo4jPort
	RealNeo4jRepo       *neo4j.Neo4jRepository
	TransformService    *transform.TransformService
	DeploymentAdapter   ports.DeploymentPort
	PerformanceServices *PerformanceServiceContainer
	MetricsInjector     *performance.SimpleMetricsInjector
	cleanupFuncs        []func()
}

// PerformanceServiceContainer holds all performance-related services.
type PerformanceServiceContainer struct {
	BenchmarkService    *performance.BenchmarkService
	PerformanceAnalyzer *performance.PerformanceAnalyzer
	PSAdapter           *performance.PerformanceSchemaAdapter
	GraphMapper         *performance.GraphPerformanceMapper
	RealtimeMonitor     *performance.RealtimePerformanceMonitor
}

// Cleanup releases all resources in reverse order.
func (r *Resources) Cleanup() {
	for i := len(r.cleanupFuncs) - 1; i >= 0; i-- {
		r.cleanupFuncs[i]()
	}
}

// LoadConfig loads the application configuration.
// If configPath is empty, it falls back to the default config.Load() behavior.
func LoadConfig(configPath string) (*models.Config, error) {
	if configPath != "" {
		if err := os.Setenv("CONFIG_PATH", configPath); err != nil {
			return nil, fmt.Errorf("failed to set CONFIG_PATH: %w", err)
		}
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

// InitAll initializes all application resources: config, databases, services.
func InitAll(ctx context.Context, configPath string) (*Resources, error) {
	res := &Resources{}

	// Deployment adapter
	logger := logrus.StandardLogger()
	res.DeploymentAdapter = deployment.NewDeploymentAdapter(logger)

	logrus.Infof("=== SQL Graph Visualizer Starting - Build timestamp: %s ===", time.Now().Format("2006-01-02 15:04:05"))
	logrus.Infof("Platform: %s", res.DeploymentAdapter.GetPlatformName())

	envInfo := res.DeploymentAdapter.GetEnvironmentInfo()
	for key, value := range envInfo {
		logrus.Infof("%s: %v", key, value)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	res.Config = cfg

	// Apply Railway overrides
	if res.DeploymentAdapter.GetPlatformName() == "Railway" {
		logrus.Info("Applying Railway-specific configuration overrides...")
		if railwayDeployment, ok := res.DeploymentAdapter.(*deployment.RailwayDeployment); ok {
			dbConfig := railwayDeployment.GetRailwayDatabaseConfig()
			if err := overrideConfigWithDeploymentSettings(cfg, dbConfig); err != nil {
				logrus.Warnf("Error applying deployment config: %v", err)
			}
		}
	}

	// Connect to source database
	if err := res.connectDatabase(ctx); err != nil {
		return nil, err
	}

	// Connect to Neo4j
	if err := res.connectNeo4j(); err != nil {
		res.Cleanup()
		return nil, err
	}

	// Clear Neo4j data
	if err := res.clearNeo4jData(); err != nil {
		res.Cleanup()
		return nil, err
	}

	// Init services
	res.TransformService = transform.NewTransformService(res.DBPort, res.Neo4jRepo, configrule.NewRuleRepository())

	// Performance services
	if cfg.Performance != nil && cfg.Performance.Monitoring != nil && cfg.Performance.Monitoring.Enabled {
		logrus.Info("Initializing performance monitoring services...")
		res.PerformanceServices = initPerformanceServices(cfg, res.DB)
		logrus.Info("Performance services initialized")
	} else {
		logrus.Info("Performance monitoring is disabled")
	}

	// Metrics injector
	logrus.Info("Initializing performance metrics visualization...")
	metricsConfig := &performance.SimpleMetricsConfig{
		UpdateInterval:   5 * time.Second,
		MetricsRetention: 1 * time.Hour,
		SimulationMode:   true,
	}
	res.MetricsInjector = performance.NewSimpleMetricsInjector(res.Neo4jRepo, logrus.StandardLogger(), metricsConfig)
	if err := res.MetricsInjector.Start(ctx); err != nil {
		logrus.Errorf("Failed to start metrics injector: %v", err)
	} else {
		logrus.Info("Performance metrics visualization started")
	}

	logrus.Info("Services initialized")
	return res, nil
}

// RunTransform executes the data transformation pipeline.
func (r *Resources) RunTransform(ctx context.Context) error {
	logrus.Info("Starting data transformation...")
	if err := r.TransformService.TransformAndStore(ctx); err != nil {
		if os.Getenv("FORCE_FULL_MODE") == "true" {
			logrus.Warnf("Transform service failed but FORCE_FULL_MODE enabled: %v", err)
			return nil
		}
		return fmt.Errorf("failed to transform and store data: %w", err)
	}
	logrus.Info("Data transformation successful")
	return nil
}

func (r *Resources) connectDatabase(_ context.Context) error {
	cfg := r.Config
	var err error

	if cfg.Database != nil && cfg.Database.Type != "" {
		logrus.Infof("Using multi-database configuration: %s", cfg.Database.Type)

		switch cfg.Database.Type {
		case models.DatabaseTypePostgreSQL:
			pgConfig := cfg.Database.PostgreSQL
			logrus.Infof("Connecting to PostgreSQL: %s@%s:%d/%s", pgConfig.GetUsername(), pgConfig.GetHost(), pgConfig.GetPort(), pgConfig.GetDatabase())
			postgresRepo := postgresqlrepo.NewPostgreSQLRepository(nil)
			r.DB, err = postgresRepo.ConnectToExisting(context.Background(), pgConfig)
			if err != nil {
				return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
			}
			r.DBPort = postgresqlrepo.NewPostgreSQLDatabasePort(r.DB)
			logrus.Info("Successfully connected to PostgreSQL database")

		case models.DatabaseTypeMySQL:
			mysqlConfig := cfg.Database.MySQL
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
				mysqlConfig.GetUsername(), mysqlConfig.GetPassword(),
				mysqlConfig.GetHost(), mysqlConfig.GetPort(), mysqlConfig.GetDatabase())
			r.DB, err = sql.Open("mysql", dsn)
			if err != nil {
				return fmt.Errorf("failed to connect to MySQL: %w", err)
			}
			r.DBPort = mysqlrepo.NewMySQLDatabasePort(r.DB)
			logrus.Info("Successfully connected to MySQL database")

		case models.DatabaseTypeMSSQL:
			mssqlConfig := cfg.Database.MSSQL
			if err := mssqlConfig.Validate(); err != nil {
				return fmt.Errorf("invalid MSSQL configuration: %w", err)
			}
			connString := mssqlConfig.BuildConnectionString()
			logrus.Infof("Connecting to SQL Server: %s@%s:%d/%s",
				mssqlConfig.Username, mssqlConfig.Host, mssqlConfig.Port, mssqlConfig.Database)
			r.DB, err = sql.Open("sqlserver", connString)
			if err != nil {
				return fmt.Errorf("failed to connect to SQL Server: %w", err)
			}
			// Use a generic DatabasePort — for now reuse MySQL port adapter
			// as both use database/sql underneath. Full MSSQL-specific port
			// can be added later if needed.
			r.DBPort = mysqlrepo.NewMySQLDatabasePort(r.DB)
			logrus.Info("Successfully connected to SQL Server database")

		default:
			return fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
		}
	} else {
		logrus.Info("Using legacy MySQL configuration")
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
		r.DB, err = sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %w", err)
		}
		r.DBPort = mysqlrepo.NewMySQLDatabasePort(r.DB)
		logrus.Info("MySQL connection successful")
	}

	r.cleanupFuncs = append(r.cleanupFuncs, func() {
		if err := r.DB.Close(); err != nil {
			logrus.Errorf("Error closing database connection: %v", err)
		}
	})

	return nil
}

func (r *Resources) connectNeo4j() error {
	cfg := r.Config
	logrus.Info("Initializing Neo4j connection...")

	realRepo, err := neo4j.NewNeo4jRepository(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		if os.Getenv("FORCE_FULL_MODE") == "true" {
			logrus.Warnf("Neo4j connection failed but FORCE_FULL_MODE is enabled, using Mock repository: %v", err)
			r.Neo4jRepo = neo4j.NewMockNeo4jRepository()
			logrus.Info("Using Mock Neo4j repository for visualization")
		} else {
			return fmt.Errorf("failed to create Neo4j repository: %w", err)
		}
	} else {
		r.RealNeo4jRepo = realRepo
		r.Neo4jRepo = realRepo
		logrus.Info("Neo4j connection successful")
	}

	r.cleanupFuncs = append(r.cleanupFuncs, func() {
		if err := r.Neo4jRepo.Close(); err != nil {
			logrus.Errorf("Error closing Neo4j repository: %v", err)
		}
	})

	return nil
}

func (r *Resources) clearNeo4jData() error {
	if r.RealNeo4jRepo == nil {
		logrus.Info("Using Mock Neo4j repository, skipping data deletion")
		return nil
	}

	logrus.Info("Deleting all data in Neo4j...")
	session := r.RealNeo4jRepo.NewSession(neo4jDriver.SessionConfig{})
	defer func() {
		if sessionCloser, ok := session.(interface{ Close() error }); ok {
			if err := sessionCloser.Close(); err != nil {
				logrus.Errorf("Error closing session: %v", err)
			}
		}
	}()

	result, err := session.Run("MATCH (n) DETACH DELETE n", nil)
	if err != nil {
		if os.Getenv("FORCE_FULL_MODE") == "true" {
			logrus.Info("FORCE_FULL_MODE enabled - switching to Mock Neo4j repository")
			r.Neo4jRepo = neo4j.NewMockNeo4jRepository()
			r.RealNeo4jRepo = nil
			logrus.Info("Successfully switched to Mock Neo4j repository")
		} else {
			return fmt.Errorf("error deleting data in Neo4j: %w", err)
		}
	} else {
		if _, err := result.Consume(); err != nil {
			logrus.Warnf("Error consuming result after deleting Neo4j data: %v", err)
		}
	}
	logrus.Info("All data in Neo4j deleted")
	return nil
}

func overrideConfigWithDeploymentSettings(cfg *models.Config, dbConfig map[string]string) error {
	if cfg.Database != nil && cfg.Database.MySQL != nil {
		if host := dbConfig["mysql_host"]; host != "" {
			cfg.Database.MySQL.Host = host
		}
		if user := dbConfig["mysql_user"]; user != "" {
			cfg.Database.MySQL.User = user
		}
		if password := dbConfig["mysql_password"]; password != "" {
			cfg.Database.MySQL.Password = password
		}
		if database := dbConfig["mysql_database"]; database != "" {
			cfg.Database.MySQL.Database = database
		}
		if port := dbConfig["mysql_port"]; port != "" {
			if portNum := parseInt(port); portNum > 0 {
				cfg.Database.MySQL.Port = portNum
			}
		}
	}

	if uri := dbConfig["neo4j_uri"]; uri != "" && uri != "${NEO4J_URI}" {
		cfg.Neo4j.URI = uri
	}
	if user := dbConfig["neo4j_user"]; user != "" && user != "${NEO4J_USER}" {
		cfg.Neo4j.User = user
	}
	if password := dbConfig["neo4j_password"]; password != "" && password != "${NEO4J_PASSWORD}" {
		cfg.Neo4j.Password = password
	}

	return nil
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}
