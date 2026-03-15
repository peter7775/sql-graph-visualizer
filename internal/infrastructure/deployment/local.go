package deployment

import (
	"net/http"
	"os"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// LocalDeployment provides local deployment capabilities.
type LocalDeployment struct {
	logger *logrus.Logger
}

// NewLocalDeployment creates a new local deployment instance.
func NewLocalDeployment(logger *logrus.Logger) ports.DeploymentPort {
	return &LocalDeployment{
		logger: logger,
	}
}

// GetAPIPort returns the API server port for local deployment.
func (l *LocalDeployment) GetAPIPort() string {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	l.logger.Infof("Local: Using API port %s", port)
	return port
}

// GetVisualizationPort returns the visualization server port.
func (l *LocalDeployment) GetVisualizationPort() string {
	port := os.Getenv("VIZ_PORT")
	if port == "" {
		port = "3000"
	}
	l.logger.Infof("Local: Using visualization port %s", port)
	return port
}

// ShouldStartVisualizationServer indicates if a separate visualization server should start.
func (l *LocalDeployment) ShouldStartVisualizationServer() bool {
	l.logger.Info("Local: Starting separate visualization server for local development")
	return true
}

// GetEnvironmentInfo returns local environment configuration information.
func (l *LocalDeployment) GetEnvironmentInfo() map[string]interface{} {
	return map[string]interface{}{
		"platform":    l.GetPlatformName(),
		"api_port":    l.GetAPIPort(),
		"viz_port":    l.GetVisualizationPort(),
		"go_env":      l.getEnvOrDefault("GO_ENV", "development"),
		"config_path": l.getEnvOrDefault("CONFIG_PATH", "config/config.yml"),
		"log_level":   l.getEnvOrDefault("LOG_LEVEL", "info"),
	}
}

// ConfigureServer applies local development server configuration.
func (l *LocalDeployment) ConfigureServer(server *http.Server) *http.Server {
	l.logger.Info("Local: Applying local development server configuration")

	server.ReadTimeout = 15 * time.Second
	server.ReadHeaderTimeout = 5 * time.Second
	server.WriteTimeout = 15 * time.Second
	server.IdleTimeout = 60 * time.Second

	l.logger.Infof("Local: Server configured for local development on %s", server.Addr)

	return server
}

// GetPlatformName returns the platform name for local deployment.
func (l *LocalDeployment) GetPlatformName() string {
	return "Local Development"
}

func (l *LocalDeployment) getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (l *LocalDeployment) RegisterVisualizationRoutes(router *mux.Router, neo4jRepo ports.Neo4jPort, cfg *models.Config) error {
	l.logger.Info("Local: Visualization routes handled by separate server - no registration needed")
	return nil
}

func (l *LocalDeployment) GetHomepageHandler() http.HandlerFunc {
	return nil
}
