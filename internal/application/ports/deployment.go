package ports

import (
	"net/http"
	"github.com/gorilla/mux"
	"sql-graph-visualizer/internal/domain/models"
)

// DeploymentPort defines the interface for deployment platform-specific logic
type DeploymentPort interface {
	// GetAPIPort returns the port for API server
	GetAPIPort() string
	
	// GetVisualizationPort returns the port for visualization server
	GetVisualizationPort() string
	
	// ShouldStartVisualizationServer determines if visualization server should be started
	ShouldStartVisualizationServer() bool
	
	GetEnvironmentInfo() map[string]interface{}
	
	ConfigureServer(server *http.Server) *http.Server
	
	GetPlatformName() string
	
	// RegisterVisualizationRoutes allows deployment to add visualization endpoints to API server
	RegisterVisualizationRoutes(router *mux.Router, neo4jRepo Neo4jPort, cfg *models.Config) error
	
	GetHomepageHandler() http.HandlerFunc
}
