package deployment

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"sql-graph-visualizer/internal/application/ports"
)


type RailwayDeployment struct {
	logger *logrus.Logger
}


func NewRailwayDeployment(logger *logrus.Logger) ports.DeploymentPort {
	return &RailwayDeployment{
		logger: logger,
	}
}


func (r *RailwayDeployment) GetAPIPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local development
	}
	r.logger.Infof("Railway: Using API port %s", port)
	return port
}


func (r *RailwayDeployment) GetVisualizationPort() string {

	return r.GetAPIPort()
}


func (r *RailwayDeployment) ShouldStartVisualizationServer() bool {
	isRailway := r.isRailwayEnvironment()
	r.logger.Infof("Railway: Should start visualization server: %t", !isRailway)
	return !isRailway 
}


func (r *RailwayDeployment) GetEnvironmentInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":            r.GetPlatformName(),
		"railway_environment": r.getEnvOrDefault("RAILWAY_ENVIRONMENT", "not_set"),
		"railway_service_id":  r.getEnvOrDefault("RAILWAY_SERVICE_ID", "not_set"),
		"railway_project_id":  r.getEnvOrDefault("RAILWAY_PROJECT_ID", "not_set"),
		"port":                r.getEnvOrDefault("PORT", "not_set"),
		"force_full_mode":     r.getEnvOrDefault("FORCE_FULL_MODE", "not_set"),
	}
}


func (r *RailwayDeployment) ConfigureServer(server *http.Server) *http.Server {
	if r.isRailwayEnvironment() {
		r.logger.Info("Railway: Applying Railway-specific server configuration")
		
	
		server.ReadTimeout = 30 * time.Second
		server.ReadHeaderTimeout = 10 * time.Second
		server.WriteTimeout = 30 * time.Second
		server.IdleTimeout = 120 * time.Second
		
	
		if server.Addr == "" || server.Addr[0] != ':' {
			server.Addr = ":" + r.GetAPIPort()
		}
		
		r.logger.Infof("Railway: Server configured for Railway deployment on %s", server.Addr)
	} else {
		r.logger.Info("Railway: Local development mode - using default configuration")
	}
	
	return server
}


func (r *RailwayDeployment) GetPlatformName() string {
	if r.isRailwayEnvironment() {
		return "Railway"
	}
	return "Local"
}


func (r *RailwayDeployment) isRailwayEnvironment() bool {
	return os.Getenv("RAILWAY_ENVIRONMENT") != ""
}


func (r *RailwayDeployment) getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (r *RailwayDeployment) GetRailwayDatabaseConfig() map[string]string {
	config := make(map[string]string)
	
	if r.isRailwayEnvironment() {
		r.logger.Info("Railway: Loading Railway database configuration")
		
	
		config["mysql_host"] = r.getEnvOrDefault("MYSQL_HOST", "")
		config["mysql_port"] = r.getEnvOrDefault("MYSQL_PORT", "3306")
		config["mysql_user"] = r.getEnvOrDefault("MYSQL_USER", "")
		config["mysql_password"] = r.getEnvOrDefault("MYSQL_PASSWORD", "")
		config["mysql_database"] = r.getEnvOrDefault("MYSQL_DATABASE", "")
		
	
		config["neo4j_uri"] = r.getEnvOrDefault("NEO4J_URI", "")
		config["neo4j_user"] = r.getEnvOrDefault("NEO4J_USER", "neo4j")
		config["neo4j_password"] = r.getEnvOrDefault("NEO4J_PASSWORD", "")
		
		r.logger.Infof("Railway: Database config loaded - MySQL host: %s, Neo4j URI set: %t", 
			config["mysql_host"], config["neo4j_uri"] != "")
	}
	
	return config
}


func (r *RailwayDeployment) IsProductionMode() bool {
	railwayEnv := os.Getenv("RAILWAY_ENVIRONMENT")
	return railwayEnv == "production"
}


func (r *RailwayDeployment) GetMemoryLimit() int64 {
	if !r.isRailwayEnvironment() {
		return 0
	}
	

	if memStr := os.Getenv("RAILWAY_MEMORY_LIMIT"); memStr != "" {
		if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			return mem
		}
	}
	

	return 512
}

func (r *RailwayDeployment) GetHealthCheckConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":     true,
		"path":        "/api/health",
		"port":        r.GetAPIPort(),
		"timeout":     30,
		"interval":    10,
		"retries":     3,
		"start_period": 60,
	}
}