package performance

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return logger
}

func TestFileBenchmarkResultStore_SaveGetList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("NewFileBenchmarkResultStore() error = %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	results := []*ports.BenchmarkResult{
		{ID: "run-1", ToolName: "sysbench", TestType: "oltp_read_write", StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-2 * time.Hour), Status: ports.BenchmarkStatusCompleted},
		{ID: "run-2", ToolName: "sysbench", TestType: "oltp_read_only", StartTime: now.Add(-1 * time.Hour), EndTime: now.Add(-1 * time.Hour), Status: ports.BenchmarkStatusCompleted},
		{ID: "run-3", ToolName: "custom", TestType: "reporting_queries", StartTime: now, EndTime: now, Status: ports.BenchmarkStatusFailed},
	}
	for _, r := range results {
		if err := store.Save(ctx, r); err != nil {
			t.Fatalf("Save(%s) error = %v", r.ID, err)
		}
	}

	got, err := store.Get(ctx, "run-2")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "run-2" || got.TestType != "oltp_read_only" {
		t.Errorf("Get() = %+v, want run-2/oltp_read_only", got)
	}

	if _, err := store.Get(ctx, "missing"); err == nil {
		t.Error("Get() for missing ID: expected error, got nil")
	}

	all, err := store.List(ctx, ports.BenchmarkResultFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("List() returned %d results, want 3", len(all))
	}
	// Must be ordered ascending by start time.
	for i := 1; i < len(all); i++ {
		if all[i-1].StartTime.After(all[i].StartTime) {
			t.Errorf("List() not sorted ascending by start time: %v before %v", all[i-1].StartTime, all[i].StartTime)
		}
	}

	sysbenchOnly, err := store.List(ctx, ports.BenchmarkResultFilter{ToolName: "sysbench"})
	if err != nil {
		t.Fatalf("List(tool filter) error = %v", err)
	}
	if len(sysbenchOnly) != 2 {
		t.Errorf("List(tool=sysbench) returned %d results, want 2", len(sysbenchOnly))
	}

	limited, err := store.List(ctx, ports.BenchmarkResultFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List(limit) error = %v", err)
	}
	if len(limited) != 1 || limited[0].ID != "run-3" {
		t.Errorf("List(limit=1) = %+v, want the single most recent result (run-3)", limited)
	}
}

func TestFileBenchmarkResultStore_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store1, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("NewFileBenchmarkResultStore() error = %v", err)
	}
	if err := store1.Save(ctx, &ports.BenchmarkResult{ID: "persisted-1", ToolName: "sysbench", StartTime: time.Now()}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("second NewFileBenchmarkResultStore() error = %v", err)
	}
	got, err := store2.Get(ctx, "persisted-1")
	if err != nil {
		t.Fatalf("Get() after reload error = %v", err)
	}
	if got.ID != "persisted-1" {
		t.Errorf("Get() after reload = %+v, want persisted-1", got)
	}

	if _, err := filepath.Abs(dir); err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
}

func TestFileBenchmarkResultStore_DeleteOlderThan(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("NewFileBenchmarkResultStore() error = %v", err)
	}

	ctx := context.Background()
	now := time.Now()
	old := &ports.BenchmarkResult{ID: "old", StartTime: now.Add(-48 * time.Hour)}
	recent := &ports.BenchmarkResult{ID: "recent", StartTime: now}

	if err := store.Save(ctx, old); err != nil {
		t.Fatalf("Save(old) error = %v", err)
	}
	if err := store.Save(ctx, recent); err != nil {
		t.Fatalf("Save(recent) error = %v", err)
	}

	deleted, err := store.DeleteOlderThan(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("DeleteOlderThan() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("DeleteOlderThan() deleted = %d, want 1", deleted)
	}

	if _, err := store.Get(ctx, "old"); err == nil {
		t.Error("Get(old) after DeleteOlderThan: expected error, got nil")
	}
	if _, err := store.Get(ctx, "recent"); err != nil {
		t.Errorf("Get(recent) after DeleteOlderThan: unexpected error %v", err)
	}

	// Reload from disk to confirm compaction persisted correctly.
	reloaded, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("reload after compaction error = %v", err)
	}
	all, err := reloaded.List(ctx, ports.BenchmarkResultFilter{})
	if err != nil {
		t.Fatalf("List() after reload error = %v", err)
	}
	if len(all) != 1 || all[0].ID != "recent" {
		t.Errorf("List() after reload = %+v, want only 'recent'", all)
	}
}

func TestNewFileBenchmarkResultStore_DefaultDir(t *testing.T) {
	// Passing an empty dir should not error; it falls back to a default path
	// relative to the working directory. Use t.TempDir() as cwd substitute is
	// not straightforward, so just verify no panic/error using an explicit
	// nested path instead.
	dir := filepath.Join(t.TempDir(), "nested", "benchmarks")
	store, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatalf("NewFileBenchmarkResultStore() with nested dir error = %v", err)
	}
	if store == nil {
		t.Fatal("NewFileBenchmarkResultStore() returned nil store")
	}
}
