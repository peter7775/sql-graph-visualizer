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

package models

import (
	"fmt"
	"time"
)

// ColumnInfo represents information about a database column
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   string `json:"is_nullable"` // "YES" or "NO"
	DefaultValue string `json:"default_value,omitempty"`
	MaxLength    int    `json:"max_length,omitempty"`
	IsKey        bool   `json:"is_key"`
	KeyType      string `json:"key_type,omitempty"` // PRIMARY, UNIQUE, INDEX, FOREIGN
	Extra        string `json:"extra,omitempty"`    // auto_increment, etc.
	Comment      string `json:"comment,omitempty"`
}

// IndexInfo represents information about a database index
type IndexInfo struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"is_unique"`
	Type     string   `json:"type"` // BTREE, HASH, etc.
}

// ForeignKeyInfo represents information about a foreign key relationship
type ForeignKeyInfo struct {
	Name             string `json:"name"`
	Column           string `json:"column"`
	ReferencedTable  string `json:"referenced_table"`
	ReferencedColumn string `json:"referenced_column"`
	OnDelete         string `json:"on_delete,omitempty"` // CASCADE, SET NULL, etc.
	OnUpdate         string `json:"on_update,omitempty"`
}

// Constraint represents a database constraint
type Constraint struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"`    // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK, etc.
	Columns           []string `json:"columns"` // columns involved in constraint
	TableName         string   `json:"table_name"`
	Condition         string   `json:"condition,omitempty"`          // for CHECK constraints
	ReferencedTable   string   `json:"referenced_table,omitempty"`   // for FOREIGN KEY
	ReferencedColumns []string `json:"referenced_columns,omitempty"` // for FOREIGN KEY
}

// TableInfo represents comprehensive information about a database table
type TableInfo struct {
	Name            string           `json:"name"`
	Schema          string           `json:"schema,omitempty"`
	Engine          string           `json:"engine,omitempty"`
	Columns         []*ColumnInfo    `json:"columns"`
	Indexes         []IndexInfo      `json:"indexes,omitempty"`
	ForeignKeys     []ForeignKeyInfo `json:"foreign_keys,omitempty"`
	Relationships   []*Relationship  `json:"relationships,omitempty"`
	EstimatedRows   int64            `json:"estimated_rows"`
	RowCount        int64            `json:"row_count,omitempty"`
	DataLength      int64            `json:"data_length,omitempty"`  // bytes
	IndexLength     int64            `json:"index_length,omitempty"` // bytes
	Comment         string           `json:"comment,omitempty"`
	GraphType       string           `json:"graph_type,omitempty"` // NODE, RELATIONSHIP
	Recommendations []string         `json:"recommendations,omitempty"`
	CreatedAt       *time.Time       `json:"created_at,omitempty"`
	UpdatedAt       *time.Time       `json:"updated_at,omitempty"`
}

// DatabaseSchema represents the complete schema of a database
type DatabaseSchema struct {
	DatabaseName string               `json:"database_name"`
	Tables       map[string]TableInfo `json:"tables"`
	Version      string               `json:"version,omitempty"`
	Charset      string               `json:"charset,omitempty"`
	Collation    string               `json:"collation,omitempty"`
	TotalTables  int                  `json:"total_tables"`
	TotalRows    int64                `json:"total_rows"`
	TotalSize    int64                `json:"total_size"` // bytes
	AnalyzedAt   time.Time            `json:"analyzed_at"`
}

// RelationshipInfo represents a relationship discovered between tables
type RelationshipInfo struct {
	FromTable    string  `json:"from_table"`
	FromColumn   string  `json:"from_column"`
	ToTable      string  `json:"to_table"`
	ToColumn     string  `json:"to_column"`
	RelationType string  `json:"relation_type"` // "ONE_TO_ONE", "ONE_TO_MANY", "MANY_TO_MANY"
	IsForeignKey bool    `json:"is_foreign_key"`
	IsImplicit   bool    `json:"is_implicit"` // Discovered by naming convention
	Confidence   float64 `json:"confidence"`  // 0.0 - 1.0 for implicit relationships
}

// SchemaAnalysisResult represents the result of database schema analysis
type SchemaAnalysisResult struct {
	DatabaseName   string                `json:"database_name"`
	Tables         []*TableInfo          `json:"tables"`
	GraphPatterns  []*GraphPattern       `json:"graph_patterns"`
	GeneratedRules []*TransformationRule `json:"generated_rules"`
	DatasetInfo    *DatasetInfo          `json:"dataset_info,omitempty"`
	DiscoveredAt   time.Time             `json:"discovered_at"`
	Suggestions    []string              `json:"suggestions,omitempty"`
	Warnings       []string              `json:"warnings,omitempty"`
}

// GraphPattern represents identified graph database patterns
type GraphPattern struct {
	PatternType string   `json:"pattern_type"` // STAR_SCHEMA, HIERARCHY, etc.
	CenterTable string   `json:"center_table,omitempty"`
	Description string   `json:"description"`
	Confidence  float64  `json:"confidence"`
	Tables      []string `json:"tables,omitempty"`
}

// TransformationRule represents an auto-generated transformation rule
type TransformationRule struct {
	RuleID        string  `json:"rule_id"`
	RuleType      string  `json:"rule_type"` // "NODE_CREATION", "RELATIONSHIP_CREATION"
	SourceTable   string  `json:"source_table"`
	CypherQuery   string  `json:"cypher_query"`
	Description   string  `json:"description"`
	AutoGenerated bool    `json:"auto_generated"`
	Confidence    float64 `json:"confidence"` // 0.0-1.0 confidence score
}

// Relationship represents a database relationship
type Relationship struct {
	SourceTable      string `json:"source_table"`
	SourceColumn     string `json:"source_column"`
	TargetTable      string `json:"target_table"`
	TargetColumn     string `json:"target_column"`
	RelationshipType string `json:"relationship_type"`
	ConstraintName   string `json:"constraint_name,omitempty"`
}

// AnalysisStatistics provides statistics about the analysis process
type AnalysisStatistics struct {
	ProcessingTime    time.Duration `json:"processing_time"`
	TablesAnalyzed    int           `json:"tables_analyzed"`
	RelationsFound    int           `json:"relations_found"`
	ImplicitRelations int           `json:"implicit_relations"`
	Warnings          int           `json:"warnings"`
	LargestTable      string        `json:"largest_table,omitempty"`
	LargestTableSize  int64         `json:"largest_table_size"`
}

// DatasetInfo represents information about data that will be extracted
type DatasetInfo struct {
	TotalTables       int              `json:"total_tables"`
	TotalRows         int64            `json:"total_rows"`
	TableSizes        map[string]int64 `json:"table_sizes"` // table_name -> estimated_rows
	EstimatedSizeMB   float64          `json:"estimated_size_mb"`
	AnalyzedAt        time.Time        `json:"analyzed_at"`
	FilteredTables    []string         `json:"filtered_tables,omitempty"`
	ProcessingOrder   []string         `json:"processing_order,omitempty"`
	EstimatedDuration time.Duration    `json:"estimated_duration,omitempty"`
}

// ValidationError represents a database validation error
type ValidationError struct {
	Type       string `json:"type"` // "CONNECTION", "PERMISSION", "SCHEMA", "DATA"
	Message    string `json:"message"`
	Table      string `json:"table,omitempty"`
	Column     string `json:"column,omitempty"`
	Severity   string `json:"severity"` // "ERROR", "WARNING", "INFO"
	Suggestion string `json:"suggestion,omitempty"`
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return e.Message
}

// ConnectionValidationResult represents the result of database connection validation
type ConnectionValidationResult struct {
	IsValid             bool              `json:"is_valid"`
	ErrorMessage        string            `json:"error_message,omitempty"`
	Errors              []ValidationError `json:"errors,omitempty"`
	Warnings            []ValidationError `json:"warnings,omitempty"`
	DatabaseInfo        map[string]string `json:"database_info"` // version, charset, etc.
	ServerInfo          map[string]string `json:"server_info"`   // server version, etc.
	Permissions         []string          `json:"permissions"`   // granted permissions
	ConnectionTime      time.Duration     `json:"connection_time"`
	HasWritePermissions bool              `json:"has_write_permissions"`
}

// DirectDatabaseAnalysisResult contains complete analysis results
type DirectDatabaseAnalysisResult struct {
	StartTime            time.Time                   `json:"start_time"`
	EndTime              time.Time                   `json:"end_time"`
	ProcessingDuration   time.Duration               `json:"processing_duration"`
	Success              bool                        `json:"success"`
	ErrorMessage         string                      `json:"error_message,omitempty"`
	DatabaseInfo         *DatabaseConnectionInfo     `json:"database_info"`
	SecurityValidation   *SecurityValidationResult   `json:"security_validation"`
	ConnectionValidation *ConnectionValidationResult `json:"connection_validation"`
	SchemaAnalysis       *SchemaAnalysisResult       `json:"schema_analysis"`
	Summary              *AnalysisSummary            `json:"summary"`
}

// DatabaseConnectionInfo contains database connection information
type DatabaseConnectionInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Version  string `json:"version"`
}

// AnalysisSummary provides high-level analysis summary
type AnalysisSummary struct {
	TotalTables       int      `json:"total_tables"`
	TotalRules        int      `json:"total_rules"`
	NodeRules         int      `json:"node_rules"`
	RelationshipRules int      `json:"relationship_rules"`
	TotalPatterns     int      `json:"total_patterns"`
	Recommendations   []string `json:"recommendations"`
	Warnings          []string `json:"warnings"`
}

// ConnectionTestResult contains simple connection test results
type ConnectionTestResult struct {
	TestedAt       time.Time `json:"tested_at"`
	Success        bool      `json:"success"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	DatabaseName   string    `json:"database_name,omitempty"`
	ServerVersion  string    `json:"server_version,omitempty"`
	UserName       string    `json:"user_name,omitempty"`
	TableCount     int       `json:"table_count"`
	SecurityIssues []string  `json:"security_issues,omitempty"`
	Warnings       []string  `json:"warnings,omitempty"`
}

// SecurityValidationResult represents security validation results
type SecurityValidationResult struct {
	IsValid         bool                        `json:"is_valid"`
	SecurityLevel   string                      `json:"security_level"` // HIGH, MEDIUM, LOW, CRITICAL_RISK
	Validations     map[string]*ValidationCheck `json:"validations"`
	Recommendations []string                    `json:"recommendations"`
	ErrorMessage    string                      `json:"error_message,omitempty"`
}

// ValidationCheck represents individual security validation check
type ValidationCheck struct {
	CheckName   string `json:"check_name"`
	Passed      bool   `json:"passed"`
	Severity    string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Message     string `json:"message"`
	Description string `json:"description"`
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Universal Database Analysis Models for multi-database support

// UniversalDatabaseAnalysisResult contains complete analysis results for any database type
type UniversalDatabaseAnalysisResult struct {
	StartTime          time.Time                      `json:"start_time"`
	EndTime            time.Time                      `json:"end_time"`
	ProcessingDuration time.Duration                  `json:"processing_duration"`
	Success            bool                           `json:"success"`
	DatabaseType       DatabaseType                   `json:"database_type"`
	ErrorMessage       string                         `json:"error_message,omitempty"`
	DatabaseInfo       *DatabaseConnectionInfo        `json:"database_info"`
	SecurityValidation *SecurityValidationResult      `json:"security_validation,omitempty"`
	SchemaAnalysis     *UniversalSchemaAnalysisResult `json:"schema_analysis"`
	Summary            *UniversalAnalysisSummary      `json:"summary"`
}

// UniversalConnectionTestResult contains connection test results for any database type
type UniversalConnectionTestResult struct {
	TestedAt       time.Time    `json:"tested_at"`
	Success        bool         `json:"success"`
	DatabaseType   DatabaseType `json:"database_type"`
	ErrorMessage   string       `json:"error_message,omitempty"`
	DatabaseName   string       `json:"database_name,omitempty"`
	ServerVersion  string       `json:"server_version,omitempty"`
	UserName       string       `json:"user_name,omitempty"`
	TableCount     int          `json:"table_count"`
	SecurityIssues []string     `json:"security_issues,omitempty"`
	Warnings       []string     `json:"warnings,omitempty"`
}

// UniversalSchemaAnalysisResult represents schema analysis for any database type
type UniversalSchemaAnalysisResult struct {
	DatabaseName string                `json:"database_name"`
	Tables       []*UniversalTableInfo `json:"tables"`
	DiscoveredAt time.Time             `json:"discovered_at"`
	Suggestions  []string              `json:"suggestions,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

// UniversalTableInfo represents table information for any database type
type UniversalTableInfo struct {
	Name            string          `json:"name"`
	Schema          string          `json:"schema,omitempty"`
	Columns         []*ColumnInfo   `json:"columns"`
	Relationships   []*Relationship `json:"relationships,omitempty"`
	EstimatedRows   int64           `json:"estimated_rows"`
	Recommendations []string        `json:"recommendations,omitempty"`
}

// UniversalAnalysisSummary provides high-level analysis summary for any database type
type UniversalAnalysisSummary struct {
	TotalTables     int      `json:"total_tables"`
	Recommendations []string `json:"recommendations"`
	Warnings        []string `json:"warnings"`
}

// ColumnStatistics contains statistical information about a database column
type ColumnStatistics struct {
	ColumnName  string            `json:"column_name"`
	DataType    string            `json:"data_type"`
	TotalCount  int64             `json:"total_count"`
	NullCount   int64             `json:"null_count"`
	UniqueCount int64             `json:"unique_count"`
	MinValue    interface{}       `json:"min_value,omitempty"`
	MaxValue    interface{}       `json:"max_value,omitempty"`
	AvgValue    interface{}       `json:"avg_value,omitempty"`
	MinLength   int               `json:"min_length,omitempty"`
	MaxLength   int               `json:"max_length,omitempty"`
	AvgLength   float64           `json:"avg_length,omitempty"`
	TopValues   []ValueFreq       `json:"top_values,omitempty"`
	Histogram   []HistogramBucket `json:"histogram,omitempty"`
	Cardinality float64           `json:"cardinality"` // unique_count / total_count
}

// ValueFreq represents a value and its frequency
type ValueFreq struct {
	Value     interface{} `json:"value"`
	Frequency int64       `json:"frequency"`
}

// HistogramBucket represents a histogram bucket
type HistogramBucket struct {
	LowerBound interface{} `json:"lower_bound"`
	UpperBound interface{} `json:"upper_bound"`
	Count      int64       `json:"count"`
}

// TableSize contains information about table storage size
type TableSize struct {
	TableName     string  `json:"table_name"`
	RowCount      int64   `json:"row_count"`
	DataSize      int64   `json:"data_size"`  // bytes
	IndexSize     int64   `json:"index_size"` // bytes
	TotalSize     int64   `json:"total_size"` // bytes
	DataSizeMB    float64 `json:"data_size_mb"`
	IndexSizeMB   float64 `json:"index_size_mb"`
	TotalSizeMB   float64 `json:"total_size_mb"`
	Compression   float64 `json:"compression,omitempty"`   // compression ratio
	Fragmentation float64 `json:"fragmentation,omitempty"` // fragmentation percentage
}

// UserPrivileges contains information about database user privileges
type UserPrivileges struct {
	UserName      string                         `json:"user_name"`
	Host          string                         `json:"host"`
	DatabaseName  string                         `json:"database_name,omitempty"`
	GlobalPrivs   []string                       `json:"global_privileges"`
	DatabasePrivs map[string][]string            `json:"database_privileges"` // database -> privileges
	TablePrivs    map[string][]string            `json:"table_privileges"`    // table -> privileges
	ColumnPrivs   map[string]map[string][]string `json:"column_privileges"`   // table -> column -> privileges
	HasSelect     bool                           `json:"has_select"`
	HasInsert     bool                           `json:"has_insert"`
	HasUpdate     bool                           `json:"has_update"`
	HasDelete     bool                           `json:"has_delete"`
	HasCreate     bool                           `json:"has_create"`
	HasDrop       bool                           `json:"has_drop"`
	HasAdmin      bool                           `json:"has_admin"`
}

// DatabaseError represents a database-specific error
type DatabaseError struct {
	ErrorCode      string `json:"error_code,omitempty"`
	SQLState       string `json:"sql_state,omitempty"`
	Message        string `json:"message"`
	OriginalErr    error  `json:"-"`          // Original error, not serialized
	ErrorType      string `json:"error_type"` // CONNECTION, TIMEOUT, PERMISSION, CONSTRAINT, SYNTAX, etc.
	Retryable      bool   `json:"retryable"`
	TableName      string `json:"table_name,omitempty"`
	ColumnName     string `json:"column_name,omitempty"`
	ConstraintName string `json:"constraint_name,omitempty"`
}

// Error implements the error interface
func (e *DatabaseError) Error() string {
	if e.ErrorCode != "" {
		return fmt.Sprintf("%s (%s): %s", e.ErrorType, e.ErrorCode, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.ErrorType, e.Message)
}

// Unwrap returns the original error
func (e *DatabaseError) Unwrap() error {
	return e.OriginalErr
}
