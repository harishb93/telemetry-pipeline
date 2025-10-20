package collector

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

func TestCollectorConfig_DefaultValues(t *testing.T) {
	config := CollectorConfig{
		Workers:           2,
		DataDir:           "/tmp/collector-test",
		MaxEntriesPerGPU:  500,
		CheckpointEnabled: true,
		CheckpointDir:     "/tmp/checkpoints",
		HealthPort:        "8083",
		MQTopic:           "telemetry",
	}

	if config.Workers != 2 {
		t.Errorf("Expected 2 workers, got %d", config.Workers)
	}
	if config.DataDir != "/tmp/collector-test" {
		t.Errorf("Expected '/tmp/collector-test' data dir, got %s", config.DataDir)
	}
	if config.MaxEntriesPerGPU != 500 {
		t.Errorf("Expected 500 max entries, got %d", config.MaxEntriesPerGPU)
	}
	if !config.CheckpointEnabled {
		t.Error("Expected checkpoint enabled to be true")
	}
	if config.HealthPort != "8083" {
		t.Errorf("Expected health port '8083', got %s", config.HealthPort)
	}
}

func TestCollector_Creation(t *testing.T) {
	config := CollectorConfig{
		Workers:           3,
		DataDir:           "/tmp/test-creation",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8184",
		MQTopic:           "test-topic",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	defer broker.Close()

	collector := NewCollector(broker, config)

	if collector == nil {
		t.Fatal("Failed to create collector")
	}
}

func TestCollector_BasicTelemetryRetrieval(t *testing.T) {
	config := CollectorConfig{
		Workers:           2,
		DataDir:           "/tmp/test-basic",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8285",
		MQTopic:           "test-basic",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	defer broker.Close()

	collector := NewCollector(broker, config)

	// Test GetTelemetryForGPU with a test GPU ID
	entries := collector.GetTelemetryForGPU("test-gpu-0", 10)
	// Should return empty slice for non-existent GPU
	if len(entries) != 0 {
		t.Errorf("Expected empty slice for non-existent GPU, got %d entries", len(entries))
	}
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	config := CollectorConfig{
		Workers:           2,
		DataDir:           "/tmp/test-concurrent",
		MaxEntriesPerGPU:  500,
		CheckpointEnabled: false,
		HealthPort:        "8286",
		MQTopic:           "test-concurrent",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	defer broker.Close()

	collector := NewCollector(broker, config)

	var wg sync.WaitGroup
	numGoroutines := 5

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				gpuID := fmt.Sprintf("gpu-%d", id)
				entries := collector.GetTelemetryForGPU(gpuID, 10)
				_ = entries // Use the entries to avoid "unused" warnings
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Test passes if no panics occur during concurrent access
}

func TestCollector_StartStop(t *testing.T) {
	config := CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test-start-stop",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8287",
		MQTopic:           "test-start-stop",
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	defer broker.Close()

	collector := NewCollector(broker, config)

	// Test starting collector
	go func() {
		if err := collector.Start(); err != nil {
			t.Errorf("Failed to start collector: %v", err)
		}
	}()

	// Let it run for a short time
	time.Sleep(100 * time.Millisecond)

	// Test stopping collector
	collector.Stop()

	// Test passes if start/stop cycle completes without panics
}

func TestCollector_ConfigValidation(t *testing.T) {
	testCases := []struct {
		name    string
		config  CollectorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CollectorConfig{
				Workers:           2,
				DataDir:           "/tmp/test",
				MaxEntriesPerGPU:  100,
				CheckpointEnabled: false,
				HealthPort:        "8290",
				MQTopic:           "test",
			},
			wantErr: false,
		},
		{
			name: "zero workers",
			config: CollectorConfig{
				Workers:           0,
				DataDir:           "/tmp/test",
				MaxEntriesPerGPU:  100,
				CheckpointEnabled: false,
				HealthPort:        "8291",
				MQTopic:           "test",
			},
			wantErr: false, // Collector should handle this gracefully
		},
	}

	broker := mq.NewBroker(mq.DefaultBrokerConfig())
	defer broker.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			collector := NewCollector(broker, tc.config)
			if collector == nil && !tc.wantErr {
				t.Error("Expected collector to be created")
			}
		})
	}
}
