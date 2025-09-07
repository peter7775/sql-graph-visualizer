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
	"log"
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgreSQLRepository struct {
	db *sql.DB
}

func NewPostgreSQLRepository(db *sql.DB) ports.PostgreSQLPort {
	return &PostgreSQLRepository{db: db}
}

// NewPostgreSQLDatabasePort creates a PostgreSQL repository that implements the generic DatabasePort interface
func NewPostgreSQLDatabasePort(db *sql.DB) ports.DatabasePort {
	return &PostgreSQLRepository{db: db}
}

func (r *PostgreSQLRepository) FetchData() ([]map[string]any, error) {
	logrus.Infof("💾 FetchData called - returning empty slice (data loading moved to transform service)")
	return []map[string]any{}, nil
}

func (r *PostgreSQLRepository) Close() error {
	return r.db.Close()
}

func (r *PostgreSQLRepository) ExecuteQuery(query string) ([]map[string]any, error) {
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		columnPointers := make([]any, len(columns))
		for i := range columns {
			columnPointers[i] = new(any)
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colName := range columns {
			row[colName] = *(columnPointers[i].(*any))
		}

		results = append(results, row)
	}

	return results, nil
}

// EscapeIdentifier escapes PostgreSQL identifiers (table names, column names)
func (r *PostgreSQLRepository) EscapeIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, strings.Replace(identifier, `"`, `""`, -1))
}

// ConnectToExisting creates a new connection to existing PostgreSQL database
func (r *PostgreSQLRepository) ConnectToExisting(ctx context.Context, config *models.PostgreSQLConfig) (*sql.DB, error) {
	// Use Username if set, otherwise fallback to User
	username := config.Username
	if username == "" {
		username = config.User
	}

	// Build PostgreSQL connection string
	var connString strings.Builder
	connString.WriteString(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		config.Host, config.Port, username, config.Password, config.Database))

	// Add SSL configuration
	if config.SSLConfig.Mode != "" {
		connString.WriteString(fmt.Sprintf(" sslmode=%s", config.SSLConfig.Mode))
	} else {
		connString.WriteString(" sslmode=prefer") // Default to prefer
	}

	if config.SSLConfig.CertFile != "" {
		connString.WriteString(fmt.Sprintf(" sslcert=%s", config.SSLConfig.CertFile))
	}
	if config.SSLConfig.KeyFile != "" {
		connString.WriteString(fmt.Sprintf(" sslkey=%s", config.SSLConfig.KeyFile))
	}
	if config.SSLConfig.CAFile != "" {
		connString.WriteString(fmt.Sprintf(" sslrootcert=%s", config.SSLConfig.CAFile))
	}

	// Add timeout configurations
	if config.Security.ConnectionTimeout > 0 {
		connString.WriteString(fmt.Sprintf(" connect_timeout=%d", config.Security.ConnectionTimeout))
	}
	if config.StatementTimeout > 0 {
		connString.WriteString(fmt.Sprintf(" statement_timeout=%dms", config.StatementTimeout*1000))
	}

	// Set application name for .monitoring
	appName := config.ApplicationName
	if appName == "" {
		appName = "sql-graph-visualizer"
	}
	connString.WriteString(fmt.Sprintf(" application_name=%s", appName))

	logrus.Infof("Connecting to PostgreSQL database: %s@%s:%d/%s", username, config.Host, config.Port, config.Database)

	db, err := sql.Open("postgres", connString.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool limits
	db.SetMaxOpenConns(config.Security.MaxConnections)
	db.SetMaxIdleConns(config.Security.MaxConnections / 2)
	db.SetConnMaxLifetime(10 * time.Minute)

	// Test connection
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(config.Security.ConnectionTimeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctxTimeout); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Infof("Successfully connected to PostgreSQL database")
	return db, nil
}

// ValidateConnection performs comprehensive connection validation
func (r *PostgreSQLRepository) ValidateConnection(ctx context.Context, db *sql.DB) (*models.ConnectionValidationResult, error) {
	result := &models.ConnectionValidationResult{
		IsValid:      true,
		DatabaseInfo: make(map[string]string),
		Permissions:  []string{},
		ServerInfo:   make(map[string]string),
	}

	// Test basic connectivity
	if err := db.PingContext(ctx); err != nil {
		result.IsValid = false
		result.ErrorMessage = fmt.Sprintf("Database ping failed: %v", err)
		return result, nil
	}

	// Get database info
	if err := r.collectDatabaseInfo(ctx, db, result); err != nil {
		logrus.Warnf("Failed to collect database info: %v", err)
	}

	// Check permissions
	if err := r.checkDatabasePermissions(ctx, db, result); err != nil {
		logrus.Warnf("Failed to check permissions: %v", err)
	}

	return result, nil
}

// collectDatabaseInfo gathers basic PostgreSQL database information
func (r *PostgreSQLRepository) collectDatabaseInfo(ctx context.Context, db *sql.DB, result *models.ConnectionValidationResult) error {
	// Get PostgreSQL version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err == nil {
		result.ServerInfo["version"] = version
	}

	// Get current database
	var database string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database); err == nil {
		result.DatabaseInfo["current_database"] = database
	}

	// Get current user
	var user string
	if err := db.QueryRowContext(ctx, "SELECT current_user").Scan(&user); err == nil {
		result.DatabaseInfo["current_user"] = user
	}

	// Get current schema
	var schema string
	if err := db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schema); err == nil {
		result.DatabaseInfo["current_schema"] = schema
	}

	return nil
}

// checkDatabasePermissions verifies read-only access for PostgreSQL
func (r *PostgreSQLRepository) checkDatabasePermissions(ctx context.Context, db *sql.DB, result *models.ConnectionValidationResult) error {
	// Check table privileges in information_schema
	query := `
		SELECT DISTINCT privilege_type 
		FROM information_schema.role_table_grants 
		WHERE grantee = current_user
		UNION
		SELECT DISTINCT privilege_type 
		FROM information_schema.table_privileges 
		WHERE grantee = current_user
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var privilege string
		if err := rows.Scan(&privilege); err == nil {
			result.Permissions = append(result.Permissions, privilege)

			// Check for dangerous permissions
			privUpper := strings.ToUpper(privilege)
			if strings.Contains(privUpper, "INSERT") ||
				strings.Contains(privUpper, "UPDATE") ||
				strings.Contains(privUpper, "DELETE") ||
				strings.Contains(privUpper, "DROP") {
				result.HasWritePermissions = true
			}
		}
	}

	return nil
}

// DiscoverSchema analyzes PostgreSQL database schema and structure
func (r *PostgreSQLRepository) DiscoverSchema(ctx context.Context, db *sql.DB, filterConfig *models.DataFilteringConfig) (*models.SchemaAnalysisResult, error) {
	logrus.Infof("Starting PostgreSQL database schema discovery")

	result := &models.SchemaAnalysisResult{
		DatabaseName: "",
		Tables:       []*models.TableInfo{},
		DiscoveredAt: time.Now(),
	}

	// Get current database name
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}
	result.DatabaseName = dbName

	// Get all tables
	tableNames, err := r.GetTables(ctx, db, filterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	logrus.Infof("Found %d tables to analyze", len(tableNames))

	// Analyze each table
	for _, tableName := range tableNames {
		logrus.Debugf("Analyzing table: %s", tableName)
		tableInfo, err := r.GetTableInfo(ctx, db, tableName)
		if err != nil {
			logrus.Warnf("Failed to analyze table %s: %v", tableName, err)
			continue
		}
		result.Tables = append(result.Tables, tableInfo)
	}

	logrus.Infof("PostgreSQL schema discovery completed: %d tables analyzed", len(result.Tables))
	return result, nil
}

// GetTables returns list of PostgreSQL tables based on filtering configuration
func (r *PostgreSQLRepository) GetTables(ctx context.Context, db *sql.DB, filters *models.DataFilteringConfig) ([]string, error) {
	// Get tables from current schema (typically 'public' if not specified)
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = current_schema() 
			AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.QueryContext(ctx, query)
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

// applyTableFiltering applies whitelist/blacklist filtering (same logic as MySQL)
func (r *PostgreSQLRepository) applyTableFiltering(tables []string, filters *models.DataFilteringConfig) []string {
	if filters == nil {
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

// isInList checks if item exists in list (case-insensitive)
func (r *PostgreSQLRepository) isInList(item string, list []string) bool {
	for _, listItem := range list {
		if strings.EqualFold(item, listItem) {
			return true
		}
	}
	return false
}

// GetTableInfo analyzes individual PostgreSQL table structure
func (r *PostgreSQLRepository) GetTableInfo(ctx context.Context, db *sql.DB, tableName string) (*models.TableInfo, error) {
	tableInfo := &models.TableInfo{
		Name:            tableName,
		Columns:         []*models.ColumnInfo{},
		Relationships:   []*models.Relationship{},
		Recommendations: []string{},
	}

	// Get column information
	columns, err := r.getTableColumns(ctx, db, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}
	tableInfo.Columns = columns

	// Get row count estimate
	rowCount, err := r.getTableRowCount(ctx, db, tableName)
	if err != nil {
		logrus.Warnf("Failed to get row count for table %s: %v", tableName, err)
		rowCount = 0
	}
	tableInfo.EstimatedRows = rowCount

	return tableInfo, nil
}

// getTableColumns retrieves column information for a PostgreSQL table
func (r *PostgreSQLRepository) getTableColumns(ctx context.Context, db *sql.DB, tableName string) ([]*models.ColumnInfo, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length
		FROM information_schema.columns 
		WHERE table_schema = current_schema() 
			AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, tableName)
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
		)
		if err != nil {
			return nil, err
		}

		if defaultVal.Valid {
			col.DefaultValue = defaultVal.String
		}

		if maxLength.Valid {
			// Convert string to int if needed for MaxLength field
			// For now, storing in Extra field
			col.Extra = maxLength.String
		}

		// Check if this is a primary key or has constraints
		// This would require additional queries to pg_constraint, etc.
		// For now, basic implementation

		columns = append(columns, &col)
	}

	return columns, nil
}

// getTableRowCount estimates number of rows in PostgreSQL table
func (r *PostgreSQLRepository) getTableRowCount(ctx context.Context, db *sql.DB, tableName string) (int64, error) {
	// Use pg_stat_user_tables for quick estimate (PostgreSQL equivalent of MySQL's TABLE_ROWS)
	query := `
		SELECT COALESCE(n_tup_ins - n_tup_del, 0) as estimated_rows
		FROM pg_stat_user_tables 
		WHERE relname = $1
	`

	var rowCount sql.NullInt64
	err := db.QueryRowContext(ctx, query, tableName).Scan(&rowCount)
	if err != nil {
		// If pg_stat_user_tables doesn't have the data, fall back to COUNT(*)
		// This is slower but more accurate
		// #nosec G201 - tableName is escaped using EscapeIdentifier
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.EscapeIdentifier(tableName))
		err = db.QueryRowContext(ctx, countQuery).Scan(&rowCount)
		if err != nil {
			return 0, err
		}
	}

	if rowCount.Valid {
		return rowCount.Int64, nil
	}

	return 0, nil
}

// ExtractTableData extracts data from a PostgreSQL table with filtering
func (r *PostgreSQLRepository) ExtractTableData(ctx context.Context, db *sql.DB, tableName string, config *models.DataFilteringConfig) ([]map[string]any, error) {
	logrus.Infof("📤 Extracting data from PostgreSQL table: %s", tableName)

	// Build query with optional WHERE conditions
	query := fmt.Sprintf("SELECT * FROM %s", tableName)

	// Add WHERE condition if specified
	if config != nil && config.WhereConditions != nil {
		if condition, exists := config.WhereConditions[tableName]; exists && condition != "" {
			query += fmt.Sprintf(" WHERE %s", condition)
		}
	}

	// Add LIMIT if specified
	if config != nil && config.RowLimitPerTable > 0 {
		query += fmt.Sprintf(" LIMIT %d", config.RowLimitPerTable)
	}

	logrus.Debugf("Executing PostgreSQL query: %s", query)

	// Set query timeout
	ctxWithTimeout := ctx
	if config != nil && config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctxWithTimeout, cancel = context.WithTimeout(ctx, time.Duration(config.QueryTimeout)*time.Second)
		defer cancel()
	}

	return r.ExecuteQueryWithContext(ctxWithTimeout, query)
}

// ExecuteQueryWithContext executes PostgreSQL query with context timeout
func (r *PostgreSQLRepository) ExecuteQueryWithContext(ctx context.Context, query string) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		columnPointers := make([]any, len(columns))
		for i := range columns {
			columnPointers[i] = new(any)
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colName := range columns {
			row[colName] = *(columnPointers[i].(*any))
		}

		results = append(results, row)
	}

	return results, nil
}

// EstimateDataSize provides PostgreSQL dataset size estimation
func (r *PostgreSQLRepository) EstimateDataSize(ctx context.Context, db *sql.DB, config *models.DataFilteringConfig) (*models.DatasetInfo, error) {
	logrus.Infof("Estimating PostgreSQL dataset size")

	tables, err := r.GetTables(ctx, db, config)
	if err != nil {
		return nil, err
	}

	datasetInfo := &models.DatasetInfo{
		TotalTables: len(tables),
		TableSizes:  make(map[string]int64),
		AnalyzedAt:  time.Now(),
	}

	var totalRows int64
	for _, tableName := range tables {
		rowCount, err := r.getTableRowCount(ctx, db, tableName)
		if err != nil {
			logrus.Warnf("Failed to get row count for table %s: %v", tableName, err)
			continue
		}

		// Apply row limit if configured
		if config != nil && config.RowLimitPerTable > 0 && rowCount > int64(config.RowLimitPerTable) {
			rowCount = int64(config.RowLimitPerTable)
		}

		datasetInfo.TableSizes[tableName] = rowCount
		totalRows += rowCount
	}

	datasetInfo.TotalRows = totalRows
	datasetInfo.EstimatedSizeMB = r.estimateDataSizeInMB(totalRows)

	logrus.Infof("PostgreSQL dataset estimation: %d tables, %d total rows (~%.2f MB)",
		datasetInfo.TotalTables, datasetInfo.TotalRows, datasetInfo.EstimatedSizeMB)

	return datasetInfo, nil
}

// estimateDataSizeInMB provides rough size estimation for PostgreSQL
func (r *PostgreSQLRepository) estimateDataSizeInMB(totalRows int64) float64 {
	// Rough estimation: average 500 bytes per row (same as MySQL for consistency)
	const avgBytesPerRow = 500
	totalBytes := float64(totalRows) * avgBytesPerRow
	return totalBytes / (1024 * 1024) // Convert to MB
}
