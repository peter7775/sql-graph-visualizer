package api //nolint:revive // api is a clear and conventional package name

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"sql-graph-visualizer/internal/application/services/performance"

	"github.com/gorilla/mux"
)

const (
	// sseWriteTimeout bounds a single SSE write. It is re-armed before every
	// write so long-running streams are not cut by the server-wide
	// WriteTimeout.
	sseWriteTimeout = 15 * time.Second
	// sseKeepAliveInterval keeps idle connections (and proxies) alive.
	sseKeepAliveInterval = 15 * time.Second
)

// registerLiveRoutes registers the live benchmarking endpoints. The compare
// route must be registered before the generic {id} route.
func (ph *PerformanceHandlers) registerLiveRoutes(router *mux.Router) {
	router.HandleFunc("/api/performance/benchmarks/compare", ph.CompareBenchmarks).Methods("GET")
	router.HandleFunc("/api/performance/benchmarks/{id}/stream", ph.StreamBenchmark).Methods("GET")
	router.HandleFunc("/api/performance/benchmarks/{id}/samples", ph.GetBenchmarkSamples).Methods("GET")
}

// sseStatusEvent is the payload of the "status" SSE event.
type sseStatusEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Phase  string `json:"phase"`
}

// sseDoneEvent is the payload of the final "done" SSE event.
type sseDoneEvent struct {
	ID      string                        `json:"id"`
	Status  string                        `json:"status"`
	Error   string                        `json:"error"`
	Summary *performance.BenchmarkSummary `json:"summary,omitempty"`
}

// StreamBenchmark streams the live samples of a benchmark as Server-Sent
// Events. Buffered samples are replayed first (optionally only those after
// ?since=<seq> or the Last-Event-ID header), then new samples follow until the
// run ends, at which point "status" and "done" events are sent.
func (ph *PerformanceHandlers) StreamBenchmark(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	since := parseSinceSeq(r)

	backlog, live, cancel, err := ph.benchmarkService.SubscribeLive(id, since)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark not found", err.Error())
		return
	}
	defer cancel()

	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(event string, seq int, payload any) error {
		data, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return marshalErr
		}
		// Re-arm the write deadline; ignore "not supported" (e.g. test recorders).
		if dlErr := rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout)); dlErr != nil && !errors.Is(dlErr, http.ErrNotSupported) {
			return dlErr
		}
		if seq > 0 {
			if _, werr := fmt.Fprintf(w, "id: %d\n", seq); werr != nil {
				return werr
			}
		}
		if _, werr := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); werr != nil {
			return werr
		}
		if ferr := rc.Flush(); ferr != nil && !errors.Is(ferr, http.ErrNotSupported) {
			return ferr
		}
		return nil
	}

	status, _ := ph.benchmarkService.ExecutionStatus(id)
	if err := send("status", 0, sseStatusEvent{ID: id, Status: string(status), Phase: "connected"}); err != nil {
		return
	}
	for _, s := range backlog {
		if err := send("sample", s.Seq, s); err != nil {
			return
		}
	}

	keepAlive := time.NewTicker(sseKeepAliveInterval)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case s, ok := <-live:
			if !ok {
				ph.sendStreamDone(id, send)
				return
			}
			if err := send("sample", s.Seq, s); err != nil {
				return
			}
		case <-keepAlive.C:
			_ = rc.SetWriteDeadline(time.Now().Add(sseWriteTimeout))
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			_ = rc.Flush()
		}
	}
}

// sendStreamDone emits the terminal "status" and "done" events.
func (ph *PerformanceHandlers) sendStreamDone(id string, send func(string, int, any) error) {
	status, _ := ph.benchmarkService.ExecutionStatus(id)
	done := sseDoneEvent{ID: id, Status: string(status)}

	summary, err := ph.benchmarkService.GetBenchmarkSummary(context.Background(), id)
	if err == nil {
		done.Summary = summary
		done.Error = summary.ErrorMessage
		if status == "" {
			done.Status = summary.Status
		}
	} else {
		done.Error = err.Error()
	}
	_ = send("status", 0, sseStatusEvent{ID: id, Status: done.Status, Phase: "finished"})
	_ = send("done", 0, done)
}

// GetBenchmarkSamples is the polling fallback of the SSE stream.
func (ph *PerformanceHandlers) GetBenchmarkSamples(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	samples, status, err := ph.benchmarkService.GetLiveSamples(id, parseSinceSeq(r))
	if err != nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark not found", err.Error())
		return
	}
	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"status":  string(status),
			"samples": samples,
		},
		Timestamp: time.Now(),
	})
}

// BenchmarkDelta is the percentage change of run B relative to run A.
type BenchmarkDelta struct {
	QPSPct        float64 `json:"qps_pct"`
	AvgLatencyPct float64 `json:"avg_latency_pct"`
	P95Pct        float64 `json:"p95_pct"`
	P99Pct        float64 `json:"p99_pct"`
}

// CompareBenchmarks compares two finished runs: GET ...?a=<id>&b=<id>.
func (ph *PerformanceHandlers) CompareBenchmarks(w http.ResponseWriter, r *http.Request) {
	aID, bID := r.URL.Query().Get("a"), r.URL.Query().Get("b")
	if aID == "" || bID == "" {
		ph.sendErrorResponse(w, http.StatusBadRequest, "invalid_request", "Query parameters a and b are required", "")
		return
	}

	a, err := ph.benchmarkService.GetBenchmarkSummary(r.Context(), aID)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark a not available", err.Error())
		return
	}
	b, err := ph.benchmarkService.GetBenchmarkSummary(r.Context(), bID)
	if err != nil {
		ph.sendErrorResponse(w, http.StatusNotFound, "not_found", "Benchmark b not available", err.Error())
		return
	}

	ph.sendJSONResponse(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"a": a,
			"b": b,
			"delta": BenchmarkDelta{
				QPSPct:        pctChange(a.QPS, b.QPS),
				AvgLatencyPct: pctChange(a.AvgLatencyMs, b.AvgLatencyMs),
				P95Pct:        pctChange(a.P95LatencyMs, b.P95LatencyMs),
				P99Pct:        pctChange(a.P99LatencyMs, b.P99LatencyMs),
			},
		},
		Timestamp: time.Now(),
	})
}

// pctChange returns the percentage change from a to b (0 when a is 0).
func pctChange(a, b float64) float64 {
	if a == 0 {
		return 0
	}
	return (b - a) / a * 100
}

// parseSinceSeq reads the resume position from ?since= or Last-Event-ID.
func parseSinceSeq(r *http.Request) int {
	for _, raw := range []string{r.URL.Query().Get("since"), r.Header.Get("Last-Event-ID")} {
		if raw == "" {
			continue
		}
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			return n
		}
	}
	return 0
}
