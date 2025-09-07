package performance

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"sql-graph-visualizer/internal/domain/models"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// RealtimePerformanceMonitor provides real-time performance .monitoring with WebSocket streaming
type RealtimePerformanceMonitor struct {
	logger      *logrus.Logger
	config      *RealtimeMonitorConfig
	psAdapter   *PerformanceSchemaAdapter
	analyzer    *PerformanceAnalyzer
	graphMapper *GraphPerformanceMapper

	// WebSocket management
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]*ClientInfo
	clientMutex sync.RWMutex

	// Monitoring control
	isRunning    bool
	runningMutex sync.RWMutex
	stopChannel  chan struct{}

	// Data channels
	performanceData chan *PerformanceGraphData
	alertsChannel   chan *PerformanceAlert

	// Cache and state
	lastGraphData *PerformanceGraphData
	lastUpdate    time.Time
	stateMutex    sync.RWMutex
}

// RealtimeMonitorConfig contains configuration for real-time .monitoring
type RealtimeMonitorConfig struct {
	// Update intervals
	DataUpdateInterval time.Duration `yaml:"data_update_interval" json:"data_update_interval"`
	HeartbeatInterval  time.Duration `yaml:"heartbeat_interval" json:"heartbeat_interval"`

	// WebSocket settings
	MaxConnections int           `yaml:"max_connections" json:"max_connections"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	PingTimeout    time.Duration `yaml:"ping_timeout" json:"ping_timeout"`
	MaxMessageSize int64         `yaml:"max_message_size" json:"max_message_size"`

	// Performance .monitoring
	AlertThresholds    AlertThresholds `yaml:"alert_thresholds" json:"alert_thresholds"`
	MetricsRetention   time.Duration   `yaml:"metrics_retention" json:"metrics_retention"`
	CompressionEnabled bool            `yaml:"compression_enabled" json:"compression_enabled"`

	// Resource limits
	MaxConcurrentQueries int     `yaml:"max_concurrent_queries" json:"max_concurrent_queries"`
	MemoryLimitMB        int     `yaml:"memory_limit_mb" json:"memory_limit_mb"`
	CPUThreshold         float64 `yaml:"cpu_threshold" json:"cpu_threshold"`
}

// AlertThresholds defines performance alert thresholds
type AlertThresholds struct {
	HighLatency        float64 `yaml:"high_latency" json:"high_latency"`
	HighErrorRate      float64 `yaml:"high_error_rate" json:"high_error_rate"`
	HighCPUUsage       float64 `yaml:"high_cpu_usage" json:"high_cpu_usage"`
	HighMemoryUsage    float64 `yaml:"high_memory_usage" json:"high_memory_usage"`
	SlowQueryThreshold float64 `yaml:"slow_query_threshold" json:"slow_query_threshold"`
	DeadlockThreshold  int     `yaml:"deadlock_threshold" json:"deadlock_threshold"`
}

// ClientInfo stores information about connected WebSocket clients
type ClientInfo struct {
	ID               string                 `json:"id"`
	ConnectedAt      time.Time              `json:"connected_at"`
	LastPingAt       time.Time              `json:"last_ping_at"`
	SubscribedTopics []string               `json:"subscribed_topics"`
	Filters          map[string]interface{} `json:"filters"`
	Compression      bool                   `json:"compression"`
}

// WebSocketMessage represents a WebSocket message structure
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Topic     string      `json:"topic"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	ID        string      `json:"id"`
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	TableName   string                 `json:"table_name,omitempty"`
	QueryID     string                 `json:"query_id,omitempty"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RealtimeMetrics contains real-time performance metrics
type RealtimeMetrics struct {
	Timestamp         time.Time                `json:"timestamp"`
	SystemMetrics     *SystemMetrics           `json:"system_metrics"`
	DatabaseMetrics   *DatabaseMetrics         `json:"database_metrics"`
	ActiveConnections int                      `json:"active_connections"`
	TopQueries        []QueryPerformanceMetric `json:"top_queries"`
	Alerts            []PerformanceAlert       `json:"alerts"`
	GraphData         *PerformanceGraphData    `json:"graph_data,omitempty"`
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskUsage     float64 `json:"disk_usage"`
	NetworkIO     struct {
		BytesSent     int64 `json:"bytes_sent"`
		BytesReceived int64 `json:"bytes_received"`
	} `json:"network_io"`
}

// DatabaseMetrics contains database-specific metrics
type DatabaseMetrics struct {
	QueriesPerSecond float64 `json:"queries_per_second"`
	SlowQueries      int64   `json:"slow_queries"`
	ConnectionsUsed  int     `json:"connections_used"`
	ConnectionsMax   int     `json:"connections_max"`
	InnoDBBufferPool struct {
		HitRatio     float64 `json:"hit_ratio"`
		Usage        float64 `json:"usage"`
		PagesRead    int64   `json:"pages_read"`
		PagesWritten int64   `json:"pages_written"`
	} `json:"innodb_buffer_pool"`
}

// QueryPerformanceMetric represents performance metrics for individual queries
type QueryPerformanceMetric struct {
	QueryID          string  `json:"query_id"`
	DigestText       string  `json:"digest_text"`
	ExecutionCount   int64   `json:"execution_count"`
	AvgExecutionTime float64 `json:"avg_execution_time"`
	MaxExecutionTime float64 `json:"max_execution_time"`
	RowsAffected     int64   `json:"rows_affected"`
	ErrorCount       int64   `json:"error_count"`
}

// NewRealtimePerformanceMonitor creates a new real-time performance monitor
func NewRealtimePerformanceMonitor(
	logger *logrus.Logger,
	config *RealtimeMonitorConfig,
	psAdapter *PerformanceSchemaAdapter,
	analyzer *PerformanceAnalyzer,
	graphMapper *GraphPerformanceMapper,
) *RealtimePerformanceMonitor {
	if config == nil {
		config = defaultRealtimeMonitorConfig()
	}

	return &RealtimePerformanceMonitor{
		logger:      logger,
		config:      config,
		psAdapter:   psAdapter,
		analyzer:    analyzer,
		graphMapper: graphMapper,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking in production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:         make(map[*websocket.Conn]*ClientInfo),
		performanceData: make(chan *PerformanceGraphData, 100),
		alertsChannel:   make(chan *PerformanceAlert, 200),
		stopChannel:     make(chan struct{}),
	}
}

// Start begins real-time .monitoring
func (rpm *RealtimePerformanceMonitor) Start(ctx context.Context) error {
	rpm.runningMutex.Lock()
	if rpm.isRunning {
		rpm.runningMutex.Unlock()
		return fmt.Errorf("monitor is already running")
	}
	rpm.isRunning = true
	rpm.runningMutex.Unlock()

	rpm.logger.Info("Starting real-time performance monitor")

	// Start .monitoring goroutines
	go rpm.performanceCollectionLoop(ctx)
	go rpm.alertProcessingLoop(ctx)
	go rpm.clientCleanupLoop(ctx)

	return nil
}

// Stop stops real-time .monitoring
func (rpm *RealtimePerformanceMonitor) Stop() error {
	rpm.runningMutex.Lock()
	defer rpm.runningMutex.Unlock()

	if !rpm.isRunning {
		return fmt.Errorf("monitor is not running")
	}

	rpm.logger.Info("Stopping real-time performance monitor")

	// Signal stop
	close(rpm.stopChannel)

	// Close all WebSocket connections
	rpm.clientMutex.Lock()
	for conn := range rpm.clients {
		conn.Close()
	}
	rpm.clients = make(map[*websocket.Conn]*ClientInfo)
	rpm.clientMutex.Unlock()

	rpm.isRunning = false
	return nil
}

// HandleWebSocket handles new WebSocket connections
func (rpm *RealtimePerformanceMonitor) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check connection limit
	rpm.clientMutex.RLock()
	clientCount := len(rpm.clients)
	rpm.clientMutex.RUnlock()

	if clientCount >= rpm.config.MaxConnections {
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	// Upgrade connection
	conn, err := rpm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		rpm.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Create client info
	clientInfo := &ClientInfo{
		ID:               fmt.Sprintf("client-%d", time.Now().UnixNano()),
		ConnectedAt:      time.Now(),
		LastPingAt:       time.Now(),
		SubscribedTopics: []string{"performance", "alerts"},
		Filters:          make(map[string]interface{}),
		Compression:      false,
	}

	// Register client
	rpm.clientMutex.Lock()
	rpm.clients[conn] = clientInfo
	rpm.clientMutex.Unlock()

	rpm.logger.WithFields(logrus.Fields{
		"client_id":     clientInfo.ID,
		"remote_addr":   r.RemoteAddr,
		"total_clients": len(rpm.clients),
	}).Info("New WebSocket client connected")

	// Send initial data
	rpm.sendInitialData(conn, clientInfo)

	// Handle client messages
	go rpm.handleClientMessages(conn, clientInfo)
}

// Private methods for .monitoring loops and client handling

func (rpm *RealtimePerformanceMonitor) performanceCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(rpm.config.DataUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rpm.stopChannel:
			return
		case <-ticker.C:
			if err := rpm.collectAndBroadcastPerformanceData(ctx); err != nil {
				rpm.logger.WithError(err).Error("Failed to collect performance data")
			}
		}
	}
}

func (rpm *RealtimePerformanceMonitor) alertProcessingLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-rpm.stopChannel:
			return
		case alert := <-rpm.alertsChannel:
			rpm.broadcastAlert(alert)
		}
	}
}

func (rpm *RealtimePerformanceMonitor) clientCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rpm.stopChannel:
			return
		case <-ticker.C:
			rpm.cleanupInactiveClients()
		}
	}
}

func (rpm *RealtimePerformanceMonitor) collectAndBroadcastPerformanceData(ctx context.Context) error {
	// Collect performance data
	perfData, err := rpm.psAdapter.CollectPerformanceData(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect performance data: %w", err)
	}

	// TODO: Get base graph from graph service
	var baseGraph *models.Graph // This would come from the graph service

	// Map performance data to graph visualization
	if baseGraph != nil {
		graphData, err := rpm.graphMapper.MapPerformanceToGraph(ctx, baseGraph, perfData)
		if err != nil {
			rpm.logger.WithError(err).Error("Failed to map performance to graph")
		} else {
			rpm.stateMutex.Lock()
			rpm.lastGraphData = graphData
			rpm.lastUpdate = time.Now()
			rpm.stateMutex.Unlock()

			// Broadcast to clients
			rpm.broadcastToClients("performance", graphData)
		}
	}

	// Generate metrics summary
	metrics := rpm.generateRealtimeMetrics(perfData)
	rpm.broadcastToClients("metrics", metrics)

	// Check for alerts
	rpm.checkAndGenerateAlerts(perfData)

	return nil
}

func (rpm *RealtimePerformanceMonitor) generateRealtimeMetrics(perfData *PerformanceSchemaData) *RealtimeMetrics {
	// Extract top slow queries
	var topQueries []QueryPerformanceMetric
	for i, stmt := range perfData.StatementStats {
		if i >= 10 { // Limit to top 10
			break
		}
		topQueries = append(topQueries, QueryPerformanceMetric{
			QueryID:          stmt.SchemaName,
			DigestText:       stmt.DigestText,
			ExecutionCount:   stmt.CountStar,
			AvgExecutionTime: float64(stmt.AvgTimerWait) / 1000000.0, // Convert to milliseconds
			MaxExecutionTime: float64(stmt.MaxTimerWait) / 1000000.0,
			RowsAffected:     stmt.SumRowsAffected,
			ErrorCount:       0, // SumErrors field not available - use 0
		})
	}

	return &RealtimeMetrics{
		Timestamp:       time.Now(),
		SystemMetrics:   rpm.collectSystemMetrics(),
		DatabaseMetrics: rpm.collectDatabaseMetrics(perfData),
		TopQueries:      topQueries,
		Alerts:          make([]PerformanceAlert, 0),
	}
}

func (rpm *RealtimePerformanceMonitor) collectSystemMetrics() *SystemMetrics {
	// TODO: Implement actual system metrics collection
	return &SystemMetrics{
		CPUPercent:    0.0,
		MemoryPercent: 0.0,
		DiskUsage:     0.0,
	}
}

func (rpm *RealtimePerformanceMonitor) collectDatabaseMetrics(perfData *PerformanceSchemaData) *DatabaseMetrics {
	// Calculate QPS from statements
	var totalQueries int64
	for _, stmt := range perfData.StatementStats {
		totalQueries += stmt.CountStar
	}

	return &DatabaseMetrics{
		QueriesPerSecond: float64(totalQueries) / rpm.config.DataUpdateInterval.Seconds(),
		SlowQueries:      0,    // TODO: Calculate from perfData
		ConnectionsUsed:  1,    // ConnectionStats is a struct, not slice - use 1
		ConnectionsMax:   1000, // TODO: Get from MySQL configuration
	}
}

func (rpm *RealtimePerformanceMonitor) checkAndGenerateAlerts(perfData *PerformanceSchemaData) {
	// Check for slow queries
	for _, stmt := range perfData.StatementStats {
		avgTime := float64(stmt.AvgTimerWait) / 1000000.0 // Convert to milliseconds
		if avgTime > rpm.config.AlertThresholds.SlowQueryThreshold {
			alert := &PerformanceAlert{
				ID:          fmt.Sprintf("slow-query-%d", time.Now().UnixNano()),
				Type:        "slow_query",
				Severity:    rpm.determineSeverity(avgTime),
				Title:       "Slow Query Detected",
				Description: fmt.Sprintf("Query execution time %.2fms exceeds threshold", avgTime),
				QueryID:     stmt.DigestText,
				Value:       avgTime,
				Threshold:   rpm.config.AlertThresholds.SlowQueryThreshold,
				Timestamp:   time.Now(),
			}
			select {
			case rpm.alertsChannel <- alert:
			default:
				rpm.logger.Warn("Alert channel full, dropping alert")
			}
		}
	}
}

func (rpm *RealtimePerformanceMonitor) determineSeverity(value float64) string {
	ratio := value / rpm.config.AlertThresholds.SlowQueryThreshold
	if ratio > 3.0 {
		return "critical"
	} else if ratio > 2.0 {
		return "high"
	} else if ratio > 1.5 {
		return "medium"
	}
	return "low"
}

func (rpm *RealtimePerformanceMonitor) broadcastToClients(topic string, data interface{}) {
	message := &WebSocketMessage{
		Type:      "data",
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now(),
		ID:        fmt.Sprintf("msg-%d", time.Now().UnixNano()),
	}

	rpm.clientMutex.RLock()
	defer rpm.clientMutex.RUnlock()

	for conn, clientInfo := range rpm.clients {
		if rpm.clientSubscribedToTopic(clientInfo, topic) {
			go rpm.sendMessageToClient(conn, clientInfo, message)
		}
	}
}

func (rpm *RealtimePerformanceMonitor) broadcastAlert(alert *PerformanceAlert) {
	rpm.broadcastToClients("alerts", alert)
}

func (rpm *RealtimePerformanceMonitor) sendInitialData(conn *websocket.Conn, clientInfo *ClientInfo) {
	rpm.stateMutex.RLock()
	lastGraph := rpm.lastGraphData
	rpm.stateMutex.RUnlock()

	if lastGraph != nil {
		message := &WebSocketMessage{
			Type:      "initial",
			Topic:     "performance",
			Data:      lastGraph,
			Timestamp: time.Now(),
			ID:        "initial-data",
		}
		rpm.sendMessageToClient(conn, clientInfo, message)
	}
}

func (rpm *RealtimePerformanceMonitor) handleClientMessages(conn *websocket.Conn, clientInfo *ClientInfo) {
	defer func() {
		rpm.clientMutex.Lock()
		delete(rpm.clients, conn)
		rpm.clientMutex.Unlock()
		conn.Close()
	}()

	conn.SetReadLimit(rpm.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(rpm.config.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(rpm.config.ReadTimeout))
		clientInfo.LastPingAt = time.Now()
		return nil
	})

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				rpm.logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		// Process client message
		rpm.processClientMessage(conn, clientInfo, msg)
	}
}

func (rpm *RealtimePerformanceMonitor) processClientMessage(conn *websocket.Conn, clientInfo *ClientInfo, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		if topic, ok := msg["topic"].(string); ok {
			rpm.subscribeClientToTopic(clientInfo, topic)
		}
	case "unsubscribe":
		if topic, ok := msg["topic"].(string); ok {
			rpm.unsubscribeClientFromTopic(clientInfo, topic)
		}
	case "filter":
		if filters, ok := msg["filters"].(map[string]interface{}); ok {
			clientInfo.Filters = filters
		}
	case "ping":
		rpm.sendPong(conn, clientInfo)
	}
}

func (rpm *RealtimePerformanceMonitor) sendMessageToClient(conn *websocket.Conn, clientInfo *ClientInfo, message *WebSocketMessage) {
	conn.SetWriteDeadline(time.Now().Add(rpm.config.WriteTimeout))
	if err := conn.WriteJSON(message); err != nil {
		rpm.logger.WithError(err).WithField("client_id", clientInfo.ID).Error("Failed to send message to client")
	}
}

func (rpm *RealtimePerformanceMonitor) clientSubscribedToTopic(clientInfo *ClientInfo, topic string) bool {
	for _, subscribedTopic := range clientInfo.SubscribedTopics {
		if subscribedTopic == topic {
			return true
		}
	}
	return false
}

func (rpm *RealtimePerformanceMonitor) subscribeClientToTopic(clientInfo *ClientInfo, topic string) {
	if !rpm.clientSubscribedToTopic(clientInfo, topic) {
		clientInfo.SubscribedTopics = append(clientInfo.SubscribedTopics, topic)
	}
}

func (rpm *RealtimePerformanceMonitor) unsubscribeClientFromTopic(clientInfo *ClientInfo, topic string) {
	for i, subscribedTopic := range clientInfo.SubscribedTopics {
		if subscribedTopic == topic {
			clientInfo.SubscribedTopics = append(clientInfo.SubscribedTopics[:i], clientInfo.SubscribedTopics[i+1:]...)
			break
		}
	}
}

func (rpm *RealtimePerformanceMonitor) sendPong(conn *websocket.Conn, clientInfo *ClientInfo) {
	message := &WebSocketMessage{
		Type:      "pong",
		Timestamp: time.Now(),
		ID:        "pong",
	}
	rpm.sendMessageToClient(conn, clientInfo, message)
}

func (rpm *RealtimePerformanceMonitor) cleanupInactiveClients() {
	rpm.clientMutex.Lock()
	defer rpm.clientMutex.Unlock()

	cutoff := time.Now().Add(-rpm.config.PingTimeout)
	for conn, clientInfo := range rpm.clients {
		if clientInfo.LastPingAt.Before(cutoff) {
			conn.Close()
			delete(rpm.clients, conn)
			rpm.logger.WithField("client_id", clientInfo.ID).Info("Cleaned up inactive client")
		}
	}
}

// GetConnectedClients returns information about connected clients
func (rpm *RealtimePerformanceMonitor) GetConnectedClients() []*ClientInfo {
	rpm.clientMutex.RLock()
	defer rpm.clientMutex.RUnlock()

	clients := make([]*ClientInfo, 0, len(rpm.clients))
	for _, clientInfo := range rpm.clients {
		clients = append(clients, clientInfo)
	}
	return clients
}

// GetLastGraphData returns the most recent performance graph data
func (rpm *RealtimePerformanceMonitor) GetLastGraphData() *PerformanceGraphData {
	rpm.stateMutex.RLock()
	defer rpm.stateMutex.RUnlock()
	return rpm.lastGraphData
}

// Default configuration
func defaultRealtimeMonitorConfig() *RealtimeMonitorConfig {
	return &RealtimeMonitorConfig{
		DataUpdateInterval:   5 * time.Second,
		HeartbeatInterval:    30 * time.Second,
		MaxConnections:       100,
		WriteTimeout:         10 * time.Second,
		ReadTimeout:          60 * time.Second,
		PingTimeout:          90 * time.Second,
		MaxMessageSize:       512,
		MetricsRetention:     1 * time.Hour,
		CompressionEnabled:   true,
		MaxConcurrentQueries: 10,
		MemoryLimitMB:        100,
		CPUThreshold:         80.0,
		AlertThresholds: AlertThresholds{
			HighLatency:        1000.0, // 1 second
			HighErrorRate:      5.0,    // 5%
			HighCPUUsage:       80.0,   // 80%
			HighMemoryUsage:    85.0,   // 85%
			SlowQueryThreshold: 200.0,  // 200ms
			DeadlockThreshold:  5,      // 5 deadlocks per minute
		},
	}
}
