package performance

import (
	"context"
	"math"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
)

func TestLatencyHist_Percentiles(t *testing.T) {
	var h latencyHist
	// 1..100 ms, one observation each: p50 ~ 50 ms, p95 ~ 95 ms, p99 ~ 99 ms.
	for i := 1; i <= 100; i++ {
		h.record(time.Duration(i) * time.Millisecond)
	}
	s := h.snapshot()

	if s.count != 100 {
		t.Fatalf("count = %d, want 100", s.count)
	}
	// Buckets are 5% wide, so allow ~6% relative error.
	for _, tc := range []struct {
		p    float64
		want float64
	}{{0.50, 50}, {0.95, 95}, {0.99, 99}} {
		got := s.percentileMs(tc.p)
		if math.Abs(got-tc.want)/tc.want > 0.06 {
			t.Errorf("percentile(%v) = %.2f ms, want ~%.0f ms", tc.p, got, tc.want)
		}
	}
	if got := s.maxMs(); math.Abs(got-100) > 0.001 {
		t.Errorf("max = %v, want 100", got)
	}
	if got := s.avgMs(); math.Abs(got-50.5) > 0.001 {
		t.Errorf("avg = %v, want 50.5", got)
	}
}

func TestLatencyHist_SubMillisecondAndEmpty(t *testing.T) {
	var h latencyHist
	empty := h.snapshot()
	if empty.percentileMs(0.95) != 0 || empty.avgMs() != 0 {
		t.Error("empty histogram must report zeros")
	}

	h.record(200 * time.Microsecond)
	s := h.snapshot()
	if got := s.avgMs(); math.Abs(got-0.2) > 1e-9 {
		t.Errorf("avgMs = %v, want 0.2 (no truncation to whole ms)", got)
	}
	if got := s.percentileMs(0.99); got <= 0 || got > 0.21 {
		t.Errorf("p99 = %v ms, want in (0, 0.21]", got)
	}
}

func TestLatencyHist_DrainResets(t *testing.T) {
	var h latencyHist
	h.record(5 * time.Millisecond)
	h.record(7 * time.Millisecond)

	first := h.drain()
	if first.count != 2 {
		t.Errorf("first drain count = %d, want 2", first.count)
	}
	second := h.drain()
	if second.count != 0 || second.maxNs != 0 || second.sumNs != 0 {
		t.Errorf("second drain not empty: %+v", second.count)
	}
}

func TestRankAndHeat(t *testing.T) {
	cases := []struct {
		ms   float64
		rank string
	}{{1, "FAST"}, {24.9, "FAST"}, {25, "MEDIUM"}, {99, "MEDIUM"}, {100, "SLOW"}, {500, "SLOW"}}
	for _, c := range cases {
		if got := rankFor(c.ms); got != c.rank {
			t.Errorf("rankFor(%v) = %s, want %s", c.ms, got, c.rank)
		}
	}
	if heatFor(1) != 0 {
		t.Error("heat below the low bound must be 0")
	}
	if heatFor(10000) != 1 {
		t.Error("heat above the high bound must be clamped to 1")
	}
	if a, b := heatFor(10), heatFor(100); !(a > 0 && a < b && b < 1) {
		t.Errorf("heat must increase monotonically: heat(10)=%v heat(100)=%v", a, b)
	}
}

func TestQueryTables(t *testing.T) {
	tests := []struct {
		name string
		def  ports.CustomQueryDefinition
		want []string
	}{
		{"configured list wins", ports.CustomQueryDefinition{Query: "SELECT 1", Tables: []string{"a", " b "}}, []string{"a", "b"}},
		{"from + join chain", ports.CustomQueryDefinition{Query: "SELECT * FROM orders o JOIN customers c ON c.id=o.customer_id JOIN `products` p ON 1=1"}, []string{"orders", "customers", "products"}},
		{"update", ports.CustomQueryDefinition{Query: "UPDATE customers SET x=1 WHERE id=1"}, []string{"customers"}},
		{"duplicates collapsed", ports.CustomQueryDefinition{Query: "SELECT * FROM a JOIN a ON 1=1"}, []string{"a"}},
		{"no tables", ports.CustomQueryDefinition{Query: "SELECT 1"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queryTables(tt.def)
			if len(got) != len(tt.want) {
				t.Fatalf("queryTables() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("queryTables() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestLiveCollector_Sample(t *testing.T) {
	defs := []ports.CustomQueryDefinition{
		{Query: "q-orders-customers", Description: "slow join", Tables: []string{"orders", "customers"}},
		{Query: "q-items", Description: "fast single", Tables: []string{"order_items"}},
		{Query: "q-chain", Tables: []string{"order_items", "orders", "customers"}},
	}
	start := time.Now()
	c := newLiveCollector(defs, 8, start)

	// Interval 1: 10 slow executions on the join (150 ms), 40 fast ones (1 ms).
	for i := 0; i < 10; i++ {
		c.record(0, 150*time.Millisecond, false)
	}
	for i := 0; i < 40; i++ {
		c.record(1, time.Millisecond, false)
	}
	c.record(1, time.Millisecond, true)

	s := c.sample(start.Add(time.Second))

	if s.Threads != 8 || s.ElapsedSeconds != 1 {
		t.Errorf("threads/elapsed = %d/%v, want 8/1", s.Threads, s.ElapsedSeconds)
	}
	if s.Interval.Queries != 51 || s.Interval.Errors != 1 {
		t.Errorf("interval queries/errors = %d/%d, want 51/1", s.Interval.Queries, s.Interval.Errors)
	}
	if s.Interval.QPS != 51 {
		t.Errorf("interval QPS = %v, want 51", s.Interval.QPS)
	}
	if s.Interval.P99LatencyMs < 100 || s.Interval.P50LatencyMs > 5 {
		t.Errorf("p50/p99 = %v/%v, want p50 small and p99 large", s.Interval.P50LatencyMs, s.Interval.P99LatencyMs)
	}
	if len(s.Queries) != 3 || s.Queries[0].Count != 10 || s.Queries[2].Count != 0 {
		t.Errorf("unexpected per-query stats: %+v", s.Queries)
	}

	// Nodes: every table of the query set is present, in stable order.
	if len(s.Graph.Nodes) != 3 || s.Graph.Nodes[0].ID != "customers" || s.Graph.Nodes[2].ID != "orders" {
		t.Fatalf("nodes = %+v, want customers/order_items/orders", s.Graph.Nodes)
	}
	nodes := map[string]ports.LiveGraphNode{}
	for _, n := range s.Graph.Nodes {
		nodes[n.ID] = n
	}
	if nodes["orders"].Heat <= nodes["order_items"].Heat {
		t.Errorf("orders (slow) must be hotter than order_items (fast): %v vs %v", nodes["orders"].Heat, nodes["order_items"].Heat)
	}

	// Edges are adjacent pairs along the join path, not a star.
	edges := map[string]ports.LiveGraphEdge{}
	for _, e := range s.Graph.Edges {
		edges[e.ID] = e
	}
	if len(edges) != 2 {
		t.Fatalf("edges = %+v, want exactly orders-customers and order_items-orders", s.Graph.Edges)
	}
	oc, ok := edges[ports.GraphEdgeID("orders", "customers")]
	if !ok {
		t.Fatalf("missing orders-customers edge in %v", edges)
	}
	if oc.Rank != "SLOW" || oc.Load < 0.9 {
		t.Errorf("orders-customers rank/load = %s/%v, want SLOW and ~all of the busy time", oc.Rank, oc.Load)
	}
	if _, ok := edges[ports.GraphEdgeID("order_items", "customers")]; ok {
		t.Error("order_items-customers must not exist (chain, not star)")
	}
	if idle := edges[ports.GraphEdgeID("order_items", "orders")]; idle.QPS != 0 || idle.Rank != "FAST" {
		t.Errorf("idle edge = %+v, want zero qps and FAST", idle)
	}

	// Interval counters were drained: an empty second interval is all zeros.
	s2 := c.sample(start.Add(2 * time.Second))
	if s2.Interval.Queries != 0 || s2.Interval.QPS != 0 {
		t.Errorf("second interval = %+v, want empty", s2.Interval)
	}

	// Cumulative stats still reflect the whole run.
	metrics, results := c.finalStats(2 * time.Second)
	if metrics.QueriesPerSecond != 25.5 {
		t.Errorf("final QPS = %v, want 25.5", metrics.QueriesPerSecond)
	}
	if metrics.Percentile99 < 100 || metrics.Percentile50 > 5 {
		t.Errorf("final p50/p99 = %v/%v", metrics.Percentile50, metrics.Percentile99)
	}
	if len(results) != 2 {
		t.Fatalf("final query results = %d, want 2 (idle query skipped)", len(results))
	}
	if results[0].SourceTables[0] != "orders" || results[0].JoinedTables[0] != "customers" {
		t.Errorf("tables not mapped: %+v / %+v", results[0].SourceTables, results[0].JoinedTables)
	}
}

func TestLiveHub_ReplayAndLive(t *testing.T) {
	h := newLiveHub("run-1")
	h.publish(ports.LiveSample{})
	h.publish(ports.LiveSample{})

	backlog, ch, cancel := h.subscribe(0)
	defer cancel()
	if len(backlog) != 2 || backlog[0].Seq != 1 || backlog[1].Seq != 2 || backlog[0].BenchmarkID != "run-1" {
		t.Fatalf("backlog = %+v, want seq 1,2 with benchmark id", backlog)
	}

	h.publish(ports.LiveSample{})
	select {
	case s := <-ch:
		if s.Seq != 3 {
			t.Errorf("live sample seq = %d, want 3 (no gap, no duplicate)", s.Seq)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive the live sample")
	}

	if got := h.since(2); len(got) != 1 || got[0].Seq != 3 {
		t.Errorf("since(2) = %+v, want only seq 3", got)
	}

	h.close()
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected closed channel after close()")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed by hub.close()")
	}

	// Subscribing after the run ended replays and ends immediately.
	late, lateCh, lateCancel := h.subscribe(1)
	defer lateCancel()
	if len(late) != 2 {
		t.Errorf("late backlog = %d, want 2", len(late))
	}
	if _, ok := <-lateCh; ok {
		t.Error("late subscriber channel must be closed")
	}

	h.publish(ports.LiveSample{}) // must be ignored after close, no panic
	if len(h.since(0)) != 3 {
		t.Error("publish after close must be dropped")
	}
}

func TestLiveHub_SlowSubscriberDoesNotBlock(t *testing.T) {
	h := newLiveHub("run-2")
	_, _, cancel := h.subscribe(0) // never read from the channel
	defer cancel()

	done := make(chan struct{})
	go func() {
		for i := 0; i < liveSubscriberBuffer*3; i++ {
			h.publish(ports.LiveSample{})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked on a slow subscriber")
	}
}

// fakeStreamingTool emits a fixed number of samples, then completes (or stops
// when the context is cancelled).
type fakeStreamingTool struct {
	samples int
	hold    chan struct{} // when set, the tool waits for ctx or close
}

func (f *fakeStreamingTool) ExecuteStream(ctx context.Context, cfg ports.BenchmarkConfig, emit func(ports.LiveSample)) (*ports.BenchmarkResult, error) {
	for i := 0; i < f.samples; i++ {
		emit(ports.LiveSample{Threads: cfg.Threads, Interval: ports.LiveIntervalStats{QPS: float64(i + 1)}})
	}
	if f.hold != nil {
		select {
		case <-ctx.Done():
		case <-f.hold:
		}
	}
	return &ports.BenchmarkResult{
		ToolName: "fake-stream",
		TestType: "set-a",
		Metrics:  &ports.PerformanceMetrics{QueriesPerSecond: 99, AverageLatency: 2, Percentile95: 5, Percentile99: 9},
		QueryResults: []ports.QueryPerformance{
			{ExecutionCount: 7},
		},
	}, nil
}

func (f *fakeStreamingTool) Execute(ctx context.Context, cfg ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	return f.ExecuteStream(ctx, cfg, func(ports.LiveSample) {})
}
func (f *fakeStreamingTool) Validate(ports.BenchmarkConfig) error { return nil }
func (f *fakeStreamingTool) GetSupportedTests() []string          { return []string{"set-a"} }
func (f *fakeStreamingTool) IsAvailable() bool                    { return true }
func (f *fakeStreamingTool) GetVersion() (string, error)          { return "fake-stream/1", nil }

func TestBenchmarkService_LiveStreaming(t *testing.T) {
	svc := newTestBenchmarkService(t)
	if err := svc.RegisterBenchmarkTool("stream", &fakeStreamingTool{samples: 3}); err != nil {
		t.Fatal(err)
	}

	id, err := svc.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{
		Threads: 2, Duration: time.Second,
		CustomParams: map[string]interface{}{"scenario": "checkout-peak"},
	}, "stream")
	if err != nil {
		t.Fatalf("ExecuteBenchmark() error = %v", err)
	}

	backlog, ch, cancel, err := svc.SubscribeLive(id, 0)
	if err != nil {
		t.Fatalf("SubscribeLive() error = %v", err)
	}
	defer cancel()

	// Backlog + live samples together must yield exactly seq 1..3 in order.
	got := append([]ports.LiveSample{}, backlog...)
	timeout := time.After(2 * time.Second)
	for open := true; open; {
		select {
		case s, ok := <-ch:
			if !ok {
				open = false
				break
			}
			got = append(got, s)
		case <-timeout:
			t.Fatal("stream did not finish")
		}
	}
	if len(got) != 3 {
		t.Fatalf("received %d samples, want 3", len(got))
	}
	for i, s := range got {
		if s.Seq != i+1 || s.BenchmarkID != id {
			t.Errorf("sample %d = seq %d id %q, want seq %d id %q", i, s.Seq, s.BenchmarkID, i+1, id)
		}
	}

	// After the stream ended, the final result and summary are available.
	summary, err := svc.GetBenchmarkSummary(context.Background(), id)
	if err != nil {
		t.Fatalf("GetBenchmarkSummary() error = %v", err)
	}
	if summary.Scenario != "checkout-peak" || summary.QPS != 99 || summary.P95LatencyMs != 5 || summary.TotalQueries != 7 {
		t.Errorf("unexpected summary: %+v", summary)
	}
	if status, _ := svc.ExecutionStatus(id); status != ports.BenchmarkStatusCompleted {
		t.Errorf("status = %v, want completed", status)
	}

	if _, _, err := svc.GetLiveSamples("missing", 0); err == nil {
		t.Error("GetLiveSamples for unknown id: expected error")
	}
}

func TestBenchmarkService_StoppedRunStaysCancelled(t *testing.T) {
	svc := newTestBenchmarkService(t)
	tool := &fakeStreamingTool{samples: 1, hold: make(chan struct{})}
	if err := svc.RegisterBenchmarkTool("stream", tool); err != nil {
		t.Fatal(err)
	}
	id, err := svc.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{Threads: 1, Duration: time.Minute}, "stream")
	if err != nil {
		t.Fatal(err)
	}
	_, ch, cancel, err := svc.SubscribeLive(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	// Wait until the run is actually running, then stop it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st, _ := svc.ExecutionStatus(id); st == ports.BenchmarkStatusRunning {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := svc.StopBenchmark(context.Background(), id); err != nil {
		t.Fatalf("StopBenchmark() error = %v", err)
	}

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not end after stop")
	case <-func() chan struct{} {
		done := make(chan struct{})
		go func() {
			for range ch {
			}
			close(done)
		}()
		return done
	}():
	}

	if st, _ := svc.ExecutionStatus(id); st != ports.BenchmarkStatusCancelled {
		t.Errorf("status after stop = %v, want cancelled", st)
	}
	summary, err := svc.GetBenchmarkSummary(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Status != string(ports.BenchmarkStatusCancelled) {
		t.Errorf("summary status = %q, want cancelled", summary.Status)
	}
}
