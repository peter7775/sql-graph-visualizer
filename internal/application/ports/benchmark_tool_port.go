package ports

import (
	"context"
	"time"

	"sql-graph-visualizer/internal/domain/models"
)

// BenchmarkToolPort defines the interface for benchmark tool integrations
type BenchmarkToolPort interface {
	// Execute runs a benchmark with the given configuration
	Execute(ctx context.Context, config BenchmarkConfig) (*BenchmarkResult, error)
	
	// Validate checks if the benchmark configuration is valid
	Validate(config BenchmarkConfig) error
	
	// GetSupportedTests returns list of supported test types
	GetSupportedTests() []string
	
	// IsAvailable checks if the benchmark tool is available and configured
	IsAvailable() bool
	
	// GetVersion returns the version of the benchmark tool
	GetVersion() (string, error)
}

// BenchmarkConfig represents configuration for benchmark execution
type BenchmarkConfig struct {
	// Tool-specific configuration
	TestType     string                 `json:"test_type" yaml:"test_type"`
	Duration     time.Duration         `json:"duration" yaml:"duration"`
	Threads      int                   `json:"threads" yaml:"threads"`
	TableSize    int                   `json:"table_size,omitempty" yaml:"table_size,omitempty"`
	Tables       int                   `json:"tables,omitempty" yaml:"tables,omitempty"`
	WarmupTime   time.Duration         `json:"warmup_time,omitempty" yaml:"warmup_time,omitempty"`
	
	// Database connection
	DatabaseType string                `json:"database_type" yaml:"database_type"`
	DatabaseURL  string                `json:"database_url" yaml:"database_url"`
	
	// Additional parameters
	CustomParams map[string]interface{} `json:"custom_params,omitempty" yaml:"custom_params,omitempty"`
}

// BenchmarkResult represents the result of a benchmark execution
type BenchmarkResult struct {
	// Execution metadata
	ID          string                 `json:"id"`
	ToolName    string                 `json:"tool_name"`
	TestType    string                 `json:"test_type"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time"`
	Duration    time.Duration         `json:"duration"`
	
	// Performance metrics
	Metrics     *PerformanceMetrics   `json:"metrics"`
	
	// Query-level results for graph mapping
	QueryResults []QueryPerformance    `json:"query_results,omitempty"`
	
	// Raw output from tool (for debugging)
	RawOutput   string                `json:"raw_output,omitempty"`
	
	// Status and errors
	Status      BenchmarkStatus       `json:"status"`
	Error       string                `json:"error,omitempty"`
}

// PerformanceMetrics contains aggregated performance data
type PerformanceMetrics struct {
	// Throughput metrics
	QueriesPerSecond    float64 `json:"queries_per_second"`
	TransactionsPerSec  float64 `json:"transactions_per_second"`
	ReadQPS            float64 `json:"read_qps"`
	WriteQPS           float64 `json:"write_qps"`
	
	// Latency metrics (in milliseconds)
	AverageLatency     float64 `json:"average_latency"`
	MinLatency         float64 `json:"min_latency"`
	MaxLatency         float64 `json:"max_latency"`
	Percentile95       float64 `json:"percentile_95"`
	Percentile99       float64 `json:"percentile_99"`
	
	// Resource utilization
	RowsRead           int64   `json:"rows_read"`
	RowsWritten        int64   `json:"rows_written"`
	DataTransferred    int64   `json:"data_transferred"`
	
	// Error metrics
	TotalErrors        int     `json:"total_errors"`
	ErrorRate          float64 `json:"error_rate"`
	Timeouts           int     `json:"timeouts"`
	
	// Database-specific metrics
	IndexHits          int64   `json:"index_hits,omitempty"`
	IndexMisses        int64   `json:"index_misses,omitempty"`
	TableScans         int64   `json:"table_scans,omitempty"`
	TemporaryTables    int     `json:"temporary_tables,omitempty"`
	FileSorts          int     `json:"file_sorts,omitempty"`
}

// QueryPerformance represents performance data for individual query patterns
type QueryPerformance struct {
	QueryPattern       string    `json:"query_pattern"`
	QueryType          string    `json:"query_type"` // SELECT, INSERT, UPDATE, DELETE
	ExecutionCount     int64     `json:"execution_count"`
	TotalTime          time.Duration `json:"total_time"`
	AverageTime        time.Duration `json:"average_time"`
	MinTime            time.Duration `json:"min_time"`
	MaxTime            time.Duration `json:"max_time"`
	
	// Tables involved in the query
	SourceTables       []string  `json:"source_tables"`
	JoinedTables       []string  `json:"joined_tables,omitempty"`
	
	// Performance characteristics
	RowsExamined       int64     `json:"rows_examined"`
	RowsReturned       int64     `json:"rows_returned"`
	IndexUsed          bool      `json:"index_used"`
	IndexName          string    `json:"index_name,omitempty"`
	
	// Resource usage
	CPUTime            time.Duration `json:"cpu_time,omitempty"`
	IOReads            int64     `json:"io_reads,omitempty"`
	IOWrites           int64     `json:"io_writes,omitempty"`
	
	// Classification for graph mapping
	RelationshipType   string    `json:"relationship_type,omitempty"` // JOIN, FK_LOOKUP, etc.
	PerformanceImpact  string    `json:"performance_impact"` // LOW, MEDIUM, HIGH
}

// BenchmarkStatus represents the status of benchmark execution
type BenchmarkStatus string

const (
	BenchmarkStatusPending   BenchmarkStatus = "pending"
	BenchmarkStatusRunning   BenchmarkStatus = "running" 
	BenchmarkStatusCompleted BenchmarkStatus = "completed"
	BenchmarkStatusFailed    BenchmarkStatus = "failed"
	BenchmarkStatusCancelled BenchmarkStatus = "cancelled"
)

// Custom benchmark configuration for user-defined queries
type CustomBenchmarkConfig struct {
	Name        string                    `json:"name" yaml:"name"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Duration    time.Duration            `json:"duration" yaml:"duration"`
	Threads     int                      `json:"threads" yaml:"threads"`
	Queries     []CustomQueryDefinition   `json:"queries" yaml:"queries"`
}

// CustomQueryDefinition defines a query for custom benchmarking
type CustomQueryDefinition struct {
	Query       string                 `json:"query" yaml:"query"`
	Weight      int                   `json:"weight" yaml:"weight"` // Relative frequency
	Parameters  []interface{}         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	
	// Performance expectations (optional)
	ExpectedLatency time.Duration     `json:"expected_latency,omitempty" yaml:"expected_latency,omitempty"`
	TargetQPS      float64           `json:"target_qps,omitempty" yaml:"target_qps,omitempty"`
}
