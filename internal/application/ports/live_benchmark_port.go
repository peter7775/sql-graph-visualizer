package ports

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StreamingBenchmarkTool is implemented by benchmark tools that can report
// per-interval samples while a run is in progress (in addition to the final
// BenchmarkResult). The emit callback is invoked roughly once per sample
// interval from a single goroutine and must not block for long.
type StreamingBenchmarkTool interface {
	BenchmarkToolPort

	// ExecuteStream behaves like Execute but additionally emits LiveSample
	// values while running. Sequence number and benchmark ID are assigned by
	// the caller; the tool fills in everything else.
	ExecuteStream(ctx context.Context, config BenchmarkConfig, emit func(LiveSample)) (*BenchmarkResult, error)
}

// LiveSample is a one-interval snapshot of a running benchmark. Its JSON
// shape is consumed directly by the live benchmarking UI.
type LiveSample struct {
	BenchmarkID    string            `json:"benchmark_id"`
	Seq            int               `json:"seq"`
	Timestamp      time.Time         `json:"timestamp"`
	ElapsedSeconds float64           `json:"elapsed_seconds"`
	Threads        int               `json:"threads"`
	Interval       LiveIntervalStats `json:"interval"`
	Queries        []LiveQueryStats  `json:"queries"`
	Graph          LiveGraph         `json:"graph"`
}

// LiveIntervalStats aggregates all queries executed during one interval.
type LiveIntervalStats struct {
	Queries      int64   `json:"queries"`
	QPS          float64 `json:"qps"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	MaxLatencyMs float64 `json:"max_latency_ms"`
	Errors       int64   `json:"errors"`
}

// LiveQueryStats describes one configured query during an interval.
type LiveQueryStats struct {
	Pattern      string   `json:"pattern"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Count        int64    `json:"count"`
	QPS          float64  `json:"qps"`
	AvgLatencyMs float64  `json:"avg_latency_ms"`
	P95LatencyMs float64  `json:"p95_latency_ms"`
	Tables       []string `json:"tables"`
}

// LiveGraph is the per-interval table/relationship performance graph.
type LiveGraph struct {
	Nodes []LiveGraphNode `json:"nodes"`
	Edges []LiveGraphEdge `json:"edges"`
}

// LiveGraphNode is a table with its activity during the interval. Heat is a
// 0..1 absolute latency indicator.
type LiveGraphNode struct {
	ID           string  `json:"id"`
	Table        string  `json:"table"`
	QPS          float64 `json:"qps"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Heat         float64 `json:"heat"`
	Queries      int64   `json:"queries"`
}

// LiveGraphEdge is a join relationship between two tables. Load is the 0..1
// share of total database busy time spent in queries using this relationship;
// Rank is an absolute FAST/MEDIUM/SLOW classification of its latency.
type LiveGraphEdge struct {
	ID           string  `json:"id"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	QPS          float64 `json:"qps"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	Load         float64 `json:"load"`
	Rank         string  `json:"rank"`
}

// GraphEdgeID returns the canonical, order-insensitive identifier of the
// relationship between two tables. Samples and the demo topology both use it
// so the UI can match edges by ID.
func GraphEdgeID(a, b string) string {
	pair := []string{strings.ToLower(a), strings.ToLower(b)}
	sort.Strings(pair)
	return pair[0] + "-" + pair[1]
}
