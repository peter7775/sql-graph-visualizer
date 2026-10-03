package performance

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"sql-graph-visualizer/internal/application/ports"
)

// Latency histogram: logarithmic buckets (5% wide) over microseconds. It uses
// constant memory regardless of run length and supports lock-free recording
// from many worker goroutines.
const (
	histGrowth  = 1.05
	histBuckets = 440 // covers 1us .. well beyond 1000s
)

var histLogGrowth = math.Log(histGrowth)

type latencyHist struct {
	buckets [histBuckets]atomic.Int64
	count   atomic.Int64
	sumNs   atomic.Int64
	maxNs   atomic.Int64
}

func (h *latencyHist) record(d time.Duration) {
	ns := d.Nanoseconds()
	if ns < 0 {
		ns = 0
	}
	us := float64(ns) / 1000.0
	idx := 0
	if us > 1 {
		idx = int(math.Log(us) / histLogGrowth)
	}
	if idx >= histBuckets {
		idx = histBuckets - 1
	}
	h.buckets[idx].Add(1)
	h.count.Add(1)
	h.sumNs.Add(ns)
	for {
		cur := h.maxNs.Load()
		if ns <= cur || h.maxNs.CompareAndSwap(cur, ns) {
			break
		}
	}
}

// histSnapshot is a point-in-time copy of a histogram.
type histSnapshot struct {
	buckets [histBuckets]int64
	count   int64
	sumNs   int64
	maxNs   int64
}

// drain atomically takes the accumulated data and resets the histogram.
func (h *latencyHist) drain() histSnapshot {
	var s histSnapshot
	for i := range h.buckets {
		s.buckets[i] = h.buckets[i].Swap(0)
	}
	s.count = h.count.Swap(0)
	s.sumNs = h.sumNs.Swap(0)
	s.maxNs = h.maxNs.Swap(0)
	return s
}

// snapshot copies the current data without resetting it.
func (h *latencyHist) snapshot() histSnapshot {
	var s histSnapshot
	for i := range h.buckets {
		s.buckets[i] = h.buckets[i].Load()
	}
	s.count = h.count.Load()
	s.sumNs = h.sumNs.Load()
	s.maxNs = h.maxNs.Load()
	return s
}

func (s *histSnapshot) merge(o *histSnapshot) {
	for i := range s.buckets {
		s.buckets[i] += o.buckets[i]
	}
	s.count += o.count
	s.sumNs += o.sumNs
	if o.maxNs > s.maxNs {
		s.maxNs = o.maxNs
	}
}

func (s *histSnapshot) avgMs() float64 {
	if s.count == 0 {
		return 0
	}
	return float64(s.sumNs) / float64(s.count) / 1e6
}

func (s *histSnapshot) maxMs() float64 { return float64(s.maxNs) / 1e6 }

// percentileMs returns the p-th percentile (0..1) in milliseconds, using the
// geometric midpoint of the matching bucket and clamped to the observed max.
func (s *histSnapshot) percentileMs(p float64) float64 {
	if s.count == 0 {
		return 0
	}
	target := int64(math.Ceil(p * float64(s.count)))
	if target < 1 {
		target = 1
	}
	var cum int64
	for i, c := range s.buckets {
		cum += c
		if cum >= target {
			lowUs := math.Pow(histGrowth, float64(i))
			highUs := math.Pow(histGrowth, float64(i+1))
			ms := math.Sqrt(lowUs*highUs) / 1000.0
			if mx := s.maxMs(); ms > mx && mx > 0 {
				ms = mx
			}
			return ms
		}
	}
	return s.maxMs()
}

// liveQuery holds the static description and the live counters of one query.
type liveQuery struct {
	def      ports.CustomQueryDefinition
	stmtType string
	tables   []string
	interval latencyHist
	total    latencyHist
	errIntvl atomic.Int64
	errTotal atomic.Int64
}

// Rank thresholds (milliseconds) and heat scale used for the live graph.
const (
	liveFastBelowMs = 25.0
	liveSlowFromMs  = 100.0
	liveHeatLowMs   = 5.0
	liveHeatHighMs  = 500.0
)

// liveCollector records per-query latencies from worker goroutines and turns
// them into LiveSample snapshots.
type liveCollector struct {
	queries []*liveQuery
	threads int
	start   time.Time
	last    time.Time
	nodeIDs []string // stable ordering of every known table
	edges   [][2]string
}

func newLiveCollector(defs []ports.CustomQueryDefinition, threads int, start time.Time) *liveCollector {
	c := &liveCollector{threads: threads, start: start, last: start}
	tableSet := map[string]bool{}
	edgeSet := map[string]bool{}
	for _, d := range defs {
		q := &liveQuery{def: d, stmtType: statementTypeOf(d.Query), tables: queryTables(d)}
		c.queries = append(c.queries, q)
		for _, t := range q.tables {
			tableSet[t] = true
		}
		// The table list is ordered along the join path, so each adjacent
		// pair is one relationship (a chain, not a star around the first table).
		for i := 0; i+1 < len(q.tables); i++ {
			id := ports.GraphEdgeID(q.tables[i], q.tables[i+1])
			if !edgeSet[id] {
				edgeSet[id] = true
				c.edges = append(c.edges, [2]string{q.tables[i], q.tables[i+1]})
			}
		}
	}
	for t := range tableSet {
		c.nodeIDs = append(c.nodeIDs, t)
	}
	sort.Strings(c.nodeIDs)
	return c
}

func (c *liveCollector) record(idx int, d time.Duration, failed bool) {
	q := c.queries[idx]
	q.interval.record(d)
	q.total.record(d)
	if failed {
		q.errIntvl.Add(1)
		q.errTotal.Add(1)
	}
}

func rankFor(avgMs float64) string {
	switch {
	case avgMs < liveFastBelowMs:
		return "FAST"
	case avgMs < liveSlowFromMs:
		return "MEDIUM"
	default:
		return "SLOW"
	}
}

func heatFor(avgMs float64) float64 {
	if avgMs <= liveHeatLowMs {
		return 0
	}
	h := math.Log(avgMs/liveHeatLowMs) / math.Log(liveHeatHighMs/liveHeatLowMs)
	return math.Max(0, math.Min(1, h))
}

// sample drains the interval counters and builds a LiveSample. now is passed
// in so tests can control time.
func (c *liveCollector) sample(now time.Time) ports.LiveSample {
	intervalSec := now.Sub(c.last).Seconds()
	if intervalSec <= 0 {
		intervalSec = 1
	}
	c.last = now

	var all histSnapshot
	var totalErrors int64
	snaps := make([]histSnapshot, len(c.queries))
	for i, q := range c.queries {
		snaps[i] = q.interval.drain()
		all.merge(&snaps[i])
		totalErrors += q.errIntvl.Swap(0)
	}

	sample := ports.LiveSample{
		Timestamp:      now,
		ElapsedSeconds: now.Sub(c.start).Seconds(),
		Threads:        c.threads,
		Interval: ports.LiveIntervalStats{
			Queries:      all.count,
			QPS:          float64(all.count) / intervalSec,
			AvgLatencyMs: all.avgMs(),
			P50LatencyMs: all.percentileMs(0.50),
			P95LatencyMs: all.percentileMs(0.95),
			P99LatencyMs: all.percentileMs(0.99),
			MaxLatencyMs: all.maxMs(),
			Errors:       totalErrors,
		},
		Queries: make([]ports.LiveQueryStats, 0, len(c.queries)),
	}

	type agg struct {
		count int64
		sumNs int64
	}
	nodeAgg := map[string]*agg{}
	edgeAgg := map[string]*agg{}

	for i, q := range c.queries {
		s := &snaps[i]
		sample.Queries = append(sample.Queries, ports.LiveQueryStats{
			Pattern:      q.def.Query,
			Description:  q.def.Description,
			Type:         q.stmtType,
			Count:        s.count,
			QPS:          float64(s.count) / intervalSec,
			AvgLatencyMs: s.avgMs(),
			P95LatencyMs: s.percentileMs(0.95),
			Tables:       q.tables,
		})
		for _, t := range q.tables {
			a := nodeAgg[t]
			if a == nil {
				a = &agg{}
				nodeAgg[t] = a
			}
			a.count += s.count
			a.sumNs += s.sumNs
		}
		for i := 0; i+1 < len(q.tables); i++ {
			id := ports.GraphEdgeID(q.tables[i], q.tables[i+1])
			a := edgeAgg[id]
			if a == nil {
				a = &agg{}
				edgeAgg[id] = a
			}
			a.count += s.count
			a.sumNs += s.sumNs
		}
	}

	sample.Graph.Nodes = make([]ports.LiveGraphNode, 0, len(c.nodeIDs))
	for _, t := range c.nodeIDs {
		a := nodeAgg[t]
		node := ports.LiveGraphNode{ID: t, Table: t}
		if a != nil && a.count > 0 {
			node.Queries = a.count
			node.QPS = float64(a.count) / intervalSec
			node.AvgLatencyMs = float64(a.sumNs) / float64(a.count) / 1e6
			node.Heat = heatFor(node.AvgLatencyMs)
		}
		sample.Graph.Nodes = append(sample.Graph.Nodes, node)
	}

	sample.Graph.Edges = make([]ports.LiveGraphEdge, 0, len(c.edges))
	for _, e := range c.edges {
		id := ports.GraphEdgeID(e[0], e[1])
		edge := ports.LiveGraphEdge{ID: id, Source: e[0], Target: e[1], Rank: "FAST"}
		if a := edgeAgg[id]; a != nil && a.count > 0 {
			edge.QPS = float64(a.count) / intervalSec
			edge.AvgLatencyMs = float64(a.sumNs) / float64(a.count) / 1e6
			edge.Rank = rankFor(edge.AvgLatencyMs)
			if all.sumNs > 0 {
				edge.Load = math.Min(1, float64(a.sumNs)/float64(all.sumNs))
			}
		}
		sample.Graph.Edges = append(sample.Graph.Edges, edge)
	}
	return sample
}

// finalStats converts the cumulative counters into the shared result shapes.
func (c *liveCollector) finalStats(wallClock time.Duration) (*ports.PerformanceMetrics, []ports.QueryPerformance) {
	metrics := &ports.PerformanceMetrics{}
	results := make([]ports.QueryPerformance, 0, len(c.queries))

	var all histSnapshot
	var totalErrors int64
	var minNs int64
	for _, q := range c.queries {
		s := q.total.snapshot()
		if s.count == 0 {
			continue
		}
		all.merge(&s)
		totalErrors += q.errTotal.Load()

		avg := time.Duration(s.sumNs / s.count)
		results = append(results, ports.QueryPerformance{
			QueryPattern:      q.def.Query,
			QueryType:         q.stmtType,
			ExecutionCount:    s.count,
			TotalTime:         time.Duration(s.sumNs),
			AverageTime:       avg,
			MaxTime:           time.Duration(s.maxNs),
			SourceTables:      q.tables[:min(1, len(q.tables))],
			JoinedTables:      q.tables[min(1, len(q.tables)):],
			RelationshipType:  "CUSTOM_QUERY",
			PerformanceImpact: classifyCustomQueryImpact(avg),
		})
		if minNs == 0 || avg.Nanoseconds() < minNs {
			minNs = avg.Nanoseconds()
		}
	}

	if wallClock > 0 {
		metrics.QueriesPerSecond = float64(all.count) / wallClock.Seconds()
		metrics.TransactionsPerSec = metrics.QueriesPerSecond
	}
	if all.count > 0 {
		metrics.AverageLatency = all.avgMs()
		metrics.MinLatency = float64(minNs) / 1e6
		metrics.MaxLatency = all.maxMs()
		metrics.Percentile50 = all.percentileMs(0.50)
		metrics.Percentile95 = all.percentileMs(0.95)
		metrics.Percentile99 = all.percentileMs(0.99)
		metrics.ErrorRate = float64(totalErrors) / float64(all.count) * 100
	}
	metrics.TotalErrors = int(totalErrors)
	return metrics, results
}

func statementTypeOf(query string) string {
	if fields := strings.Fields(strings.TrimSpace(query)); len(fields) > 0 {
		return strings.ToUpper(fields[0])
	}
	return "SELECT"
}

var tableRefRe = regexp.MustCompile("(?i)\\b(?:from|join|into|update)\\s+`?([a-z_][a-z0-9_]*)`?")

// queryTables returns the tables used by a query: the configured list if
// present, otherwise tables found in FROM/JOIN/INTO/UPDATE clauses in order
// of first appearance.
func queryTables(def ports.CustomQueryDefinition) []string {
	if len(def.Tables) > 0 {
		out := make([]string, 0, len(def.Tables))
		for _, t := range def.Tables {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range tableRefRe.FindAllStringSubmatch(def.Query, -1) {
		name := m[1]
		if strings.EqualFold(name, "select") || strings.EqualFold(name, "dual") || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		out = append(out, name)
	}
	return out
}
