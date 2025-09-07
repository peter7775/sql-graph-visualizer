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

package migration

import (
	"context"
	"fmt"
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/transform"
	"sql-graph-visualizer/internal/domain/valueobjects"
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
	if mysqlPort == nil || neo4jPort == nil || transform == nil {
		panic("ports and transform service must not be nil")
	}
	return &MigrationService{
		mysqlPort: mysqlPort,
		neo4jPort: neo4jPort,
		transform: transform,
	}
}

func (s *MigrationService) MigrateData(ctx context.Context, config valueobjects.TransformConfig) error {
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}
	return s.transform.TransformAndStore(ctx)
}
