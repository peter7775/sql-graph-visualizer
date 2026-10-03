package performance

import (
	"context"
	"fmt"
	"sync"

	"sql-graph-visualizer/internal/application/ports"
)

// maxLiveSamplesPerRun bounds the replay buffer of one run (about 2 hours at
// the default 1 s interval).
const maxLiveSamplesPerRun = 7200

// liveSubscriberBuffer is the per-subscriber channel buffer. A subscriber that
// falls this far behind starts losing samples (it can resync via /samples).
const liveSubscriberBuffer = 256

// liveHub stores the samples of one benchmark run and fans them out to
// subscribers. The replay backlog and the live channel are handed out
// atomically, so a subscriber never misses or duplicates a sample.
type liveHub struct {
	mu      sync.Mutex
	samples []ports.LiveSample
	nextSeq int
	subs    map[chan ports.LiveSample]struct{}
	closed  bool
	id      string
}

func newLiveHub(benchmarkID string) *liveHub {
	return &liveHub{id: benchmarkID, nextSeq: 1, subs: make(map[chan ports.LiveSample]struct{})}
}

// publish assigns the sequence number and benchmark ID, stores the sample and
// delivers it to all subscribers without blocking.
func (h *liveHub) publish(s ports.LiveSample) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	s.BenchmarkID = h.id
	s.Seq = h.nextSeq
	h.nextSeq++
	if len(h.samples) >= maxLiveSamplesPerRun {
		h.samples = h.samples[1:]
	}
	h.samples = append(h.samples, s)
	for ch := range h.subs {
		select {
		case ch <- s:
		default: // slow consumer: drop, it can resync from the backlog
		}
	}
}

// since returns the stored samples with Seq > afterSeq.
func (h *liveHub) since(afterSeq int) []ports.LiveSample {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sinceLocked(afterSeq)
}

func (h *liveHub) sinceLocked(afterSeq int) []ports.LiveSample {
	out := make([]ports.LiveSample, 0, len(h.samples))
	for _, s := range h.samples {
		if s.Seq > afterSeq {
			out = append(out, s)
		}
	}
	return out
}

// subscribe returns the backlog (Seq > afterSeq) and a channel with all
// following samples. The channel is closed when the run ends. cancel must be
// called when the subscriber goes away.
func (h *liveHub) subscribe(afterSeq int) (backlog []ports.LiveSample, ch <-chan ports.LiveSample, cancel func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	backlog = h.sinceLocked(afterSeq)

	c := make(chan ports.LiveSample, liveSubscriberBuffer)
	if h.closed {
		close(c)
		return backlog, c, func() {}
	}
	h.subs[c] = struct{}{}
	return backlog, c, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subs[c]; ok {
			delete(h.subs, c)
			close(c)
		}
	}
}

// close ends the stream: all subscriber channels are closed.
func (h *liveHub) close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for c := range h.subs {
		close(c)
		delete(h.subs, c)
	}
}

// BenchmarkSummary is the compact result shape used by the live UI and the
// run comparison endpoint.
type BenchmarkSummary struct {
	ID            string  `json:"id"`
	Scenario      string  `json:"scenario"`
	QPS           float64 `json:"qps"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	ErrorRate     float64 `json:"error_rate"`
	TotalQueries  int64   `json:"total_queries"`
	DurationSecs  float64 `json:"duration_seconds,omitempty"`
	Status        string  `json:"status,omitempty"`
	ErrorMessage  string  `json:"error,omitempty"`
	StartedAtUnix int64   `json:"started_at_unix,omitempty"`
}

// SummarizeResult converts a benchmark result into a BenchmarkSummary.
func SummarizeResult(r *ports.BenchmarkResult) *BenchmarkSummary {
	s := &BenchmarkSummary{
		ID:           r.ID,
		Scenario:     r.Labels["scenario"],
		Status:       string(r.Status),
		ErrorMessage: r.Error,
		DurationSecs: r.Duration.Seconds(),
	}
	if s.Scenario == "" {
		s.Scenario = r.TestType
	}
	if m := r.Metrics; m != nil {
		s.QPS = m.QueriesPerSecond
		s.AvgLatencyMs = m.AverageLatency
		s.P50LatencyMs = m.Percentile50
		s.P95LatencyMs = m.Percentile95
		s.P99LatencyMs = m.Percentile99
		s.ErrorRate = m.ErrorRate
	}
	for _, q := range r.QueryResults {
		s.TotalQueries += q.ExecutionCount
	}
	if !r.StartTime.IsZero() {
		s.StartedAtUnix = r.StartTime.Unix()
	}
	return s
}

// SubscribeLive attaches to the live stream of a run. See liveHub.subscribe.
func (s *BenchmarkService) SubscribeLive(executionID string, afterSeq int) ([]ports.LiveSample, <-chan ports.LiveSample, func(), error) {
	hub, err := s.liveHubFor(executionID)
	if err != nil {
		return nil, nil, nil, err
	}
	backlog, ch, cancel := hub.subscribe(afterSeq)
	return backlog, ch, cancel, nil
}

// GetLiveSamples returns the buffered samples of a run with Seq > afterSeq and
// the current run status.
func (s *BenchmarkService) GetLiveSamples(executionID string, afterSeq int) ([]ports.LiveSample, ports.BenchmarkStatus, error) {
	hub, err := s.liveHubFor(executionID)
	if err != nil {
		return nil, "", err
	}
	status, _ := s.executionStatus(executionID)
	return hub.since(afterSeq), status, nil
}

// GetBenchmarkSummary returns the summary of a finished run, from memory or
// from the persistent result store.
func (s *BenchmarkService) GetBenchmarkSummary(ctx context.Context, executionID string) (*BenchmarkSummary, error) {
	s.runsMutex.RLock()
	execution, ok := s.activeRuns[executionID]
	s.runsMutex.RUnlock()
	if ok {
		execution.mutex.RLock()
		result := execution.Result
		execution.mutex.RUnlock()
		if result != nil {
			return SummarizeResult(result), nil
		}
		return nil, fmt.Errorf("benchmark %s has not finished yet", executionID)
	}
	if store := s.getResultStore(); store != nil {
		result, err := store.Get(ctx, executionID)
		if err == nil && result != nil {
			return SummarizeResult(result), nil
		}
	}
	return nil, fmt.Errorf("benchmark %s not found", executionID)
}

// ExecutionStatus returns the current status of a run held in memory.
func (s *BenchmarkService) ExecutionStatus(executionID string) (ports.BenchmarkStatus, bool) {
	return s.executionStatus(executionID)
}

func (s *BenchmarkService) executionStatus(executionID string) (ports.BenchmarkStatus, bool) {
	s.runsMutex.RLock()
	execution, ok := s.activeRuns[executionID]
	s.runsMutex.RUnlock()
	if !ok {
		return "", false
	}
	execution.mutex.RLock()
	defer execution.mutex.RUnlock()
	return execution.Status, true
}

func (s *BenchmarkService) liveHubFor(executionID string) (*liveHub, error) {
	s.runsMutex.RLock()
	execution, ok := s.activeRuns[executionID]
	s.runsMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}
	return execution.live, nil
}
