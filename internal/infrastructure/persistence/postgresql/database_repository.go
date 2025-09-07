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

package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repository"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// PostgreSQLDatabaseRepository implements DatabaseRepository for PostgreSQL
type PostgreSQLDatabaseRepository struct {
	db *sql.DB
}

// NewPostgreSQLDatabaseRepository creates a new PostgreSQL database repository
func NewPostgreSQLDatabaseRepository() repository.DatabaseRepository {
	return &PostgreSQLDatabaseRepository{}
}

// Connect establishes connection to PostgreSQL database
func (r *PostgreSQLDatabaseRepository) Connect(ctx context.Context, config models.DatabaseConfig) (*sql.DB, error) {
	pgConfig, ok := config.(*models.PostgreSQLConfig)
	if !ok {
		return nil, fmt.Errorf("expected PostgreSQLConfig, got %T", config)
	}

	// Use Username if set, otherwise fallback to User
	username := pgConfig.GetUsername()

	// Build PostgreSQL connection string
	var connString strings.Builder
	connString.WriteString(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		pgConfig.GetHost(), pgConfig.GetPort(), username, pgConfig.GetPassword(), pgConfig.GetDatabase()))

	// Add SSL configuration
	if pgConfig.SSLConfig.Mode != "" {
		connString.WriteString(fmt.Sprintf(" sslmode=%s", pgConfig.SSLConfig.Mode))
	} else {
		connString.WriteString(" sslmode=prefer") // Default to prefer
	}

	if pgConfig.SSLConfig.CertFile != "" {
		connString.WriteString(fmt.Sprintf(" sslcert=%s", pgConfig.SSLConfig.CertFile))
	}
	if pgConfig.SSLConfig.KeyFile != "" {
		connString.WriteString(fmt.Sprintf(" sslkey=%s", pgConfig.SSLConfig.KeyFile))
	}
	if pgConfig.SSLConfig.CAFile != "" {
		connString.WriteString(fmt.Sprintf(" sslrootcert=%s", pgConfig.SSLConfig.CAFile))
	}

	// Add timeout configurations
	security := pgConfig.GetSecurity()
	if security.ConnectionTimeout > 0 {
		connString.WriteString(fmt.Sprintf(" connect_timeout=%d", security.ConnectionTimeout))
	}
	if pgConfig.StatementTimeout > 0 {
		connString.WriteString(fmt.Sprintf(" statement_timeout=%dms", pgConfig.StatementTimeout*1000))
	}

	// Set application name for .monitoring
	appName := pgConfig.ApplicationName
	if appName == "" {
		appName = "sql-graph-visualizer"
	}
	connString.WriteString(fmt.Sprintf(" application_name=%s", appName))

	logrus.Infof("Connecting to PostgreSQL database: %s@%s:%d/%s", username, pgConfig.GetHost(), pgConfig.GetPort(), pgConfig.GetDatabase())

	db, err := sql.Open("postgres", connString.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database connection: %w", err)
	}

	// Set connection pool limits
	db.SetMaxOpenConns(security.MaxConnections)
	db.SetMaxIdleConns(security.MaxConnections / 2)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Test connection
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(security.ConnectionTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctxTimeout); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	r.db = db
	logrus.Infof("Successfully connected to PostgreSQL database")
	return db, nil
}

// Close closes the database connection
func (r *PostgreSQLDatabaseRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// TestConnection tests the database connection
func (r *PostgreSQLDatabaseRepository) TestConnection(ctx context.Context) error {
	if r.db == nil {
		return fmt.Errorf("no active database connection")
	}
	return r.db.PingContext(ctx)
}

// GetTables returns list of tables based on filtering configuration
func (r *PostgreSQLDatabaseRepository) GetTables(ctx context.Context, filters models.DataFilteringConfig) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	// Get tables from current schema (typically 'public' if not specified)
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = current_schema() 
			AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

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

// GetColumns retrieves column information for a PostgreSQL table
func (r *PostgreSQLDatabaseRepository) GetColumns(ctx context.Context, tableName string) ([]*models.ColumnInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			CASE 
				WHEN column_default LIKE 'nextval%' THEN 'auto_increment'
				ELSE ''
			END as extra
		FROM information_schema.columns 
		WHERE table_schema = current_schema() 
			AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var defaultVal, maxLength sql.NullString

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&defaultVal,
			&maxLength,
			&col.Extra,
		)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.DefaultValue = defaultVal.String
		}

		if maxLength.Valid {
			// Store max length info in Comment for now
			col.Comment = fmt.Sprintf("max_length: %s", maxLength.String)
		}

		// Check if this column is part of primary key
		r.enrichColumnWithConstraintInfo(ctx, tableName, &col)

		columns = append(columns, &col)
	}

	return columns, nil
}

// enrichColumnWithConstraintInfo adds primary key and other constraint information
func (r *PostgreSQLDatabaseRepository) enrichColumnWithConstraintInfo(ctx context.Context, tableName string, col *models.ColumnInfo) {
	// Check if column is primary key
	pkQuery := `
		SELECT COUNT(*) 
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = current_schema()
			AND tc.table_name = $1
			AND kcu.column_name = $2
	`

	var pkCount int
	if err := r.db.QueryRowContext(ctx, pkQuery, tableName, col.Name).Scan(&pkCount); err == nil && pkCount > 0 {
		col.KeyType = "PRI"
		col.IsKey = true
	}
}

// GetForeignKeys retrieves foreign key information for a PostgreSQL table
func (r *PostgreSQLDatabaseRepository) GetForeignKeys(ctx context.Context, tableName string) ([]models.ForeignKeyInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS referenced_table_name,
			ccu.column_name AS referenced_column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu 
			ON tc.constraint_name = kcu.constraint_name 
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu 
			ON ccu.constraint_name = tc.constraint_name 
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = current_schema()
			AND tc.table_name = $1
		ORDER BY tc.constraint_name
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

// GetIndexes retrieves index information for a PostgreSQL table
func (r *PostgreSQLDatabaseRepository) GetIndexes(ctx context.Context, tableName string) ([]models.IndexInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			i.relname as index_name,
			a.attname as column_name,
			ix.indisunique as is_unique,
			am.amname as index_type
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_am am ON i.relam = am.oid
		WHERE t.relname = $1
			AND t.relkind = 'r'
		ORDER BY i.relname, a.attnum
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*models.IndexInfo)
	for rows.Next() {
		var indexName, columnName, indexType string
		var isUnique bool

		err := rows.Scan(&indexName, &columnName, &isUnique, &indexType)
		if err != nil {
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

// GetConstraints retrieves constraint information for a PostgreSQL table
func (r *PostgreSQLDatabaseRepository) GetConstraints(ctx context.Context, tableName string) ([]models.Constraint, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT 
			tc.constraint_name,
			tc.constraint_type,
			COALESCE(cc.check_clause, '') as condition
		FROM information_schema.table_constraints tc
		LEFT JOIN information_schema.check_constraints cc 
			ON tc.constraint_name = cc.constraint_name 
			AND tc.constraint_schema = cc.constraint_schema
		WHERE tc.table_schema = current_schema()
			AND tc.table_name = $1
		ORDER BY tc.constraint_name
	`

	rows, err := r.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []models.Constraint
	for rows.Next() {
		var constraint models.Constraint
		var condition sql.NullString

		err := rows.Scan(
			&constraint.Name,
			&constraint.Type,
			&condition,
		)
		if err != nil {
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

// GetDatabaseName returns the current database name
func (r *PostgreSQLDatabaseRepository) GetDatabaseName(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var dbName string
	err := r.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName)
	return dbName, err
}

// GetDatabaseVersion returns the database version
func (r *PostgreSQLDatabaseRepository) GetDatabaseVersion(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var version string
	err := r.db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
	return version, err
}

// GetSchemaNames returns list of schema names in PostgreSQL
func (r *PostgreSQLDatabaseRepository) GetSchemaNames(ctx context.Context) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT schema_name 
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
			AND schema_name NOT LIKE 'pg_temp_%'
			AND schema_name NOT LIKE 'pg_toast_temp_%'
		ORDER BY schema_name
	`

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
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// GetTableRowCount returns the row count for a PostgreSQL table
func (r *PostgreSQLDatabaseRepository) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("no active database connection")
	}

	// Use pg_stat_user_tables for quick estimate
	query := `
		SELECT COALESCE(n_tup_ins - n_tup_del, 0) as estimated_rows
		FROM pg_stat_user_tables 
		WHERE relname = $1
	`

	var rowCount sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, tableName).Scan(&rowCount)
	if err != nil {
		// If pg_stat_user_tables doesn't have the data, fall back to COUNT(*)
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

// Implement remaining methods...
func (r *PostgreSQLDatabaseRepository) SampleTableData(ctx context.Context, tableName string, limit int) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *PostgreSQLDatabaseRepository) AnalyzeColumnStatistics(ctx context.Context, tableName, columnName string) (*models.ColumnStatistics, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *PostgreSQLDatabaseRepository) GetTableSize(ctx context.Context, tableName string) (*models.TableSize, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *PostgreSQLDatabaseRepository) GetQueryExecutionPlan(ctx context.Context, query string) (string, error) {
	return "", fmt.Errorf("not implemented yet")
}

func (r *PostgreSQLDatabaseRepository) ValidatePermissions(ctx context.Context, requiredPerms []string) error {
	return fmt.Errorf("not implemented yet")
}

func (r *PostgreSQLDatabaseRepository) CheckUserPrivileges(ctx context.Context) (*models.UserPrivileges, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (r *PostgreSQLDatabaseRepository) EscapeIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.Replace(identifier, `"`, `""`, -1))
}

func (r *PostgreSQLDatabaseRepository) GetQuoteChar() string {
	return `"`
}

func (r *PostgreSQLDatabaseRepository) GetDatabaseType() models.DatabaseType {
	return models.DatabaseTypePostgreSQL
}

func (r *PostgreSQLDatabaseRepository) GetConnectionString(config models.DatabaseConfig) string {
	pgConfig, ok := config.(*models.PostgreSQLConfig)
	if !ok {
		return ""
	}

	username := pgConfig.GetUsername()
	return fmt.Sprintf("%s@%s:%d/%s", username, pgConfig.GetHost(), pgConfig.GetPort(), pgConfig.GetDatabase())
}

// Helper methods
func (r *PostgreSQLDatabaseRepository) applyTableFiltering(tables []string, filters models.DataFilteringConfig) []string {
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

func (r *PostgreSQLDatabaseRepository) isInList(item string, list []string) bool {
	for _, listItem := range list {
		if strings.EqualFold(item, listItem) {
			return true
		}
	}
	return false
}
