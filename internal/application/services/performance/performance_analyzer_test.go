package performance

import (
	"context"
	"testing"

	"sql-graph-visualizer/internal/application/ports"
)

func newTestAnalyzer() *PerformanceAnalyzer {
	return NewPerformanceAnalyzer(newTestLogger(), nil)
}

func TestPerformanceAnalyzer_DetectRegressions(t *testing.T) {
	analyzer := newTestAnalyzer()
	ctx := context.Background()

	t.Run("requires both metrics", func(t *testing.T) {
		if _, err := analyzer.DetectRegressions(ctx, nil, &ports.PerformanceMetrics{}); err == nil {
			t.Error("DetectRegressions() with nil current: expected error, got nil")
		}
	})

	t.Run("detects latency regression", func(t *testing.T) {
		previous := &ports.PerformanceMetrics{AverageLatency: 50, QueriesPerSecond: 100}
		current := &ports.PerformanceMetrics{AverageLatency: 100, QueriesPerSecond: 100}

		regressions, err := analyzer.DetectRegressions(ctx, current, previous)
		if err != nil {
			t.Fatalf("DetectRegressions() error = %v", err)
		}
		found := false
		for _, r := range regressions {
			if r.MetricName == "average_latency" {
				found = true
				if r.RegressionAmount <= 0 {
					t.Errorf("RegressionAmount = %v, want positive", r.RegressionAmount)
				}
			}
		}
		if !found {
			t.Errorf("DetectRegressions() = %+v, want a latency regression to be detected", regressions)
		}
	})

	t.Run("no regression when performance improves", func(t *testing.T) {
		previous := &ports.PerformanceMetrics{AverageLatency: 100, QueriesPerSecond: 50}
		current := &ports.PerformanceMetrics{AverageLatency: 50, QueriesPerSecond: 100}

		regressions, err := analyzer.DetectRegressions(ctx, current, previous)
		if err != nil {
			t.Fatalf("DetectRegressions() error = %v", err)
		}
		if len(regressions) != 0 {
			t.Errorf("DetectRegressions() = %+v, want no regressions when performance improves", regressions)
		}
	})
}

func TestPerformanceAnalyzer_CalculatePerformanceScore(t *testing.T) {
	analyzer := newTestAnalyzer()
	ctx := context.Background()

	if _, err := analyzer.CalculatePerformanceScore(ctx, nil); err == nil {
		t.Error("CalculatePerformanceScore(nil) expected error, got nil")
	}

	good, err := analyzer.CalculatePerformanceScore(ctx, &ports.PerformanceMetrics{
		AverageLatency: 5, QueriesPerSecond: 2000, ErrorRate: 0,
	})
	if err != nil {
		t.Fatalf("CalculatePerformanceScore() error = %v", err)
	}
	poor, err := analyzer.CalculatePerformanceScore(ctx, &ports.PerformanceMetrics{
		AverageLatency: 500, QueriesPerSecond: 1, ErrorRate: 10,
	})
	if err != nil {
		t.Fatalf("CalculatePerformanceScore() error = %v", err)
	}

	if good.OverallScore <= poor.OverallScore {
		t.Errorf("expected good metrics score (%v) > poor metrics score (%v)", good.OverallScore, poor.OverallScore)
	}
}

func TestPerformanceAnalyzer_IdentifyBottlenecks(t *testing.T) {
	analyzer := newTestAnalyzer()
	ctx := context.Background()

	if _, err := analyzer.IdentifyBottlenecks(ctx, nil); err == nil {
		t.Error("IdentifyBottlenecks(nil) expected error, got nil")
	}

	result := &ports.BenchmarkResult{
		Metrics: &ports.PerformanceMetrics{AverageLatency: 5000, QueriesPerSecond: 1},
	}
	bottlenecks, err := analyzer.IdentifyBottlenecks(ctx, result)
	if err != nil {
		t.Fatalf("IdentifyBottlenecks() error = %v", err)
	}
	if len(bottlenecks) == 0 {
		t.Error("IdentifyBottlenecks() with clearly bad metrics: expected at least one bottleneck")
	}
}

func TestPerformanceAnalyzer_ClassifyRegressionSeverity(t *testing.T) {
	analyzer := newTestAnalyzer()

	tests := []struct {
		pct  float64
		want ports.SeverityLevel
	}{
		{60, ports.SeverityCritical},
		{30, ports.SeverityHigh},
		{15, ports.SeverityMedium},
		{5, ports.SeverityLow},
	}
	for _, tt := range tests {
		if got := analyzer.classifyRegressionSeverity(tt.pct); got != tt.want {
			t.Errorf("classifyRegressionSeverity(%v) = %v, want %v", tt.pct, got, tt.want)
		}
	}
}
