package performance

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"sql-graph-visualizer/internal/application/ports"
)

// SimpleMetricsInjector injects simulated performance metrics as Neo4j relationships
type SimpleMetricsInjector struct {
	neo4jRepo       ports.Neo4jPort
	logger          *logrus.Logger
	config          *SimpleMetricsConfig
	isRunning       bool
	stopChan        chan struct{}
	mutex           sync.RWMutex
}

// SimpleMetricsConfig configuration for simple metrics injection
type SimpleMetricsConfig struct {
	UpdateInterval   time.Duration `json:"update_interval"`
	MetricsRetention time.Duration `json:"metrics_retention"`
	SimulationMode   bool          `json:"simulation_mode"`
}

// NewSimpleMetricsInjector creates new simple metrics injector
func NewSimpleMetricsInjector(
	neo4jRepo ports.Neo4jPort,
	logger *logrus.Logger,
	config *SimpleMetricsConfig,
) *SimpleMetricsInjector {
	if config == nil {
		config = &SimpleMetricsConfig{
			UpdateInterval:   5 * time.Second,
			MetricsRetention: 1 * time.Hour,
			SimulationMode:   true,
		}
	}

	return &SimpleMetricsInjector{
		neo4jRepo: neo4jRepo,
		logger:    logger,
		config:    config,
		stopChan:  make(chan struct{}),
	}
}

// Start begins injecting performance metrics
func (s *SimpleMetricsInjector) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("simple metrics injector is already running")
	}

	s.isRunning = true
	s.logger.Info("🚀 Starting simple performance metrics injection service")

	// Start the injection goroutine
	go s.injectionLoop(ctx)

	return nil
}

// Stop stops the metrics injection
func (s *SimpleMetricsInjector) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return nil
	}

	s.logger.Info("Stopping simple performance metrics injection service")
	close(s.stopChan)
	s.isRunning = false

	return nil
}

// injectionLoop main loop for injecting metrics
func (s *SimpleMetricsInjector) injectionLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Simple metrics injection stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("Simple metrics injection stopped")
			return
		case <-ticker.C:
			if err := s.injectMetrics(ctx); err != nil {
				s.logger.WithError(err).Error("Failed to inject simple metrics")
			}
		}
	}
}

// injectMetrics injects performance metrics into the graph
func (s *SimpleMetricsInjector) injectMetrics(ctx context.Context) error {
	// Get existing nodes using Neo4j repository
	query := `MATCH (n) 
			  WHERE n:Artist OR n:Album OR n:Track OR n:Genre OR n:Customer
			  RETURN id(n) as node_id, labels(n) as labels, n.name as name
			  LIMIT 20`
	
	results, err := s.neo4jRepo.ExecuteQuery(query, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to get existing nodes: %w", err)
	}

	if len(results) == 0 {
		s.logger.Debug("No nodes found to add metrics to")
		return nil
	}

	// Generate and inject metrics between nodes
	metricsCount := 0
	for i, sourceResult := range results {
		for j, targetResult := range results {
			if i >= j || metricsCount >= 30 { // Limit for demo
				continue
			}

			if err := s.createPerformanceMetric(ctx, sourceResult, targetResult); err != nil {
				s.logger.WithError(err).Debug("Failed to create performance metric")
			} else {
				metricsCount++
			}
		}
	}

	// Cleanup old metrics
	if err := s.cleanupOldMetrics(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to cleanup old metrics")
	}

	s.logger.WithField("metrics_count", metricsCount).Info("📊 Successfully injected performance metrics")
	return nil
}

// createPerformanceMetric creates a performance metric relationship between two nodes
func (s *SimpleMetricsInjector) createPerformanceMetric(ctx context.Context, source, target map[string]interface{}) error {
	// Generate random metric type and value
	metricTypes := []string{
		"QUERIES_PER_SEC",
		"AVG_LATENCY_MS",
		"JOIN_FREQUENCY",
		"INDEX_EFFICIENCY",
		"HOTSPOT_SCORE",
		"LOAD_FACTOR",
	}

	metricType := metricTypes[rand.Intn(len(metricTypes))]
	value := s.generateMetricValue(metricType)
	timestamp := time.Now().Unix()
	trend := s.generateTrend()
	severity := s.generateSeverity()

	// First, delete any existing metric of the same type between these nodes
	deleteQuery := fmt.Sprintf(`
		MATCH (a)-[r:%s]->(b) 
		WHERE id(a) = $source_id AND id(b) = $target_id 
		DELETE r
	`, metricType)

	_, err := s.neo4jRepo.ExecuteQuery(deleteQuery, map[string]interface{}{
		"source_id": source["node_id"],
		"target_id": target["node_id"],
	})
	if err != nil {
		// It's ok if there's no existing relationship to delete
		s.logger.WithError(err).Debug("No existing metric relationship to delete")
	}

	// Create new metric relationship
	createQuery := fmt.Sprintf(`
		MATCH (a), (b) 
		WHERE id(a) = $source_id AND id(b) = $target_id
		CREATE (a)-[r:%s {
			value: $value,
			timestamp: $timestamp,
			source_name: $source_name,
			target_name: $target_name,
			trend: $trend,
			severity: $severity,
			updated_at: datetime()
		}]->(b)
		RETURN r
	`, metricType)

	_, err = s.neo4jRepo.ExecuteQuery(createQuery, map[string]interface{}{
		"source_id":   source["node_id"],
		"target_id":   target["node_id"],
		"value":       value,
		"timestamp":   timestamp,
		"source_name": source["name"],
		"target_name": target["name"],
		"trend":       trend,
		"severity":    severity,
	})

	return err
}

// generateMetricValue generates realistic values for different metric types
func (s *SimpleMetricsInjector) generateMetricValue(metricType string) float64 {
	switch metricType {
	case "QUERIES_PER_SEC":
		return float64(rand.Intn(100) + 10) // 10-110 QPS
	case "AVG_LATENCY_MS":
		return float64(rand.Intn(500) + 20) // 20-520ms
	case "JOIN_FREQUENCY":
		return float64(rand.Intn(1000) + 50) // 50-1050 joins
	case "INDEX_EFFICIENCY":
		return rand.Float64() // 0.0-1.0
	case "HOTSPOT_SCORE":
		return rand.Float64() * 100 // 0-100 score
	case "LOAD_FACTOR":
		return rand.Float64() * 2.0 // 0.0-2.0 load
	default:
		return rand.Float64() * 100
	}
}

// generateTrend generates trend information
func (s *SimpleMetricsInjector) generateTrend() string {
	trends := []string{"increasing", "decreasing", "stable", "volatile"}
	return trends[rand.Intn(len(trends))]
}

// generateSeverity generates severity level
func (s *SimpleMetricsInjector) generateSeverity() string {
	severities := []string{"low", "medium", "high", "critical"}
	weights := []int{40, 30, 20, 10} // Probability distribution
	
	total := 0
	for _, weight := range weights {
		total += weight
	}
	
	r := rand.Intn(total)
	sum := 0
	for i, weight := range weights {
		sum += weight
		if r < sum {
			return severities[i]
		}
	}
	
	return "low"
}

// cleanupOldMetrics removes old performance metric relationships
func (s *SimpleMetricsInjector) cleanupOldMetrics(ctx context.Context) error {
	cutoffTime := time.Now().Add(-s.config.MetricsRetention).Unix()
	
	metricTypes := []string{
		"QUERIES_PER_SEC", "AVG_LATENCY_MS", "JOIN_FREQUENCY",
		"INDEX_EFFICIENCY", "HOTSPOT_SCORE", "LOAD_FACTOR",
	}

	for _, metricType := range metricTypes {
		query := fmt.Sprintf(`
			MATCH ()-[r:%s]->()
			WHERE r.timestamp < $cutoff_time
			DELETE r
		`, metricType)

		_, err := s.neo4jRepo.ExecuteQuery(query, map[string]interface{}{
			"cutoff_time": cutoffTime,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// GetCurrentMetrics returns current performance metrics
func (s *SimpleMetricsInjector) GetCurrentMetrics(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
		MATCH (a)-[r]->(b)
		WHERE type(r) IN ['QUERIES_PER_SEC', 'AVG_LATENCY_MS', 'JOIN_FREQUENCY', 'INDEX_EFFICIENCY', 'HOTSPOT_SCORE', 'LOAD_FACTOR']
		RETURN type(r) as metric_type, id(a) as source_id, id(b) as target_id, 
			   r.value as value, r.timestamp as timestamp, r.source_name as source_name,
			   r.target_name as target_name, r.trend as trend, r.severity as severity
		ORDER BY r.timestamp DESC
		LIMIT 100
	`

	results, err := s.neo4jRepo.ExecuteQuery(query, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	return results, nil
}
