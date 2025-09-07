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

package ports

import (
	"context"
	"database/sql"
	"sql-graph-visualizer/internal/domain/models"
)

type PostgreSQLPort interface {
	FetchData() ([]map[string]any, error)
	Close() error
	ExecuteQuery(query string) ([]map[string]any, error)

	ConnectToExisting(ctx context.Context, config *models.PostgreSQLConfig) (*sql.DB, error)
	ValidateConnection(ctx context.Context, db *sql.DB) (*models.ConnectionValidationResult, error)

	DiscoverSchema(ctx context.Context, db *sql.DB, config *models.DataFilteringConfig) (*models.SchemaAnalysisResult, error)
	GetTables(ctx context.Context, db *sql.DB, filters *models.DataFilteringConfig) ([]string, error)
	GetTableInfo(ctx context.Context, db *sql.DB, tableName string) (*models.TableInfo, error)

	ExtractTableData(ctx context.Context, db *sql.DB, tableName string, config *models.DataFilteringConfig) ([]map[string]any, error)
	EstimateDataSize(ctx context.Context, db *sql.DB, config *models.DataFilteringConfig) (*models.DatasetInfo, error)
}
