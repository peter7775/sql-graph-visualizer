package performance

import (
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
)

func TestSysbenchAdapter_ExtractFloat(t *testing.T) {
	adapter := &SysbenchAdapter{logger: newTestLogger()}

	tests := []struct {
		text    string
		pattern string
		want    float64
	}{
		{"queries: 1234.56 queries/sec", `([0-9]+\.?[0-9]*)\s*queries/sec`, 1234.56},
		{"avg: 12.34", `avg:\s*([0-9]+\.?[0-9]*)`, 12.34},
		{"no match here", `avg:\s*([0-9]+\.?[0-9]*)`, 0},
	}

	for _, tt := range tests {
		if got := adapter.extractFloat(tt.text, tt.pattern); got != tt.want {
			t.Errorf("extractFloat(%q, %q) = %v, want %v", tt.text, tt.pattern, got, tt.want)
		}
	}
}

func TestSysbenchAdapter_ParseOutput(t *testing.T) {
	adapter := &SysbenchAdapter{logger: newTestLogger()}
	output := `
SQL statistics:
    queries performed:
        read:                            140000
        write:                           40000
        total:                           180000
    transactions:                        18000  (300.00 transactions/sec)
    queries:                             180000 (3000.00 queries/sec)
    ignored errors:                      0      (0.00 errors/s)

Latency (ms):
         min:                                    1.20
         avg:                                    5.43
         max:                                   87.10
         95th percentile:                       12.50
         99th percentile:                       45.00
`
	config := ports.BenchmarkConfig{TestType: "oltp_read_write"}
	metrics, queryResults, err := adapter.parseOutput(output, config)
	if err != nil {
		t.Fatalf("parseOutput() error = %v", err)
	}

	if metrics.QueriesPerSecond != 3000.00 {
		t.Errorf("QueriesPerSecond = %v, want 3000.00", metrics.QueriesPerSecond)
	}
	if metrics.TransactionsPerSec != 300.00 {
		t.Errorf("TransactionsPerSec = %v, want 300.00", metrics.TransactionsPerSec)
	}
	if metrics.AverageLatency != 5.43 {
		t.Errorf("AverageLatency = %v, want 5.43", metrics.AverageLatency)
	}
	if metrics.MinLatency != 1.20 {
		t.Errorf("MinLatency = %v, want 1.20", metrics.MinLatency)
	}
	if metrics.MaxLatency != 87.10 {
		t.Errorf("MaxLatency = %v, want 87.10", metrics.MaxLatency)
	}
	if metrics.Percentile95 != 12.50 {
		t.Errorf("Percentile95 = %v, want 12.50", metrics.Percentile95)
	}
	if metrics.Percentile99 != 45.00 {
		t.Errorf("Percentile99 = %v, want 45.00", metrics.Percentile99)
	}

	if len(queryResults) == 0 {
		t.Error("parseOutput() returned no query results for oltp_read_write")
	}
}

func TestSysbenchAdapter_GetSupportedTests(t *testing.T) {
	adapter := &SysbenchAdapter{logger: newTestLogger()}
	tests := adapter.GetSupportedTests()
	if len(tests) == 0 {
		t.Fatal("GetSupportedTests() returned no tests")
	}
	found := false
	for _, tt := range tests {
		if tt == "oltp_read_write" {
			found = true
		}
	}
	if !found {
		t.Errorf("GetSupportedTests() = %v, want it to include oltp_read_write", tests)
	}
}

func TestSysbenchAdapter_Validate(t *testing.T) {
	adapter := &SysbenchAdapter{logger: newTestLogger()}

	tests := []struct {
		name    string
		config  ports.BenchmarkConfig
		wantErr bool
	}{
		{
			name: "valid oltp config",
			config: ports.BenchmarkConfig{
				TestType: "oltp_read_write", DatabaseURL: "mysql://u:p@h:3306/db",
				Threads: 4, Duration: 30 * time.Second, TableSize: 1000, Tables: 2,
			},
			wantErr: false,
		},
		{"unsupported test type", ports.BenchmarkConfig{TestType: "not_a_real_test", DatabaseURL: "x", Threads: 1, Duration: time.Second}, true},
		{"missing database url", ports.BenchmarkConfig{TestType: "oltp_read_only", Threads: 1, Duration: time.Second}, true},
		{"non-positive threads", ports.BenchmarkConfig{TestType: "oltp_read_only", DatabaseURL: "x", Threads: 0, Duration: time.Second}, true},
		{"non-positive duration", ports.BenchmarkConfig{TestType: "oltp_read_only", DatabaseURL: "x", Threads: 1, Duration: 0}, true},
		{
			name:    "oltp test missing table size",
			config:  ports.BenchmarkConfig{TestType: "oltp_read_write", DatabaseURL: "x", Threads: 1, Duration: time.Second, Tables: 2},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.Validate(tt.config)
			if tt.wantErr && err == nil {
				t.Error("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestSysbenchAdapter_ParseDatabaseURL(t *testing.T) {
	adapter := &SysbenchAdapter{logger: newTestLogger()}

	args, err := adapter.parseDatabaseURL("mysql://user:pass@localhost:3306/mydb", "mysql")
	if err != nil {
		t.Fatalf("parseDatabaseURL() error = %v", err)
	}

	want := map[string]bool{
		"--db-user=user":      true,
		"--db-password=pass":  true,
		"--db-host=localhost": true,
		"--db-port=3306":      true,
		"--db-name=mydb":      true,
		"--db-driver=mysql":   true,
	}
	if len(args) != len(want) {
		t.Errorf("parseDatabaseURL() returned %d args, want %d: %v", len(args), len(want), args)
	}
	for _, arg := range args {
		if !want[arg] {
			t.Errorf("parseDatabaseURL() produced unexpected arg %q", arg)
		}
	}
}

func TestSysbenchAdapter_IsAvailableAndVersion_WhenBinaryMissing(t *testing.T) {
	adapter := NewSysbenchAdapter(newTestLogger(), &SysbenchConfig{BinaryPath: "/nonexistent/sysbench-binary"})
	if adapter.IsAvailable() {
		t.Error("IsAvailable() with nonexistent binary: want false, got true")
	}
	if _, err := adapter.GetVersion(); err == nil {
		t.Error("GetVersion() with unavailable adapter: expected error, got nil")
	}
}
