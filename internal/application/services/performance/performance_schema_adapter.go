package performance

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	
	"github.com/sirupsen/logrus"
)

// PerformanceSchemaAdapter collects performance data from MySQL Performance Schema
type PerformanceSchemaAdapter struct {
	db          *sql.DB
	logger      *logrus.Logger
	config      *PerformanceSchemaConfig
	
	// Caching and state management
	lastCollection  time.Time
	mutex          sync.RWMutex
	isConnected    bool
	
	// Query cache for performance schema queries
	queryCache     map[string]*sql.Stmt
	queryCacheMux  sync.RWMutex
}

// PerformanceSchemaConfig contains configuration for Performance Schema data collection
type PerformanceSchemaConfig struct {
	// Collection settings
	CollectionInterval  time.Duration `yaml:"collection_interval" json:"collection_interval"`
	SlowQueryThreshold  time.Duration `yaml:"slow_query_threshold" json:"slow_query_threshold"`
	MaxHistoryRetention time.Duration `yaml:"max_history_retention" json:"max_history_retention"`
	
	// Data collection toggles
	CollectStatements   bool `yaml:"collect_statements" json:"collect_statements"`
	CollectTableIO      bool `yaml:"collect_table_io" json:"collect_table_io"`
	CollectIndexUsage   bool `yaml:"collect_index_usage" json:"collect_index_usage"`
	CollectWaitEvents   bool `yaml:"collect_wait_events" json:"collect_wait_events"`
	CollectConnections  bool `yaml:"collect_connections" json:"collect_connections"`
	CollectReplication  bool `yaml:"collect_replication" json:"collect_replication"`
	
	// Query limits
	MaxStatements       int `yaml:"max_statements" json:"max_statements"`
	MaxTables           int `yaml:"max_tables" json:"max_tables"`
	
	// Filtering options
	IgnoredSchemas      []string `yaml:"ignored_schemas" json:"ignored_schemas"`
	IgnoredUsers        []string `yaml:"ignored_users" json:"ignored_users"`
	FocusedTables       []string `yaml:"focused_tables" json:"focused_tables"`
	
	// Advanced settings
	EnableDigestText    bool    `yaml:"enable_digest_text" json:"enable_digest_text"`
	MinExecutionCount   int64   `yaml:"min_execution_count" json:"min_execution_count"`
	MinAvgLatency       float64 `yaml:"min_avg_latency" json:"min_avg_latency"` // milliseconds
}

// PerformanceSchemaData contains collected performance data
type PerformanceSchemaData struct {
	CollectionTime     time.Time                    `json:"collection_time"`
	GlobalStatus       *GlobalStatusData            `json:"global_status"`
	StatementStats     []StatementStatistic         `json:"statement_stats"`
	TableIOStats       []TableIOStatistic           `json:"table_io_stats"`
	IndexStats         []IndexStatistic             `json:"index_stats"`
	WaitEventStats     []WaitEventStatistic         `json:"wait_event_stats"`
	ConnectionStats    *ConnectionStatistics        `json:"connection_stats"`
	ReplicationStats   *ReplicationStatistics       `json:"replication_stats"`
	SlowQueries        []SlowQueryInfo              `json:"slow_queries"`
}

// GlobalStatusData contains global MySQL status information
type GlobalStatusData struct {
	QueriesPerSecond       float64 `json:"queries_per_second"`
	ConnectionsPerSecond   float64 `json:"connections_per_second"`
	SlowQueries            int64   `json:"slow_queries"`
	OpenTables             int64   `json:"open_tables"`
	ThreadsRunning         int64   `json:"threads_running"`
	ThreadsConnected       int64   `json:"threads_connected"`
	InnodbBufferPoolHitRate float64 `json:"innodb_buffer_pool_hit_rate"`
	KeyCacheHitRate        float64 `json:"key_cache_hit_rate"`
	TmpTablesCreated       int64   `json:"tmp_tables_created"`
	TmpDiskTablesCreated   int64   `json:"tmp_disk_tables_created"`
}

// StatementStatistic contains per-statement performance data
type StatementStatistic struct {
	SchemaName       string        `json:"schema_name"`
	Digest           string        `json:"digest"`
	DigestText       string        `json:"digest_text,omitempty"`
	CountStar        int64         `json:"count_star"`
	SumTimerWait     time.Duration `json:"sum_timer_wait"`
	MinTimerWait     time.Duration `json:"min_timer_wait"`
	AvgTimerWait     time.Duration `json:"avg_timer_wait"`
	MaxTimerWait     time.Duration `json:"max_timer_wait"`
	SumRowsAffected  int64         `json:"sum_rows_affected"`
	SumRowsSent      int64         `json:"sum_rows_sent"`
	SumRowsExamined  int64         `json:"sum_rows_examined"`
	SumCreatedTmpTables int64      `json:"sum_created_tmp_tables"`
	SumCreatedTmpDiskTables int64  `json:"sum_created_tmp_disk_tables"`
	SumSelectFullJoin int64        `json:"sum_select_full_join"`
	SumSelectScan    int64         `json:"sum_select_scan"`
	SumSortScan      int64         `json:"sum_sort_scan"`
	SumSortRows      int64         `json:"sum_sort_rows"`
	SumNoIndexUsed   int64         `json:"sum_no_index_used"`
	SumNoGoodIndexUsed int64       `json:"sum_no_good_index_used"`
	FirstSeen        time.Time     `json:"first_seen"`
	LastSeen         time.Time     `json:"last_seen"`
}

// TableIOStatistic contains per-table I/O performance data
type TableIOStatistic struct {
	SchemaName       string  `json:"schema_name"`
	TableName        string  `json:"table_name"`
	CountRead        int64   `json:"count_read"`
	SumTimerRead     time.Duration `json:"sum_timer_read"`
	CountWrite       int64   `json:"count_write"`
	SumTimerWrite    time.Duration `json:"sum_timer_write"`
	CountFetch       int64   `json:"count_fetch"`
	SumTimerFetch    time.Duration `json:"sum_timer_fetch"`
	CountInsert      int64   `json:"count_insert"`
	SumTimerInsert   time.Duration `json:"sum_timer_insert"`
	CountUpdate      int64   `json:"count_update"`
	SumTimerUpdate   time.Duration `json:"sum_timer_update"`
	CountDelete      int64   `json:"count_delete"`
	SumTimerDelete   time.Duration `json:"sum_timer_delete"`
}

// IndexStatistic contains index usage statistics
type IndexStatistic struct {
	SchemaName     string  `json:"schema_name"`
	TableName      string  `json:"table_name"`
	IndexName      string  `json:"index_name"`
	CountFetch     int64   `json:"count_fetch"`
	SumTimerFetch  time.Duration `json:"sum_timer_fetch"`
	CountInsert    int64   `json:"count_insert"`
	SumTimerInsert time.Duration `json:"sum_timer_insert"`
	CountUpdate    int64   `json:"count_update"`
	SumTimerUpdate time.Duration `json:"sum_timer_update"`
	CountDelete    int64   `json:"count_delete"`
	SumTimerDelete time.Duration `json:"sum_timer_delete"`
}

// WaitEventStatistic contains wait event statistics
type WaitEventStatistic struct {
	EventName     string        `json:"event_name"`
	CountStar     int64         `json:"count_star"`
	SumTimerWait  time.Duration `json:"sum_timer_wait"`
	MinTimerWait  time.Duration `json:"min_timer_wait"`
	AvgTimerWait  time.Duration `json:"avg_timer_wait"`
	MaxTimerWait  time.Duration `json:"max_timer_wait"`
}

// ConnectionStatistics contains connection-related statistics
type ConnectionStatistics struct {
	CurrentConnections int64   `json:"current_connections"`
	TotalConnections   int64   `json:"total_connections"`
	ConnectionsPerSec  float64 `json:"connections_per_sec"`
	AbortedConnections int64   `json:"aborted_connections"`
	AbortedClients     int64   `json:"aborted_clients"`
	MaxUsedConnections int64   `json:"max_used_connections"`
}

// ReplicationStatistics contains replication-related statistics
type ReplicationStatistics struct {
	SlaveRunning        bool          `json:"slave_running"`
	SecondsBehindMaster *int64        `json:"seconds_behind_master"`
	MasterLogFile       string        `json:"master_log_file"`
	MasterLogPos        int64         `json:"master_log_pos"`
	RelayLogFile        string        `json:"relay_log_file"`
	RelayLogPos         int64         `json:"relay_log_pos"`
	LastIOError         string        `json:"last_io_error,omitempty"`
	LastSQLError        string        `json:"last_sql_error,omitempty"`
}

// SlowQueryInfo contains information about slow queries
type SlowQueryInfo struct {
	StartTime      time.Time     `json:"start_time"`
	UserHost       string        `json:"user_host"`
	QueryTime      time.Duration `json:"query_time"`
	LockTime       time.Duration `json:"lock_time"`
	RowsSent       int64         `json:"rows_sent"`
	RowsExamined   int64         `json:"rows_examined"`
	SQLText        string        `json:"sql_text"`
	Schema         string        `json:"schema"`
}

// NewPerformanceSchemaAdapter creates a new Performance Schema adapter
func NewPerformanceSchemaAdapter(db *sql.DB, logger *logrus.Logger, config *PerformanceSchemaConfig) *PerformanceSchemaAdapter {
	if config == nil {
		config = defaultPerformanceSchemaConfig()
	}

	adapter := &PerformanceSchemaAdapter{
		db:         db,
		logger:     logger,
		config:     config,
		queryCache: make(map[string]*sql.Stmt),
	}

	// Test connection and Performance Schema availability
	adapter.testConnection()

	return adapter
}

// CollectPerformanceData collects current performance data from Performance Schema
func (p *PerformanceSchemaAdapter) CollectPerformanceData(ctx context.Context) (*PerformanceSchemaData, error) {
	if !p.isConnected {
		return nil, fmt.Errorf("not connected to MySQL Performance Schema")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	data := &PerformanceSchemaData{
		CollectionTime: time.Now(),
	}

	// Collect global status
	if globalStatus, err := p.collectGlobalStatus(ctx); err != nil {
		p.logger.WithError(err).Warn("Failed to collect global status")
	} else {
		data.GlobalStatus = globalStatus
	}

	// Collect statement statistics
	if p.config.CollectStatements {
		if statements, err := p.collectStatementStats(ctx); err != nil {
			p.logger.WithError(err).Warn("Failed to collect statement statistics")
		} else {
			data.StatementStats = statements
		}
	}

	// Collect table I/O statistics
	if p.config.CollectTableIO {
		if tableIO, err := p.collectTableIOStats(ctx); err != nil {
			p.logger.WithError(err).Warn("Failed to collect table I/O statistics")
		} else {
			data.TableIOStats = tableIO
		}
	}

	// Collect index statistics
	if p.config.CollectIndexUsage {
		if indexes, err := p.collectIndexStats(ctx); err != nil {
			p.logger.WithError(err).Warn("Failed to collect index statistics")
		} else {
			data.IndexStats = indexes
		}
	}

	// Collect wait event statistics
	if p.config.CollectWaitEvents {
		if waitEvents, err := p.collectWaitEventStats(ctx); err != nil {
			p.logger.WithError(err).Warn("Failed to collect wait event statistics")
		} else {
			data.WaitEventStats = waitEvents
		}
	}

	// Collect connection statistics
	if p.config.CollectConnections {
		if connections, err := p.collectConnectionStats(ctx); err != nil {
			p.logger.WithError(err).Warn("Failed to collect connection statistics")
		} else {
			data.ConnectionStats = connections
		}
	}

	// Collect replication statistics
	if p.config.CollectReplication {
		if replication, err := p.collectReplicationStats(ctx); err != nil {
			p.logger.WithError(err).Debug("Failed to collect replication statistics (may not be configured)")
		} else {
			data.ReplicationStats = replication
		}
	}

	// Collect slow queries
	if slowQueries, err := p.collectSlowQueries(ctx); err != nil {
		p.logger.WithError(err).Warn("Failed to collect slow queries")
	} else {
		data.SlowQueries = slowQueries
	}

	p.lastCollection = data.CollectionTime

	p.logger.WithFields(logrus.Fields{
		"statements_collected": len(data.StatementStats),
		"tables_collected":     len(data.TableIOStats),
		"slow_queries":         len(data.SlowQueries),
	}).Debug("Performance Schema data collection completed")

	return data, nil
}

// ConvertToPerformanceMetrics converts Performance Schema data to standard performance metrics
func (p *PerformanceSchemaAdapter) ConvertToPerformanceMetrics(data *PerformanceSchemaData) *ports.PerformanceMetrics {
	metrics := &ports.PerformanceMetrics{}

	if data.GlobalStatus != nil {
		metrics.QueriesPerSecond = data.GlobalStatus.QueriesPerSecond
		// Additional global metrics mapping
	}

	// Aggregate statement statistics
	if len(data.StatementStats) > 0 {
		var totalQueries int64
		var totalLatency time.Duration
		var minLatency, maxLatency time.Duration = time.Hour, 0

		for _, stmt := range data.StatementStats {
			totalQueries += stmt.CountStar
			totalLatency += stmt.SumTimerWait
			
			if stmt.MinTimerWait < minLatency {
				minLatency = stmt.MinTimerWait
			}
			if stmt.MaxTimerWait > maxLatency {
				maxLatency = stmt.MaxTimerWait
			}
		}

		if totalQueries > 0 {
			metrics.AverageLatency = float64(totalLatency.Milliseconds()) / float64(totalQueries)
			metrics.MinLatency = float64(minLatency.Milliseconds())
			metrics.MaxLatency = float64(maxLatency.Milliseconds())
		}
	}

	return metrics
}

// ConvertToQueryPerformance converts statement statistics to query performance data
func (p *PerformanceSchemaAdapter) ConvertToQueryPerformance(data *PerformanceSchemaData) []ports.QueryPerformance {
	queryPerformance := make([]ports.QueryPerformance, 0, len(data.StatementStats))

	for _, stmt := range data.StatementStats {
		// Extract table names from digest text (basic implementation)
		tables := p.extractTableNames(stmt.DigestText)
		
		perf := ports.QueryPerformance{
			QueryPattern:       stmt.DigestText,
			QueryType:          p.identifyQueryType(stmt.DigestText),
			ExecutionCount:     stmt.CountStar,
			TotalTime:          stmt.SumTimerWait,
			AverageTime:        stmt.AvgTimerWait,
			MinTime:            stmt.MinTimerWait,
			MaxTime:            stmt.MaxTimerWait,
			SourceTables:       tables,
			RowsExamined:       stmt.SumRowsExamined,
			RowsReturned:       stmt.SumRowsSent,
			IndexUsed:          stmt.SumNoIndexUsed == 0,
			RelationshipType:   p.determineRelationshipType(stmt),
			PerformanceImpact:  p.classifyPerformanceImpact(stmt.AvgTimerWait),
		}

		queryPerformance = append(queryPerformance, perf)
	}

	return queryPerformance
}

// Private implementation methods

func (p *PerformanceSchemaAdapter) testConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic connection
	if err := p.db.PingContext(ctx); err != nil {
		p.logger.WithError(err).Error("Failed to ping MySQL database")
		p.isConnected = false
		return
	}

	// Test Performance Schema availability
	var psEnabled int
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'performance_schema'"
	if err := p.db.QueryRowContext(ctx, query).Scan(&psEnabled); err != nil {
		p.logger.WithError(err).Error("Failed to check Performance Schema availability")
		p.isConnected = false
		return
	}

	if psEnabled == 0 {
		p.logger.Error("Performance Schema is not available")
		p.isConnected = false
		return
	}

	p.isConnected = true
	p.logger.Info("Connected to MySQL Performance Schema")
}

func (p *PerformanceSchemaAdapter) getOrCreateStatement(query string) (*sql.Stmt, error) {
	p.queryCacheMux.RLock()
	if stmt, exists := p.queryCache[query]; exists {
		p.queryCacheMux.RUnlock()
		return stmt, nil
	}
	p.queryCacheMux.RUnlock()

	p.queryCacheMux.Lock()
	defer p.queryCacheMux.Unlock()

	// Double-check after acquiring write lock
	if stmt, exists := p.queryCache[query]; exists {
		return stmt, nil
	}

	stmt, err := p.db.Prepare(query)
	if err != nil {
		return nil, err
	}

	p.queryCache[query] = stmt
	return stmt, nil
}

func (p *PerformanceSchemaAdapter) collectGlobalStatus(ctx context.Context) (*GlobalStatusData, error) {
	query := `
		SELECT 
			variable_name, 
			variable_value 
		FROM performance_schema.global_status 
		WHERE variable_name IN (
			'Queries', 'Connections', 'Slow_queries', 'Open_tables',
			'Threads_running', 'Threads_connected',
			'Innodb_buffer_pool_read_requests', 'Innodb_buffer_pool_reads',
			'Key_read_requests', 'Key_reads',
			'Created_tmp_tables', 'Created_tmp_disk_tables'
		)`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query global status: %w", err)
	}
	defer rows.Close()

	statusMap := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		statusMap[name] = value
	}

	status := &GlobalStatusData{}
	
	// Parse numeric values
	if val, exists := statusMap["Queries"]; exists {
		if queries, err := strconv.ParseInt(val, 10, 64); err == nil {
			// Calculate QPS based on uptime (simplified)
			status.QueriesPerSecond = float64(queries) / 60.0 // Rough estimate
		}
	}

	if val, exists := statusMap["Connections"]; exists {
		if connections, err := strconv.ParseInt(val, 10, 64); err == nil {
			status.ConnectionsPerSecond = float64(connections) / 60.0
		}
	}

	if val, exists := statusMap["Slow_queries"]; exists {
		if slowQueries, err := strconv.ParseInt(val, 10, 64); err == nil {
			status.SlowQueries = slowQueries
		}
	}

	if val, exists := statusMap["Threads_running"]; exists {
		if threadsRunning, err := strconv.ParseInt(val, 10, 64); err == nil {
			status.ThreadsRunning = threadsRunning
		}
	}

	// Calculate buffer pool hit rate
	if readRequests, exists1 := statusMap["Innodb_buffer_pool_read_requests"]; exists1 {
		if reads, exists2 := statusMap["Innodb_buffer_pool_reads"]; exists2 {
			if reqVal, err1 := strconv.ParseFloat(readRequests, 64); err1 == nil {
				if readsVal, err2 := strconv.ParseFloat(reads, 64); err2 == nil && reqVal > 0 {
					status.InnodbBufferPoolHitRate = (reqVal - readsVal) / reqVal * 100
				}
			}
		}
	}

	return status, nil
}

func (p *PerformanceSchemaAdapter) collectStatementStats(ctx context.Context) ([]StatementStatistic, error) {
	query := `
		SELECT 
			COALESCE(schema_name, 'NULL') as schema_name,
			digest,
			COALESCE(digest_text, '') as digest_text,
			count_star,
			sum_timer_wait,
			min_timer_wait,
			avg_timer_wait,
			max_timer_wait,
			sum_rows_affected,
			sum_rows_sent,
			sum_rows_examined,
			sum_created_tmp_tables,
			sum_created_tmp_disk_tables,
			sum_select_full_join,
			sum_select_scan,
			sum_sort_scan,
			sum_sort_rows,
			sum_no_index_used,
			sum_no_good_index_used,
			first_seen,
			last_seen
		FROM performance_schema.events_statements_summary_by_digest 
		WHERE count_star >= ?
		  AND avg_timer_wait >= ?
		ORDER BY sum_timer_wait DESC
		LIMIT ?`

	minLatencyNanos := int64(p.config.MinAvgLatency * 1000000) // Convert ms to nanoseconds
	
	rows, err := p.db.QueryContext(ctx, query, 
		p.config.MinExecutionCount,
		minLatencyNanos,
		p.config.MaxStatements)
	if err != nil {
		return nil, fmt.Errorf("failed to query statement statistics: %w", err)
	}
	defer rows.Close()

	var statements []StatementStatistic
	for rows.Next() {
		var stmt StatementStatistic
		var digestText sql.NullString
		
		err := rows.Scan(
			&stmt.SchemaName,
			&stmt.Digest,
			&digestText,
			&stmt.CountStar,
			&stmt.SumTimerWait,
			&stmt.MinTimerWait,
			&stmt.AvgTimerWait,
			&stmt.MaxTimerWait,
			&stmt.SumRowsAffected,
			&stmt.SumRowsSent,
			&stmt.SumRowsExamined,
			&stmt.SumCreatedTmpTables,
			&stmt.SumCreatedTmpDiskTables,
			&stmt.SumSelectFullJoin,
			&stmt.SumSelectScan,
			&stmt.SumSortScan,
			&stmt.SumSortRows,
			&stmt.SumNoIndexUsed,
			&stmt.SumNoGoodIndexUsed,
			&stmt.FirstSeen,
			&stmt.LastSeen,
		)
		
		if err != nil {
			p.logger.WithError(err).Debug("Failed to scan statement row")
			continue
		}

		if digestText.Valid {
			stmt.DigestText = digestText.String
		}

		// Skip ignored schemas
		if p.shouldIgnoreSchema(stmt.SchemaName) {
			continue
		}

		statements = append(statements, stmt)
	}

	return statements, nil
}

func (p *PerformanceSchemaAdapter) collectTableIOStats(ctx context.Context) ([]TableIOStatistic, error) {
	query := `
		SELECT 
			object_schema,
			object_name,
			count_read,
			sum_timer_read,
			count_write,
			sum_timer_write,
			count_fetch,
			sum_timer_fetch,
			count_insert,
			sum_timer_insert,
			count_update,
			sum_timer_update,
			count_delete,
			sum_timer_delete
		FROM performance_schema.table_io_waits_summary_by_table
		WHERE object_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
		  AND (count_read > 0 OR count_write > 0)
		ORDER BY (sum_timer_read + sum_timer_write) DESC
		LIMIT ?`

	rows, err := p.db.QueryContext(ctx, query, p.config.MaxTables)
	if err != nil {
		return nil, fmt.Errorf("failed to query table I/O statistics: %w", err)
	}
	defer rows.Close()

	var tableStats []TableIOStatistic
	for rows.Next() {
		var stat TableIOStatistic
		
		err := rows.Scan(
			&stat.SchemaName,
			&stat.TableName,
			&stat.CountRead,
			&stat.SumTimerRead,
			&stat.CountWrite,
			&stat.SumTimerWrite,
			&stat.CountFetch,
			&stat.SumTimerFetch,
			&stat.CountInsert,
			&stat.SumTimerInsert,
			&stat.CountUpdate,
			&stat.SumTimerUpdate,
			&stat.CountDelete,
			&stat.SumTimerDelete,
		)
		
		if err != nil {
			p.logger.WithError(err).Debug("Failed to scan table I/O row")
			continue
		}

		if p.shouldIgnoreSchema(stat.SchemaName) {
			continue
		}

		tableStats = append(tableStats, stat)
	}

	return tableStats, nil
}

func (p *PerformanceSchemaAdapter) collectIndexStats(ctx context.Context) ([]IndexStatistic, error) {
	// Implementation for index statistics collection
	// This would query performance_schema.table_io_waits_summary_by_index_usage
	return []IndexStatistic{}, nil // Placeholder
}

func (p *PerformanceSchemaAdapter) collectWaitEventStats(ctx context.Context) ([]WaitEventStatistic, error) {
	// Implementation for wait event statistics
	// This would query performance_schema.events_waits_summary_global_by_event_name
	return []WaitEventStatistic{}, nil // Placeholder
}

func (p *PerformanceSchemaAdapter) collectConnectionStats(ctx context.Context) (*ConnectionStatistics, error) {
	// Implementation for connection statistics
	return &ConnectionStatistics{}, nil // Placeholder
}

func (p *PerformanceSchemaAdapter) collectReplicationStats(ctx context.Context) (*ReplicationStatistics, error) {
	// Implementation for replication statistics
	return &ReplicationStatistics{}, nil // Placeholder
}

func (p *PerformanceSchemaAdapter) collectSlowQueries(ctx context.Context) ([]SlowQueryInfo, error) {
	// Implementation for slow query collection from performance_schema.events_statements_history_long
	return []SlowQueryInfo{}, nil // Placeholder
}

// Helper methods

func (p *PerformanceSchemaAdapter) shouldIgnoreSchema(schema string) bool {
	for _, ignored := range p.config.IgnoredSchemas {
		if schema == ignored {
			return true
		}
	}
	return false
}

func (p *PerformanceSchemaAdapter) extractTableNames(digestText string) []string {
	// Simple table name extraction from SQL text
	// This is a basic implementation - could be enhanced with proper SQL parsing
	tables := make([]string, 0)
	
	if digestText == "" {
		return tables
	}

	// Look for FROM clauses and JOIN clauses
	lowerText := strings.ToLower(digestText)
	words := strings.Fields(lowerText)
	
	for i, word := range words {
		if word == "from" || word == "join" || word == "update" || word == "into" {
			if i+1 < len(words) && !strings.Contains(words[i+1], "(") {
				tableName := strings.Trim(words[i+1], "`,")
				if tableName != "" && !strings.Contains(tableName, " ") {
					tables = append(tables, tableName)
				}
			}
		}
	}
	
	return tables
}

func (p *PerformanceSchemaAdapter) identifyQueryType(digestText string) string {
	if digestText == "" {
		return "UNKNOWN"
	}
	
	lowerText := strings.ToLower(strings.TrimSpace(digestText))
	
	if strings.HasPrefix(lowerText, "select") {
		return "SELECT"
	} else if strings.HasPrefix(lowerText, "insert") {
		return "INSERT"
	} else if strings.HasPrefix(lowerText, "update") {
		return "UPDATE"
	} else if strings.HasPrefix(lowerText, "delete") {
		return "DELETE"
	} else if strings.HasPrefix(lowerText, "create") {
		return "CREATE"
	} else if strings.HasPrefix(lowerText, "drop") {
		return "DROP"
	} else if strings.HasPrefix(lowerText, "alter") {
		return "ALTER"
	}
	
	return "OTHER"
}

func (p *PerformanceSchemaAdapter) determineRelationshipType(stmt StatementStatistic) string {
	if stmt.SumSelectFullJoin > 0 {
		return "FULL_JOIN"
	}
	if strings.Contains(strings.ToLower(stmt.DigestText), "join") {
		return "JOIN"
	}
	if strings.Contains(strings.ToLower(stmt.DigestText), "where") {
		return "FILTERED"
	}
	return "SINGLE_TABLE"
}

func (p *PerformanceSchemaAdapter) classifyPerformanceImpact(avgLatency time.Duration) string {
	latencyMs := float64(avgLatency.Milliseconds())
	
	if latencyMs < 10 {
		return "LOW"
	} else if latencyMs < 100 {
		return "MEDIUM"
	}
	return "HIGH"
}

// IsConnected returns the connection status
func (p *PerformanceSchemaAdapter) IsConnected() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.isConnected
}

// Close closes the adapter and cleans up resources
func (p *PerformanceSchemaAdapter) Close() error {
	p.queryCacheMux.Lock()
	defer p.queryCacheMux.Unlock()
	
	for _, stmt := range p.queryCache {
		stmt.Close()
	}
	p.queryCache = make(map[string]*sql.Stmt)
	
	return nil
}

// defaultPerformanceSchemaConfig returns default configuration
func defaultPerformanceSchemaConfig() *PerformanceSchemaConfig {
	return &PerformanceSchemaConfig{
		CollectionInterval:  30 * time.Second,
		SlowQueryThreshold:  1 * time.Second,
		MaxHistoryRetention: 1 * time.Hour,
		
		CollectStatements:   true,
		CollectTableIO:      true,
		CollectIndexUsage:   true,
		CollectWaitEvents:   true,
		CollectConnections:  true,
		CollectReplication:  false,
		
		MaxStatements:       100,
		MaxTables:          50,
		
		IgnoredSchemas:      []string{"mysql", "information_schema", "performance_schema", "sys"},
		IgnoredUsers:        []string{"root", "mysql.sys", "mysql.session"},
		FocusedTables:       []string{},
		
		EnableDigestText:    true,
		MinExecutionCount:   10,
		MinAvgLatency:       10.0, // 10ms
	}
}
