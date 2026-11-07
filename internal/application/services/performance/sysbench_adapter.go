package performance

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
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

var sysbenchInterim = regexp.MustCompile(`^\[\s*([0-9.]+)s\s*\]\s+thds:\s*(\d+).*?\bqps:\s*([0-9.]+).*?lat \(ms,95%\):\s*([0-9.]+).*?err/s:\s*([0-9.]+)`)

func parseSysbenchInterim(line string, previousElapsed float64) (ports.LiveSample, bool) {
	m := sysbenchInterim.FindStringSubmatch(line)
	if m == nil {
		return ports.LiveSample{}, false
	}
	values := make([]float64, 5)
	for n := range values {
		value, err := strconv.ParseFloat(m[n+1], 64)
		if err != nil {
			return ports.LiveSample{}, false
		}
		values[n] = value
	}
	interval := values[0] - previousElapsed
	if interval <= 0 {
		return ports.LiveSample{}, false
	}
	return ports.LiveSample{
		Timestamp: time.Now(), ElapsedSeconds: values[0], Threads: int(values[1]),
		Interval: ports.LiveIntervalStats{Queries: int64(values[2]*interval + 0.5), QPS: values[2], P95LatencyMs: values[3], Errors: int64(values[4]*interval + 0.5)},
		Queries:  []ports.LiveQueryStats{}, Graph: ports.LiveGraph{Nodes: []ports.LiveGraphNode{}, Edges: []ports.LiveGraphEdge{}},
	}, true
}

// SysbenchConfig contains sysbench-specific configuration
type SysbenchConfig struct {
	ReportInterval int `yaml:"report_interval" json:"report_interval"`
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

	adapter.checkAvailability()

	return adapter
}

// Execute runs a sysbench benchmark
func (s *SysbenchAdapter) Execute(ctx context.Context, config ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	return s.ExecuteStream(ctx, config, nil)
}

// ExecuteStream runs sysbench and emits its measured interim statistics.
func (s *SysbenchAdapter) ExecuteStream(ctx context.Context, config ports.BenchmarkConfig, emit func(ports.LiveSample)) (*ports.BenchmarkResult, error) {
	if !s.isAvailable {
		return nil, fmt.Errorf("sysbench is not available")
	}

	startTime := time.Now()

	cmd, err := s.buildCommand(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build command: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"test_type": config.TestType,
		"threads":   config.Threads,
		"duration":  config.Duration,
	}).Info("Starting sysbench benchmark")

	output, err := s.executeStreamCommand(ctx, cmd, emit)
	if err != nil {
		return nil, fmt.Errorf("sysbench execution failed: %w", err)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

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

	dbParams, err := s.parseDatabaseURL(config.DatabaseURL, config.DatabaseType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	args := []string{binaryPath}

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
		switch config.DatabaseType {
		case "mysql":
			if s.config.MySQLDefaults.StorageEngine != "" {
				args = append(args, fmt.Sprintf("--mysql-storage-engine=%s", s.config.MySQLDefaults.StorageEngine))
			}
		case "postgresql":
			if schema := s.config.PostgreSQLDefaults.Schema; schema != "" && schema != "public" {
				return nil, fmt.Errorf("sysbench uses the PostgreSQL role search_path; configure that role for schema %q", schema)
			}
		}
	}

	args = append(args, fmt.Sprintf("--threads=%d", config.Threads))
	args = append(args, fmt.Sprintf("--time=%d", int(config.Duration.Seconds())))
	interval := s.config.ReportInterval
	if interval <= 0 {
		interval = 1
	}
	args = append(args, fmt.Sprintf("--report-interval=%d", interval))

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

	args = append(args, config.TestType, "run")

	// #nosec G204 - args are validated and constructed internally
	return exec.Command(args[0], args[1:]...), nil
}

func (s *SysbenchAdapter) parseDatabaseURL(dbURL, dbType string) ([]string, error) {
	prefix, driver := "mysql", "mysql"
	if dbType == "postgresql" {
		prefix, driver = "pgsql", "pgsql"
	} else if dbType != "mysql" && dbType != "" {
		return nil, fmt.Errorf("unsupported sysbench database type: %s", dbType)
	}
	if !strings.Contains(dbURL, "://") {
		dbURL = "db://" + dbURL
	}
	parsed, err := url.Parse(dbURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL")
	}
	args := []string{"--db-driver=" + driver}
	add := func(key, value string) {
		if value != "" {
			args = append(args, "--"+prefix+"-"+key+"="+value)
		}
	}
	if parsed.User != nil {
		add("user", parsed.User.Username())
		if password, exists := parsed.User.Password(); exists {
			add("password", password)
		}
	}
	add("host", parsed.Hostname())
	add("port", parsed.Port())
	add("db", strings.TrimPrefix(parsed.Path, "/"))
	if driver == "pgsql" {
		add("sslmode", parsed.Query().Get("sslmode"))
	}
	return args, nil
}

func (s *SysbenchAdapter) executeStreamCommand(ctx context.Context, cmd *exec.Cmd, emit func(ports.LiveSample)) (string, error) {
	// Set context for command cancellation
	// #nosec G204 - cmd.Args are validated and constructed internally
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)

	cmd.WaitDelay = time.Second
	var output, stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	previousElapsed := 0.0
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")
		if sample, ok := parseSysbenchInterim(line, previousElapsed); ok {
			previousElapsed = sample.ElapsedSeconds
			if emit != nil {
				emit(sample)
			}
		}
	}
	scanErr := scanner.Err()
	if scanErr != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return output.String(), ctx.Err()
	}
	if scanErr != nil {
		return output.String(), scanErr
	}
	if waitErr != nil {
		return output.String(), fmt.Errorf("%w: %s", waitErr, stderr.String())
	}
	return output.String(), nil
}

func (s *SysbenchAdapter) parseOutput(output string, _ ports.BenchmarkConfig) (*ports.PerformanceMetrics, []ports.QueryPerformance, error) {
	lines := strings.Split(output, "\n")

	metrics := &ports.PerformanceMetrics{}

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
			if minVal := s.extractFloat(line, `min:\s*([0-9]+\.?[0-9]*)`); minVal > 0 {
				metrics.MinLatency = minVal
			}
		}

		if strings.Contains(line, "max:") {
			if maxVal := s.extractFloat(line, `max:\s*([0-9]+\.?[0-9]*)`); maxVal > 0 {
				metrics.MaxLatency = maxVal
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

	queryResults := []ports.QueryPerformance{}

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

// DefaultSysbenchConfig returns the default sysbench adapter configuration.
// Exported so callers (e.g. bootstrap) can start from sane defaults and
// override individual fields from user configuration.
func DefaultSysbenchConfig() *SysbenchConfig {
	return defaultSysbenchConfig()
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
