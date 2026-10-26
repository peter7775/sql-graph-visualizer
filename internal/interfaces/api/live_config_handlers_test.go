package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func testQuerySets() map[string]ports.CustomBenchmarkConfig {
	return map[string]ports.CustomBenchmarkConfig{
		"checkout": {
			Name: "checkout", Description: "OLTP", Threads: 12, Duration: 30 * time.Second,
			Queries: []ports.CustomQueryDefinition{
				{Query: "SELECT 1", Tables: []string{"customers", "orders"}},
				{Query: "SELECT 2", Tables: []string{"orders", "order_items"}},
			},
		},
		"reporting": {Name: "reporting", Queries: []ports.CustomQueryDefinition{{Query: "SELECT x FROM orders"}}},
	}
}

func newLiveConfigRouter(t *testing.T, demoMode bool, optimization func(context.Context) ports.OptimizationState) *mux.Router {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	querySets := testQuerySets()

	demo := &models.LiveDemoConfig{
		Enabled: true,
		Scenarios: []models.LiveDemoScenario{
			{Name: "checkout-peak", Title: "Checkout peak", QuerySet: "checkout", Threads: 8, Duration: "20s"},
			{Name: "orphan", QuerySet: "missing-set"}, // refers to an unknown query set: hidden
		},
	}
	mapping := performance.BuildLabelTableMapping([]models.TransformationConfig{
		{RuleType: "node", TargetType: "Order", Source: models.SourceConfig{Type: "table", Value: "orders"}},
	})

	router := mux.NewRouter()
	NewLiveConfigHandlers(LiveConfigDeps{
		Logger:       logger,
		Topology:     performance.NewTopologyService(logger, nil, "", querySets, nil),
		Mapping:      mapping,
		QuerySets:    querySets,
		LiveDemo:     demo,
		DemoMode:     demoMode,
		DatabaseType: "postgresql",
		Optimization: optimization,
	}).RegisterRoutes(router)
	return router
}

func getJSON(t *testing.T, router http.Handler, url string, into any) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if into != nil && rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
			t.Fatalf("%s: invalid JSON: %v\n%s", url, err, rec.Body.String())
		}
	}
	return rec
}

type liveConfigResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Mode         string             `json:"mode"`
		DatabaseType string             `json:"database_type"`
		QuerySets    []liveQuerySetInfo `json:"query_sets"`
		Scenarios    []liveScenarioInfo `json:"scenarios"`
		Optimization ports.OptimizationState
		Topology     ports.Topology `json:"topology"`
	} `json:"data"`
}

func TestLiveConfig_GenericMode(t *testing.T) {
	router := newLiveConfigRouter(t, false, nil)

	var resp liveConfigResponse
	rec := getJSON(t, router, "/api/performance/live/config", &resp)
	if rec.Code != http.StatusOK || !resp.Success {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	d := resp.Data
	if d.Mode != "generic" || d.DatabaseType != "postgresql" {
		t.Errorf("mode/db = %q/%q", d.Mode, d.DatabaseType)
	}
	if len(d.Scenarios) != 0 {
		t.Errorf("generic mode must not expose demo scenarios: %+v", d.Scenarios)
	}
	if len(d.QuerySets) != 2 || d.QuerySets[0].Name != "checkout" || d.QuerySets[1].Name != "reporting" {
		t.Fatalf("query sets must be sorted by name: %+v", d.QuerySets)
	}
	checkout := d.QuerySets[0]
	if checkout.Threads != 12 || checkout.DurationSec != 30 || checkout.QueryCount != 2 || len(checkout.Tables) != 3 {
		t.Errorf("checkout info = %+v", checkout)
	}
	if got := d.QuerySets[1].Tables; len(got) != 1 || got[0] != "orders" {
		t.Errorf("tables derived from SQL = %v", got)
	}
	if d.Topology.Source != "queries+schema" || len(d.Topology.Nodes) != 3 {
		t.Errorf("topology = %+v", d.Topology)
	}

	var raw map[string]map[string]json.RawMessage
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	var opt map[string]any
	if err := json.Unmarshal(raw["data"]["optimization"], &opt); err != nil {
		t.Fatal(err)
	}
	if opt["available"] != false || opt["can_apply"] != false {
		t.Errorf("optimization without a provider must be unavailable: %v", opt)
	}
}

func TestLiveConfig_DemoModeWithOptimization(t *testing.T) {
	provider := func(context.Context) ports.OptimizationState {
		return ports.OptimizationState{
			Available: true, CanApply: true, Name: "idx", Title: "Index", Applied: true,
			Statements: []string{"CREATE INDEX i ON t (c)"},
		}
	}
	router := newLiveConfigRouter(t, true, provider)

	var resp struct {
		Data struct {
			Mode         string                  `json:"mode"`
			Scenarios    []liveScenarioInfo      `json:"scenarios"`
			Optimization ports.OptimizationState `json:"optimization"`
		} `json:"data"`
	}
	if rec := getJSON(t, router, "/api/performance/live/config", &resp); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if resp.Data.Mode != "demo" {
		t.Errorf("mode = %q, want demo", resp.Data.Mode)
	}
	if len(resp.Data.Scenarios) != 1 || resp.Data.Scenarios[0].Name != "checkout-peak" ||
		resp.Data.Scenarios[0].DurationSec != 20 || resp.Data.Scenarios[0].QuerySet != "checkout" {
		t.Errorf("scenarios (unknown query sets hidden) = %+v", resp.Data.Scenarios)
	}
	opt := resp.Data.Optimization
	if !opt.Available || !opt.CanApply || !opt.Applied || len(opt.Statements) != 1 {
		t.Errorf("optimization = %+v", opt)
	}
}

func TestLiveConfig_TopologyEndpoint(t *testing.T) {
	router := newLiveConfigRouter(t, false, nil)

	var all struct {
		Data ports.Topology `json:"data"`
	}
	getJSON(t, router, "/api/performance/topology", &all)
	if len(all.Data.Nodes) != 3 || len(all.Data.Edges) != 2 {
		t.Errorf("full topology = %+v", all.Data)
	}

	var one struct {
		Data ports.Topology `json:"data"`
	}
	getJSON(t, router, "/api/performance/topology?query_set=reporting", &one)
	if len(one.Data.Nodes) != 1 || one.Data.Nodes[0].ID != "orders" {
		t.Errorf("query set topology = %+v", one.Data)
	}

	if rec := getJSON(t, router, "/api/performance/topology?query_set=missing", nil); rec.Code != http.StatusNotFound {
		t.Errorf("unknown query set status = %d, want 404", rec.Code)
	}
}

func TestLiveConfig_MappingEndpoint(t *testing.T) {
	router := newLiveConfigRouter(t, false, nil)
	var resp struct {
		Data ports.LabelTableMapping `json:"data"`
	}
	getJSON(t, router, "/api/performance/live/mapping", &resp)
	if resp.Data.Source != "transform_rules" || resp.Data.Labels["Order"].Table != "orders" {
		t.Errorf("mapping = %+v", resp.Data)
	}
	if resp.Data.Relationships == nil {
		t.Error("relationships must serialize as an object, not null")
	}
}

func TestBenchmarkHistoryEndpoint(t *testing.T) {
	ph, router := newLiveTestHandlers(t, map[string]ports.BenchmarkToolPort{"custom": &streamTool{n: 1, qps: 7, p95: 3}})

	var ids []string
	for i := 0; i < 3; i++ {
		id, err := ph.benchmarkService.ExecuteBenchmark(context.Background(), ports.BenchmarkConfig{
			Threads: 1, Duration: time.Second,
			CustomParams: map[string]interface{}{"query_set": "checkout", "labels": map[string]string{"index_applied": "true"}},
		}, "custom")
		if err != nil {
			t.Fatal(err)
		}
		waitFinished(t, ph, id)
		ids = append(ids, id)
		time.Sleep(5 * time.Millisecond)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Runs []performance.BenchmarkHistoryEntry `json:"runs"`
		} `json:"data"`
	}
	if rec := getJSON(t, router, "/api/performance/benchmarks/history?limit=2", &resp); rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	runs := resp.Data.Runs
	if len(runs) != 2 || runs[0].ID != ids[2] || runs[1].ID != ids[1] {
		t.Fatalf("runs = %+v, want the two newest first", runs)
	}
	if runs[0].QuerySet != "checkout" || runs[0].Labels["index_applied"] != "true" || runs[0].QPS != 7 || runs[0].Status != "completed" {
		t.Errorf("entry = %+v", runs[0])
	}

	var filtered struct {
		Data struct {
			Runs []performance.BenchmarkHistoryEntry `json:"runs"`
		} `json:"data"`
	}
	getJSON(t, router, "/api/performance/benchmarks/history?query_set=other", &filtered)
	if filtered.Data.Runs == nil || len(filtered.Data.Runs) != 0 {
		t.Errorf("filtered runs must be an empty array, got %+v", filtered.Data.Runs)
	}

	if rec := getJSON(t, router, "/api/performance/benchmarks/history?limit=abc", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid limit status = %d, want 400", rec.Code)
	}
}
