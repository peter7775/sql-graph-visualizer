package deployment

import (
	"os"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

func NewDeploymentAdapter(logger *logrus.Logger) ports.DeploymentPort {

	if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
		logger.Info("Deployment: Detected Railway environment, using Railway adapter")
		return NewRailwayDeployment(logger)
	}

	logger.Info("Deployment: Using local development adapter")
	return NewLocalDeployment(logger)
}
