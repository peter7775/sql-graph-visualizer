/*
 * SQL Graph Visualizer - Universal Database Service
 *
 * Copyright (c) 2025
 * Licensed under Dual License: AGPL-3.0 OR Commercial License
 * See LICENSE file for details
 * Patent Pending - Application filed for innovative database transformation techniques
 */

package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"sql-graph-visualizer/internal/domain/models"
	"sql-graph-visualizer/internal/domain/repository"

	"github.com/sirupsen/logrus"
)

// UniversalDatabaseService orchestrates database operations for any supported database type
// Works with MySQL, PostgreSQL, and future database types through the generic DatabaseRepository interface
type UniversalDatabaseService struct {
	repo           repository.DatabaseRepository
	config         models.DatabaseConfig
	dbType         models.DatabaseType
	securityValidator *SecurityValidationService
}

// NewUniversalDatabaseService creates a new universal database service
func NewUniversalDatabaseService(
	repo repository.DatabaseRepository,
	config models.DatabaseConfig,
) *UniversalDatabaseService {
	
	// Initialize security validator based on config
	var securityValidator *SecurityValidationService
	switch config.GetDatabaseType() {
	case models.DatabaseTypeMySQL:
		if mysqlConfig, ok := config.(*models.MySQLConfig); ok {
			securityValidator = NewSecurityValidationService(&mysqlConfig.Security)
		}
	case models.DatabaseTypePostgreSQL:
		if pgConfig, ok := config.(*models.PostgreSQLConfig); ok {
			securityValidator = NewSecurityValidationService(&pgConfig.Security)
		}
	}

	return &UniversalDatabaseService{
		repo:              repo,
		config:            config,
		dbType:            config.GetDatabaseType(),
		securityValidator: securityValidator,
	}
}

// ConnectAndAnalyze performs the complete workflow for any database type:
// 1. Security validation of connection parameters
// 2. Connection to existing database
// 3. Schema discovery and analysis
// 4. Transformation rule generation (basic)
func (s *UniversalDatabaseService) ConnectAndAnalyze(ctx context.Context) (*models.UniversalDatabaseAnalysisResult, error) {
	logrus.Infof("Starting universal database connection and analysis workflow for %s", s.dbType)
	
	result := &models.UniversalDatabaseAnalysisResult{
		StartTime:    time.Now(),
		Success:      false,
		DatabaseType: s.dbType,
		DatabaseInfo: &models.DatabaseConnectionInfo{},
	}

	// Step 1: Security validation (if validator is available)
	if s.securityValidator != nil {
		logrus.Infof("Step 1: Validating connection security")
		
		// Convert config for security validation
		var securityResult *models.SecurityValidationResult
		var err error
		
		switch s.dbType {
		case models.DatabaseTypeMySQL:
			if mysqlConfig, ok := s.config.(*models.MySQLConfig); ok {
				securityResult, err = s.securityValidator.ValidateConnectionSecurity(ctx, mysqlConfig)
			}
		case models.DatabaseTypePostgreSQL:
			// For PostgreSQL, create a basic security result for now
			securityResult = &models.SecurityValidationResult{
				IsValid:       true,
				SecurityLevel: "BASIC",
				Validations:   make(map[string]*models.ValidationCheck),
				Recommendations: []string{},
			}
		}
		
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Security validation failed: %v", err)
			result.EndTime = time.Now()
			return result, nil
		}
		
		result.SecurityValidation = securityResult
		if securityResult != nil && !securityResult.IsValid {
			result.ErrorMessage = "Connection failed security validation"
			result.EndTime = time.Now()
			return result, nil
		}
		
		logrus.Infof("Security validation passed")
	}

	// Step 2: Establish database connection
	logrus.Infof("Step 2: Connecting to %s database", s.dbType)
	db, err := s.repo.Connect(ctx, s.config)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Database connection failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			logrus.Warnf("Failed to close database connection: %v", closeErr)
		}
	}()

	// Step 3: Test connection and gather basic info
	logrus.Infof("Step 3: Validating database connection")
	if err := db.PingContext(ctx); err != nil {
		result.ErrorMessage = fmt.Sprintf("Database ping failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}

	// Populate database connection info
	result.DatabaseInfo.Host = s.config.GetHost()
	result.DatabaseInfo.Port = s.config.GetPort()
	result.DatabaseInfo.User = s.config.GetUsername()

	dbName, err := s.repo.GetDatabaseName(ctx)
	if err == nil {
		result.DatabaseInfo.Database = dbName
	}

	version, err := s.repo.GetDatabaseVersion(ctx)
	if err == nil {
		result.DatabaseInfo.Version = version
	}

	logrus.Infof("Connected to %s (User: %s, Version: %s)", 
		result.DatabaseInfo.Database, 
		result.DatabaseInfo.User, 
		result.DatabaseInfo.Version)

	// Step 4: Analyze database schema
	logrus.Infof("Step 4: Analyzing database schema")
	schemaResult, err := s.analyzeSchema(ctx, db)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Schema analysis failed: %v", err)
		result.EndTime = time.Now()
		return result, nil
	}
	
	result.SchemaAnalysis = schemaResult
	
	logrus.Infof("Schema analysis completed: %d tables analyzed", len(schemaResult.Tables))

	// Step 5: Generate summary and recommendations
	logrus.Infof("Step 5: Generating analysis summary")
	s.generateAnalysisSummary(result)

	result.Success = true
	result.EndTime = time.Now()
	result.ProcessingDuration = result.EndTime.Sub(result.StartTime)

	logrus.Infof("Universal database analysis completed successfully in %v", result.ProcessingDuration)
	
	return result, nil
}

// TestConnection performs a quick connection test without full analysis
func (s *UniversalDatabaseService) TestConnection(ctx context.Context) (*models.UniversalConnectionTestResult, error) {
	logrus.Infof("Testing %s database connection", s.dbType)
	
	testResult := &models.UniversalConnectionTestResult{
		TestedAt:     time.Now(),
		Success:      false,
		DatabaseType: s.dbType,
	}

	// Security validation if available
	if s.securityValidator != nil {
		switch s.dbType {
		case models.DatabaseTypeMySQL:
			if mysqlConfig, ok := s.config.(*models.MySQLConfig); ok {
				securityResult, err := s.securityValidator.ValidateConnectionSecurity(ctx, mysqlConfig)
				if err != nil || !securityResult.IsValid {
					testResult.ErrorMessage = "Connection failed security validation"
					if securityResult != nil {
						testResult.SecurityIssues = securityResult.Recommendations
					}
					return testResult, nil
				}
			}
		}
	}

	// Attempt connection
	db, err := s.repo.Connect(ctx, s.config)
	if err != nil {
		testResult.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return testResult, nil
	}
	defer db.Close()

	// Test connectivity
	if err := db.PingContext(ctx); err != nil {
		testResult.ErrorMessage = fmt.Sprintf("Database ping failed: %v", err)
		return testResult, nil
	}

	// Get database information
	dbName, err := s.repo.GetDatabaseName(ctx)
	if err == nil {
		testResult.DatabaseName = dbName
	}

	version, err := s.repo.GetDatabaseVersion(ctx)
	if err == nil {
		testResult.ServerVersion = version
	}

	testResult.UserName = s.config.GetUsername()

	// Get table count
	tables, err := s.repo.GetTables(ctx, s.config.GetDataFiltering())
	if err != nil {
		testResult.Warnings = append(testResult.Warnings, "Could not list tables")
	} else {
		testResult.TableCount = len(tables)
	}

	testResult.Success = true
	logrus.Infof("Connection test successful: %s@%s (%d tables)",
		testResult.UserName, testResult.DatabaseName, testResult.TableCount)

	return testResult, nil
}

// analyzeSchema performs schema analysis using the generic repository interface
func (s *UniversalDatabaseService) analyzeSchema(ctx context.Context, db *sql.DB) (*models.UniversalSchemaAnalysisResult, error) {
	result := &models.UniversalSchemaAnalysisResult{
		DatabaseName: "",
		Tables:       []*models.UniversalTableInfo{},
		DiscoveredAt: time.Now(),
	}

	// Get current database name
	dbName, err := s.repo.GetDatabaseName(ctx)
	if err == nil {
		result.DatabaseName = dbName
	}

	// Get all tables
	tableNames, err := s.repo.GetTables(ctx, s.config.GetDataFiltering())
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	logrus.Infof("Found %d tables to analyze", len(tableNames))

	// Analyze each table
	for _, tableName := range tableNames {
		logrus.Debugf("Analyzing table: %s", tableName)
		tableInfo, err := s.analyzeTable(ctx, tableName)
		if err != nil {
			logrus.Warnf("Failed to analyze table %s: %v", tableName, err)
			continue
		}
		result.Tables = append(result.Tables, tableInfo)
	}

	logrus.Infof("Schema analysis completed: %d tables analyzed", len(result.Tables))
	return result, nil
}

// analyzeTable analyzes individual table structure
func (s *UniversalDatabaseService) analyzeTable(ctx context.Context, tableName string) (*models.UniversalTableInfo, error) {
	tableInfo := &models.UniversalTableInfo{
		Name:            tableName,
		Columns:         []*models.ColumnInfo{},
		Relationships:   []*models.Relationship{},
		Recommendations: []string{},
	}

	// Get column information
	columns, err := s.repo.GetColumns(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
	}
	tableInfo.Columns = columns

	// Get row count estimate
	rowCount, err := s.repo.GetTableRowCount(ctx, tableName)
	if err == nil {
		tableInfo.EstimatedRows = rowCount
	} else {
		logrus.Warnf("Failed to get row count for table %s: %v", tableName, err)
	}

	return tableInfo, nil
}

// generateAnalysisSummary creates analysis summary and recommendations
func (s *UniversalDatabaseService) generateAnalysisSummary(result *models.UniversalDatabaseAnalysisResult) {
	summary := &models.UniversalAnalysisSummary{
		TotalTables:     len(result.SchemaAnalysis.Tables),
		Recommendations: []string{},
		Warnings:        []string{},
	}

	// Generate database-specific recommendations
	switch s.dbType {
	case models.DatabaseTypePostgreSQL:
		summary.Recommendations = append(summary.Recommendations,
			"PostgreSQL detected - consider using schema-specific filtering for better performance")
		if pgConfig, ok := s.config.(*models.PostgreSQLConfig); ok {
			if pgConfig.SSLConfig.Mode == "disable" {
				summary.Warnings = append(summary.Warnings,
					"SSL is disabled - consider enabling SSL for production databases")
			}
		}
	case models.DatabaseTypeMySQL:
		summary.Recommendations = append(summary.Recommendations,
			"MySQL detected - ensure proper indexing for large tables")
	}

	// Performance recommendations
	if summary.TotalTables > 50 {
		summary.Recommendations = append(summary.Recommendations,
			"Large number of tables detected - consider using table filtering to focus on important entities")
	}

	// Check for potential issues
	totalRows := int64(0)
	for _, table := range result.SchemaAnalysis.Tables {
		totalRows += table.EstimatedRows
	}

	if totalRows > 1000000 {
		summary.Recommendations = append(summary.Recommendations,
			"High row count detected (>1M rows) - consider using row limits for better performance")
	}

	result.Summary = summary

	logrus.Infof("Analysis summary: %d tables, %d warnings, %d recommendations",
		summary.TotalTables, len(summary.Warnings), len(summary.Recommendations))
}

// ValidateConfiguration validates the service configuration
func (s *UniversalDatabaseService) ValidateConfiguration() error {
	return s.config.Validate()
}

// GetConfiguration returns current service configuration
func (s *UniversalDatabaseService) GetConfiguration() models.DatabaseConfig {
	return s.config
}

// GetDatabaseType returns the database type
func (s *UniversalDatabaseService) GetDatabaseType() models.DatabaseType {
	return s.dbType
}
