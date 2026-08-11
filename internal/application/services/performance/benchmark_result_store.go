package performance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

// FileBenchmarkResultStore is a simple, dependency-free implementation of
// ports.BenchmarkResultStorePort that appends benchmark results as
// newline-delimited JSON (JSONL) to a file on disk. All stored results are
// also kept in an in-memory cache so reads never need to re-parse the file.
//
// This is intentionally simple: benchmark executions are infrequent
// (human-triggered or scheduled), so a single append-only file with an
// in-memory index is sufficient and avoids introducing a new database
// dependency (e.g. SQLite) purely for this feature.
type FileBenchmarkResultStore struct {
	logger   *logrus.Logger
	dir      string
	filePath string

	mu      sync.Mutex
	results map[string]*ports.BenchmarkResult
}

// NewFileBenchmarkResultStore creates a store rooted at dir, creating the
// directory if necessary and loading any previously persisted results into
// memory.
func NewFileBenchmarkResultStore(logger *logrus.Logger, dir string) (*FileBenchmarkResultStore, error) {
	if dir == "" {
		dir = "data/performance/benchmarks"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create benchmark results directory %q: %w", dir, err)
	}

	store := &FileBenchmarkResultStore{
		logger:   logger,
		dir:      dir,
		filePath: filepath.Join(dir, "benchmark_results.jsonl"),
		results:  make(map[string]*ports.BenchmarkResult),
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// load reads all previously persisted results from the JSONL file into the
// in-memory cache. Malformed lines are logged and skipped rather than
// aborting startup.
func (s *FileBenchmarkResultStore) load() error {
	f, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open benchmark results file %q: %w", s.filePath, err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	// Benchmark results can carry a sizeable number of query results; allow
	// lines larger than bufio's 64KB default.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	loaded := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var result ports.BenchmarkResult
		if err := json.Unmarshal(line, &result); err != nil {
			s.logger.WithError(err).Warn("Skipping malformed benchmark result record")
			continue
		}
		s.results[result.ID] = &result
		loaded++
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read benchmark results file %q: %w", s.filePath, err)
	}

	s.logger.WithFields(logrus.Fields{"count": loaded, "path": s.filePath}).Info("Loaded persisted benchmark results")
	return nil
}

// Save persists a benchmark result, appending it to the JSONL file and
// updating the in-memory cache.
func (s *FileBenchmarkResultStore) Save(_ context.Context, result *ports.BenchmarkResult) error {
	if result == nil {
		return fmt.Errorf("cannot save a nil benchmark result")
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark result: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("failed to open benchmark results file %q: %w", s.filePath, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to append benchmark result: %w", err)
	}

	resultCopy := *result
	s.results[result.ID] = &resultCopy
	return nil
}

// Get retrieves a single stored result by ID.
func (s *FileBenchmarkResultStore) Get(_ context.Context, id string) (*ports.BenchmarkResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, exists := s.results[id]
	if !exists {
		return nil, fmt.Errorf("benchmark result %q not found", id)
	}
	resultCopy := *result
	return &resultCopy, nil
}

// List returns stored results matching the filter, ordered by start time
// ascending (oldest first). When filter.Limit > 0, only the most recent
// matching results (up to the limit) are returned, still ordered ascending.
func (s *FileBenchmarkResultStore) List(_ context.Context, filter ports.BenchmarkResultFilter) ([]*ports.BenchmarkResult, error) {
	s.mu.Lock()
	matches := make([]*ports.BenchmarkResult, 0, len(s.results))
	for _, result := range s.results {
		if filter.ToolName != "" && result.ToolName != filter.ToolName {
			continue
		}
		if filter.TestType != "" && result.TestType != filter.TestType {
			continue
		}
		if !filter.Since.IsZero() && result.StartTime.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && result.StartTime.After(filter.Until) {
			continue
		}
		resultCopy := *result
		matches = append(matches, &resultCopy)
	}
	s.mu.Unlock()

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].StartTime.Before(matches[j].StartTime)
	})

	if filter.Limit > 0 && len(matches) > filter.Limit {
		matches = matches[len(matches)-filter.Limit:]
	}

	return matches, nil
}

// DeleteOlderThan removes results started before cutoff, compacting the
// backing file to reflect the remaining set.
func (s *FileBenchmarkResultStore) DeleteOlderThan(_ context.Context, cutoff time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0
	for id, result := range s.results {
		if result.StartTime.Before(cutoff) {
			delete(s.results, id)
			deleted++
		}
	}

	if deleted == 0 {
		return 0, nil
	}

	if err := s.rewriteLocked(); err != nil {
		return 0, err
	}

	s.logger.WithField("deleted_count", deleted).Info("Compacted persisted benchmark results")
	return deleted, nil
}

// rewriteLocked rewrites the backing file from the current in-memory cache.
// Callers must hold s.mu.
func (s *FileBenchmarkResultStore) rewriteLocked() error {
	tmpPath := s.filePath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("failed to create temporary benchmark results file: %w", err)
	}

	writer := bufio.NewWriter(f)
	for _, result := range s.results {
		data, err := json.Marshal(result)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("failed to marshal benchmark result during compaction: %w", err)
		}
		if _, err := writer.Write(append(data, '\n')); err != nil {
			_ = f.Close()
			return fmt.Errorf("failed to write benchmark result during compaction: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to flush compacted benchmark results file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close compacted benchmark results file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("failed to replace benchmark results file: %w", err)
	}
	return nil
}
