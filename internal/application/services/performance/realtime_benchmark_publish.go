package performance

import "sql-graph-visualizer/internal/application/ports"

// BenchmarkTopic is the WebSocket topic that carries live benchmark samples.
// Clients opt in with {"type":"subscribe","topic":"benchmark"}.
const BenchmarkTopic = "benchmark"

// BroadcastBenchmarkSample publishes a live benchmark sample to every client
// subscribed to the benchmark topic. It is intended as the sample observer of
// BenchmarkService, so it must stay cheap and non-blocking: delivery to each
// client happens on its own goroutine.
func (rpm *RealtimePerformanceMonitor) BroadcastBenchmarkSample(sample ports.LiveSample) {
	rpm.broadcastToClients(BenchmarkTopic, sample)
}
