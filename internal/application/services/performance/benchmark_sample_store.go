package performance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"sql-graph-visualizer/internal/application/ports"

	"github.com/sirupsen/logrus"
)

// sampleFileExt is the extension of a persisted sample file.
const sampleFileExt = ".jsonl"

// validBenchmarkID guards the file name derived from a benchmark ID against
// path traversal; IDs are UUIDs but arrive through HTTP paths.
var validBenchmarkID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// FileBenchmarkSampleStore persists the live samples of each benchmark run as
// one JSONL file (<dir>/samples/<id>.jsonl), written incrementally so a crash
// or restart keeps everything up to the last second. It implements
// ports.BenchmarkSampleStorePort.
type FileBenchmarkSampleStore struct {
	logger *logrus.Logger
	dir    string

	mu sync.Mutex
}

var _ ports.BenchmarkSampleStorePort = (*FileBenchmarkSampleStore)(nil)

// NewFileBenchmarkSampleStore creates the store below dir (the benchmark
// results directory); the samples go to the "samples" subdirectory.
func NewFileBenchmarkSampleStore(logger *logrus.Logger, dir string) (*FileBenchmarkSampleStore, error) {
	if dir == "" {
		dir = "data/performance/benchmarks"
	}
	samplesDir := filepath.Join(dir, "samples")
	if err := os.MkdirAll(samplesDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create benchmark samples directory %q: %w", samplesDir, err)
	}
	return &FileBenchmarkSampleStore{logger: logger, dir: samplesDir}, nil
}

func (s *FileBenchmarkSampleStore) path(benchmarkID string) (string, error) {
	if !validBenchmarkID.MatchString(benchmarkID) {
		return "", fmt.Errorf("invalid benchmark id %q", benchmarkID)
	}
	return filepath.Join(s.dir, benchmarkID+sampleFileExt), nil
}

// AppendSample appends one sample to the run's file.
func (s *FileBenchmarkSampleStore) AppendSample(_ context.Context, benchmarkID string, sample ports.LiveSample) error {
	path, err := s.path(benchmarkID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(sample)
	if err != nil {
		return fmt.Errorf("failed to marshal live sample: %w", err)
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- path built from a validated id
	if err != nil {
		return fmt.Errorf("failed to open sample file %q: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write sample file %q: %w", path, err)
	}
	return f.Close()
}

// GetSamples returns the stored samples with Seq > afterSeq. Malformed lines
// (for example a torn last write after a crash) are skipped.
func (s *FileBenchmarkSampleStore) GetSamples(_ context.Context, benchmarkID string, afterSeq int) ([]ports.LiveSample, error) {
	path, err := s.path(benchmarkID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	f, err := os.Open(path) // #nosec G304 -- path built from a validated id
	s.mu.Unlock()
	if err != nil {
		if os.IsNotExist(err) {
			return []ports.LiveSample{}, nil
		}
		return nil, fmt.Errorf("failed to open sample file %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	samples := make([]ports.LiveSample, 0, 64)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var sample ports.LiveSample
		if err := json.Unmarshal(line, &sample); err != nil {
			s.logger.WithError(err).WithField("benchmark_id", benchmarkID).Warn("Skipping malformed live sample record")
			continue
		}
		if sample.Seq > afterSeq {
			samples = append(samples, sample)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read sample file %q: %w", path, err)
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].Seq < samples[j].Seq })
	return samples, nil
}

// HasSamples reports whether a non-empty sample file exists for the run.
func (s *FileBenchmarkSampleStore) HasSamples(_ context.Context, benchmarkID string) bool {
	path, err := s.path(benchmarkID)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// DeleteSamples removes the run's sample file. A missing file is not an error.
func (s *FileBenchmarkSampleStore) DeleteSamples(_ context.Context, benchmarkID string) error {
	path, err := s.path(benchmarkID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete sample file %q: %w", path, err)
	}
	return nil
}

// DeleteSamplesOlderThan removes sample files last modified before cutoff and
// returns how many were deleted. It is used by the retention cleanup.
func (s *FileBenchmarkSampleStore) DeleteSamplesOlderThan(_ context.Context, cutoff time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, fmt.Errorf("failed to list sample directory %q: %w", s.dir, err)
	}
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), sampleFileExt) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, entry.Name())); err == nil {
			deleted++
		}
	}
	return deleted, nil
}
