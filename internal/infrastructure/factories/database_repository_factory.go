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

package factories

import (
	"fmt"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repository"
	"sql-graph-visualizer/internal/infrastructure/persistence/mysql"
	"sql-graph-visualizer/internal/infrastructure/persistence/postgresql"
)

// DatabaseRepositoryFactory creates database-specific repository implementations
type DatabaseRepositoryFactory struct{}

// NewDatabaseRepositoryFactory creates a new repository factory
func NewDatabaseRepositoryFactory() repository.DatabaseRepositoryFactory {
	return &DatabaseRepositoryFactory{}
}

// CreateRepository creates a database-specific repository based on database type
func (f *DatabaseRepositoryFactory) CreateRepository(dbType models.DatabaseType) (repository.DatabaseRepository, error) {
	switch dbType {
	case models.DatabaseTypeMySQL:
		return mysql.NewMySQLDatabaseRepository(), nil
	case models.DatabaseTypePostgreSQL:
		return postgresql.NewPostgreSQLDatabaseRepository(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// GetSupportedDatabaseTypes returns list of supported database types
func (f *DatabaseRepositoryFactory) GetSupportedDatabaseTypes() []models.DatabaseType {
	return []models.DatabaseType{
		models.DatabaseTypeMySQL,
		models.DatabaseTypePostgreSQL,
	}
}
