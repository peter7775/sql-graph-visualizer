package performance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
)

func TestFileBenchmarkSampleStore_RoundTrip(t *testing.T) {
	store, err := NewFileBenchmarkSampleStore(newTestLogger(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if store.HasSamples(ctx, "run-1") {
		t.Fatal("HasSamples must be false for an unknown run")
	}
	if got, err := store.GetSamples(ctx, "run-1", 0); err != nil || len(got) != 0 {
		t.Fatalf("GetSamples(unknown) = %v, %v; want empty and no error", got, err)
	}

	for seq := 1; seq <= 5; seq++ {
		sample := ports.LiveSample{BenchmarkID: "run-1", Seq: seq, Interval: ports.LiveIntervalStats{QPS: float64(seq)}}
		if err := store.AppendSample(ctx, "run-1", sample); err != nil {
			t.Fatalf("AppendSample(%d) error = %v", seq, err)
		}
	}
	if !store.HasSamples(ctx, "run-1") {
		t.Fatal("HasSamples must be true after appending")
	}

	all, err := store.GetSamples(ctx, "run-1", 0)
	if err != nil || len(all) != 5 {
		t.Fatalf("GetSamples(0) = %d samples, err %v; want 5", len(all), err)
	}
	after, _ := store.GetSamples(ctx, "run-1", 3)
	if len(after) != 2 || after[0].Seq != 4 || after[1].Seq != 5 {
		t.Errorf("GetSamples(3) = %+v, want seq 4,5", after)
	}

	if err := store.DeleteSamples(ctx, "run-1"); err != nil {
		t.Fatal(err)
	}
	if store.HasSamples(ctx, "run-1") {
		t.Error("samples must be gone after DeleteSamples")
	}
	if err := store.DeleteSamples(ctx, "run-1"); err != nil {
		t.Errorf("deleting a missing run must not fail: %v", err)
	}
}

func TestFileBenchmarkSampleStore_RejectsUnsafeIDs(t *testing.T) {
	store, err := NewFileBenchmarkSampleStore(newTestLogger(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "../etc/passwd", "a/b", `a\b`, "with space", "x.jsonl/../../y"} {
		if err := store.AppendSample(context.Background(), id, ports.LiveSample{}); err == nil {
			t.Errorf("AppendSample(%q) must be rejected", id)
		}
		if store.HasSamples(context.Background(), id) {
			t.Errorf("HasSamples(%q) must be false", id)
		}
	}
}

func TestFileBenchmarkSampleStore_SkipsTornLastLine(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileBenchmarkSampleStore(newTestLogger(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.AppendSample(ctx, "run-2", ports.LiveSample{Seq: 1}); err != nil {
		t.Fatal(err)
	}
	// Simulate a crash in the middle of the next write.
	f, err := os.OpenFile(filepath.Join(dir, "samples", "run-2.jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"seq":2,"inter`); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	got, err := store.GetSamples(ctx, "run-2", 0)
	if err != nil || len(got) != 1 || got[0].Seq != 1 {
		t.Errorf("GetSamples = %+v, %v; want only the intact sample", got, err)
	}
}

func TestFileBenchmarkSampleStore_DeleteOlderThan(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileBenchmarkSampleStore(newTestLogger(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, id := range []string{"old", "new"} {
		if err := store.AppendSample(ctx, id, ports.LiveSample{Seq: 1}); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "samples", "old.jsonl"), past, past); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.DeleteSamplesOlderThan(ctx, time.Now().Add(-time.Hour))
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteSamplesOlderThan = %d, %v; want 1", deleted, err)
	}
	if store.HasSamples(ctx, "old") || !store.HasSamples(ctx, "new") {
		t.Error("only the old run may be removed")
	}
}

func waitForStatus(t *testing.T, svc *BenchmarkService, id string, want ports.BenchmarkStatus) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if status, ok := svc.executionStatus(id); ok && status == want {
			// The result is stored right after the status flips.
			time.Sleep(50 * time.Millisecond)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("run %s did not reach status %s", id, want)
}

// newPersistingService builds a service with file-backed result and sample
// stores in dir, so a second instance on the same dir simulates a restart.
func newPersistingService(t *testing.T, dir string) *BenchmarkService {
	t.Helper()
	svc := newTestBenchmarkService(t)
	results, err := NewFileBenchmarkResultStore(newTestLogger(), dir)
	if err != nil {
		t.Fatal(err)
	}
	samples, err := NewFileBenchmarkSampleStore(newTestLogger(), dir)
	if err != nil {
		t.Fatal(err)
	}
	svc.SetResultStore(results)
	svc.SetSampleStore(samples)
	if err := svc.RegisterBenchmarkTool("stream", &fakeStreamingTool{samples: 4}); err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestBenchmarkService_PersistsSamplesAndReplaysAfterRestart(t *testing.T) {
	dir := t.TempDir()
	svc := newPersistingService(t, dir)

	var observed []ports.LiveSample
	svc.SetSampleObserver(func(s ports.LiveSample) { observed = append(observed, s) })

	id, err := svc.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{
		Threads: 2, Duration: time.Second,
		CustomParams: map[string]interface{}{
			"query_set": "set-a",
			"labels":    map[string]string{"index_applied": "false"},
		},
	}, "stream")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, svc, id, ports.BenchmarkStatusCompleted)

	if len(observed) != 4 {
		t.Fatalf("observer saw %d samples, want 4", len(observed))
	}
	last, ok := svc.LatestLiveSample()
	if !ok || last.Seq != 4 || last.BenchmarkID != id {
		t.Errorf("LatestLiveSample = %+v, %v; want seq 4 of the run", last, ok)
	}

	// "Restart": a new service on the same directories has no in-memory runs.
	restarted := newPersistingService(t, dir)

	backlog, ch, cancel, err := restarted.SubscribeLive(id, 0)
	if err != nil {
		t.Fatalf("SubscribeLive after restart error = %v", err)
	}
	defer cancel()
	if len(backlog) != 4 || backlog[0].Seq != 1 || backlog[3].Seq != 4 {
		t.Errorf("replayed backlog = %d samples, want seq 1..4", len(backlog))
	}
	if _, open := <-ch; open {
		t.Error("a replayed run must end the stream immediately")
	}

	resumed, _, cancel2, err := restarted.SubscribeLive(id, 2)
	if err != nil {
		t.Fatal(err)
	}
	cancel2()
	if len(resumed) != 2 || resumed[0].Seq != 3 {
		t.Errorf("resume after seq 2 = %d samples, want seq 3,4", len(resumed))
	}

	samples, status, err := restarted.GetLiveSamples(id, 0)
	if err != nil || len(samples) != 4 || status != ports.BenchmarkStatusCompleted {
		t.Errorf("GetLiveSamples = %d samples, %q, %v; want 4, completed", len(samples), status, err)
	}
	if status, ok := restarted.ExecutionStatus(id); !ok || status != ports.BenchmarkStatusCompleted {
		t.Errorf("ExecutionStatus after restart = %q, %v", status, ok)
	}
	if _, _, err := restarted.GetLiveSamples("unknown-run", 0); err == nil {
		t.Error("an unknown run must still be an error")
	}

	history, err := restarted.History(context.Background(), 10, "")
	if err != nil || len(history) != 1 {
		t.Fatalf("History = %d entries, %v; want 1", len(history), err)
	}
	h := history[0]
	if h.ID != id || h.QuerySet != "set-a" || h.Scenario != "set-a" || !h.HasSamples || h.Status != "completed" {
		t.Errorf("unexpected history entry: %+v", h)
	}
	if h.Labels["index_applied"] != "false" || h.QPS != 99 || h.P95LatencyMs != 5 {
		t.Errorf("history labels/metrics not carried over: %+v", h)
	}

	if other, _ := restarted.History(context.Background(), 10, "another-set"); len(other) != 0 {
		t.Errorf("query_set filter returned %d entries, want 0", len(other))
	}
}

func TestBenchmarkService_HistoryOrderAndLimit(t *testing.T) {
	svc := newTestBenchmarkService(t)
	if err := svc.RegisterBenchmarkTool("stream", &fakeStreamingTool{samples: 1}); err != nil {
		t.Fatal(err)
	}

	var ids []string
	for i := 0; i < 3; i++ {
		id, err := svc.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{
			Threads: 1, Duration: time.Second,
			CustomParams: map[string]interface{}{"scenario": "demo-scenario"},
		}, "stream")
		if err != nil {
			t.Fatal(err)
		}
		waitForStatus(t, svc, id, ports.BenchmarkStatusCompleted)
		ids = append(ids, id)
		time.Sleep(5 * time.Millisecond) // distinct start times
	}

	history, err := svc.History(context.Background(), 2, "")
	if err != nil || len(history) != 2 {
		t.Fatalf("History(limit 2) = %d, %v; want 2", len(history), err)
	}
	if history[0].ID != ids[2] || history[1].ID != ids[1] {
		t.Errorf("history must be newest first: got %s, %s", history[0].ID, history[1].ID)
	}
	if history[0].Scenario != "demo-scenario" || history[0].HasSamples {
		t.Errorf("scenario label missing or samples reported without a store: %+v", history[0])
	}
}

func TestApplyResultLabels(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   map[string]string
	}{
		{"nothing", nil, nil},
		{"scenario only", map[string]interface{}{"scenario": "s"}, map[string]string{"scenario": "s"}},
		{"query set doubles as scenario", map[string]interface{}{"query_set": "q"}, map[string]string{"scenario": "q", "query_set": "q"}},
		{"explicit scenario wins", map[string]interface{}{"query_set": "q", "scenario": "s"}, map[string]string{"scenario": "s", "query_set": "q"}},
		{"string map labels", map[string]interface{}{"labels": map[string]string{"index_applied": "true"}}, map[string]string{"index_applied": "true"}},
		{"interface map labels skip non strings", map[string]interface{}{"labels": map[string]interface{}{"a": "1", "b": 2}}, map[string]string{"a": "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ports.BenchmarkResult{}
			applyResultLabels(result, tt.params)
			if len(result.Labels) != len(tt.want) {
				t.Fatalf("labels = %v, want %v", result.Labels, tt.want)
			}
			for k, v := range tt.want {
				if result.Labels[k] != v {
					t.Errorf("label %q = %q, want %q", k, result.Labels[k], v)
				}
			}
		})
	}
}
