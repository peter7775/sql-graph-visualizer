package api //nolint:revive // conventional API package name

import (
	"encoding/json"
	"net/http"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// OptimizationHandlers serves measured benchmark index recommendations.
type OptimizationHandlers struct {
	logger    *logrus.Logger
	service   *performance.BenchmarkService
	advisor   ports.IndexAdvisorPort
	querySets map[string]ports.CustomBenchmarkConfig
}

// NewOptimizationHandlers constructs the read-only suggestions API.
func NewOptimizationHandlers(logger *logrus.Logger, svc *performance.BenchmarkService, advisor ports.IndexAdvisorPort, querySets map[string]ports.CustomBenchmarkConfig) *OptimizationHandlers {
	return &OptimizationHandlers{logger: logger, service: svc, advisor: advisor, querySets: querySets}
}

// RegisterRoutes registers the suggestions endpoint.
func (h *OptimizationHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/performance/optimization/suggestions", h.Suggestions).Methods(http.MethodGet)
}

// Suggestions returns recommendations for a finished run.
func (h *OptimizationHandlers) Suggestions(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("benchmark_id")
	fail := func(status int, code, message string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(Response{Success: false, Error: &Error{Code: code, Message: message}, Timestamp: time.Now()})
	}
	if id == "" {
		fail(http.StatusBadRequest, "invalid_request", "benchmark_id is required")
		return
	}
	if h.service == nil {
		fail(http.StatusServiceUnavailable, "unavailable", "Benchmark service is unavailable")
		return
	}
	if status, ok := h.service.ExecutionStatus(id); ok && (status == ports.BenchmarkStatusRunning || status == ports.BenchmarkStatusPending) {
		fail(http.StatusConflict, "benchmark_running", "Benchmark has not finished yet")
		return
	}
	result, err := h.service.FindResult(r.Context(), id)
	if err != nil || result == nil {
		fail(http.StatusNotFound, "not_found", "Benchmark not found")
		return
	}
	suggestions := []ports.IndexSuggestion{}
	if h.advisor != nil {
		suggestions, err = h.advisor.SuggestIndexes(r.Context(), result, h.querySets[result.TestType].Queries)
		if err != nil {
			h.logger.WithError(err).Warn("Index recommendations failed")
			fail(http.StatusInternalServerError, "index_advisor_error", "Failed to analyze index recommendations")
			return
		}
		if suggestions == nil {
			suggestions = []ports.IndexSuggestion{}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: map[string]any{"benchmark_id": id, "suggestions": suggestions}, Timestamp: time.Now()})
}
