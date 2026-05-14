/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 */

// Package mssql provides Microsoft SQL Server database persistence.
package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repositories"
	"strings"
	"time"

	_ "github.com/microsoft/go-mssqldb" // SQL Server driver registration
	"github.com/sirupsen/logrus"
)

// MSSQLDatabaseRepository implements DatabaseRepository for Microsoft SQL Server.
//
//nolint:revive // MSSQLDatabaseRepository is descriptive and follows project conventions
type MSSQLDatabaseRepository struct {
	db *sql.DB
}

// NewMSSQLDatabaseRepository creates a new MSSQL database repository.
func NewMSSQLDatabaseRepository() repositories.DatabaseRepository {
	return &MSSQLDatabaseRepository{}
}

// Connect establishes connection to SQL Server database.
func (r *MSSQLDatabaseRepository) Connect(ctx context.Context, config models.DatabaseConfig) (*sql.DB, error) {
	mssqlConfig, ok := config.GetEffectiveConfig().(*models.MSSQLConfig)
	if !ok {
		return nil, fmt.Errorf("expected MSSQLConfig, got %T", config.GetEffectiveConfig())
	}

	if err := mssqlConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MSSQL configuration: %w", err)
	}

	connString := mssqlConfig.BuildConnectionString()

	logrus.Infof("Connecting to SQL Server: %s@%s:%d/%s",
		mssqlConfig.Username, mssqlConfig.Host, mssqlConfig.Port, mssqlConfig.Database)

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL Server database connection: %w", err)
	}

	db.SetMaxOpenConns(mssqlConfig.MaxConnections)
	db.SetMaxIdleConns(mssqlConfig.MaxConnections / 2)
	db.SetConnMaxLifetime(10 * time.Minute)

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(mssqlConfig.ConnectionTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctxTimeout); err != nil {
		return nil, fmt.Errorf("failed to ping SQL Server database: %w", err)
	}

	r.db = db
	logrus.Info("Successfully connected to SQL Server database")
	return db, nil
}

// Close closes the database connection.
func (r *MSSQLDatabaseRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// TestConnection tests the database connection.
func (r *MSSQLDatabaseRepository) TestConnection(ctx context.Context) error {
	if r.db == nil {
		return fmt.Errorf("no active database connection")
	}
	return r.db.PingContext(ctx)
}

// GetTables returns list of tables based on filtering configuration.
func (r *MSSQLDatabaseRepository) GetTables(ctx context.Context, filters models.DataFilteringConfig) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE'
			AND TABLE_SCHEMA = SCHEMA_NAME()
		ORDER BY TABLE_NAME
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var allTables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		allTables = append(allTables, tableName)
	}

	return applyTableFiltering(allTables, filters), nil
}

// GetColumns retrieves column information for a SQL Server table.
func (r *MSSQLDatabaseRepository) GetColumns(ctx context.Context, tableName string) ([]*models.ColumnInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			c.COLUMN_NAME,
			c.DATA_TYPE,
			c.IS_NULLABLE,
			ISNULL(c.COLUMN_DEFAULT, '') AS column_default,
			c.CHARACTER_MAXIMUM_LENGTH,
			CASE
				WHEN pk.COLUMN_NAME IS NOT NULL THEN 'PRIMARY'
				WHEN uq.COLUMN_NAME IS NOT NULL THEN 'UNIQUE'
				ELSE ''
			END AS key_type,
			CASE
				WHEN COLUMNPROPERTY(OBJECT_ID(c.TABLE_SCHEMA + '.' + c.TABLE_NAME), c.COLUMN_NAME, 'IsIdentity') = 1
				THEN 'auto_increment'
				ELSE ''
			END AS extra
		FROM INFORMATION_SCHEMA.COLUMNS c
		LEFT JOIN (
			SELECT ku.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
				ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
				AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
				AND tc.TABLE_NAME = @p1
				AND tc.TABLE_SCHEMA = SCHEMA_NAME()
		) pk ON c.COLUMN_NAME = pk.COLUMN_NAME
		LEFT JOIN (
			SELECT ku.COLUMN_NAME
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
				ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
				AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
			WHERE tc.CONSTRAINT_TYPE = 'UNIQUE'
				AND tc.TABLE_NAME = @p2
				AND tc.TABLE_SCHEMA = SCHEMA_NAME()
		) uq ON c.COLUMN_NAME = uq.COLUMN_NAME
		WHERE c.TABLE_NAME = @p3
			AND c.TABLE_SCHEMA = SCHEMA_NAME()
		ORDER BY c.ORDINAL_POSITION
	`

	rows, err := r.db.QueryContext(ctx, query, tableName, tableName, tableName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var columns []*models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var defaultVal, keyType, maxLength sql.NullString

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&defaultVal,
			&maxLength,
			&keyType,
			&col.Extra,
		)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.DefaultValue = strings.TrimSpace(defaultVal.String)
		}
		if keyType.Valid && keyType.String != "" {
			col.KeyType = keyType.String
			col.IsKey = true
		}

		columns = append(columns, &col)
	}

	return columns, nil
}

// GetForeignKeys retrieves foreign key information for a SQL Server table.
func (r *MSSQLDatabaseRepository) GetForeignKeys(ctx context.Context, tableName string) ([]models.ForeignKeyInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			fk.name AS constraint_name,
			COL_NAME(fkc.parent_object_id, fkc.parent_column_id) AS column_name,
			OBJECT_NAME(fkc.referenced_object_id) AS referenced_table,
			COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) AS referenced_column
		FROM sys.foreign_keys fk
		JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		WHERE OBJECT_NAME(fk.parent_object_id) = @p1
			AND SCHEMA_NAME(fk.schema_id) = SCHEMA_NAME()
		ORDER BY fk.name, fkc.constraint_column_id
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var foreignKeys []models.ForeignKeyInfo
	for rows.Next() {
		var fk models.ForeignKeyInfo
		var constraintName string
		err := rows.Scan(
			&constraintName,
			&fk.Column,
			&fk.ReferencedTable,
			&fk.ReferencedColumn,
		)
		if err != nil {
			return nil, err
		}
		fk.Name = constraintName
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// GetIndexes retrieves index information for a SQL Server table.
func (r *MSSQLDatabaseRepository) GetIndexes(ctx context.Context, tableName string) ([]models.IndexInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			i.name AS index_name,
			COL_NAME(ic.object_id, ic.column_id) AS column_name,
			i.is_unique,
			i.type_desc AS index_type
		FROM sys.indexes i
		JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		WHERE OBJECT_NAME(i.object_id) = @p1
			AND SCHEMA_NAME(OBJECTPROPERTY(i.object_id, 'SchemaId')) = SCHEMA_NAME()
			AND i.name IS NOT NULL
		ORDER BY i.name, ic.key_ordinal
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	indexMap := make(map[string]*models.IndexInfo)
	for rows.Next() {
		var indexName, columnName, indexType string
		var isUnique bool

		if err := rows.Scan(&indexName, &columnName, &isUnique, &indexType); err != nil {
			return nil, err
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &models.IndexInfo{
				Name:     indexName,
				Columns:  []string{columnName},
				IsUnique: isUnique,
				Type:     indexType,
			}
		}
	}

	var indexes []models.IndexInfo
	for _, idx := range indexMap {
		indexes = append(indexes, *idx)
	}

	return indexes, nil
}

// GetConstraints retrieves constraint information for a SQL Server table.
func (r *MSSQLDatabaseRepository) GetConstraints(ctx context.Context, tableName string) ([]models.Constraint, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			tc.CONSTRAINT_NAME,
			tc.CONSTRAINT_TYPE,
			ISNULL(cc.CHECK_CLAUSE, '') AS condition
		FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
		LEFT JOIN INFORMATION_SCHEMA.CHECK_CONSTRAINTS cc
			ON tc.CONSTRAINT_NAME = cc.CONSTRAINT_NAME
			AND tc.CONSTRAINT_SCHEMA = cc.CONSTRAINT_SCHEMA
		WHERE tc.TABLE_NAME = @p1
			AND tc.TABLE_SCHEMA = SCHEMA_NAME()
		ORDER BY tc.CONSTRAINT_NAME
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var constraints []models.Constraint
	for rows.Next() {
		var constraint models.Constraint
		var condition sql.NullString

		if err := rows.Scan(&constraint.Name, &constraint.Type, &condition); err != nil {
			return nil, err
		}

		constraint.TableName = tableName
		if condition.Valid {
			constraint.Condition = condition.String
		}

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

// GetDatabaseName returns the current SQL Server database name.
func (r *MSSQLDatabaseRepository) GetDatabaseName(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var dbName string
	err := r.db.QueryRowContext(ctx, "SELECT DB_NAME()").Scan(&dbName)
	return dbName, err
}

// GetDatabaseVersion returns the SQL Server version.
func (r *MSSQLDatabaseRepository) GetDatabaseVersion(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var version string
	err := r.db.QueryRowContext(ctx, "SELECT @@VERSION").Scan(&version)
	return version, err
}

// GetSchemaNames returns list of schemas in SQL Server.
func (r *MSSQLDatabaseRepository) GetSchemaNames(ctx context.Context) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT SCHEMA_NAME
		FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('guest', 'INFORMATION_SCHEMA', 'sys',
			'db_owner', 'db_accessadmin', 'db_securityadmin', 'db_ddladmin',
			'db_backupoperator', 'db_datareader', 'db_datawriter', 'db_denydatareader',
			'db_denydatawriter')
		ORDER BY SCHEMA_NAME
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// GetTableRowCount returns the estimated row count for a SQL Server table.
func (r *MSSQLDatabaseRepository) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("no active database connection")
	}

	// Use sys.partitions for fast estimate
	query := `
		SELECT SUM(p.rows)
		FROM sys.partitions p
		JOIN sys.tables t ON p.object_id = t.object_id
		WHERE t.name = @p1
			AND SCHEMA_NAME(t.schema_id) = SCHEMA_NAME()
			AND p.index_id IN (0, 1)
	`

	var rowCount sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, tableName).Scan(&rowCount)
	if err != nil || !rowCount.Valid {
		// Fall back to COUNT(*)
		// #nosec G201 - tableName is escaped using EscapeIdentifier
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.EscapeIdentifier(tableName))
		err = r.db.QueryRowContext(ctx, countQuery).Scan(&rowCount)
		if err != nil {
			return 0, err
		}
	}

	if rowCount.Valid {
		return rowCount.Int64, nil
	}

	return 0, nil
}

// SampleTableData retrieves a sample of data from the specified table.
func (r *MSSQLDatabaseRepository) SampleTableData(_ context.Context, _ string, _ int) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// AnalyzeColumnStatistics analyzes statistical information for table columns.
func (r *MSSQLDatabaseRepository) AnalyzeColumnStatistics(_ context.Context, _, _ string) (*models.ColumnStatistics, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// GetTableSize returns the size of the specified table.
func (r *MSSQLDatabaseRepository) GetTableSize(_ context.Context, _ string) (*models.TableSize, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// GetQueryExecutionPlan returns the execution plan for the given query.
func (r *MSSQLDatabaseRepository) GetQueryExecutionPlan(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not implemented yet")
}

// ValidatePermissions validates database connection permissions.
func (r *MSSQLDatabaseRepository) ValidatePermissions(_ context.Context, _ []string) error {
	return fmt.Errorf("not implemented yet")
}

// CheckUserPrivileges checks user privileges for database operations.
func (r *MSSQLDatabaseRepository) CheckUserPrivileges(_ context.Context) (*models.UserPrivileges, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// EscapeIdentifier escapes SQL Server identifiers using bracket notation.
func (r *MSSQLDatabaseRepository) EscapeIdentifier(identifier string) string {
	return fmt.Sprintf("[%s]", strings.ReplaceAll(identifier, "]", "]]"))
}

// GetQuoteChar returns the character used for quoting identifiers in SQL Server.
func (r *MSSQLDatabaseRepository) GetQuoteChar() string {
	return "["
}

// GetDatabaseType returns the MSSQL database type.
func (r *MSSQLDatabaseRepository) GetDatabaseType() models.DatabaseType {
	return models.DatabaseTypeMSSQL
}

// GetConnectionString builds SQL Server connection string from config.
func (r *MSSQLDatabaseRepository) GetConnectionString(config models.DatabaseConfig) string {
	mssqlConfig, ok := config.GetEffectiveConfig().(*models.MSSQLConfig)
	if !ok || mssqlConfig == nil {
		return ""
	}
	return mssqlConfig.BuildConnectionString()
}

// Helper functions

func closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		logrus.WithError(err).Error("Failed to close rows")
	}
}

func applyTableFiltering(tables []string, filters models.DataFilteringConfig) []string {
	if len(filters.TableBlacklist) == 0 && len(filters.TableWhitelist) == 0 {
		return tables
	}

	var filtered []string
	for _, table := range tables {
		if isInList(table, filters.TableBlacklist) {
			continue
		}
		if len(filters.TableWhitelist) > 0 {
			if !isInList(table, filters.TableWhitelist) {
				continue
			}
		}
		filtered = append(filtered, table)
	}

	return filtered
}

func isInList(item string, list []string) bool {
	for _, listItem := range list {
		if strings.EqualFold(item, listItem) {
			return true
		}
	}
	return false
}
