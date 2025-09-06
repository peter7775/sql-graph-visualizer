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

package repository

import (
	"context"
	"database/sql"
	"sql-graph-visualizer/internal/domain/models"
)

// DatabaseRepository defines the interface for database-specific operations
type DatabaseRepository interface {
	// Connection management
	Connect(ctx context.Context, config models.DatabaseConfig) (*sql.DB, error)
	Close() error
	TestConnection(ctx context.Context) error
	
	// Schema introspection
	GetTables(ctx context.Context, filters models.DataFilteringConfig) ([]string, error)
	GetColumns(ctx context.Context, tableName string) ([]*models.ColumnInfo, error)
	GetForeignKeys(ctx context.Context, tableName string) ([]models.ForeignKeyInfo, error)
	GetIndexes(ctx context.Context, tableName string) ([]models.IndexInfo, error)
	GetConstraints(ctx context.Context, tableName string) ([]models.Constraint, error)
	
	// Database metadata
	GetDatabaseName(ctx context.Context) (string, error)
	GetDatabaseVersion(ctx context.Context) (string, error)
	GetSchemaNames(ctx context.Context) ([]string, error)
	
	// Data sampling and analysis
	GetTableRowCount(ctx context.Context, tableName string) (int64, error)
	SampleTableData(ctx context.Context, tableName string, limit int) ([]map[string]interface{}, error)
	AnalyzeColumnStatistics(ctx context.Context, tableName, columnName string) (*models.ColumnStatistics, error)
	
	// Performance and optimization
	GetTableSize(ctx context.Context, tableName string) (*models.TableSize, error)
	GetQueryExecutionPlan(ctx context.Context, query string) (string, error)
	
	// Security validation
	ValidatePermissions(ctx context.Context, requiredPerms []string) error
	CheckUserPrivileges(ctx context.Context) (*models.UserPrivileges, error)
	
	// Database-specific utility methods
	EscapeIdentifier(identifier string) string
	GetQuoteChar() string
	GetDatabaseType() models.DatabaseType
	GetConnectionString(config models.DatabaseConfig) string
}

// DatabaseRepositoryFactory creates database-specific repository implementations
type DatabaseRepositoryFactory interface {
	CreateRepository(dbType models.DatabaseType) (DatabaseRepository, error)
	GetSupportedDatabaseTypes() []models.DatabaseType
}

// QueryBuilder defines interface for building database-specific queries
type QueryBuilder interface {
	// Schema queries
	BuildTablesQuery(filters models.DataFilteringConfig) string
	BuildColumnsQuery(tableName string) string
	BuildForeignKeysQuery(tableName string) string
	BuildIndexesQuery(tableName string) string
	BuildConstraintsQuery(tableName string) string
	
	// Metadata queries
	BuildDatabaseVersionQuery() string
	BuildSchemaNamesQuery() string
	BuildTableRowCountQuery(tableName string) string
	BuildTableSizeQuery(tableName string) string
	
	// Data analysis queries
	BuildSampleDataQuery(tableName string, limit int) string
	BuildColumnStatisticsQuery(tableName, columnName string) string
	
	// Security queries
	BuildUserPrivilegesQuery() string
	BuildPermissionCheckQuery(permission string) string
	
	// Utility methods
	FormatTableName(tableName string, schema ...string) string
	FormatColumnName(columnName string) string
	BuildLimitClause(limit int) string
	BuildOffsetClause(offset int) string
}

// ConnectionManager handles database connections with pooling and retry logic
type ConnectionManager interface {
	GetConnection(ctx context.Context) (*sql.DB, error)
	ReturnConnection(db *sql.DB)
	Health() error
	Stats() ConnectionStats
	Close() error
}

// ConnectionStats provides connection pool statistics
type ConnectionStats struct {
	ActiveConnections int
	IdleConnections   int
	TotalConnections  int
	MaxConnections    int
	ConnectErrors     int64
	QueryErrors       int64
}

// RepositoryMetrics tracks repository performance and usage
type RepositoryMetrics interface {
	RecordQuery(queryType string, duration int64, success bool)
	RecordConnection(success bool)
	GetQueryStats() map[string]QueryStats
	GetConnectionStats() ConnectionStats
	Reset()
}

// QueryStats provides statistics for specific query types
type QueryStats struct {
	Count           int64
	TotalDuration   int64
	AverageDuration float64
	SuccessRate     float64
	LastExecuted    int64
}

// TransactionManager handles database transactions
type TransactionManager interface {
	BeginTransaction(ctx context.Context) (*sql.Tx, error)
	CommitTransaction(tx *sql.Tx) error
	RollbackTransaction(tx *sql.Tx) error
	ExecuteInTransaction(ctx context.Context, fn func(*sql.Tx) error) error
}

// SchemaCache provides caching for schema metadata
type SchemaCache interface {
	GetTables(key string) ([]string, bool)
	SetTables(key string, tables []string)
	GetColumns(tableName string) ([]*models.ColumnInfo, bool)
	SetColumns(tableName string, columns []*models.ColumnInfo)
	InvalidateTable(tableName string)
	InvalidateAll()
	Stats() CacheStats
}

// CacheStats provides cache performance statistics
type CacheStats struct {
	HitCount    int64
	MissCount   int64
	HitRate     float64
	Size        int
	MaxSize     int
	Evictions   int64
}

// DatabaseSpecificConfig holds database-specific configuration and behaviors
type DatabaseSpecificConfig struct {
	QuoteChar          string
	IdentifierMaxLen   int
	CaseSensitive      bool
	DefaultPort        int
	DefaultSchema      string
	SupportsSchemas    bool
	SupportsCatalogs   bool
	SupportsTransactions bool
	
	// Query limits and timeouts
	MaxQueryTimeout    int // seconds
	MaxResultSetSize   int64
	DefaultBatchSize   int
	
	// Feature support
	SupportsWindowFunctions bool
	SupportsCTE            bool
	SupportsRecursiveCTE   bool
	SupportsArrayTypes     bool
	SupportsJSONType       bool
}

// ErrorHandler provides database-specific error handling
type ErrorHandler interface {
	IsConnectionError(err error) bool
	IsTimeoutError(err error) bool
	IsPermissionError(err error) bool
	IsConstraintViolation(err error) bool
	MapError(err error) models.DatabaseError
	ShouldRetry(err error) bool
}
