package collector

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
	"github.com/harishb93/telemetry-pipeline/internal/persistence"
)

// TestTelemetryConversion tests the conversion from StreamerMessage to Telemetry
func TestTelemetryConversion(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8082",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	timestamp := time.Now()
	streamerMsg := StreamerMessage{
		Timestamp: timestamp,
		Fields: map[string]interface{}{
			"gpu_id":           "gpu_0",
			"temperature":      75.5,
			"utilization":      85.2,
			"memory_used":      4096.0,
			"power_draw":       "250.5", // String that should be converted to float
			"fan_speed":        "2500",  // String that should be converted to float
			"some_other_field": "not_a_number",
		},
	}

	telemetry, err := collector.convertToTelemetry(streamerMsg)
	if err != nil {
		t.Fatalf("Failed to convert telemetry: %v", err)
	}

	// Verify basic fields
	if telemetry.GPUId != "gpu_0" {
		t.Errorf("Expected GPU ID 'gpu_0', got '%s'", telemetry.GPUId)
	}

	if !telemetry.Timestamp.Equal(timestamp) {
		t.Errorf("Expected timestamp %v, got %v", timestamp, telemetry.Timestamp)
	}

	// Verify numeric conversions
	expectedMetrics := map[string]float64{
		"temperature": 75.5,
		"utilization": 85.2,
		"memory_used": 4096.0,
		"power_draw":  250.5,
		"fan_speed":   2500.0,
	}

	for key, expected := range expectedMetrics {
		if actual, exists := telemetry.Metrics[key]; !exists {
			t.Errorf("Expected metric '%s' not found", key)
		} else if actual != expected {
			t.Errorf("Expected metric '%s' = %f, got %f", key, expected, actual)
		}
	}

	// Verify non-numeric field is excluded
	if _, exists := telemetry.Metrics["some_other_field"]; exists {
		t.Error("Non-numeric field should not be included in metrics")
	}
}

// TestTelemetryConversionWithoutGPUID tests handling of messages without GPU ID
func TestTelemetryConversionWithoutGPUID(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8082",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	streamerMsg := StreamerMessage{
		Timestamp: time.Now(),
		Fields: map[string]interface{}{
			"temperature": 75.5,
			"utilization": 85.2,
		},
	}

	_, err := collector.convertToTelemetry(streamerMsg)
	if err == nil {
		t.Error("Expected error for message without GPU ID, but got none")
	}
}

// TestTelemetryConversionWithZeroTimestamp tests handling of zero timestamps
func TestTelemetryConversionWithZeroTimestamp(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8082",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	streamerMsg := StreamerMessage{
		Timestamp: time.Time{}, // Zero timestamp
		Fields: map[string]interface{}{
			"gpu_id":      "gpu_0",
			"temperature": 75.5,
		},
	}

	before := time.Now()
	telemetry, err := collector.convertToTelemetry(streamerMsg)
	after := time.Now()

	if err != nil {
		t.Fatalf("Failed to convert telemetry: %v", err)
	}

	// Should use current time when timestamp is zero
	if telemetry.Timestamp.Before(before) || telemetry.Timestamp.After(after) {
		t.Error("Expected current time to be used for zero timestamp")
	}
}

// TestMemoryStorageIntegration tests the integration with memory storage
func TestMemoryStorageIntegration(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8082",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Test storing and retrieving telemetry
	telemetry := &Telemetry{
		GPUId:     "gpu_0",
		Metrics:   map[string]float64{"temperature": 75.5, "utilization": 85.2},
		Timestamp: time.Now(),
	}

	// Convert to persistence type and store
	persistenceTelemetry := persistence.Telemetry{
		GPUId:     telemetry.GPUId,
		Metrics:   telemetry.Metrics,
		Timestamp: telemetry.Timestamp,
	}
	collector.memoryStorage.StoreTelemetry(persistenceTelemetry)

	// Retrieve and verify
	retrieved := collector.GetTelemetryForGPU("gpu_0", 10)
	if len(retrieved) != 1 {
		t.Fatalf("Expected 1 telemetry entry, got %d", len(retrieved))
	}

	if retrieved[0].GPUId != "gpu_0" {
		t.Errorf("Expected GPU ID 'gpu_0', got '%s'", retrieved[0].GPUId)
	}

	if retrieved[0].Metrics["temperature"] != 75.5 {
		t.Errorf("Expected temperature 75.5, got %f", retrieved[0].Metrics["temperature"])
	}
}

// TestGetMemoryStats tests memory storage statistics
func TestGetMemoryStats(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8082",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Initially should be empty
	stats := collector.GetMemoryStats()
	if stats["total_entries"].(int) != 0 {
		t.Errorf("Expected 0 total entries, got %v", stats["total_entries"])
	}

	// Add some telemetry
	for i := 0; i < 3; i++ {
		telemetry := persistence.Telemetry{
			GPUId:     "gpu_0",
			Metrics:   map[string]float64{"temperature": 75.5},
			Timestamp: time.Now(),
		}
		collector.memoryStorage.StoreTelemetry(telemetry)
	}

	// Check updated stats
	stats = collector.GetMemoryStats()
	if stats["total_entries"].(int) != 3 {
		t.Errorf("Expected 3 total entries, got %v", stats["total_entries"])
	}

	if stats["total_gpus"].(int) != 1 {
		t.Errorf("Expected 1 GPU, got %v", stats["total_gpus"])
	}
}

// TestCollectorLifecycle tests starting and stopping the collector
func TestCollectorLifecycle(t *testing.T) {
	config := CollectorConfig{
		Workers:           2,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8083", // Different port to avoid conflicts
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Start collector
	go func() {
		if err := collector.Start(); err != nil {
			t.Errorf("Failed to start collector: %v", err)
		}
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop collector
	collector.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)
}

// TestCheckpointManager tests checkpoint functionality
func TestCheckpointManager(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: true,
		CheckpointDir:     "/tmp/test-checkpoint",
		HealthPort:        "8084",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Clean up any existing checkpoint data for isolation
	checkpointName := "worker-1"
	collector.checkpointMgr.DeleteCheckpoint(checkpointName)

	// Test updating processed count
	err := collector.checkpointMgr.UpdateProcessedCount(checkpointName, 100)
	if err != nil {
		t.Fatalf("Failed to update processed count: %v", err)
	}

	// Load and verify checkpoint
	checkpoint, err := collector.checkpointMgr.LoadCheckpoint(checkpointName)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	if checkpoint.ProcessedCount != 100 {
		t.Errorf("Expected processed count 100, got %d", checkpoint.ProcessedCount)
	}

	// Update again
	err = collector.checkpointMgr.UpdateProcessedCount(checkpointName, 50)
	if err != nil {
		t.Fatalf("Failed to update processed count again: %v", err)
	}

	// Verify incremental update
	checkpoint, err = collector.checkpointMgr.LoadCheckpoint(checkpointName)
	if err != nil {
		t.Fatalf("Failed to load checkpoint after update: %v", err)
	}

	if checkpoint.ProcessedCount != 150 {
		t.Errorf("Expected processed count 150, got %d", checkpoint.ProcessedCount)
	}
}

// TestMessageHandling tests the complete message handling pipeline
func TestMessageHandling(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8086",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Create a test message
	streamerMsg := StreamerMessage{
		Timestamp: time.Now(),
		Fields: map[string]interface{}{
			"gpu_id":      "gpu_test",
			"temperature": 80.0,
			"utilization": 90.0,
		},
	}

	msgBytes, err := json.Marshal(streamerMsg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Create a message with an acknowledgment function
	msg := mq.Message{
		Payload: msgBytes,
		Ack: func() {
			// Message acknowledged
		},
	}

	// Handle the message
	err = collector.handleMessage(1, msg)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	// Verify telemetry was stored in memory
	retrieved := collector.GetTelemetryForGPU("gpu_test", 10)
	if len(retrieved) != 1 {
		t.Fatalf("Expected 1 telemetry entry, got %d", len(retrieved))
	}

	if retrieved[0].Metrics["temperature"] != 80.0 {
		t.Errorf("Expected temperature 80.0, got %f", retrieved[0].Metrics["temperature"])
	}
}

// Benchmark telemetry conversion
func BenchmarkTelemetryConversion(b *testing.B) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8085",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	streamerMsg := StreamerMessage{
		Timestamp: time.Now(),
		Fields: map[string]interface{}{
			"gpu_id":      "gpu_0",
			"temperature": 75.5,
			"utilization": 85.2,
			"memory_used": 4096.0,
			"power_draw":  250.5,
			"fan_speed":   2500.0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.convertToTelemetry(streamerMsg)
		if err != nil {
			b.Fatalf("Conversion failed: %v", err)
		}
	}
}

// TestGetAllHosts tests the GetAllHosts method
func TestGetAllHosts(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8084",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Initially should be empty
	hosts := collector.GetAllHosts()
	if len(hosts) != 0 {
		t.Errorf("Expected 0 hosts initially, got %d", len(hosts))
	}

	// Add telemetry data with different hostnames
	testData := []struct {
		hostname string
		gpuID    string
	}{
		{"host1", "gpu_0"},
		{"host1", "gpu_1"},
		{"host2", "gpu_2"},
		{"host3", "gpu_3"},
	}

	for _, data := range testData {
		telemetry := persistence.Telemetry{
			GPUId:     data.gpuID,
			Hostname:  data.hostname,
			Metrics:   map[string]float64{"temperature": 75.5},
			Timestamp: time.Now(),
		}
		collector.memoryStorage.StoreTelemetry(telemetry)
	}

	// Check hosts
	hosts = collector.GetAllHosts()
	expectedHosts := 3
	if len(hosts) != expectedHosts {
		t.Errorf("Expected %d hosts, got %d", expectedHosts, len(hosts))
	}

	// Verify unique hostnames
	hostSet := make(map[string]bool)
	for _, host := range hosts {
		if hostSet[host] {
			t.Errorf("Duplicate host found: %s", host)
		}
		hostSet[host] = true
	}

	// Verify expected hostnames are present
	expectedHostnames := []string{"host1", "host2", "host3"}
	for _, expected := range expectedHostnames {
		if !hostSet[expected] {
			t.Errorf("Expected hostname %s not found in results", expected)
		}
	}
}

// TestGetGPUsForHost tests the GetGPUsForHost method
func TestGetGPUsForHost(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8085",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	collector := NewCollector(broker, config)

	// Add telemetry data
	testData := []struct {
		hostname string
		gpuID    string
	}{
		{"host1", "gpu_0"},
		{"host1", "gpu_1"},
		{"host1", "gpu_2"},
		{"host2", "gpu_3"},
		{"host2", "gpu_4"},
	}

	for _, data := range testData {
		telemetry := persistence.Telemetry{
			GPUId:     data.gpuID,
			Hostname:  data.hostname,
			Metrics:   map[string]float64{"temperature": 75.5},
			Timestamp: time.Now(),
		}
		collector.memoryStorage.StoreTelemetry(telemetry)
	}

	// Test getting GPUs for host1
	gpus := collector.GetGPUsForHost("host1")
	expectedGPUs := 3
	if len(gpus) != expectedGPUs {
		t.Errorf("Expected %d GPUs for host1, got %d", expectedGPUs, len(gpus))
	}

	// Verify unique GPU IDs
	gpuSet := make(map[string]bool)
	for _, gpu := range gpus {
		if gpuSet[gpu] {
			t.Errorf("Duplicate GPU found for host1: %s", gpu)
		}
		gpuSet[gpu] = true
	}

	// Test getting GPUs for host2
	gpus = collector.GetGPUsForHost("host2")
	expectedGPUs = 2
	if len(gpus) != expectedGPUs {
		t.Errorf("Expected %d GPUs for host2, got %d", expectedGPUs, len(gpus))
	}

	// Test getting GPUs for non-existent host
	gpus = collector.GetGPUsForHost("non-existent-host")
	if len(gpus) != 0 {
		t.Errorf("Expected 0 GPUs for non-existent host, got %d", len(gpus))
	}
}
