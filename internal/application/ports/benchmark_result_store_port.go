// Package ports defines interfaces for application layer dependencies.
package ports

import (
	"context"
	"time"
)

// BenchmarkResultStorePort defines persistence for historical benchmark
// results, allowing trend analysis and regression detection to survive
// process restarts.
type BenchmarkResultStorePort interface {
	// Save persists a completed (or failed/cancelled) benchmark result.
	Save(ctx context.Context, result *BenchmarkResult) error

	// Get retrieves a single stored result by its execution ID.
	Get(ctx context.Context, id string) (*BenchmarkResult, error)

	// List returns stored results matching the given filter, ordered by
	// start time ascending.
	List(ctx context.Context, filter BenchmarkResultFilter) ([]*BenchmarkResult, error)

	// DeleteOlderThan removes stored results whose start time is before the
	// given cutoff. It returns the number of deleted results.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// BenchmarkResultFilter narrows down which stored benchmark results to
// return from List. Zero-valued fields are treated as "no filter" for that
// dimension.
type BenchmarkResultFilter struct {
	ToolName string
	TestType string
	Since    time.Time
	Until    time.Time
	// Limit caps the number of results returned (most recent first). Zero
	// or negative means no limit.
	Limit int
}
