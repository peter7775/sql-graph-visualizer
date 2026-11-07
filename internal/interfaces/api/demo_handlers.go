package api //nolint:revive // api is a clear and conventional package name

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"sql-graph-visualizer/internal/application/ports"
	"sql-graph-visualizer/internal/application/services/performance"
	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	identPattern = `[A-Za-z_][A-Za-z0-9_]*`
	// ddlTimeout bounds one CREATE/DROP INDEX statement.
	ddlTimeout = 10 * time.Minute
)

// Only these two statement shapes are ever executed by the demo optimization
// endpoints, regardless of what the configuration contains.
var (
	createIndexRe = regexp.MustCompile(`(?i)^\s*CREATE\s+INDEX\s+(` + identPattern + `)\s+ON\s+(` + identPattern + `)\s*\(\s*` +
		identPattern + `(?:\s*,\s*` + identPattern + `)*\s*\)\s*;?\s*$`)
	dropIndexRe = regexp.MustCompile(`(?i)^\s*DROP\s+INDEX\s+(` + identPattern + `)\s+ON\s+(` + identPattern + `)\s*;?\s*$`)
)

// DemoHandlers serves the live benchmarking demo API: scenario presets, the
// table topology and a reversible one-click index optimization. It must only
// be registered when DEMO_MODE=true.
type DemoHandlers struct {
	logger           *logrus.Logger
	benchmarkService *performance.BenchmarkService
	db               *sql.DB
	demo             *models.LiveDemoConfig
	inspector        ports.IndexInspector

	scenarios map[string]models.LiveDemoScenario
	optMu     sync.Mutex
}

func (d *DemoHandlers) optimizationSpecs() ([]ports.IndexSpec, error) {
	opt := d.demo.Optimization
	specs := []ports.IndexSpec{}
	if opt == nil {
		return specs, nil
	}
	if len(opt.Indexes) > 0 {
		for _, index := range opt.Indexes {
			spec := ports.IndexSpec{Name: index.Name, Table: index.Table, Columns: index.Columns}
			if _, err := d.inspector.BuildCreateDDL(spec); err != nil {
				return nil, err
			}
			specs = append(specs, spec)
		}
		return specs, nil
	}
	if err := validateOptimization(opt); err != nil {
		return nil, err
	}
	for _, statement := range opt.Apply {
		m := createIndexRe.FindStringSubmatch(statement)
		start, end := strings.Index(statement, "("), strings.LastIndex(statement, ")")
		columns := strings.Split(statement[start+1:end], ",")
		for n := range columns {
			columns[n] = strings.TrimSpace(columns[n])
		}
		spec := ports.IndexSpec{Name: m[1], Table: m[2], Columns: columns}
		if _, err := d.inspector.BuildCreateDDL(spec); err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// OptimizationState exposes validated optimization metadata without writing an HTTP response.
func (d *DemoHandlers) OptimizationState(ctx context.Context) ports.OptimizationState {
	state := ports.OptimizationState{}
	opt := d.demo.Optimization
	if opt == nil {
		return state
	}
	specs, err := d.optimizationSpecs()
	if err != nil || len(specs) == 0 {
		return state
	}
	state.Available, state.CanApply = true, true
	state.Name, state.Title, state.Description = opt.Name, opt.Title, opt.Description
	if state.Title == "" {
		state.Title = opt.Name
	}
	state.Applied = true
	state.Statements = []string{}
	for _, spec := range specs {
		ddl, _ := d.inspector.BuildCreateDDL(spec)
		state.Statements = append(state.Statements, ddl)
		exists, inspectErr := d.inspector.IndexExists(ctx, spec.Table, spec.Name)
		state.Applied = state.Applied && inspectErr == nil && exists
	}
	return state
}

// NewDemoHandlers validates the demo configuration and creates the handlers.
// An invalid optimization section disables only the optimization endpoints.
func NewDemoHandlers(logger *logrus.Logger, svc *performance.BenchmarkService, db *sql.DB, demo *models.LiveDemoConfig, inspectors ...ports.IndexInspector) *DemoHandlers {
	inspector, _ := performance.NewIndexInspector(db, "mysql")
	if len(inspectors) > 0 && inspectors[0] != nil {
		inspector = inspectors[0]
	}
	if demo == nil {
		demo = &models.LiveDemoConfig{}
	}
	d := &DemoHandlers{
		logger:           logger,
		benchmarkService: svc,
		db:               db,
		demo:             demo,
		inspector:        inspector,
		scenarios:        make(map[string]models.LiveDemoScenario, len(demo.Scenarios)),
	}
	for _, s := range demo.Scenarios {
		if s.Name == "" || s.QuerySet == "" {
			logger.Warnf("Ignoring demo scenario with missing name or query_set: %+v", s)
			continue
		}
		d.scenarios[s.Name] = s
	}
	if demo.Optimization != nil {
		if _, err := d.optimizationSpecs(); err != nil {
			logger.Errorf("Demo optimization disabled: %v", err)
			d.demo = &models.LiveDemoConfig{
				Enabled: demo.Enabled, Scenarios: demo.Scenarios, Topology: demo.Topology,
			}
		}
	}
	return d
}

// RegisterRoutes registers the demo endpoints.
func (d *DemoHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/demo/scenarios", d.ListScenarios).Methods("GET")
	router.HandleFunc("/api/demo/scenarios/{name}/run", d.RunScenario).Methods("POST")
	router.HandleFunc("/api/demo/topology", d.GetTopology).Methods("GET")
	router.HandleFunc("/api/demo/optimization", d.GetOptimization).Methods("GET")
	router.HandleFunc("/api/demo/optimization/apply", d.ApplyOptimization).Methods("POST")
	router.HandleFunc("/api/demo/optimization/revert", d.RevertOptimization).Methods("POST")
}

// validateOptimization checks that every statement is a plain CREATE INDEX
// (apply) or DROP INDEX (revert) statement.
func validateOptimization(o *models.LiveDemoOptimization) error {
	if len(o.Apply) == 0 || len(o.Revert) == 0 {
		return errors.New("optimization needs at least one apply and one revert statement")
	}
	for _, s := range o.Apply {
		if !createIndexRe.MatchString(s) {
			return fmt.Errorf("apply statement is not an allowed CREATE INDEX: %q", s)
		}
	}
	for _, s := range o.Revert {
		if !dropIndexRe.MatchString(s) {
			return fmt.Errorf("revert statement is not an allowed DROP INDEX: %q", s)
		}
	}
	return nil
}

// Scenario handlers

type demoScenarioInfo struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Threads     int    `json:"threads"`
	DurationSec int    `json:"duration_seconds"`
}

func scenarioDuration(s models.LiveDemoScenario) time.Duration {
	if dur, err := time.ParseDuration(s.Duration); err == nil && dur > 0 {
		return dur
	}
	return 30 * time.Second
}

// ListScenarios returns the configured scenario presets in config order.
func (d *DemoHandlers) ListScenarios(w http.ResponseWriter, _ *http.Request) {
	out := make([]demoScenarioInfo, 0, len(d.demo.Scenarios))
	for _, s := range d.demo.Scenarios {
		if _, ok := d.scenarios[s.Name]; !ok {
			continue
		}
		title := s.Title
		if title == "" {
			title = s.Name
		}
		out = append(out, demoScenarioInfo{
			Name: s.Name, Title: title, Description: s.Description,
			Threads: s.Threads, DurationSec: int(scenarioDuration(s).Seconds()),
		})
	}
	d.respond(w, http.StatusOK, out)
}

type runScenarioRequest struct {
	Threads         int `json:"threads"`
	DurationSeconds int `json:"duration_seconds"`
}

// RunScenario starts the query set of a scenario as a benchmark run.
func (d *DemoHandlers) RunScenario(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	sc, ok := d.scenarios[name]
	if !ok {
		d.fail(w, http.StatusNotFound, "not_found", "Unknown scenario", name)
		return
	}

	var req runScenarioRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			d.fail(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body", err.Error())
			return
		}
	}

	threads := sc.Threads
	if req.Threads > 0 {
		threads = req.Threads
	}
	duration := scenarioDuration(sc)
	if req.DurationSeconds > 0 {
		duration = time.Duration(req.DurationSeconds) * time.Second
	}

	started := time.Now()
	id, err := d.benchmarkService.ExecuteBenchmark(r.Context(), ports.BenchmarkConfig{
		Threads:  threads,
		Duration: duration,
		CustomParams: map[string]interface{}{
			"query_set": sc.QuerySet,
			"scenario":  sc.Name,
			"labels":    map[string]string{"index_applied": fmt.Sprint(d.OptimizationState(r.Context()).Applied)},
		},
	}, "custom")
	if err != nil {
		d.fail(w, http.StatusInternalServerError, "benchmark_error", "Failed to start scenario", err.Error())
		return
	}

	d.respond(w, http.StatusCreated, map[string]any{
		"id":         id,
		"scenario":   sc.Name,
		"start_time": started.Format(time.RFC3339),
	})
}

// Topology handler

type demoTopologyNode struct {
	ID    string `json:"id"`
	Table string `json:"table"`
	Label string `json:"label"`
}

type demoTopologyEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// GetTopology returns the table graph skeleton shown before a run starts.
func (d *DemoHandlers) GetTopology(w http.ResponseWriter, _ *http.Request) {
	nodes := []demoTopologyNode{}
	edges := []demoTopologyEdge{}
	if t := d.demo.Topology; t != nil {
		for _, n := range t.Nodes {
			table, label := n.Table, n.Label
			if table == "" {
				table = n.ID
			}
			if label == "" {
				label = n.ID
			}
			nodes = append(nodes, demoTopologyNode{ID: n.ID, Table: table, Label: label})
		}
		for _, e := range t.Edges {
			edges = append(edges, demoTopologyEdge{
				ID: ports.GraphEdgeID(e.Source, e.Target), Source: e.Source, Target: e.Target, Relation: e.Relation,
			})
		}
	}
	d.respond(w, http.StatusOK, map[string]any{"nodes": nodes, "edges": edges})
}

// Optimization handlers

type demoOptimizationInfo struct {
	DatabaseType string   `json:"database_type"`
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Applied      bool     `json:"applied"`
	Statements   []string `json:"statements"`
}

// GetOptimization returns the optimization description and whether its
// indexes currently exist.
func (d *DemoHandlers) GetOptimization(w http.ResponseWriter, r *http.Request) {
	info, ok := d.optimizationInfo(r.Context(), w)
	if !ok {
		return
	}
	d.respond(w, http.StatusOK, info)
}

// ApplyOptimization creates the configured indexes (idempotent).
func (d *DemoHandlers) ApplyOptimization(w http.ResponseWriter, r *http.Request) {
	d.changeOptimization(w, r, true)
}

// RevertOptimization drops the configured indexes (idempotent).
func (d *DemoHandlers) RevertOptimization(w http.ResponseWriter, r *http.Request) {
	d.changeOptimization(w, r, false)
}

func (d *DemoHandlers) changeOptimization(w http.ResponseWriter, r *http.Request, apply bool) {
	opt := d.demo.Optimization
	if opt == nil {
		d.fail(w, http.StatusNotFound, "not_configured", "No optimization is configured", "")
		return
	}

	d.optMu.Lock()
	defer d.optMu.Unlock()

	// Do not tie DDL to the request: a browser tab closing mid-way would
	// otherwise leave a half-applied optimization.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), ddlTimeout)
	defer cancel()

	specs, err := d.optimizationSpecs()
	if err != nil {
		d.fail(w, http.StatusInternalServerError, "invalid_statement", "Statement not allowed", err.Error())
		return
	}
	if !apply && len(opt.Indexes) == 0 {
		specs = []ports.IndexSpec{}
		for _, statement := range opt.Revert {
			m := dropIndexRe.FindStringSubmatch(statement)
			if m == nil {
				d.fail(w, http.StatusInternalServerError, "invalid_statement", "Statement not allowed", "")
				return
			}
			specs = append(specs, ports.IndexSpec{Name: m[1], Table: m[2]})
		}
	}
	for _, spec := range specs {
		stmt, buildErr := d.inspector.BuildDropDDL(spec)
		if apply {
			stmt, buildErr = d.inspector.BuildCreateDDL(spec)
		}
		if buildErr != nil {
			d.fail(w, http.StatusInternalServerError, "invalid_statement", "Statement not allowed", buildErr.Error())
			return
		}
		exists, err := d.indexExists(ctx, spec.Table, spec.Name)
		if err != nil {
			d.fail(w, http.StatusInternalServerError, "ddl_error", "Failed to inspect indexes", err.Error())
			return
		}
		if exists == apply { // already in the requested state
			continue
		}
		started := time.Now()
		if _, err := d.db.ExecContext(ctx, strings.TrimSuffix(strings.TrimSpace(stmt), ";")); err != nil {
			d.fail(w, http.StatusInternalServerError, "ddl_error", "Failed to execute statement", err.Error())
			return
		}
		d.logger.WithFields(logrus.Fields{"statement": stmt, "took": time.Since(started)}).Info("Demo optimization statement executed")
	}

	info, ok := d.optimizationInfo(r.Context(), w)
	if !ok {
		return
	}
	d.respond(w, http.StatusOK, info)
}

func (d *DemoHandlers) optimizationInfo(ctx context.Context, w http.ResponseWriter) (*demoOptimizationInfo, bool) {
	opt := d.demo.Optimization
	if opt == nil {
		d.fail(w, http.StatusNotFound, "not_configured", "No optimization is configured", "")
		return nil, false
	}
	applied := true
	specs, err := d.optimizationSpecs()
	if err != nil {
		d.fail(w, http.StatusInternalServerError, "invalid_statement", "Invalid optimization", err.Error())
		return nil, false
	}
	statements := []string{}
	for _, spec := range specs {
		ddl, _ := d.inspector.BuildCreateDDL(spec)
		statements = append(statements, ddl)
		exists, err := d.indexExists(ctx, spec.Table, spec.Name)
		if err != nil {
			d.fail(w, http.StatusInternalServerError, "ddl_error", "Failed to inspect indexes", err.Error())
			return nil, false
		}
		applied = applied && exists
	}
	title := opt.Title
	if title == "" {
		title = opt.Name
	}
	return &demoOptimizationInfo{
		Name: opt.Name, Title: title, Description: opt.Description,
		Applied: applied, Statements: statements, DatabaseType: d.inspector.DatabaseType(),
	}, true
}

// indexExists delegates current-schema inspection to the selected dialect.
func (d *DemoHandlers) indexExists(ctx context.Context, table, index string) (bool, error) {
	return d.inspector.IndexExists(ctx, table, index)
}

func (d *DemoHandlers) respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{Success: true, Data: data, Timestamp: time.Now()}); err != nil {
		d.logger.WithError(err).Error("Failed to encode demo response")
	}
}

func (d *DemoHandlers) fail(w http.ResponseWriter, status int, code, message, details string) {
	d.logger.WithFields(logrus.Fields{"code": code, "details": details}).Warn(message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success:   false,
		Error:     &Error{Code: code, Message: message, Details: details},
		Timestamp: time.Now(),
	})
}
