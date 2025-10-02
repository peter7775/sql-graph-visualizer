package ports

import "net/http"

// DeploymentPort defines the interface for deployment platform-specific logic
type DeploymentPort interface {
	// GetAPIPort returns the port for API server
	GetAPIPort() string
	
	// GetVisualizationPort returns the port for visualization server
	GetVisualizationPort() string
	
	// ShouldStartVisualizationServer determines if visualization server should be started
	ShouldStartVisualizationServer() bool
	
	// GetEnvironmentInfo returns environment-specific information for debugging
	GetEnvironmentInfo() map[string]interface{}
	
	// ConfigureServer applies deployment-specific server configuration
	ConfigureServer(server *http.Server) *http.Server
	
	// GetPlatformName returns the name of the deployment platform
	GetPlatformName() string
}