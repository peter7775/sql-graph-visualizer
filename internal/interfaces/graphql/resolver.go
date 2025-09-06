package graphql

import (
	"mysql-graph-visualizer/internal/application/ports"
	"mysql-graph-visualizer/internal/domain/models"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	Neo4jRepo ports.Neo4jPort
	Config    *models.Config
}
