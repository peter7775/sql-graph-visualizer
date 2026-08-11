package performance

import (
	"context"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
)

// fakeBenchmarkTool is a minimal ports.BenchmarkToolPort implementation used
// to exercise BenchmarkService without depending on sysbench or a real
// database connection.
type fakeBenchmarkTool struct {
	available   bool
	executeFunc func(ctx context.Context, config ports.BenchmarkConfig) (*ports.BenchmarkResult, error)
	validateErr error
}

func (f *fakeBenchmarkTool) Execute(ctx context.Context, config ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	if f.executeFunc != nil {
		return f.executeFunc(ctx, config)
	}
	return &ports.BenchmarkResult{
		ToolName: "fake",
		TestType: config.TestType,
		Status:   ports.BenchmarkStatusCompleted,
		Metrics:  &ports.PerformanceMetrics{QueriesPerSecond: 42},
	}, nil
}

func (f *fakeBenchmarkTool) Validate(_ ports.BenchmarkConfig) error { return f.validateErr }
func (f *fakeBenchmarkTool) GetSupportedTests() []string            { return []string{"fake_test"} }
func (f *fakeBenchmarkTool) IsAvailable() bool                      { return f.available }
func (f *fakeBenchmarkTool) GetVersion() (string, error)            { return "fake/1.0", nil }

func newTestBenchmarkService(t *testing.T) *BenchmarkService {
	t.Helper()
	config := defaultBenchmarkServiceConfig()
	config.CleanupInterval = time.Hour // avoid the cleanup goroutine racing with test assertions
	return NewBenchmarkService(nil, nil, nil, nil, newTestLogger(), config)
}

func TestBenchmarkService_RegisterBenchmarkTool(t *testing.T) {
	svc := newTestBenchmarkService(t)

	if err := svc.RegisterBenchmarkTool("unavailable", &fakeBenchmarkTool{available: false}); err == nil {
		t.Error("RegisterBenchmarkTool() with unavailable tool: expected error, got nil")
	}

	if err := svc.RegisterBenchmarkTool("fake", &fakeBenchmarkTool{available: true}); err != nil {
		t.Fatalf("RegisterBenchmarkTool() error = %v", err)
	}

	tools := svc.GetAvailableTools()
	if len(tools) != 1 || tools[0] != "fake" {
		t.Errorf("GetAvailableTools() = %v, want [fake]", tools)
	}
}

func TestBenchmarkService_ApplyConfigDefaults(t *testing.T) {
	svc := newTestBenchmarkService(t)
	svc.config.DefaultTestType = "oltp_read_write"
	svc.config.DefaultDatabaseURL = "mysql://user:pass@localhost:3306/db"
	svc.config.DefaultDatabaseType = "mysql"
	svc.config.DefaultThreads = 4
	svc.config.DefaultTables = 2
	svc.config.DefaultTableSize = 1000
	svc.config.DefaultTestDuration = 30 * time.Second

	config := ports.BenchmarkConfig{}
	svc.applyConfigDefaults(&config)

	if config.TestType != "oltp_read_write" {
		t.Errorf("TestType = %q, want oltp_read_write", config.TestType)
	}
	if config.DatabaseURL != "mysql://user:pass@localhost:3306/db" {
		t.Errorf("DatabaseURL = %q, want default", config.DatabaseURL)
	}
	if config.Threads != 4 || config.Tables != 2 || config.TableSize != 1000 {
		t.Errorf("Threads/Tables/TableSize = %d/%d/%d, want 4/2/1000", config.Threads, config.Tables, config.TableSize)
	}
	if config.Duration != 30*time.Second {
		t.Errorf("Duration = %v, want 30s", config.Duration)
	}

	// Explicitly set fields must not be overridden.
	explicit := ports.BenchmarkConfig{TestType: "oltp_read_only", Threads: 16}
	svc.applyConfigDefaults(&explicit)
	if explicit.TestType != "oltp_read_only" || explicit.Threads != 16 {
		t.Errorf("applyConfigDefaults() overrode explicit values: %+v", explicit)
	}
}

func TestBenchmarkService_ValidateConfig(t *testing.T) {
	svc := newTestBenchmarkService(t)
	svc.config.MaxDuration = time.Minute
	svc.config.MaxThreads = 8
	svc.config.MaxTableSize = 1000

	tests := []struct {
		name    string
		config  ports.BenchmarkConfig
		wantErr bool
	}{
		{"within limits", ports.BenchmarkConfig{Duration: 30 * time.Second, Threads: 4, TableSize: 500}, false},
		{"duration exceeds max", ports.BenchmarkConfig{Duration: 2 * time.Minute, Threads: 4, TableSize: 500}, true},
		{"threads exceed max", ports.BenchmarkConfig{Duration: 30 * time.Second, Threads: 100, TableSize: 500}, true},
		{"table size exceeds max", ports.BenchmarkConfig{Duration: 30 * time.Second, Threads: 4, TableSize: 10000}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateConfig(tt.config)
			if tt.wantErr && err == nil {
				t.Error("validateConfig() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateConfig() unexpected error = %v", err)
			}
		})
	}
}

func TestBenchmarkService_ExecuteBenchmark_PersistsResult(t *testing.T) {
	svc := newTestBenchmarkService(t)
	if err := svc.RegisterBenchmarkTool("fake", &fakeBenchmarkTool{available: true}); err != nil {
		t.Fatalf("RegisterBenchmarkTool() error = %v", err)
	}

	dir := t.TempDir()
	store, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("NewFileBenchmarkResultStore() error = %v", err)
	}
	svc.SetResultStore(store)

	executionID, err := svc.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{
		TestType: "fake_test",
		Duration: time.Second,
		Threads:  1,
	}, "fake")
	if err != nil {
		t.Fatalf("ExecuteBenchmark() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		result, resultErr := svc.GetBenchmarkResult(executionID)
		if resultErr == nil && result.Status == ports.BenchmarkStatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	result, err := svc.GetBenchmarkResult(executionID)
	if err != nil {
		t.Fatalf("GetBenchmarkResult() error = %v", err)
	}
	if result.Status != ports.BenchmarkStatusCompleted {
		t.Fatalf("benchmark did not complete in time, status = %v", result.Status)
	}

	// The completed result should have been persisted to the store.
	history, err := svc.GetBenchmarkHistory(context.Background(), ports.BenchmarkResultFilter{})
	if err != nil {
		t.Fatalf("GetBenchmarkHistory() error = %v", err)
	}
	if len(history) != 1 || history[0].ID != executionID {
		t.Errorf("GetBenchmarkHistory() = %+v, want a single entry with ID %q", history, executionID)
	}
}

func TestBenchmarkService_GetBenchmarkHistory_NoStoreConfigured(t *testing.T) {
	svc := newTestBenchmarkService(t)
	if _, err := svc.GetBenchmarkHistory(context.Background(), ports.BenchmarkResultFilter{}); err == nil {
		t.Error("GetBenchmarkHistory() with no store configured: expected error, got nil")
	}
}

func TestBenchmarkService_ExecuteBenchmark_UnknownTool(t *testing.T) {
	svc := newTestBenchmarkService(t)
	if _, err := svc.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{
		TestType: "fake_test",
		Duration: time.Second,
		Threads:  1,
	}, "does-not-exist"); err == nil {
		t.Error("ExecuteBenchmark() with unregistered tool: expected error, got nil")
	}
}

func TestBenchmarkService_MaxConcurrentRuns(t *testing.T) {
	svc := newTestBenchmarkService(t)
	svc.config.MaxConcurrentRuns = 1

	block := make(chan struct{})
	defer close(block)

	tool := &fakeBenchmarkTool{
		available: true,
		executeFunc: func(ctx context.Context, _ ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
			select {
			case <-block:
			case <-ctx.Done():
			}
			return &ports.BenchmarkResult{Status: ports.BenchmarkStatusCompleted, Metrics: &ports.PerformanceMetrics{}}, nil
		},
	}
	if err := svc.RegisterBenchmarkTool("fake", tool); err != nil {
		t.Fatalf("RegisterBenchmarkTool() error = %v", err)
	}

	config := ports.BenchmarkConfig{TestType: "fake_test", Duration: 5 * time.Second, Threads: 1}
	if _, err := svc.ExecuteBenchmark(context.Background(), config, "fake"); err != nil {
		t.Fatalf("first ExecuteBenchmark() error = %v", err)
	}

	// Give the async goroutine a moment to register as "running".
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && svc.getActiveRunCount() == 0 {
		time.Sleep(5 * time.Millisecond)
	}

	if _, err := svc.ExecuteBenchmark(context.Background(), config, "fake"); err == nil {
		t.Error("ExecuteBenchmark() exceeding MaxConcurrentRuns: expected error, got nil")
	}
}
