package performance

import (
	"context"
	"sort"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

const (
	// defaultHistoryLimit is the number of runs returned when no limit is given.
	defaultHistoryLimit = 20
	// maxHistoryLimit caps the number of runs returned by History.
	maxHistoryLimit = 200
)

// SetSampleStore configures persistent storage for live samples. It is safe to
// call at any time; nil disables persistence.
func (s *BenchmarkService) SetSampleStore(store ports.BenchmarkSampleStorePort) {
	s.sampleMutex.Lock()
	defer s.sampleMutex.Unlock()
	s.sampleStore = store
}

func (s *BenchmarkService) getSampleStore() ports.BenchmarkSampleStorePort {
	s.sampleMutex.RLock()
	defer s.sampleMutex.RUnlock()
	return s.sampleStore
}

// SetSampleObserver registers a callback invoked for every live sample after it
// was stamped and stored (for example to publish it on a WebSocket topic). The
// callback runs on the benchmark's sampling goroutine and must not block.
func (s *BenchmarkService) SetSampleObserver(observer func(ports.LiveSample)) {
	s.sampleMutex.Lock()
	defer s.sampleMutex.Unlock()
	s.sampleObserver = observer
}

// LatestLiveSample returns a copy of the most recent live sample of any run.
func (s *BenchmarkService) LatestLiveSample() (ports.LiveSample, bool) {
	sample := s.lastSample.Load()
	if sample == nil {
		return ports.LiveSample{}, false
	}
	return *sample, true
}

// afterSample runs for every stamped sample: persist, remember, notify.
func (s *BenchmarkService) afterSample(executionID string, sample ports.LiveSample) {
	s.lastSample.Store(&sample)

	if store := s.getSampleStore(); store != nil {
		if err := store.AppendSample(context.Background(), executionID, sample); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"execution_id": executionID,
				"seq":          sample.Seq,
			}).Warn("Failed to persist live sample")
		}
	}

	s.sampleMutex.RLock()
	observer := s.sampleObserver
	s.sampleMutex.RUnlock()
	if observer != nil {
		observer(sample)
	}
}

// storedSamples returns the persisted samples of a run with Seq > afterSeq.
// The boolean is false when nothing is stored for the run.
func (s *BenchmarkService) storedSamples(executionID string, afterSeq int) ([]ports.LiveSample, bool) {
	store := s.getSampleStore()
	if store == nil {
		return nil, false
	}
	ctx := context.Background()
	if !store.HasSamples(ctx, executionID) {
		return nil, false
	}
	samples, err := store.GetSamples(ctx, executionID, afterSeq)
	if err != nil {
		s.logger.WithError(err).WithField("execution_id", executionID).Warn("Failed to read persisted live samples")
		return nil, false
	}
	return samples, true
}

// applyResultLabels records run metadata on the result: the scenario (defaults
// to the query set), the query set and any labels passed by the caller in
// CustomParams["labels"].
func applyResultLabels(result *ports.BenchmarkResult, params map[string]interface{}) {
	set := func(key, value string) {
		if value == "" {
			return
		}
		if result.Labels == nil {
			result.Labels = make(map[string]string)
		}
		result.Labels[key] = value
	}

	querySet, _ := params["query_set"].(string)
	scenario, _ := params["scenario"].(string)
	if scenario == "" {
		scenario = querySet
	}
	set("scenario", scenario)
	set("query_set", querySet)

	switch labels := params["labels"].(type) {
	case map[string]string:
		for k, v := range labels {
			set(k, v)
		}
	case map[string]interface{}:
		for k, v := range labels {
			if str, ok := v.(string); ok {
				set(k, str)
			}
		}
	}
}

// BenchmarkHistoryEntry is one finished run in the run history.
type BenchmarkHistoryEntry struct {
	ID           string            `json:"id"`
	Scenario     string            `json:"scenario"`
	QuerySet     string            `json:"query_set"`
	Tool         string            `json:"tool"`
	Status       string            `json:"status"`
	StartedAt    time.Time         `json:"started_at"`
	DurationSecs float64           `json:"duration_seconds"`
	QPS          float64           `json:"qps"`
	AvgLatencyMs float64           `json:"avg_latency_ms"`
	P95LatencyMs float64           `json:"p95_latency_ms"`
	P99LatencyMs float64           `json:"p99_latency_ms"`
	TotalQueries int64             `json:"total_queries"`
	HasSamples   bool              `json:"has_samples"`
	Labels       map[string]string `json:"labels"`
}

// History returns finished runs, newest first, from memory and from the
// persistent result store. querySet filters by query set when not empty; limit
// defaults to 20 and is capped at 200.
func (s *BenchmarkService) History(ctx context.Context, limit int, querySet string) ([]BenchmarkHistoryEntry, error) {
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}

	seen := make(map[string]bool)
	results := make([]*ports.BenchmarkResult, 0)

	s.runsMutex.RLock()
	for _, execution := range s.activeRuns {
		execution.mutex.RLock()
		result := execution.Result
		execution.mutex.RUnlock()
		if result != nil {
			results = append(results, result)
			seen[result.ID] = true
		}
	}
	s.runsMutex.RUnlock()

	if store := s.getResultStore(); store != nil {
		stored, err := store.List(ctx, ports.BenchmarkResultFilter{})
		if err != nil {
			return nil, err
		}
		for _, result := range stored {
			if !seen[result.ID] {
				results = append(results, result)
				seen[result.ID] = true
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].StartTime.After(results[j].StartTime) })

	sampleStore := s.getSampleStore()
	entries := make([]BenchmarkHistoryEntry, 0, limit)
	for _, result := range results {
		if len(entries) >= limit {
			break
		}
		summary := SummarizeResult(result)
		runSet := result.Labels["query_set"]
		if runSet == "" {
			runSet = result.TestType
		}
		if querySet != "" && runSet != querySet {
			continue
		}
		labels := result.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		entries = append(entries, BenchmarkHistoryEntry{
			ID:           result.ID,
			Scenario:     summary.Scenario,
			QuerySet:     runSet,
			Tool:         result.ToolName,
			Status:       string(result.Status),
			StartedAt:    result.StartTime,
			DurationSecs: summary.DurationSecs,
			QPS:          summary.QPS,
			AvgLatencyMs: summary.AvgLatencyMs,
			P95LatencyMs: summary.P95LatencyMs,
			P99LatencyMs: summary.P99LatencyMs,
			TotalQueries: summary.TotalQueries,
			HasSamples:   sampleStore != nil && sampleStore.HasSamples(ctx, result.ID),
			Labels:       labels,
		})
	}
	return entries, nil
}
