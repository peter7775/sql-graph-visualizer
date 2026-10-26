package api //nolint:revive // api is a clear and conventional package name

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// LiveConfigDeps are the dependencies of LiveConfigHandlers.
type LiveConfigDeps struct {
	Logger   *logrus.Logger
	Topology *performance.TopologyService
	// Mapping maps Neo4j labels and relationship types to the table graph.
	Mapping ports.LabelTableMapping
	// QuerySets are the configured custom query sets.
	QuerySets map[string]ports.CustomBenchmarkConfig
	// LiveDemo is the live_demo configuration (may be nil).
	LiveDemo *models.LiveDemoConfig
	// DemoMode reports whether the demo API is served (DEMO_MODE=true).
	DemoMode bool
	// DatabaseType is "mysql" or "postgresql" (or another configured type).
	DatabaseType string
	// Optimization reports the state of the demo optimization; nil means that
	// no optimization is available.
	Optimization func(ctx context.Context) ports.OptimizationState
}

// LiveConfigHandlers serves what the live benchmark UI needs to work for any
// configured query set: capabilities, query sets, topology and the mapping
// between the Neo4j graph and the table graph.
type LiveConfigHandlers struct {
	deps LiveConfigDeps
}

// NewLiveConfigHandlers creates the handlers.
func NewLiveConfigHandlers(deps LiveConfigDeps) *LiveConfigHandlers {
	return &LiveConfigHandlers{deps: deps}
}

// RegisterRoutes registers the endpoints.
func (h *LiveConfigHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/performance/live/config", h.GetLiveConfig).Methods("GET")
	router.HandleFunc("/api/performance/live/mapping", h.GetMapping).Methods("GET")
	router.HandleFunc("/api/performance/topology", h.GetTopology).Methods("GET")
}

type liveQuerySetInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Threads     int      `json:"threads"`
	DurationSec int      `json:"duration_seconds"`
	QueryCount  int      `json:"query_count"`
	Tables      []string `json:"tables"`
}

type liveScenarioInfo struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	QuerySet    string `json:"query_set"`
	Threads     int    `json:"threads"`
	DurationSec int    `json:"duration_seconds"`
}

// GetLiveConfig returns the capabilities and content of the live benchmark UI.
func (h *LiveConfigHandlers) GetLiveConfig(w http.ResponseWriter, r *http.Request) {
	mode := "generic"
	if h.deps.DemoMode {
		mode = "demo"
	}

	topology, err := h.deps.Topology.Build(r.Context(), "")
	if err != nil {
		h.fail(w, http.StatusInternalServerError, "topology_error", "Failed to build the topology", err.Error())
		return
	}

	optimization := ports.OptimizationState{}
	if h.deps.Optimization != nil {
		optimization = h.deps.Optimization(r.Context())
	}

	respond(h.deps.Logger, w, http.StatusOK, map[string]any{
		"mode":          mode,
		"database_type": h.deps.DatabaseType,
		"query_sets":    h.querySets(),
		"scenarios":     h.scenarios(),
		"optimization":  optimization,
		"topology":      topology,
	})
}

func (h *LiveConfigHandlers) querySets() []liveQuerySetInfo {
	names := make([]string, 0, len(h.deps.QuerySets))
	for name := range h.deps.QuerySets {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]liveQuerySetInfo, 0, len(names))
	for _, name := range names {
		set := h.deps.QuerySets[name]
		out = append(out, liveQuerySetInfo{
			Name:        name,
			Description: set.Description,
			Threads:     set.Threads,
			DurationSec: int(set.Duration.Seconds()),
			QueryCount:  len(set.Queries),
			Tables:      performance.QuerySetTables(set),
		})
	}
	return out
}

func (h *LiveConfigHandlers) scenarios() []liveScenarioInfo {
	out := []liveScenarioInfo{}
	if !h.deps.DemoMode || h.deps.LiveDemo == nil {
		return out
	}
	for _, s := range h.deps.LiveDemo.Scenarios {
		if s.Name == "" || s.QuerySet == "" {
			continue
		}
		if _, ok := h.deps.QuerySets[s.QuerySet]; !ok {
			continue
		}
		title := s.Title
		if title == "" {
			title = s.Name
		}
		out = append(out, liveScenarioInfo{
			Name: s.Name, Title: title, Description: s.Description, QuerySet: s.QuerySet,
			Threads: s.Threads, DurationSec: int(scenarioDuration(s).Seconds()),
		})
	}
	return out
}

// GetTopology returns the table graph skeleton, optionally restricted to one
// query set with ?query_set=NAME.
func (h *LiveConfigHandlers) GetTopology(w http.ResponseWriter, r *http.Request) {
	topology, err := h.deps.Topology.Build(r.Context(), r.URL.Query().Get("query_set"))
	if err != nil {
		if errors.Is(err, performance.ErrUnknownQuerySet) {
			h.fail(w, http.StatusNotFound, "not_found", "Unknown query set", err.Error())
			return
		}
		h.fail(w, http.StatusInternalServerError, "topology_error", "Failed to build the topology", err.Error())
		return
	}
	respond(h.deps.Logger, w, http.StatusOK, topology)
}

// GetMapping returns the Neo4j label and relationship mapping derived from the
// transformation rules.
func (h *LiveConfigHandlers) GetMapping(w http.ResponseWriter, _ *http.Request) {
	respond(h.deps.Logger, w, http.StatusOK, h.deps.Mapping)
}

func (h *LiveConfigHandlers) fail(w http.ResponseWriter, status int, code, message, details string) {
	h.deps.Logger.WithFields(logrus.Fields{"code": code, "details": details}).Warn(message)
	writeJSON(h.deps.Logger, w, status, Response{
		Success:   false,
		Error:     &Error{Code: code, Message: message, Details: details},
		Timestamp: time.Now(),
	})
}
