package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

type testIndexAdvisor struct {
	err error
}

type pendingIndexTool struct{ release chan struct{} }

func (tool *pendingIndexTool) Execute(ctx context.Context, _ ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-tool.release:
		return &ports.BenchmarkResult{}, nil
	}
}
func (*pendingIndexTool) Validate(ports.BenchmarkConfig) error { return nil }
func (*pendingIndexTool) GetSupportedTests() []string          { return []string{"set"} }
func (*pendingIndexTool) IsAvailable() bool                    { return true }
func (*pendingIndexTool) GetVersion() (string, error)          { return "test", nil }

func TestOptimizationSuggestionsRunning(t *testing.T) {
	tool := &pendingIndexTool{release: make(chan struct{})}
	defer close(tool.release)
	ph, _ := newLiveTestHandlers(t, map[string]ports.BenchmarkToolPort{"custom": tool})
	id := startRun(t, ph, "custom")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status, _ := ph.benchmarkService.ExecutionStatus(id)
		if status == ports.BenchmarkStatusRunning || status == ports.BenchmarkStatusPending {
			break
		}
		time.Sleep(time.Millisecond)
	}
	handler := NewOptimizationHandlers(logrus.New(), ph.benchmarkService, testIndexAdvisor{}, nil)
	recorder := httptest.NewRecorder()
	handler.Suggestions(recorder, httptest.NewRequest(http.MethodGet, "/?benchmark_id="+id, nil))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409 for active run, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func (a testIndexAdvisor) SuggestIndexes(context.Context, *ports.BenchmarkResult, []ports.CustomQueryDefinition) ([]ports.IndexSuggestion, error) {
	return nil, a.err
}

func TestOptimizationSuggestions(t *testing.T) {
	ph, router := newLiveTestHandlers(t, map[string]ports.BenchmarkToolPort{"custom": &streamTool{n: 1}})
	id := startRun(t, ph, "custom")
	waitFinished(t, ph, id)
	handler := NewOptimizationHandlers(logrus.New(), ph.benchmarkService, testIndexAdvisor{}, nil)
	handler.RegisterRoutes(router)
	for _, test := range []struct {
		id   string
		code int
	}{
		{"", http.StatusBadRequest},
		{"missing", http.StatusNotFound},
		{id, http.StatusOK},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/performance/optimization/suggestions?benchmark_id="+test.id, nil))
		if recorder.Code != test.code {
			t.Fatalf("status=%d want=%d body=%s", recorder.Code, test.code, recorder.Body.String())
		}
		if test.code == http.StatusOK && !strings.Contains(recorder.Body.String(), `"suggestions":[]`) {
			t.Fatalf("expected empty JSON array: %s", recorder.Body.String())
		}
	}
	handler.advisor = testIndexAdvisor{err: errors.New("catalog unavailable")}
	recorder := httptest.NewRecorder()
	handler.Suggestions(recorder, httptest.NewRequest(http.MethodGet, "/?benchmark_id="+id, nil))
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "index_advisor_error") {
		t.Fatalf("expected readable advisor failure: %s", recorder.Body.String())
	}
}
