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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repository"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

// MySQLDatabaseRepository implements DatabaseRepository for MySQL
type MySQLDatabaseRepository struct {
	db *sql.DB
}

// NewMySQLDatabaseRepository creates a new MySQL database repository
func NewMySQLDatabaseRepository() repository.DatabaseRepository {
	return &MySQLDatabaseRepository{}
}

// Connect establishes connection to MySQL database
func (r *MySQLDatabaseRepository) Connect(ctx context.Context, config models.DatabaseConfig) (*sql.DB, error) {
	mysqlConfig, ok := config.(*models.MySQLConfig)
	if !ok {
		return nil, fmt.Errorf("expected MySQLConfig, got %T", config)
	}

	// Use Username if set, otherwise fallback to User
	username := mysqlConfig.GetUsername()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%ds&readTimeout=%ds&writeTimeout=%ds",
		username,
		mysqlConfig.GetPassword(),
		mysqlConfig.GetHost(),
		mysqlConfig.GetPort(),
		mysqlConfig.GetDatabase(),
		mysqlConfig.GetSecurity().ConnectionTimeout,
		mysqlConfig.GetSecurity().QueryTimeout,
		mysqlConfig.GetSecurity().QueryTimeout,
	)

	// Add SSL configuration if enabled
	sslConfig := mysqlConfig.GetSSLConfig()
	if sslConfig.Enabled {
		dsn += "&tls=true"
		if sslConfig.InsecureSkipVerify {
			dsn += "&tls=skip-verify"
		}
	}

	logrus.Infof("Connecting to MySQL database: %s@%s:%d/%s", username, mysqlConfig.GetHost(), mysqlConfig.GetPort(), mysqlConfig.GetDatabase())

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL database connection: %w", err)
	}

	// Set connection pool limits
	security := mysqlConfig.GetSecurity()
	db.SetMaxOpenConns(security.MaxConnections)
	db.SetMaxIdleConns(security.MaxConnections / 2)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Test connection
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(security.ConnectionTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctxTimeout); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	r.db = db
	logrus.Infof("Successfully connected to MySQL database")
	return db, nil
}

// Close closes the database connection
func (r *MySQLDatabaseRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// TestConnection tests the database connection
func (r *MySQLDatabaseRepository) TestConnection(ctx context.Context) error {
	if r.db == nil {
		return fmt.Errorf("no active database connection")
	}
	return r.db.PingContext(ctx)
}

// GetTables returns list of tables based on filtering configuration
func (r *MySQLDatabaseRepository) GetTables(ctx context.Context, filters models.DataFilteringConfig) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allTables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		allTables = append(allTables, tableName)
	}

	// Apply filtering
	return r.applyTableFiltering(allTables, filters), nil
}

// GetColumns retrieves column information for a table
func (r *MySQLDatabaseRepository) GetColumns(ctx context.Context, tableName string) ([]*models.ColumnInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE,
			COLUMN_KEY,
			COLUMN_DEFAULT,
			EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var defaultVal sql.NullString

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&col.KeyType,
			&defaultVal,
			&col.Extra,
		)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.DefaultValue = defaultVal.String
		}

		columns = append(columns, &col)
	}

	return columns, nil
}

// GetForeignKeys retrieves foreign key information for a table
func (r *MySQLDatabaseRepository) GetForeignKeys(ctx context.Context, tableName string) ([]models.ForeignKeyInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY ORDINAL_POSITION
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var foreignKeys []models.ForeignKeyInfo
	for rows.Next() {
		var fk models.ForeignKeyInfo
		err := rows.Scan(
			&fk.Name,
			&fk.Column,
			&fk.ReferencedTable,
			&fk.ReferencedColumn,
		)
		if err != nil {
			return nil, err
		}
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}

// GetIndexes retrieves index information for a table
func (r *MySQLDatabaseRepository) GetIndexes(ctx context.Context, tableName string) ([]models.IndexInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*models.IndexInfo)
	for rows.Next() {
		var indexName, columnName, indexType string
		var nonUnique int

		err := rows.Scan(&indexName, &columnName, &nonUnique, &indexType)
		if err != nil {
			return nil, err
		}

		if idx, exists := indexMap[indexName]; exists {
			idx.Columns = append(idx.Columns, columnName)
		} else {
			indexMap[indexName] = &models.IndexInfo{
				Name:     indexName,
				Columns:  []string{columnName},
				IsUnique: nonUnique == 0,
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

// GetConstraints retrieves constraint information for a table
func (r *MySQLDatabaseRepository) GetConstraints(ctx context.Context, tableName string) ([]models.Constraint, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	// MySQL doesn't have a comprehensive INFORMATION_SCHEMA for all constraints
	// We'll combine multiple queries to get different types of constraints

	var constraints []models.Constraint

	// Get CHECK constraints (MySQL 8.0+)
	checkQuery := `
		SELECT 
			CONSTRAINT_NAME,
			CHECK_CLAUSE
		FROM INFORMATION_SCHEMA.CHECK_CONSTRAINTS
		WHERE CONSTRAINT_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
	`

	rows, err := r.db.QueryContext(ctx, checkQuery, tableName)
	if err == nil { // Ignore error for older MySQL versions
		defer rows.Close()
		for rows.Next() {
			var name, condition string
			if err := rows.Scan(&name, &condition); err == nil {
				constraints = append(constraints, models.Constraint{
					Name:      name,
					Type:      "CHECK",
					TableName: tableName,
					Condition: condition,
				})
			}
		}
	}

	return constraints, nil
}

// GetDatabaseName returns the current database name
func (r *MySQLDatabaseRepository) GetDatabaseName(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var dbName string
	err := r.db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName)
	return dbName, err
}

// GetDatabaseVersion returns the database version
func (r *MySQLDatabaseRepository) GetDatabaseVersion(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var version string
	err := r.db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	return version, err
}

// GetSchemaNames returns list of schema names (databases in MySQL)
func (r *MySQLDatabaseRepository) GetSchemaNames(ctx context.Context) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := "SHOW DATABASES"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		// Skip system databases
		if schema != "information_schema" && schema != "mysql" && schema != "performance_schema" && schema != "sys" {
			schemas = append(schemas, schema)
		}
	}

	return schemas, nil
}

// GetTableRowCount returns the row count for a table
func (r *MySQLDatabaseRepository) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("no active database connection")
	}

	// Use INFORMATION_SCHEMA for quick estimate
	query := `
		SELECT TABLE_ROWS 
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ?
	`

	var rowCount sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, tableName).Scan(&rowCount)
	if err != nil {
		return 0, err
	}

	if rowCount.Valid {
		return rowCount.Int64, nil
	}

	return 0, nil
}

// Implement remaining methods...
func (r *MySQLDatabaseRepository) SampleTableData(ctx context.Context, tableName string, limit int) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *MySQLDatabaseRepository) AnalyzeColumnStatistics(ctx context.Context, tableName, columnName string) (*models.ColumnStatistics, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *MySQLDatabaseRepository) GetTableSize(ctx context.Context, tableName string) (*models.TableSize, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *MySQLDatabaseRepository) GetQueryExecutionPlan(ctx context.Context, query string) (string, error) {
	return "", fmt.Errorf("not implemented yet")
}

func (r *MySQLDatabaseRepository) ValidatePermissions(ctx context.Context, requiredPerms []string) error {
	return fmt.Errorf("not implemented yet")
}

func (r *MySQLDatabaseRepository) CheckUserPrivileges(ctx context.Context) (*models.UserPrivileges, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *MySQLDatabaseRepository) EscapeIdentifier(identifier string) string {
	return fmt.Sprintf("`%s`", strings.Replace(identifier, "`", "``", -1))
}

func (r *MySQLDatabaseRepository) GetQuoteChar() string {
	return "`"
}

func (r *MySQLDatabaseRepository) GetDatabaseType() models.DatabaseType {
	return models.DatabaseTypeMySQL
}

func (r *MySQLDatabaseRepository) GetConnectionString(config models.DatabaseConfig) string {
	mysqlConfig, ok := config.(*models.MySQLConfig)
	if !ok {
		return ""
	}

	username := mysqlConfig.GetUsername()
	return fmt.Sprintf("%s@%s:%d/%s", username, mysqlConfig.GetHost(), mysqlConfig.GetPort(), mysqlConfig.GetDatabase())
}

// Helper methods
func (r *MySQLDatabaseRepository) applyTableFiltering(tables []string, filters models.DataFilteringConfig) []string {
	if len(filters.TableBlacklist) == 0 && len(filters.TableWhitelist) == 0 {
		return tables
	}

	var filtered []string
	for _, table := range tables {
		// Check blacklist first
		if r.isInList(table, filters.TableBlacklist) {
			continue
		}

		// If whitelist exists, check if table is included
		if len(filters.TableWhitelist) > 0 {
			if !r.isInList(table, filters.TableWhitelist) {
				continue
			}
		}

		filtered = append(filtered, table)
	}

	return filtered
}

func (r *MySQLDatabaseRepository) isInList(item string, list []string) bool {
	for _, listItem := range list {
		if strings.EqualFold(item, listItem) {
			return true
		}
	}
	return false
}
