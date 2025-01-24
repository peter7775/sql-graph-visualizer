package migration

import (
	"github.com/peter7775/alevisualizer/internal/application/ports"
	"github.com/peter7775/alevisualizer/internal/application/services/transform"
	"github.com/peter7775/alevisualizer/internal/domain/valueobjects"
	"context"
)

type MigrationService struct {
	mysqlPort ports.MySQLPort
	neo4jPort ports.Neo4jPort
	transform *transform.TransformService
}

func NewMigrationService(
	mysqlPort ports.MySQLPort,
	neo4jPort ports.Neo4jPort,
	transform *transform.TransformService,
) *MigrationService {
	return &MigrationService{
		mysqlPort: mysqlPort,
		neo4jPort: neo4jPort,
		transform: transform,
	}
}

func (s *MigrationService) MigrateData(ctx context.Context, config valueobjects.TransformConfig) error {
	return s.transform.TransformAndStore(ctx, config)
}
