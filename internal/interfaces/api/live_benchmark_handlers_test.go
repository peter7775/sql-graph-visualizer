package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// streamTool emits n samples and then returns a fixed result.
type streamTool struct {
	n   int
	qps float64
	p95 float64
}

func (s *streamTool) ExecuteStream(_ context.Context, _ ports.BenchmarkConfig, emit func(ports.LiveSample)) (*ports.BenchmarkResult, error) {
	for i := 0; i < s.n; i++ {
		emit(ports.LiveSample{Interval: ports.LiveIntervalStats{QPS: s.qps}})
	}
	return &ports.BenchmarkResult{
		ToolName: "custom",
		TestType: "set",
		Metrics:  &ports.PerformanceMetrics{QueriesPerSecond: s.qps, AverageLatency: s.p95 / 2, Percentile95: s.p95, Percentile99: s.p95 * 2},
	}, nil
}
func (s *streamTool) Execute(ctx context.Context, c ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	return s.ExecuteStream(ctx, c, func(ports.LiveSample) {})
}
func (s *streamTool) Validate(ports.BenchmarkConfig) error { return nil }
func (s *streamTool) GetSupportedTests() []string          { return []string{"set"} }
func (s *streamTool) IsAvailable() bool                    { return true }
func (s *streamTool) GetVersion() (string, error)          { return "stream/1", nil }

func newLiveTestHandlers(t *testing.T, tools map[string]ports.BenchmarkToolPort) (*PerformanceHandlers, *mux.Router) {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	cfg := performance.DefaultBenchmarkServiceConfig()
	cfg.CleanupInterval = time.Hour
	svc := performance.NewBenchmarkService(nil, nil, nil, nil, logger, cfg)
	for name, tool := range tools {
		if err := svc.RegisterBenchmarkTool(name, tool); err != nil {
			t.Fatal(err)
		}
	}
	ph := &PerformanceHandlers{logger: logger, benchmarkService: svc}
	router := mux.NewRouter()
	ph.registerLiveRoutes(router)
	return ph, router
}

func startRun(t *testing.T, ph *PerformanceHandlers, tool string) string {
	t.Helper()
	id, err := ph.benchmarkService.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{Threads: 1, Duration: time.Second}, tool)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func waitFinished(t *testing.T, ph *PerformanceHandlers, id string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := ph.benchmarkService.GetBenchmarkSummary(context.Background(), id); err == nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("run did not finish")
}

type sseEvent struct {
	name string
	id   string
	data string
}

func parseSSE(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	var cur sseEvent
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if cur.name != "" {
				events = append(events, cur)
			}
			cur = sseEvent{}
		case strings.HasPrefix(line, "id: "):
			cur.id = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			cur.name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			cur.data = strings.TrimPrefix(line, "data: ")
		}
	}
	return events
}

func TestStreamBenchmark_ReplayAndDone(t *testing.T) {
	ph, router := newLiveTestHandlers(t, map[string]ports.BenchmarkToolPort{"custom": &streamTool{n: 3, qps: 10, p95: 4}})
	id := startRun(t, ph, "custom")
	waitFinished(t, ph, id)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/performance/benchmarks/"+id+"/stream", nil))

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	events := parseSSE(t, rec.Body.String())

	var samples, done []sseEvent
	for _, e := range events {
		switch e.name {
		case "sample":
			samples = append(samples, e)
		case "done":
			done = append(done, e)
		}
	}
	if len(samples) != 3 {
		t.Fatalf("got %d sample events, want 3 (events: %+v)", len(samples), events)
	}
	for i, e := range samples {
		if e.id != string(rune('1'+i)) {
			t.Errorf("sample %d id = %q, want %d", i, e.id, i+1)
		}
		var s ports.LiveSample
		if err := json.Unmarshal([]byte(e.data), &s); err != nil || s.BenchmarkID != id || s.Seq != i+1 {
			t.Errorf("sample %d payload invalid: err=%v %+v", i, err, s)
		}
	}
	if len(done) != 1 {
		t.Fatalf("got %d done events, want 1", len(done))
	}
	var d sseDoneEvent
	if err := json.Unmarshal([]byte(done[0].data), &d); err != nil {
		t.Fatal(err)
	}
	if d.Status != "completed" || d.Summary == nil || d.Summary.P95LatencyMs != 4 || d.Summary.QPS != 10 {
		t.Errorf("unexpected done event: %+v summary=%+v", d, d.Summary)
	}

	// Resume: only samples after seq 2 are replayed.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/performance/benchmarks/"+id+"/stream", nil)
	req.Header.Set("Last-Event-ID", "2")
	router.ServeHTTP(rec, req)
	count := 0
	for _, e := range parseSSE(t, rec.Body.String()) {
		if e.name == "sample" {
			count++
			if e.id != "3" {
				t.Errorf("resumed sample id = %q, want 3", e.id)
			}
		}
	}
	if count != 1 {
		t.Errorf("resumed stream replayed %d samples, want 1", count)
	}
}

func TestStreamBenchmark_UnknownID(t *testing.T) {
	_, router := newLiveTestHandlers(t, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/performance/benchmarks/nope/stream", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestGetBenchmarkSamples_Since(t *testing.T) {
	ph, router := newLiveTestHandlers(t, map[string]ports.BenchmarkToolPort{"custom": &streamTool{n: 3, qps: 5, p95: 1}})
	id := startRun(t, ph, "custom")
	waitFinished(t, ph, id)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/performance/benchmarks/"+id+"/samples?since=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Status  string             `json:"status"`
			Samples []ports.LiveSample `json:"samples"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.Data.Status != "completed" || len(resp.Data.Samples) != 2 || resp.Data.Samples[0].Seq != 2 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestCompareBenchmarks(t *testing.T) {
	ph, router := newLiveTestHandlers(t, map[string]ports.BenchmarkToolPort{
		"slow": &streamTool{n: 1, qps: 100, p95: 200},
		"fast": &streamTool{n: 1, qps: 400, p95: 20},
	})
	a, b := startRun(t, ph, "slow"), startRun(t, ph, "fast")
	waitFinished(t, ph, a)
	waitFinished(t, ph, b)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/performance/benchmarks/compare?a="+a+"&b="+b, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			A     performance.BenchmarkSummary `json:"a"`
			B     performance.BenchmarkSummary `json:"b"`
			Delta BenchmarkDelta               `json:"delta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.A.ID != a || resp.Data.B.ID != b {
		t.Errorf("ids = %q/%q, want %q/%q", resp.Data.A.ID, resp.Data.B.ID, a, b)
	}
	if resp.Data.Delta.P95Pct != -90 || resp.Data.Delta.QPSPct != 300 {
		t.Errorf("delta = %+v, want p95 -90%% and qps +300%%", resp.Data.Delta)
	}

	for _, url := range []string{
		"/api/performance/benchmarks/compare",
		"/api/performance/benchmarks/compare?a=" + a,
		"/api/performance/benchmarks/compare?a=" + a + "&b=missing",
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 400/404", url, rec.Code)
		}
	}
}

func TestPctChangeAndParseSince(t *testing.T) {
	if pctChange(0, 5) != 0 {
		t.Error("pctChange with zero base must be 0")
	}
	if pctChange(200, 20) != -90 {
		t.Error("pctChange(200,20) must be -90")
	}

	r := httptest.NewRequest(http.MethodGet, "/x?since=7", nil)
	if parseSinceSeq(r) != 7 {
		t.Error("since query parameter must be honoured")
	}
	r = httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("Last-Event-ID", "4")
	if parseSinceSeq(r) != 4 {
		t.Error("Last-Event-ID must be honoured")
	}
	r = httptest.NewRequest(http.MethodGet, "/x?since=abc", nil)
	if parseSinceSeq(r) != 0 {
		t.Error("garbage must fall back to 0")
	}
}

func TestValidateOptimization(t *testing.T) {
	good := &models.LiveDemoOptimization{
		Apply:  []string{"CREATE INDEX idx_a ON orders (customer_id)", "create index idx_b on order_items (product_id, order_id);"},
		Revert: []string{"DROP INDEX idx_a ON orders", "drop index idx_b on order_items;"},
	}
	if err := validateOptimization(good); err != nil {
		t.Errorf("valid optimization rejected: %v", err)
	}

	bad := map[string]*models.LiveDemoOptimization{
		"empty":             {},
		"no revert":         {Apply: good.Apply},
		"drop table":        {Apply: []string{"DROP TABLE orders"}, Revert: good.Revert},
		"stacked":           {Apply: []string{"CREATE INDEX i ON t (c); DROP TABLE t"}, Revert: good.Revert},
		"unique idx":        {Apply: []string{"CREATE UNIQUE INDEX i ON t (c)"}, Revert: good.Revert},
		"expression":        {Apply: []string{"CREATE INDEX i ON t ((a+1))"}, Revert: good.Revert},
		"comment trick":     {Apply: []string{"CREATE INDEX i ON t (c) -- x"}, Revert: good.Revert},
		"revert not drop":   {Apply: good.Apply, Revert: []string{"DELETE FROM orders"}},
		"schema qualified":  {Apply: []string{"CREATE INDEX i ON mysql.user (c)"}, Revert: good.Revert},
		"quoted identifier": {Apply: []string{"CREATE INDEX `i` ON t (c)"}, Revert: good.Revert},
	}
	for name, opt := range bad {
		if err := validateOptimization(opt); err == nil {
			t.Errorf("%s: expected validation error, got nil", name)
		}
	}
}

func TestDemoHandlers_TopologyAndScenarios(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	cfg := performance.DefaultBenchmarkServiceConfig()
	cfg.CleanupInterval = time.Hour
	svc := performance.NewBenchmarkService(nil, nil, nil, nil, logger, cfg)
	if err := svc.RegisterBenchmarkTool("custom", &streamTool{n: 1, qps: 1, p95: 1}); err != nil {
		t.Fatal(err)
	}

	demo := &models.LiveDemoConfig{
		Enabled: true,
		Scenarios: []models.LiveDemoScenario{
			{Name: "checkout-peak", Title: "Checkout peak", QuerySet: "set", Threads: 4, Duration: "20s"},
			{Name: "broken"}, // missing query_set: must be ignored
		},
		Optimization: &models.LiveDemoOptimization{
			Name:   "bad",
			Apply:  []string{"DROP TABLE orders"},
			Revert: []string{"DROP INDEX i ON orders"},
		},
		Topology: &models.LiveDemoTopology{
			Nodes: []models.LiveDemoNode{{ID: "orders"}, {ID: "customers", Label: "Customers"}},
			Edges: []models.LiveDemoEdge{{Source: "orders", Target: "customers", Relation: "orders.customer_id -> customers.id"}},
		},
	}
	d := NewDemoHandlers(logger, svc, nil, demo)
	router := mux.NewRouter()
	d.RegisterRoutes(router)

	// Scenarios: only the valid one, with defaults filled in.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/demo/scenarios", nil))
	var sc struct {
		Data []demoScenarioInfo `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sc); err != nil {
		t.Fatal(err)
	}
	if len(sc.Data) != 1 || sc.Data[0].Name != "checkout-peak" || sc.Data[0].DurationSec != 20 || sc.Data[0].Threads != 4 {
		t.Errorf("unexpected scenarios: %+v", sc.Data)
	}

	// Topology: canonical, order-insensitive edge ids and label defaults.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/demo/topology", nil))
	var topo struct {
		Data struct {
			Nodes []demoTopologyNode `json:"nodes"`
			Edges []demoTopologyEdge `json:"edges"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &topo); err != nil {
		t.Fatal(err)
	}
	if len(topo.Data.Nodes) != 2 || topo.Data.Nodes[0].Label != "orders" || topo.Data.Nodes[1].Label != "Customers" {
		t.Errorf("unexpected nodes: %+v", topo.Data.Nodes)
	}
	if len(topo.Data.Edges) != 1 || topo.Data.Edges[0].ID != ports.GraphEdgeID("customers", "orders") {
		t.Errorf("unexpected edges: %+v", topo.Data.Edges)
	}

	// An invalid optimization disables the optimization endpoints entirely.
	for _, path := range []string{"/api/demo/optimization/apply", "/api/demo/optimization/revert"} {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s with invalid config: status = %d, want 404", path, rec.Code)
		}
	}

	// Running a scenario starts a benchmark labelled with the scenario name.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/demo/scenarios/checkout-peak/run", strings.NewReader(`{"threads":2,"duration_seconds":5}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("run status = %d body=%s", rec.Code, rec.Body.String())
	}
	var run struct {
		Data struct {
			ID       string `json:"id"`
			Scenario string `json:"scenario"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if run.Data.ID == "" || run.Data.Scenario != "checkout-peak" {
		t.Errorf("unexpected run response: %+v", run.Data)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s, err := svc.GetBenchmarkSummary(context.Background(), run.Data.ID); err == nil {
			if s.Scenario != "checkout-peak" {
				t.Errorf("summary scenario = %q, want checkout-peak", s.Scenario)
			}
			goto ranOK
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("scenario run did not finish")
ranOK:

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/demo/scenarios/nope/run", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown scenario status = %d, want 404", rec.Code)
	}
}
