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

// Package oracle provides Oracle database persistence using the go-ora pure Go driver.
package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repositories"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2" // Oracle driver registration
	"github.com/sirupsen/logrus"
)

// OracleDatabaseRepository implements DatabaseRepository for Oracle Database.
//
//nolint:revive // OracleDatabaseRepository is descriptive and follows project conventions
type OracleDatabaseRepository struct {
	db *sql.DB
}

// NewOracleDatabaseRepository creates a new Oracle database repository.
func NewOracleDatabaseRepository() repositories.DatabaseRepository {
	return &OracleDatabaseRepository{}
}

// Connect establishes connection to Oracle database.
func (r *OracleDatabaseRepository) Connect(ctx context.Context, config models.DatabaseConfig) (*sql.DB, error) {
	oracleConfig, ok := config.GetEffectiveConfig().(*models.OracleConfig)
	if !ok {
		return nil, fmt.Errorf("expected OracleConfig, got %T", config.GetEffectiveConfig())
	}

	if err := oracleConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Oracle configuration: %w", err)
	}

	connString := oracleConfig.BuildConnectionString()

	logrus.Infof("Connecting to Oracle database: %s@%s:%d/%s",
		oracleConfig.Username, oracleConfig.Host, oracleConfig.Port, oracleConfig.ServiceName)

	db, err := sql.Open("oracle", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open Oracle database connection: %w", err)
	}

	db.SetMaxOpenConns(oracleConfig.MaxOpenConns)
	db.SetMaxIdleConns(oracleConfig.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(oracleConfig.ConnMaxLifetime) * time.Minute)

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(oracleConfig.ConnectionTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctxTimeout); err != nil {
		return nil, fmt.Errorf("failed to ping Oracle database: %w", err)
	}

	r.db = db
	logrus.Info("Successfully connected to Oracle database")
	return db, nil
}

// Close closes the database connection.
func (r *OracleDatabaseRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// TestConnection tests the database connection.
func (r *OracleDatabaseRepository) TestConnection(ctx context.Context) error {
	if r.db == nil {
		return fmt.Errorf("no active database connection")
	}
	return r.db.PingContext(ctx)
}

// GetTables returns list of tables based on filtering configuration.
func (r *OracleDatabaseRepository) GetTables(ctx context.Context, filters models.DataFilteringConfig) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT table_name
		FROM all_tables
		WHERE owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
		ORDER BY table_name
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

// GetColumns retrieves column information for an Oracle table.
func (r *OracleDatabaseRepository) GetColumns(ctx context.Context, tableName string) ([]*models.ColumnInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			c.column_name,
			c.data_type,
			CASE WHEN c.nullable = 'Y' THEN 'YES' ELSE 'NO' END AS is_nullable,
			NVL(c.data_default, '') AS column_default,
			CASE
				WHEN cc.constraint_type = 'P' THEN 'PRIMARY'
				WHEN cc.constraint_type = 'U' THEN 'UNIQUE'
				ELSE ''
			END AS key_type,
			CASE WHEN c.identity_column = 'YES' THEN 'auto_increment' ELSE '' END AS extra
		FROM all_tab_columns c
		LEFT JOIN (
			SELECT col.column_name, con.constraint_type
			FROM all_cons_columns col
			JOIN all_constraints con ON col.constraint_name = con.constraint_name
				AND col.owner = con.owner
			WHERE con.table_name = :1
				AND con.owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
				AND con.constraint_type IN ('P', 'U')
		) cc ON c.column_name = cc.column_name
		WHERE c.table_name = :2
			AND c.owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
		ORDER BY c.column_id
	`

	rows, err := r.db.QueryContext(ctx, query, tableName, tableName)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var columns []*models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var defaultVal, keyType sql.NullString

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.IsNullable,
			&defaultVal,
			&keyType,
			&col.Extra,
		)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.DefaultValue = strings.TrimSpace(defaultVal.String)
		}
		if keyType.Valid {
			col.KeyType = keyType.String
			col.IsKey = keyType.String != ""
		}

		columns = append(columns, &col)
	}

	return columns, nil
}

// GetForeignKeys retrieves foreign key information for an Oracle table.
func (r *OracleDatabaseRepository) GetForeignKeys(ctx context.Context, tableName string) ([]models.ForeignKeyInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			a.constraint_name,
			a.column_name,
			c_pk.table_name AS referenced_table,
			b.column_name AS referenced_column
		FROM all_cons_columns a
		JOIN all_constraints c ON a.constraint_name = c.constraint_name AND a.owner = c.owner
		JOIN all_constraints c_pk ON c.r_constraint_name = c_pk.constraint_name AND c.r_owner = c_pk.owner
		JOIN all_cons_columns b ON c_pk.constraint_name = b.constraint_name AND c_pk.owner = b.owner AND a.position = b.position
		WHERE c.constraint_type = 'R'
			AND a.table_name = :1
			AND a.owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
		ORDER BY a.constraint_name, a.position
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

// GetIndexes retrieves index information for an Oracle table.
func (r *OracleDatabaseRepository) GetIndexes(ctx context.Context, tableName string) ([]models.IndexInfo, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			i.index_name,
			ic.column_name,
			CASE WHEN i.uniqueness = 'UNIQUE' THEN 1 ELSE 0 END AS is_unique,
			i.index_type
		FROM all_indexes i
		JOIN all_ind_columns ic ON i.index_name = ic.index_name AND i.owner = ic.index_owner
		WHERE i.table_name = :1
			AND i.owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
		ORDER BY i.index_name, ic.column_position
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

// GetConstraints retrieves constraint information for an Oracle table.
func (r *OracleDatabaseRepository) GetConstraints(ctx context.Context, tableName string) ([]models.Constraint, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT
			constraint_name,
			constraint_type,
			NVL(search_condition, '') AS condition
		FROM all_constraints
		WHERE table_name = :1
			AND owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
		ORDER BY constraint_name
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

// GetDatabaseName returns the current Oracle database name.
func (r *OracleDatabaseRepository) GetDatabaseName(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var dbName string
	err := r.db.QueryRowContext(ctx, "SELECT ORA_DATABASE_NAME FROM DUAL").Scan(&dbName)
	return dbName, err
}

// GetDatabaseVersion returns the Oracle database version.
func (r *OracleDatabaseRepository) GetDatabaseVersion(ctx context.Context) (string, error) {
	if r.db == nil {
		return "", fmt.Errorf("no active database connection")
	}

	var version string
	err := r.db.QueryRowContext(ctx, "SELECT banner FROM v$version WHERE ROWNUM = 1").Scan(&version)
	return version, err
}

// GetSchemaNames returns list of accessible schemas in Oracle.
func (r *OracleDatabaseRepository) GetSchemaNames(ctx context.Context) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("no active database connection")
	}

	query := `
		SELECT DISTINCT owner
		FROM all_tables
		WHERE owner NOT IN ('SYS', 'SYSTEM', 'DBSNMP', 'OUTLN', 'XDB',
			'CTXSYS', 'MDSYS', 'OLAPSYS', 'WMSYS', 'EXFSYS', 'ORDSYS',
			'ORDDATA', 'APPQOSSYS', 'ANONYMOUS', 'GSMADMIN_INTERNAL')
		ORDER BY owner
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

// GetTableRowCount returns the estimated row count for an Oracle table.
func (r *OracleDatabaseRepository) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("no active database connection")
	}

	// Use statistics for fast estimate
	query := `
		SELECT NVL(num_rows, 0)
		FROM all_tables
		WHERE table_name = :1
			AND owner = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
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
func (r *OracleDatabaseRepository) SampleTableData(_ context.Context, _ string, _ int) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// AnalyzeColumnStatistics analyzes statistical information for table columns.
func (r *OracleDatabaseRepository) AnalyzeColumnStatistics(_ context.Context, _, _ string) (*models.ColumnStatistics, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// GetTableSize returns the size of the specified table.
func (r *OracleDatabaseRepository) GetTableSize(_ context.Context, _ string) (*models.TableSize, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// GetQueryExecutionPlan returns the execution plan for the given query.
func (r *OracleDatabaseRepository) GetQueryExecutionPlan(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not implemented yet")
}

// ValidatePermissions validates database connection permissions.
func (r *OracleDatabaseRepository) ValidatePermissions(_ context.Context, _ []string) error {
	return fmt.Errorf("not implemented yet")
}

// CheckUserPrivileges checks user privileges for database operations.
func (r *OracleDatabaseRepository) CheckUserPrivileges(_ context.Context) (*models.UserPrivileges, error) {
	return nil, fmt.Errorf("not implemented yet")
}

// EscapeIdentifier escapes Oracle database identifiers.
func (r *OracleDatabaseRepository) EscapeIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
}

// GetQuoteChar returns the character used for quoting identifiers in Oracle.
func (r *OracleDatabaseRepository) GetQuoteChar() string {
	return `"`
}

// GetDatabaseType returns the Oracle database type.
func (r *OracleDatabaseRepository) GetDatabaseType() models.DatabaseType {
	return models.DatabaseTypeOracle
}

// GetConnectionString builds Oracle connection string from config.
func (r *OracleDatabaseRepository) GetConnectionString(config models.DatabaseConfig) string {
	oracleConfig, ok := config.GetEffectiveConfig().(*models.OracleConfig)
	if !ok || oracleConfig == nil {
		return ""
	}
	return oracleConfig.BuildConnectionString()
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
