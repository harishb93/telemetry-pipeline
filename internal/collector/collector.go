package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
	"github.com/harishb93/telemetry-pipeline/internal/persistence"
)

// Telemetry represents a typed telemetry data point
type Telemetry struct {
	GPUId     string             `json:"gpu_id"`
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
}

// Collector handles telemetry data collection and persistence
type Collector struct {
	config        CollectorConfig
	broker        *mq.Broker
	fileStorage   *persistence.FileStorage
	memoryStorage *persistence.MemoryStorage
	checkpointMgr *persistence.CheckpointManager
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	healthServer  *http.Server
}

// NewCollector creates a new collector instance
func NewCollector(broker *mq.Broker, config CollectorConfig) *Collector {
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
	}
}

// Start begins collecting telemetry data with specified number of workers
func (c *Collector) Start() error {
	log.Printf("Collector starting with %d workers", c.config.Workers)

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
	log.Println("Collector stopping...")

	// Stop health server
	if c.healthServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.healthServer.Shutdown(shutdownCtx)
	}

	// Stop workers
	c.cancel()
	c.wg.Wait()

	log.Println("Collector stopped")
}

// worker runs a single worker goroutine
func (c *Collector) worker(workerID int) {
	defer c.wg.Done()
	log.Printf("Worker %d started", workerID)

	// Subscribe to telemetry topic with acknowledgment support
	ch, unsubscribe, err := c.broker.SubscribeWithAck("telemetry")
	if err != nil {
		log.Printf("Worker %d: Failed to subscribe: %v", workerID, err)
		return
	}
	defer unsubscribe()

	// Load checkpoint if enabled
	var lastOffset int64
	if c.checkpointMgr != nil {
		checkpointName := fmt.Sprintf("worker-%d", workerID)
		if checkpoint, err := c.checkpointMgr.LoadCheckpoint(checkpointName); err == nil {
			lastOffset = checkpoint.ProcessedCount
			log.Printf("Worker %d: Loaded checkpoint offset %d", workerID, lastOffset)
		}
	}

	processedCount := 0

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("Worker %d stopping after processing %d messages", workerID, processedCount)
			return
		case msg := <-ch:
			if err := c.handleMessage(workerID, msg); err != nil {
				log.Printf("Worker %d: Error handling message: %v", workerID, err)
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
					log.Printf("Worker %d: Failed to update checkpoint: %v", workerID, err)
				}
			}

			if processedCount%1000 == 0 {
				log.Printf("Worker %d: Processed %d messages", workerID, processedCount)
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

	// Persist to file storage
	if err := c.fileStorage.WriteTelemetry(telemetry); err != nil {
		log.Printf("Worker %d: Failed to write to file storage: %v", workerID, err)
		// Continue processing even if file write fails
	}

	// Store in memory
	persistenceTelemetry := persistence.Telemetry{
		GPUId:     telemetry.GPUId,
		Metrics:   telemetry.Metrics,
		Timestamp: telemetry.Timestamp,
	}
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
	for key, value := range msg.Fields {
		if key == "gpu_id" {
			if gpuID, ok := value.(string); ok {
				telemetry.GPUId = gpuID
			}
		} else {
			// Try to convert other fields to float64 metrics
			if floatVal, err := convertToFloat64(value); err == nil {
				telemetry.Metrics[key] = floatVal
			}
		}
	}

	// Validate that we have a GPU ID
	if telemetry.GPUId == "" {
		return nil, fmt.Errorf("missing gpu_id in telemetry data")
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
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// Stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats := c.memoryStorage.GetStats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	c.healthServer = &http.Server{
		Addr:    ":" + c.config.HealthPort,
		Handler: mux,
	}

	go func() {
		log.Printf("Health server starting on port %s", c.config.HealthPort)
		if err := c.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
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
			Metrics:   pTel.Metrics,
			Timestamp: pTel.Timestamp,
		}
		result = append(result, tel)
	}
	return result
}
