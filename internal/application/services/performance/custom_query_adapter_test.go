package performance

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	_ "github.com/go-sql-driver/mysql"
)

// fakeDB returns a *sql.DB that is valid to construct (sql.Open does not
// connect eagerly) but is never actually used to run queries in these tests.
func fakeDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/testdb")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestIsAllowedCustomQuery(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"SELECT * FROM users", true},
		{"  select id from users", true},
		{"INSERT INTO logs (msg) VALUES ('x')", true},
		{"update users set name = 'x' where id = 1", true},
		{"DELETE FROM users", false},
		{"DROP TABLE users", false},
		{"TRUNCATE TABLE users", false},
		{"", false},
		{"   ", false},
	}
	for _, tt := range tests {
		if got := isAllowedCustomQuery(tt.query); got != tt.want {
			t.Errorf("isAllowedCustomQuery(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestCustomQueryAdapter_ResolveQuerySet(t *testing.T) {
	single := map[string]ports.CustomBenchmarkConfig{
		"only_set": {Name: "only_set", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
	}
	multi := map[string]ports.CustomBenchmarkConfig{
		"set_a": {Name: "set_a", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
		"set_b": {Name: "set_b", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 2"}}},
	}

	t.Run("falls back to sole configured set", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), nil, single)
		set, name, err := adapter.resolveQuerySet(ports.BenchmarkConfig{})
		if err != nil {
			t.Fatalf("resolveQuerySet() error = %v", err)
		}
		if name != "only_set" || len(set.Queries) != 1 {
			t.Errorf("resolveQuerySet() = (%+v, %q), want only_set", set, name)
		}
	})

	t.Run("requires explicit query_set when multiple sets configured", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), nil, multi)
		if _, _, err := adapter.resolveQuerySet(ports.BenchmarkConfig{}); err == nil {
			t.Error("resolveQuerySet() with no query_set and multiple sets: expected error, got nil")
		}
	})

	t.Run("resolves named query_set", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), nil, multi)
		set, name, err := adapter.resolveQuerySet(ports.BenchmarkConfig{
			CustomParams: map[string]interface{}{"query_set": "set_b"},
		})
		if err != nil {
			t.Fatalf("resolveQuerySet() error = %v", err)
		}
		if name != "set_b" || set.Queries[0].Query != "SELECT 2" {
			t.Errorf("resolveQuerySet() = (%+v, %q), want set_b", set, name)
		}
	})

	t.Run("unknown query_set errors", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), nil, multi)
		if _, _, err := adapter.resolveQuerySet(ports.BenchmarkConfig{
			CustomParams: map[string]interface{}{"query_set": "does_not_exist"},
		}); err == nil {
			t.Error("resolveQuerySet() with unknown name: expected error, got nil")
		}
	})
}

func TestCustomQueryAdapter_Validate(t *testing.T) {
	db := fakeDB(t)

	t.Run("rejects empty query set", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), db, map[string]ports.CustomBenchmarkConfig{
			"empty": {Name: "empty"},
		})
		if err := adapter.Validate(ports.BenchmarkConfig{CustomParams: map[string]interface{}{"query_set": "empty"}}); err == nil {
			t.Error("Validate() with no queries: expected error, got nil")
		}
	})

	t.Run("rejects disallowed statement type", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), db, map[string]ports.CustomBenchmarkConfig{
			"dangerous": {Name: "dangerous", Queries: []ports.CustomQueryDefinition{{Query: "DELETE FROM users"}}},
		})
		if err := adapter.Validate(ports.BenchmarkConfig{CustomParams: map[string]interface{}{"query_set": "dangerous"}}); err == nil {
			t.Error("Validate() with DELETE statement: expected error, got nil")
		}
	})

	t.Run("rejects missing database connection", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), nil, map[string]ports.CustomBenchmarkConfig{
			"ok": {Name: "ok", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
		})
		if err := adapter.Validate(ports.BenchmarkConfig{CustomParams: map[string]interface{}{"query_set": "ok"}}); err == nil {
			t.Error("Validate() with nil db: expected error, got nil")
		}
	})

	t.Run("accepts a well-formed query set", func(t *testing.T) {
		adapter := NewCustomQueryAdapter(newTestLogger(), db, map[string]ports.CustomBenchmarkConfig{
			"ok": {Name: "ok", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
		})
		if err := adapter.Validate(ports.BenchmarkConfig{CustomParams: map[string]interface{}{"query_set": "ok"}}); err != nil {
			t.Errorf("Validate() unexpected error = %v", err)
		}
	})
}

func TestCustomQueryAdapter_IsAvailable(t *testing.T) {
	db := fakeDB(t)

	if adapter := NewCustomQueryAdapter(newTestLogger(), nil, map[string]ports.CustomBenchmarkConfig{
		"ok": {Name: "ok", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
	}); adapter.IsAvailable() {
		t.Error("IsAvailable() with nil db: want false, got true")
	}

	if adapter := NewCustomQueryAdapter(newTestLogger(), db, map[string]ports.CustomBenchmarkConfig{}); adapter.IsAvailable() {
		t.Error("IsAvailable() with no query sets: want false, got true")
	}

	adapter := NewCustomQueryAdapter(newTestLogger(), db, map[string]ports.CustomBenchmarkConfig{
		"ok": {Name: "ok", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
	})
	if !adapter.IsAvailable() {
		t.Error("IsAvailable() with configured db and query set: want true, got false")
	}
}

func TestPickWeightedIndex(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))

	if got := pickWeightedIndex(rnd, nil, 0); got != 0 {
		t.Errorf("pickWeightedIndex(empty) = %d, want 0", got)
	}

	if got := pickWeightedIndex(rnd, []int{5}, 5); got != 0 {
		t.Errorf("pickWeightedIndex(single) = %d, want 0", got)
	}

	// With a zero/negative total weight, selection should still return a
	// valid index within bounds rather than panicking.
	weights := []int{1, 1, 1}
	for i := 0; i < 20; i++ {
		got := pickWeightedIndex(rnd, weights, 0)
		if got < 0 || got >= len(weights) {
			t.Fatalf("pickWeightedIndex() = %d out of bounds for weights %v", got, weights)
		}
	}

	// Heavily weighted first index should be picked far more often than the
	// second across many trials.
	heavy := []int{99, 1}
	firstCount := 0
	trials := 1000
	for i := 0; i < trials; i++ {
		if pickWeightedIndex(rnd, heavy, 100) == 0 {
			firstCount++
		}
	}
	if firstCount < trials*8/10 {
		t.Errorf("pickWeightedIndex() picked index 0 only %d/%d times with weight 99:1, expected it to dominate", firstCount, trials)
	}
}

func TestAggregateCustomQueryStats(t *testing.T) {
	stats := []*customQueryStat{
		{
			def:          ports.CustomQueryDefinition{Query: "SELECT * FROM users"},
			execCount:    10,
			totalTime:    100 * time.Millisecond,
			minTime:      5 * time.Millisecond,
			maxTime:      20 * time.Millisecond,
			rowsExamined: 100,
			rowsReturned: 100,
		},
		{
			def:        ports.CustomQueryDefinition{Query: "INSERT INTO logs VALUES (1)"},
			execCount:  5,
			errorCount: 1,
			totalTime:  50 * time.Millisecond,
		},
		{
			// A query that never executed should be skipped entirely.
			def: ports.CustomQueryDefinition{Query: "SELECT * FROM never_ran"},
		},
	}

	metrics, queryResults := aggregateCustomQueryStats(stats, 1*time.Second)

	if len(queryResults) != 2 {
		t.Fatalf("aggregateCustomQueryStats() returned %d query results, want 2", len(queryResults))
	}
	if metrics.QueriesPerSecond != 15 {
		t.Errorf("QueriesPerSecond = %v, want 15", metrics.QueriesPerSecond)
	}
	if metrics.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", metrics.TotalErrors)
	}
	wantErrorRate := (1.0 / 15.0) * 100
	if diff := metrics.ErrorRate - wantErrorRate; diff > 0.01 || diff < -0.01 {
		t.Errorf("ErrorRate = %v, want ~%v", metrics.ErrorRate, wantErrorRate)
	}
}

func TestCustomQueryAdapter_ExecuteUnknownQuerySet(t *testing.T) {
	adapter := NewCustomQueryAdapter(newTestLogger(), fakeDB(t), map[string]ports.CustomBenchmarkConfig{
		"set_a": {Name: "set_a", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
		"set_b": {Name: "set_b", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 2"}}},
	})

	if _, err := adapter.Execute(t.Context(), ports.BenchmarkConfig{}); err == nil {
		t.Error("Execute() with ambiguous query_set: expected error, got nil")
	}
}

func TestCustomQueryAdapter_GetVersionAndSupportedTests(t *testing.T) {
	adapter := NewCustomQueryAdapter(newTestLogger(), fakeDB(t), map[string]ports.CustomBenchmarkConfig{
		"set_a": {Name: "set_a", Queries: []ports.CustomQueryDefinition{{Query: "SELECT 1"}}},
	})

	version, err := adapter.GetVersion()
	if err != nil || version == "" {
		t.Errorf("GetVersion() = (%q, %v), want non-empty version and no error", version, err)
	}

	tests := adapter.GetSupportedTests()
	if len(tests) != 1 || tests[0] != "set_a" {
		t.Errorf("GetSupportedTests() = %v, want [set_a]", tests)
	}
}
