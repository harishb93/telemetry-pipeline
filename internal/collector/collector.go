package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/logger"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
	"github.com/harishb93/telemetry-pipeline/internal/persistence"
)

// Telemetry represents a typed telemetry data point
type Telemetry struct {
	GPUId     string             `json:"gpu_id"`
	Hostname  string             `json:"hostname"`
	Metrics   map[string]float64 `json:"metrics"`
	Timestamp time.Time          `json:"timestamp"`
}

// StreamerMessage represents the message format from the streamer
type StreamerMessage struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// CollectorConfig holds configuration for the collector
type CollectorConfig struct {
	Workers           int
	DataDir           string
	MaxEntriesPerGPU  int
	CheckpointEnabled bool
	CheckpointDir     string
	HealthPort        string
	MQTopic           string
}

// Collector handles telemetry data collection and persistence
type Collector struct {
	config        CollectorConfig
	broker        mq.BrokerInterface
	fileStorage   *persistence.FileStorage
	memoryStorage *persistence.MemoryStorage
	checkpointMgr *persistence.CheckpointManager
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *logger.Logger
	wg            sync.WaitGroup
	healthServer  *http.Server
}

// NewCollector creates a new collector instance
func NewCollector(broker mq.BrokerInterface, config CollectorConfig) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	fileStorage := persistence.NewFileStorage(config.DataDir)
	memoryStorage := persistence.NewMemoryStorage(config.MaxEntriesPerGPU)

	var checkpointMgr *persistence.CheckpointManager
	if config.CheckpointEnabled {
		checkpointMgr = persistence.NewCheckpointManager(config.CheckpointDir)
	}

	return &Collector{
		config:        config,
		broker:        broker,
		fileStorage:   fileStorage,
		memoryStorage: memoryStorage,
		checkpointMgr: checkpointMgr,
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger.NewFromEnv().WithComponent("collector"),
	}
}

// Start begins collecting telemetry data with specified number of workers
func (c *Collector) Start() error {
	c.logger.Info("Collector starting", "workers", c.config.Workers)

	// Start health server
	if err := c.startHealthServer(); err != nil {
		return fmt.Errorf("failed to start health server: %w", err)
	}

	// Start worker goroutines
	for i := 0; i < c.config.Workers; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	return nil
}

// Stop gracefully stops the collector
func (c *Collector) Stop() {
	c.logger.Info("Collector stopping")

	// Stop health server
	if c.healthServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.healthServer.Shutdown(shutdownCtx); err != nil {
			c.logger.Error("Failed to shutdown health server", "error", err)
		}
	}

	// Stop workers
	c.cancel()
	c.wg.Wait()

	c.logger.Info("Collector stopped")
}

// worker runs a single worker goroutine
func (c *Collector) worker(workerID int) {
	defer c.wg.Done()
	c.logger.Info("Worker started", "worker_id", workerID)

	// Subscribe to telemetry topic with acknowledgment support
	topic := c.config.MQTopic
	if topic == "" {
		topic = "telemetry" // default topic
	}
	ch, unsubscribe, err := c.broker.SubscribeWithAck(topic)
	if err != nil {
		c.logger.Error("Worker failed to subscribe", "worker_id", workerID, "error", err)
		return
	}
	defer unsubscribe()

	// Load checkpoint if enabled
	var lastOffset int64
	if c.checkpointMgr != nil {
		checkpointName := fmt.Sprintf("worker-%d", workerID)
		if checkpoint, err := c.checkpointMgr.LoadCheckpoint(checkpointName); err == nil {
			lastOffset = checkpoint.ProcessedCount
			c.logger.Debug("Worker loaded checkpoint", "worker_id", workerID, "offset", lastOffset)
		}
	}

	processedCount := 0

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Worker stopping", "worker_id", workerID, "messages_processed", processedCount)
			return
		case msg := <-ch:
			if err := c.handleMessage(workerID, msg); err != nil {
				c.logger.Error("Worker error handling message", "worker_id", workerID, "error", err)
				// Don't acknowledge failed messages for potential retry
				continue
			}

			// Acknowledge successful processing
			msg.Ack()
			processedCount++

			// Update checkpoint periodically
			if c.checkpointMgr != nil && processedCount%100 == 0 {
				checkpointName := fmt.Sprintf("worker-%d", workerID)
				if err := c.checkpointMgr.UpdateProcessedCount(checkpointName, 100); err != nil {
					c.logger.Error("Worker failed to update checkpoint", "worker_id", workerID, "error", err)
				}
			}

			if processedCount%1000 == 0 {
				c.logger.Debug("Worker batch processed", "worker_id", workerID, "messages_processed", processedCount)
			}
		}
	}
}

// handleMessage processes a single telemetry message
func (c *Collector) handleMessage(workerID int, msg mq.Message) error {
	// Parse the JSON message from streamer
	var streamerMsg StreamerMessage
	if err := json.Unmarshal(msg.Payload, &streamerMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Convert to typed Telemetry struct
	telemetry, err := c.convertToTelemetry(streamerMsg)
	if err != nil {
		return fmt.Errorf("failed to convert message: %w", err)
	}

	// Convert to persistence.Telemetry for file storage
	persistenceTelemetry := persistence.Telemetry{
		GPUId:     telemetry.GPUId,
		Hostname:  telemetry.Hostname,
		Metrics:   telemetry.Metrics,
		Timestamp: telemetry.Timestamp,
	}

	// Persist to file storage
	if err := c.fileStorage.WriteTelemetry(persistenceTelemetry); err != nil {
		c.logger.Error("Worker failed to write to file storage", "worker_id", workerID, "error", err)
		// Continue processing even if file write fails
	}

	// Store in memory
	c.memoryStorage.StoreTelemetry(persistenceTelemetry)

	return nil
}

// convertToTelemetry converts a StreamerMessage to typed Telemetry
func (c *Collector) convertToTelemetry(msg StreamerMessage) (*Telemetry, error) {
	telemetry := &Telemetry{
		Metrics:   make(map[string]float64),
		Timestamp: msg.Timestamp,
	}

	// If timestamp is zero, use current time
	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}

	// Extract GPU ID and metrics from fields
	// Handle DCGM format - use uuid as the primary identifier
	if uuidRaw, exists := msg.Fields["uuid"]; exists {
		// Use the UUID directly as the GPU identifier
		if uuidStr, ok := uuidRaw.(string); ok {
			telemetry.GPUId = uuidStr
		}
	} else if gpuIDRaw, exists := msg.Fields["gpu_id"]; exists {
		// Fallback to gpu_id if uuid is not available
		if gpuIDStr, ok := gpuIDRaw.(string); ok {
			// Use the gpu_id as-is if it's already in the expected format
			telemetry.GPUId = gpuIDStr
		} else if gpuIDFloat, ok := gpuIDRaw.(float64); ok {
			// If it's a number, format it as gpu-xxx
			telemetry.GPUId = fmt.Sprintf("gpu-%03.0f", gpuIDFloat)
		}
	}

	// Extract hostname from DCGM format
	if hostnameRaw, exists := msg.Fields["Hostname"]; exists {
		if hostnameStr, ok := hostnameRaw.(string); ok {
			telemetry.Hostname = hostnameStr
		}
	}

	// Extract the main metric value
	if valueRaw, exists := msg.Fields["value"]; exists {
		if floatVal, err := convertToFloat64(valueRaw); err == nil {
			// Determine metric name from metric_name field
			metricName := "value" // default
			if metricNameRaw, exists := msg.Fields["metric_name"]; exists {
				if metricNameStr, ok := metricNameRaw.(string); ok {
					metricName = metricNameStr
				}
			}
			telemetry.Metrics[metricName] = floatVal
		}
	}

	// Also include other numeric fields as metrics
	for key, value := range msg.Fields {
		if key != "gpu_id" && key != "value" && key != "metric_name" {
			if floatVal, err := convertToFloat64(value); err == nil {
				telemetry.Metrics[key] = floatVal
			}
		}
	}

	// Validate that we have a GPU ID
	if telemetry.GPUId == "" {
		return nil, fmt.Errorf("missing uuid or gpu_id in telemetry data")
	}

	return telemetry, nil
}

// convertToFloat64 attempts to convert interface{} to float64
func convertToFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// startHealthServer starts the HTTP health endpoint
func (c *Collector) startHealthServer() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Simple health check - could be enhanced with more detailed status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
			c.logger.Error("Failed to write health response", "error", err)
		}
	})

	// Stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats := c.memoryStorage.GetStats()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			c.logger.Error("Failed to encode stats response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Telemetry endpoint for specific GPU
	mux.HandleFunc("/api/v1/gpus/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse GPU ID from URL path: /api/v1/gpus/{gpu_id}/telemetry
		path := r.URL.Path
		if len(path) < 15 { // Minimum: "/api/v1/gpus/x/"
			http.Error(w, "Invalid GPU ID", http.StatusBadRequest)
			return
		}

		// Extract GPU ID and check for telemetry suffix
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 4 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "gpus" {
			http.Error(w, "Invalid path format", http.StatusBadRequest)
			return
		}

		gpuID := parts[3]
		if len(parts) > 4 && parts[4] != "telemetry" {
			http.Error(w, "Invalid endpoint", http.StatusBadRequest)
			return
		}

		// Get telemetry data for the GPU
		telemetryData := c.GetTelemetryForGPU(gpuID, 100) // Get last 100 entries

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"data":   telemetryData,
			"total":  len(telemetryData),
			"gpu_id": gpuID,
		}); err != nil {
			c.logger.Error("Failed to encode telemetry response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Hosts endpoint
	mux.HandleFunc("/api/v1/hosts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		hosts := c.GetAllHosts()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"hosts": hosts,
			"total": len(hosts),
		}); err != nil {
			c.logger.Error("Failed to encode hosts response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	// Host GPUs endpoint
	mux.HandleFunc("/api/v1/hosts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse hostname from URL path: /api/v1/hosts/{hostname}/gpus
		path := r.URL.Path
		if len(path) < 16 { // Minimum: "/api/v1/hosts/x/"
			http.Error(w, "Invalid hostname", http.StatusBadRequest)
			return
		}

		// Extract hostname and check for gpus suffix
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 4 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "hosts" {
			http.Error(w, "Invalid path format", http.StatusBadRequest)
			return
		}

		hostname := parts[3]
		if len(parts) > 4 && parts[4] != "gpus" {
			http.Error(w, "Invalid endpoint", http.StatusBadRequest)
			return
		}

		// Get GPUs for the host
		gpus := c.GetGPUsForHost(hostname)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"hostname": hostname,
			"gpus":     gpus,
			"total":    len(gpus),
		}); err != nil {
			c.logger.Error("Failed to encode host GPUs response", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	c.healthServer = &http.Server{
		Addr:    ":" + c.config.HealthPort,
		Handler: mux,
	}

	go func() {
		c.logger.Info("Health server starting", "port", c.config.HealthPort)
		if err := c.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Error("Health server error", "error", err)
		}
	}()

	return nil
}

// GetMemoryStats returns current memory storage statistics
func (c *Collector) GetMemoryStats() map[string]interface{} {
	return c.memoryStorage.GetStats()
}

// GetTelemetryForGPU returns telemetry data for a specific GPU
func (c *Collector) GetTelemetryForGPU(gpuID string, limit int) []*Telemetry {
	persistenceData := c.memoryStorage.GetTelemetryForGPU(gpuID)

	// Convert and apply limit
	var result []*Telemetry
	for i, pTel := range persistenceData {
		if limit > 0 && i >= limit {
			break
		}
		tel := &Telemetry{
			GPUId:     pTel.GPUId,
			Hostname:  pTel.Hostname,
			Metrics:   pTel.Metrics,
			Timestamp: pTel.Timestamp,
		}
		result = append(result, tel)
	}
	return result
}

// GetAllHosts returns all unique hostnames that have telemetry data
func (c *Collector) GetAllHosts() []string {
	return c.memoryStorage.GetAllHosts()
}

// GetGPUsForHost returns all GPU IDs associated with a specific hostname
func (c *Collector) GetGPUsForHost(hostname string) []string {
	return c.memoryStorage.GetGPUsForHost(hostname)
}
