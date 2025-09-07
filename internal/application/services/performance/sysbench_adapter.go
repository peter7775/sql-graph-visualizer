package performance

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

// SysbenchAdapter implements BenchmarkToolPort for sysbench
type SysbenchAdapter struct {
	logger *logrus.Logger
	config *SysbenchConfig

	// Cached sysbench availability
	isAvailable bool
	version     string
}

// SysbenchConfig contains sysbench-specific configuration
type SysbenchConfig struct {
	// Sysbench binary path (empty means use PATH)
	BinaryPath string `yaml:"binary_path" json:"binary_path"`

	// Default test parameters
	DefaultTableSize  int           `yaml:"default_table_size" json:"default_table_size"`
	DefaultTables     int           `yaml:"default_tables" json:"default_tables"`
	DefaultWarmupTime time.Duration `yaml:"default_warmup_time" json:"default_warmup_time"`

	// MySQL-specific settings
	MySQLDefaults      MySQLSysbenchDefaults      `yaml:"mysql_defaults" json:"mysql_defaults"`
	PostgreSQLDefaults PostgreSQLSysbenchDefaults `yaml:"postgresql_defaults" json:"postgresql_defaults"`

	// Test scenarios configuration
	TestScenarios map[string]SysbenchScenario `yaml:"test_scenarios" json:"test_scenarios"`
}

// MySQLSysbenchDefaults contains MySQL-specific defaults
type MySQLSysbenchDefaults struct {
	Engine        string `yaml:"engine" json:"engine"`
	StorageEngine string `yaml:"storage_engine" json:"storage_engine"`
}

// PostgreSQLSysbenchDefaults contains PostgreSQL-specific defaults
type PostgreSQLSysbenchDefaults struct {
	Schema string `yaml:"schema" json:"schema"`
}

// SysbenchScenario defines a predefined test scenario
type SysbenchScenario struct {
	TestType     string            `yaml:"test_type" json:"test_type"`
	Description  string            `yaml:"description" json:"description"`
	TableSize    int               `yaml:"table_size" json:"table_size"`
	Tables       int               `yaml:"tables" json:"tables"`
	Duration     time.Duration     `yaml:"duration" json:"duration"`
	Threads      []int             `yaml:"threads" json:"threads"`
	WarmupTime   time.Duration     `yaml:"warmup_time" json:"warmup_time"`
	CustomParams map[string]string `yaml:"custom_params" json:"custom_params"`
}

// NewSysbenchAdapter creates a new sysbench adapter
func NewSysbenchAdapter(logger *logrus.Logger, config *SysbenchConfig) *SysbenchAdapter {
	if config == nil {
		config = defaultSysbenchConfig()
	}

	adapter := &SysbenchAdapter{
		logger: logger,
		config: config,
	}

	// Check availability during initialization
	adapter.checkAvailability()

	return adapter
}

// Execute runs a sysbench benchmark
func (s *SysbenchAdapter) Execute(ctx context.Context, config ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	if !s.isAvailable {
		return nil, fmt.Errorf("sysbench is not available")
	}

	startTime := time.Now()

	// Build sysbench command
	cmd, err := s.buildCommand(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"test_type": config.TestType,
		"threads":   config.Threads,
		"duration":  config.Duration,
		"command":   strings.Join(cmd.Args, " "),
	}).Info("Starting sysbench benchmark")

	// Execute sysbench
	output, err := s.executeCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("sysbench execution failed: %w", err)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Parse results
	metrics, queryResults, err := s.parseOutput(output, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sysbench output: %w", err)
	}

	result := &ports.BenchmarkResult{
		ToolName:     "sysbench",
		TestType:     config.TestType,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     duration,
		Metrics:      metrics,
		QueryResults: queryResults,
		RawOutput:    output,
		Status:       ports.BenchmarkStatusCompleted,
	}

	s.logger.WithFields(logrus.Fields{
		"duration":        duration,
		"queries_per_sec": metrics.QueriesPerSecond,
		"avg_latency":     metrics.AverageLatency,
		"95p_latency":     metrics.Percentile95,
	}).Info("Sysbench benchmark completed")

	return result, nil
}

// Validate checks if the benchmark configuration is valid for sysbench
func (s *SysbenchAdapter) Validate(config ports.BenchmarkConfig) error {
	// Check if test type is supported
	supportedTests := s.GetSupportedTests()
	isSupported := false
	for _, test := range supportedTests {
		if test == config.TestType {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("unsupported test type: %s", config.TestType)
	}

	// Validate database connection
	if config.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Validate numeric parameters
	if config.Threads <= 0 {
		return fmt.Errorf("threads must be positive")
	}

	if config.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}

	// Validate test-specific parameters
	switch config.TestType {
	case "oltp_read_write", "oltp_read_only", "oltp_write_only":
		if config.TableSize <= 0 {
			return fmt.Errorf("table_size must be positive for OLTP tests")
		}
		if config.Tables <= 0 {
			return fmt.Errorf("tables must be positive for OLTP tests")
		}
	}

	return nil
}

// GetSupportedTests returns list of supported sysbench test types
func (s *SysbenchAdapter) GetSupportedTests() []string {
	return []string{
		"oltp_read_write",
		"oltp_read_only",
		"oltp_write_only",
		"oltp_point_select",
		"oltp_insert",
		"oltp_update_index",
		"oltp_update_non_index",
		"oltp_delete",
		"select_random_points",
		"select_random_ranges",
		"bulk_insert",
	}
}

// IsAvailable checks if sysbench is available and configured
func (s *SysbenchAdapter) IsAvailable() bool {
	return s.isAvailable
}

// GetVersion returns the sysbench version
func (s *SysbenchAdapter) GetVersion() (string, error) {
	if !s.isAvailable {
		return "", fmt.Errorf("sysbench is not available")
	}
	return s.version, nil
}

// Private implementation methods

func (s *SysbenchAdapter) checkAvailability() {
	binaryPath := s.config.BinaryPath
	if binaryPath == "" {
		binaryPath = "sysbench"
	}

	// Check if sysbench binary exists
	// #nosec G204 - binaryPath is configured and validated
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		s.logger.WithError(err).Warn("Sysbench is not available")
		s.isAvailable = false
		return
	}

	// Parse version from output
	versionRegex := regexp.MustCompile(`sysbench\s+([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	matches := versionRegex.FindStringSubmatch(string(output))
	if len(matches) >= 2 {
		s.version = matches[1]
	} else {
		s.version = "unknown"
	}

	s.isAvailable = true
	s.logger.WithField("version", s.version).Info("Sysbench is available")
}

func (s *SysbenchAdapter) buildCommand(config ports.BenchmarkConfig) (*exec.Cmd, error) {
	binaryPath := s.config.BinaryPath
	if binaryPath == "" {
		binaryPath = "sysbench"
	}

	// Parse database URL for connection parameters
	dbParams, err := s.parseDatabaseURL(config.DatabaseURL, config.DatabaseType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Build command arguments
	args := []string{binaryPath}

	// Add database connection parameters
	args = append(args, dbParams...)

	// Add test-specific parameters
	switch config.TestType {
	case "oltp_read_write", "oltp_read_only", "oltp_write_only",
		"oltp_point_select", "oltp_insert", "oltp_update_index",
		"oltp_update_non_index", "oltp_delete":

		// Table configuration
		if config.Tables > 0 {
			args = append(args, fmt.Sprintf("--tables=%d", config.Tables))
		} else {
			args = append(args, fmt.Sprintf("--tables=%d", s.config.DefaultTables))
		}

		if config.TableSize > 0 {
			args = append(args, fmt.Sprintf("--table-size=%d", config.TableSize))
		} else {
			args = append(args, fmt.Sprintf("--table-size=%d", s.config.DefaultTableSize))
		}

		// Database-specific parameters
		if config.DatabaseType == "mysql" {
			if s.config.MySQLDefaults.Engine != "" {
				args = append(args, fmt.Sprintf("--mysql-engine=%s", s.config.MySQLDefaults.Engine))
			}
			if s.config.MySQLDefaults.StorageEngine != "" {
				args = append(args, fmt.Sprintf("--mysql-storage-engine=%s", s.config.MySQLDefaults.StorageEngine))
			}
		} else if config.DatabaseType == "postgresql" {
			if s.config.PostgreSQLDefaults.Schema != "" {
				args = append(args, fmt.Sprintf("--pgsql-schema=%s", s.config.PostgreSQLDefaults.Schema))
			}
		}
	}

	// Add common parameters
	args = append(args, fmt.Sprintf("--threads=%d", config.Threads))
	args = append(args, fmt.Sprintf("--time=%d", int(config.Duration.Seconds())))

	// Add warmup if specified
	if config.WarmupTime > 0 {
		args = append(args, fmt.Sprintf("--warmup-time=%d", int(config.WarmupTime.Seconds())))
	} else if s.config.DefaultWarmupTime > 0 {
		args = append(args, fmt.Sprintf("--warmup-time=%d", int(s.config.DefaultWarmupTime.Seconds())))
	}

	// Add custom parameters
	for key, value := range config.CustomParams {
		if strValue, ok := value.(string); ok {
			args = append(args, fmt.Sprintf("--%s=%s", key, strValue))
		}
	}

	// Add test type and run command
	args = append(args, config.TestType, "run")

	// #nosec G204 - args are validated and constructed internally
	return exec.Command(args[0], args[1:]...), nil
}

func (s *SysbenchAdapter) parseDatabaseURL(dbURL, dbType string) ([]string, error) {
	args := make([]string, 0)

	// Simple URL parsing for MySQL and PostgreSQL
	// Format: [db_type]://[user]:[pass]@[host]:[port]/[database]

	// Remove protocol prefix
	if strings.Contains(dbURL, "://") {
		parts := strings.SplitN(dbURL, "://", 2)
		if len(parts) == 2 {
			dbURL = parts[1]
		}
	}

	// Parse user:password@host:port/database
	var userPass, hostPortDB string
	if strings.Contains(dbURL, "@") {
		parts := strings.SplitN(dbURL, "@", 2)
		userPass = parts[0]
		hostPortDB = parts[1]
	} else {
		hostPortDB = dbURL
	}

	// Parse user:password
	if userPass != "" {
		if strings.Contains(userPass, ":") {
			parts := strings.SplitN(userPass, ":", 2)
			args = append(args, fmt.Sprintf("--db-user=%s", parts[0]))
			args = append(args, fmt.Sprintf("--db-password=%s", parts[1]))
		} else {
			args = append(args, fmt.Sprintf("--db-user=%s", userPass))
		}
	}

	// Parse host:port/database
	var hostPort, database string
	if strings.Contains(hostPortDB, "/") {
		parts := strings.SplitN(hostPortDB, "/", 2)
		hostPort = parts[0]
		database = parts[1]
	} else {
		hostPort = hostPortDB
	}

	// Parse host:port
	if strings.Contains(hostPort, ":") {
		parts := strings.SplitN(hostPort, ":", 2)
		args = append(args, fmt.Sprintf("--db-host=%s", parts[0]))
		args = append(args, fmt.Sprintf("--db-port=%s", parts[1]))
	} else if hostPort != "" {
		args = append(args, fmt.Sprintf("--db-host=%s", hostPort))
	}

	// Add database name
	if database != "" {
		args = append(args, fmt.Sprintf("--db-name=%s", database))
	}

	// Add database driver
	if dbType == "mysql" {
		args = append(args, "--db-driver=mysql")
	} else if dbType == "postgresql" {
		args = append(args, "--db-driver=pgsql")
	}

	return args, nil
}

func (s *SysbenchAdapter) executeCommand(ctx context.Context, cmd *exec.Cmd) (string, error) {
	// Set context for command cancellation
	// #nosec G204 - cmd.Args are validated and constructed internally
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)

	// Execute command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"command": strings.Join(cmd.Args, " "),
			"output":  string(output),
			"error":   err.Error(),
		}).Error("Sysbench command failed")

		return string(output), err
	}

	return string(output), nil
}

func (s *SysbenchAdapter) parseOutput(output string, config ports.BenchmarkConfig) (*ports.PerformanceMetrics, []ports.QueryPerformance, error) {
	lines := strings.Split(output, "\n")

	metrics := &ports.PerformanceMetrics{}
	queryResults := make([]ports.QueryPerformance, 0)

	// Parse metrics from sysbench output
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse queries per second
		if strings.Contains(line, "queries/sec") {
			if qps := s.extractFloat(line, `([0-9]+\.?[0-9]*)\s*queries/sec`); qps > 0 {
				metrics.QueriesPerSecond = qps
			}
		}

		// Parse transactions per second
		if strings.Contains(line, "transactions/sec") {
			if tps := s.extractFloat(line, `([0-9]+\.?[0-9]*)\s*transactions/sec`); tps > 0 {
				metrics.TransactionsPerSec = tps
			}
		}

		// Parse latency metrics
		if strings.Contains(line, "avg:") {
			if avg := s.extractFloat(line, `avg:\s*([0-9]+\.?[0-9]*)`); avg > 0 {
				metrics.AverageLatency = avg
			}
		}

		if strings.Contains(line, "min:") {
			if min := s.extractFloat(line, `min:\s*([0-9]+\.?[0-9]*)`); min > 0 {
				metrics.MinLatency = min
			}
		}

		if strings.Contains(line, "max:") {
			if max := s.extractFloat(line, `max:\s*([0-9]+\.?[0-9]*)`); max > 0 {
				metrics.MaxLatency = max
			}
		}

		if strings.Contains(line, "95th percentile:") {
			if p95 := s.extractFloat(line, `95th percentile:\s*([0-9]+\.?[0-9]*)`); p95 > 0 {
				metrics.Percentile95 = p95
			}
		}

		if strings.Contains(line, "99th percentile:") {
			if p99 := s.extractFloat(line, `99th percentile:\s*([0-9]+\.?[0-9]*)`); p99 > 0 {
				metrics.Percentile99 = p99
			}
		}

		// Parse read/write statistics
		if strings.Contains(line, "reads/s:") {
			if reads := s.extractFloat(line, `reads/s:\s*([0-9]+\.?[0-9]*)`); reads > 0 {
				metrics.ReadQPS = reads
			}
		}

		if strings.Contains(line, "writes/s:") {
			if writes := s.extractFloat(line, `writes/s:\s*([0-9]+\.?[0-9]*)`); writes > 0 {
				metrics.WriteQPS = writes
			}
		}

		// Parse error information
		if strings.Contains(line, "errors/s:") {
			if errors := s.extractFloat(line, `errors/s:\s*([0-9]+\.?[0-9]*)`); errors > 0 {
				metrics.ErrorRate = errors
			}
		}
	}

	// Create synthetic query performance data based on test type
	queryResults = s.createQueryPerformanceData(config, metrics)

	return metrics, queryResults, nil
}

func (s *SysbenchAdapter) extractFloat(text, pattern string) float64 {
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	if len(matches) >= 2 {
		if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return value
		}
	}
	return 0
}

func (s *SysbenchAdapter) createQueryPerformanceData(config ports.BenchmarkConfig, metrics *ports.PerformanceMetrics) []ports.QueryPerformance {
	queryResults := make([]ports.QueryPerformance, 0)

	// Create synthetic query performance data based on sysbench test type
	switch config.TestType {
	case "oltp_read_write":
		// Mixed read/write workload
		queryResults = append(queryResults,
			s.createQueryPerformance("SELECT", "sbtest", metrics, 0.6),
			s.createQueryPerformance("UPDATE", "sbtest", metrics, 0.2),
			s.createQueryPerformance("INSERT", "sbtest", metrics, 0.1),
			s.createQueryPerformance("DELETE", "sbtest", metrics, 0.1),
		)

	case "oltp_read_only":
		// Read-only workload
		queryResults = append(queryResults,
			s.createQueryPerformance("SELECT", "sbtest", metrics, 1.0),
		)

	case "oltp_write_only":
		// Write-only workload
		queryResults = append(queryResults,
			s.createQueryPerformance("UPDATE", "sbtest", metrics, 0.5),
			s.createQueryPerformance("INSERT", "sbtest", metrics, 0.3),
			s.createQueryPerformance("DELETE", "sbtest", metrics, 0.2),
		)

	case "oltp_point_select":
		queryResults = append(queryResults,
			s.createQueryPerformance("SELECT", "sbtest", metrics, 1.0),
		)

	default:
		// Generic query performance
		queryResults = append(queryResults,
			s.createQueryPerformance("MIXED", "sbtest", metrics, 1.0),
		)
	}

	return queryResults
}

func (s *SysbenchAdapter) createQueryPerformance(queryType, tableName string, metrics *ports.PerformanceMetrics, weight float64) ports.QueryPerformance {
	// Calculate query-specific metrics based on weight
	execCount := int64(metrics.QueriesPerSecond * float64(60) * weight) // Assuming 60 second test
	avgTime := time.Duration(metrics.AverageLatency*weight) * time.Millisecond

	return ports.QueryPerformance{
		QueryPattern:      fmt.Sprintf("%s operation on %s", queryType, tableName),
		QueryType:         queryType,
		ExecutionCount:    execCount,
		TotalTime:         avgTime * time.Duration(execCount),
		AverageTime:       avgTime,
		MinTime:           time.Duration(metrics.MinLatency*weight) * time.Millisecond,
		MaxTime:           time.Duration(metrics.MaxLatency*weight) * time.Millisecond,
		SourceTables:      []string{tableName},
		RowsExamined:      execCount * 1, // Assume 1 row per query on average
		RowsReturned:      execCount * 1,
		IndexUsed:         true, // Sysbench typically uses indexes
		IndexName:         "PRIMARY",
		RelationshipType:  "SINGLE_TABLE",
		PerformanceImpact: s.classifyPerformanceImpact(metrics.AverageLatency * weight),
	}
}

func (s *SysbenchAdapter) classifyPerformanceImpact(avgLatency float64) string {
	if avgLatency < 10 {
		return "LOW"
	} else if avgLatency < 100 {
		return "MEDIUM"
	}
	return "HIGH"
}

// defaultSysbenchConfig returns default sysbench configuration
func defaultSysbenchConfig() *SysbenchConfig {
	return &SysbenchConfig{
		BinaryPath:        "", // Use system PATH
		DefaultTableSize:  100000,
		DefaultTables:     4,
		DefaultWarmupTime: 10 * time.Second,
		MySQLDefaults: MySQLSysbenchDefaults{
			Engine:        "innodb",
			StorageEngine: "innodb",
		},
		PostgreSQLDefaults: PostgreSQLSysbenchDefaults{
			Schema: "public",
		},
		TestScenarios: map[string]SysbenchScenario{
			"quick_test": {
				TestType:    "oltp_read_write",
				Description: "Quick mixed workload test",
				TableSize:   10000,
				Tables:      2,
				Duration:    30 * time.Second,
				Threads:     []int{1, 2, 4},
				WarmupTime:  5 * time.Second,
			},
			"standard_test": {
				TestType:    "oltp_read_write",
				Description: "Standard mixed workload test",
				TableSize:   100000,
				Tables:      4,
				Duration:    2 * time.Minute,
				Threads:     []int{1, 2, 4, 8, 16},
				WarmupTime:  10 * time.Second,
			},
			"heavy_load": {
				TestType:    "oltp_read_write",
				Description: "Heavy load test",
				TableSize:   1000000,
				Tables:      8,
				Duration:    5 * time.Minute,
				Threads:     []int{8, 16, 32, 64},
				WarmupTime:  30 * time.Second,
			},
		},
	}
}
