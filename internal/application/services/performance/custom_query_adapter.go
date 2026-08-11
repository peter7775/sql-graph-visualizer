package performance

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

// allowedCustomQueryPrefixes restricts custom benchmark queries to
// read/write statement types that cannot alter schema or bulk-delete data.
// This is a defense-in-depth measure: custom queries come from operator
// configuration, not end users, but running arbitrary DDL/DELETE/TRUNCATE
// statements as part of a "benchmark" is still an easy way to cause
// accidental damage.
var allowedCustomQueryPrefixes = []string{"SELECT", "INSERT", "UPDATE"}

// CustomQueryAdapter implements ports.BenchmarkToolPort by executing
// operator-configured, named sets of SQL queries against the active source
// database via the standard database/sql connection pool (the same *sql.DB
// used elsewhere in the application, e.g. by PerformanceSchemaAdapter).
type CustomQueryAdapter struct {
	logger    *logrus.Logger
	db        *sql.DB
	querySets map[string]ports.CustomBenchmarkConfig
}

// NewCustomQueryAdapter creates a new custom query benchmark adapter. Query
// sets with no queries, or whose queries are all disallowed statement types,
// are skipped with a warning.
func NewCustomQueryAdapter(logger *logrus.Logger, db *sql.DB, querySets map[string]ports.CustomBenchmarkConfig) *CustomQueryAdapter {
	return &CustomQueryAdapter{
		logger:    logger,
		db:        db,
		querySets: querySets,
	}
}

// Execute runs the custom query set named by config.CustomParams["query_set"]
// (or the sole configured set, if there is exactly one) for config.Duration
// using config.Threads concurrent workers, cycling through the set's queries
// weighted by their configured Weight.
func (c *CustomQueryAdapter) Execute(ctx context.Context, config ports.BenchmarkConfig) (*ports.BenchmarkResult, error) {
	set, setName, err := c.resolveQuerySet(config)
	if err != nil {
		return nil, err
	}

	threads := config.Threads
	if threads <= 0 {
		threads = set.Threads
	}
	if threads <= 0 {
		threads = 1
	}

	duration := config.Duration
	if duration <= 0 {
		duration = set.Duration
	}
	if duration <= 0 {
		duration = 30 * time.Second
	}

	weights := make([]int, len(set.Queries))
	totalWeight := 0
	for i, q := range set.Queries {
		w := q.Weight
		if w <= 0 {
			w = 1
		}
		weights[i] = w
		totalWeight += w
	}

	stats := make([]*customQueryStat, len(set.Queries))
	for i, q := range set.Queries {
		stats[i] = &customQueryStat{def: q}
	}
	var statsMu sync.Mutex

	startTime := time.Now()
	deadline := startTime.Add(duration)

	var wg sync.WaitGroup
	for w := 0; w < threads; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rnd := rand.New(rand.NewSource(seed)) //nolint:gosec // non-cryptographic query selection
			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				idx := pickWeightedIndex(rnd, weights, totalWeight)
				q := set.Queries[idx]

				queryStart := time.Now()
				rowsExamined, rowsReturned, execErr := c.runQuery(ctx, q)
				elapsed := time.Since(queryStart)

				statsMu.Lock()
				stats[idx].record(elapsed, rowsExamined, rowsReturned, execErr)
				statsMu.Unlock()
			}
		}(time.Now().UnixNano() + int64(w))
	}
	wg.Wait()

	endTime := time.Now()
	metrics, queryResults := aggregateCustomQueryStats(stats, endTime.Sub(startTime))

	c.logger.WithFields(logrus.Fields{
		"query_set":       setName,
		"threads":         threads,
		"duration":        duration,
		"queries_per_sec": metrics.QueriesPerSecond,
		"total_errors":    metrics.TotalErrors,
	}).Info("Custom query benchmark completed")

	return &ports.BenchmarkResult{
		ToolName:     "custom",
		TestType:     setName,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     endTime.Sub(startTime),
		Metrics:      metrics,
		QueryResults: queryResults,
		Status:       ports.BenchmarkStatusCompleted,
	}, nil
}

// Validate checks that the requested query set exists, is non-empty, and
// only contains allowed statement types.
func (c *CustomQueryAdapter) Validate(config ports.BenchmarkConfig) error {
	set, setName, err := c.resolveQuerySet(config)
	if err != nil {
		return err
	}
	if len(set.Queries) == 0 {
		return fmt.Errorf("custom query set %q has no queries configured", setName)
	}
	for _, q := range set.Queries {
		if !isAllowedCustomQuery(q.Query) {
			return fmt.Errorf("query set %q contains a statement type that is not allowed (only SELECT/INSERT/UPDATE): %q", setName, q.Query)
		}
	}
	if c.db == nil {
		return fmt.Errorf("no database connection is configured for custom query benchmarks")
	}
	return nil
}

// GetSupportedTests returns the names of configured custom query sets; each
// name can be requested via BenchmarkConfig.CustomParams["query_set"].
func (c *CustomQueryAdapter) GetSupportedTests() []string {
	names := make([]string, 0, len(c.querySets))
	for name := range c.querySets {
		names = append(names, name)
	}
	return names
}

// IsAvailable reports whether at least one valid query set is configured and
// a database connection is available.
func (c *CustomQueryAdapter) IsAvailable() bool {
	if c.db == nil || len(c.querySets) == 0 {
		return false
	}
	for _, set := range c.querySets {
		if len(set.Queries) > 0 {
			return true
		}
	}
	return false
}

// GetVersion returns a static identifier: this adapter has no external
// binary/version to report.
func (c *CustomQueryAdapter) GetVersion() (string, error) {
	return "custom-query-adapter/1.0", nil
}

// resolveQuerySet finds the query set requested via CustomParams["query_set"],
// falling back to the sole configured set when there is exactly one.
func (c *CustomQueryAdapter) resolveQuerySet(config ports.BenchmarkConfig) (ports.CustomBenchmarkConfig, string, error) {
	name, _ := config.CustomParams["query_set"].(string)
	if name == "" {
		if len(c.querySets) == 1 {
			for onlyName := range c.querySets {
				name = onlyName
			}
		} else {
			return ports.CustomBenchmarkConfig{}, "", fmt.Errorf("custom_params.query_set is required (available: %v)", c.GetSupportedTests())
		}
	}

	set, exists := c.querySets[name]
	if !exists {
		return ports.CustomBenchmarkConfig{}, name, fmt.Errorf("unknown custom query set %q (available: %v)", name, c.GetSupportedTests())
	}
	return set, name, nil
}

// runQuery executes a single query definition and returns rows
// examined/returned. SELECT statements report the number of rows scanned as
// both examined and returned; INSERT/UPDATE report affected rows as
// "examined" with zero rows "returned".
func (c *CustomQueryAdapter) runQuery(ctx context.Context, q ports.CustomQueryDefinition) (rowsExamined, rowsReturned int64, err error) {
	if !isAllowedCustomQuery(q.Query) {
		return 0, 0, fmt.Errorf("statement type not allowed: %q", q.Query)
	}

	statementType := strings.ToUpper(strings.Fields(strings.TrimSpace(q.Query))[0])
	if statementType == "SELECT" {
		rows, queryErr := c.db.QueryContext(ctx, q.Query, q.Parameters...)
		if queryErr != nil {
			return 0, 0, queryErr
		}
		defer func() { _ = rows.Close() }()

		var count int64
		for rows.Next() {
			count++
		}
		if err := rows.Err(); err != nil {
			return count, count, err
		}
		return count, count, nil
	}

	result, execErr := c.db.ExecContext(ctx, q.Query, q.Parameters...)
	if execErr != nil {
		return 0, 0, execErr
	}
	affected, _ := result.RowsAffected()
	return affected, 0, nil
}

// isAllowedCustomQuery reports whether query begins with one of the allowed
// statement keywords.
func isAllowedCustomQuery(query string) bool {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return false
	}
	fields := strings.Fields(trimmed)
	first := strings.ToUpper(fields[0])
	for _, allowed := range allowedCustomQueryPrefixes {
		if first == allowed {
			return true
		}
	}
	return false
}

// pickWeightedIndex selects a random index from weights (a slice of positive
// weights summing to totalWeight) using weighted random selection. Falls
// back to a uniform pick if the slice is empty or weights are non-positive.
func pickWeightedIndex(rnd *rand.Rand, weights []int, totalWeight int) int {
	if len(weights) == 0 {
		return 0
	}
	if totalWeight <= 0 {
		return rnd.Intn(len(weights))
	}
	target := rnd.Intn(totalWeight)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if target < cumulative {
			return i
		}
	}
	return len(weights) - 1
}

// customQueryStat accumulates execution statistics for a single query
// definition within a benchmark run.
type customQueryStat struct {
	def          ports.CustomQueryDefinition
	execCount    int64
	errorCount   int64
	totalTime    time.Duration
	minTime      time.Duration
	maxTime      time.Duration
	rowsExamined int64
	rowsReturned int64
}

func (s *customQueryStat) record(elapsed time.Duration, rowsExamined, rowsReturned int64, err error) {
	s.execCount++
	s.totalTime += elapsed
	if s.minTime == 0 || elapsed < s.minTime {
		s.minTime = elapsed
	}
	if elapsed > s.maxTime {
		s.maxTime = elapsed
	}
	if err != nil {
		s.errorCount++
		return
	}
	s.rowsExamined += rowsExamined
	s.rowsReturned += rowsReturned
}

// aggregateCustomQueryStats converts per-query statistics into the shared
// PerformanceMetrics/QueryPerformance shapes used by all benchmark tools.
func aggregateCustomQueryStats(stats []*customQueryStat, wallClock time.Duration) (*ports.PerformanceMetrics, []ports.QueryPerformance) {
	metrics := &ports.PerformanceMetrics{}
	queryResults := make([]ports.QueryPerformance, 0, len(stats))

	var totalExec int64
	var totalErrors int64
	var totalLatencyMs float64

	for _, s := range stats {
		if s.execCount == 0 {
			continue
		}
		totalExec += s.execCount
		totalErrors += s.errorCount
		avgTime := s.totalTime / time.Duration(s.execCount)
		totalLatencyMs += float64(avgTime.Milliseconds()) * float64(s.execCount)

		statementType := "SELECT"
		if fields := strings.Fields(strings.TrimSpace(s.def.Query)); len(fields) > 0 {
			statementType = strings.ToUpper(fields[0])
		}

		queryResults = append(queryResults, ports.QueryPerformance{
			QueryPattern:      s.def.Query,
			QueryType:         statementType,
			ExecutionCount:    s.execCount,
			TotalTime:         s.totalTime,
			AverageTime:       avgTime,
			MinTime:           s.minTime,
			MaxTime:           s.maxTime,
			RowsExamined:      s.rowsExamined,
			RowsReturned:      s.rowsReturned,
			RelationshipType:  "CUSTOM_QUERY",
			PerformanceImpact: classifyCustomQueryImpact(avgTime),
		})
	}

	if wallClock > 0 {
		metrics.QueriesPerSecond = float64(totalExec) / wallClock.Seconds()
	}
	if totalExec > 0 {
		metrics.AverageLatency = totalLatencyMs / float64(totalExec)
		metrics.ErrorRate = (float64(totalErrors) / float64(totalExec)) * 100
	}
	metrics.TotalErrors = int(totalErrors)

	return metrics, queryResults
}

func classifyCustomQueryImpact(avgLatency time.Duration) string {
	switch {
	case avgLatency < 10*time.Millisecond:
		return "LOW"
	case avgLatency < 100*time.Millisecond:
		return "MEDIUM"
	default:
		return "HIGH"
	}
}
