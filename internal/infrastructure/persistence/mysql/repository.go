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
	"log"
	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/domain/models"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) ports.MySQLPort {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) FetchData() ([]map[string]any, error) {
	logrus.Infof("💾 FetchData called - returning empty slice (data loading moved to transform service)")
	return []map[string]any{}, nil
}

func (r *MySQLRepository) Close() error {
	return r.db.Close()
}

func (r *MySQLRepository) ExecuteQuery(query string) ([]map[string]any, error) {
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

// New methods for direct database connection (Issue #10)

// ConnectToExisting creates a new connection to existing database
func (r *MySQLRepository) ConnectToExisting(ctx context.Context, config *models.MySQLConfig) (*sql.DB, error) {
	// Use Username if set, otherwise fallback to User
	username := config.Username
	if username == "" {
		username = config.User
	}
	
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%ds&readTimeout=%ds&writeTimeout=%ds",
		username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Security.ConnectionTimeout,
		config.Security.QueryTimeout,
		config.Security.QueryTimeout,
	)

	// Add SSL configuration if enabled
	if config.SSLConfig.Enabled {
		dsn += "&tls=true"
		if config.SSLConfig.InsecureSkipVerify {
			dsn += "&tls=skip-verify"
		}
	}

	logrus.Infof("🔌 Connecting to existing database: %s@%s:%d/%s", username, config.Host, config.Port, config.Database)

	db, err := sql.Open("mysql", dsn)
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

	logrus.Infof("✅ Successfully connected to existing database")
	return db, nil
}

// ValidateConnection performs comprehensive connection validation
func (r *MySQLRepository) ValidateConnection(ctx context.Context, db *sql.DB) (*models.ConnectionValidationResult, error) {
	result := &models.ConnectionValidationResult{
		IsValid:        true,
		DatabaseInfo:   make(map[string]string),
		Permissions:    []string{},
		ServerInfo:     make(map[string]string),
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

// collectDatabaseInfo gathers basic database information
func (r *MySQLRepository) collectDatabaseInfo(ctx context.Context, db *sql.DB, result *models.ConnectionValidationResult) error {
	// Get MySQL version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err == nil {
		result.ServerInfo["version"] = version
	}

	// Get current database
	var database string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err == nil {
		result.DatabaseInfo["current_database"] = database
	}

	// Get current user
	var user string
	if err := db.QueryRowContext(ctx, "SELECT USER()").Scan(&user); err == nil {
		result.DatabaseInfo["current_user"] = user
	}

	return nil
}

// checkDatabasePermissions verifies read-only access
func (r *MySQLRepository) checkDatabasePermissions(ctx context.Context, db *sql.DB, result *models.ConnectionValidationResult) error {
	// Check if user has SELECT privileges
	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err == nil {
			result.Permissions = append(result.Permissions, grant)
			
			// Check for dangerous permissions
			grantUpper := strings.ToUpper(grant)
			if strings.Contains(grantUpper, "INSERT") || 
			   strings.Contains(grantUpper, "UPDATE") || 
			   strings.Contains(grantUpper, "DELETE") ||
			   strings.Contains(grantUpper, "DROP") {
				result.HasWritePermissions = true
			}
		}
	}

	return nil
}

// DiscoverSchema analyzes database schema and structure
func (r *MySQLRepository) DiscoverSchema(ctx context.Context, db *sql.DB, filterConfig *models.DataFilteringConfig) (*models.SchemaAnalysisResult, error) {
	logrus.Infof("🔍 Starting database schema discovery")
	
	result := &models.SchemaAnalysisResult{
		DatabaseName: "",
		Tables:       []*models.TableInfo{},
		DiscoveredAt: time.Now(),
	}

	// Get current database name
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName); err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}
	result.DatabaseName = dbName

	// Get all tables
	tableNames, err := r.GetTables(ctx, db, filterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	logrus.Infof("📋 Found %d tables to analyze", len(tableNames))

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

	logrus.Infof("✅ Schema discovery completed: %d tables analyzed", len(result.Tables))
	return result, nil
}

// GetTables returns list of tables based on filtering configuration
func (r *MySQLRepository) GetTables(ctx context.Context, db *sql.DB, filters *models.DataFilteringConfig) ([]string, error) {
	query := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'"
	
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

// applyTableFiltering applies whitelist/blacklist filtering
func (r *MySQLRepository) applyTableFiltering(tables []string, filters *models.DataFilteringConfig) []string {
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
func (r *MySQLRepository) isInList(item string, list []string) bool {
	for _, listItem := range list {
		if strings.EqualFold(item, listItem) {
			return true
		}
	}
	return false
}

// GetTableInfo analyzes individual table structure
func (r *MySQLRepository) GetTableInfo(ctx context.Context, db *sql.DB, tableName string) (*models.TableInfo, error) {
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

// getTableColumns retrieves column information for a table
func (r *MySQLRepository) getTableColumns(ctx context.Context, db *sql.DB, tableName string) ([]*models.ColumnInfo, error) {
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

	rows, err := db.QueryContext(ctx, query, tableName)
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

// getTableRowCount estimates number of rows in table
func (r *MySQLRepository) getTableRowCount(ctx context.Context, db *sql.DB, tableName string) (int64, error) {
	// Use INFORMATION_SCHEMA for quick estimate
	query := `
		SELECT TABLE_ROWS 
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ?
	`

	var rowCount sql.NullInt64
	err := db.QueryRowContext(ctx, query, tableName).Scan(&rowCount)
	if err != nil {
		return 0, err
	}

	if rowCount.Valid {
		return rowCount.Int64, nil
	}

	return 0, nil
}

// ExtractTableData extracts data from a table with filtering
func (r *MySQLRepository) ExtractTableData(ctx context.Context, db *sql.DB, tableName string, config *models.DataFilteringConfig) ([]map[string]any, error) {
	logrus.Infof("📤 Extracting data from table: %s", tableName)
	
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

	logrus.Debugf("Executing query: %s", query)

	// Set query timeout
	ctxWithTimeout := ctx
	if config != nil && config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctxWithTimeout, cancel = context.WithTimeout(ctx, time.Duration(config.QueryTimeout)*time.Second)
		defer cancel()
	}

	return r.ExecuteQueryWithContext(ctxWithTimeout, query)
}

// ExecuteQueryWithContext executes query with context timeout
func (r *MySQLRepository) ExecuteQueryWithContext(ctx context.Context, query string) ([]map[string]any, error) {
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

// EstimateDataSize provides dataset size estimation
func (r *MySQLRepository) EstimateDataSize(ctx context.Context, db *sql.DB, config *models.DataFilteringConfig) (*models.DatasetInfo, error) {
	logrus.Infof("📊 Estimating dataset size")
	
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

	logrus.Infof("📊 Dataset estimation: %d tables, %d total rows (~%.2f MB)", 
		datasetInfo.TotalTables, datasetInfo.TotalRows, datasetInfo.EstimatedSizeMB)

	return datasetInfo, nil
}

// estimateDataSizeInMB provides rough size estimation
func (r *MySQLRepository) estimateDataSizeInMB(totalRows int64) float64 {
	// Rough estimation: average 500 bytes per row
	const avgBytesPerRow = 500
	totalBytes := float64(totalRows) * avgBytesPerRow
	return totalBytes / (1024 * 1024) // Convert to MB
}
