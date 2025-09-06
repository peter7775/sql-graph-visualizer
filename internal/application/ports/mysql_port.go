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
	"mysql-graph-visualizer/internal/domain/models"
)

type MySQLPort interface {
	// Existing methods
	FetchData() ([]map[string]any, error)
	Close() error
	ExecuteQuery(query string) ([]map[string]any, error)
	
	// New methods for direct database connection (Issue #10)
	// Connection management
	ConnectToExisting(ctx context.Context, config *models.MySQLConfig) (*sql.DB, error)
	ValidateConnection(ctx context.Context, db *sql.DB) (*models.ConnectionValidationResult, error)
	
	// Schema discovery
	DiscoverSchema(ctx context.Context, db *sql.DB, config *models.DataFilteringConfig) (*models.SchemaAnalysisResult, error)
	GetTables(ctx context.Context, db *sql.DB, filters *models.DataFilteringConfig) ([]string, error)
	GetTableInfo(ctx context.Context, db *sql.DB, tableName string) (*models.TableInfo, error)
	
	// Data extraction with filtering
	ExtractTableData(ctx context.Context, db *sql.DB, tableName string, config *models.DataFilteringConfig) ([]map[string]any, error)
	EstimateDataSize(ctx context.Context, db *sql.DB, config *models.DataFilteringConfig) (*models.DatasetInfo, error)
}
