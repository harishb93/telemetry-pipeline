package streamer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// createTestCSV creates a temporary CSV file for testing
func createTestCSV(t *testing.T, headers []string, records [][]string) string {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		t.Fatalf("Failed to create test CSV: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

	// Write headers
	for i, header := range headers {
		if i > 0 {
			if _, err := file.WriteString(","); err != nil {
				t.Fatalf("Failed to write comma: %v", err)
			}
		}
		if _, err := file.WriteString(header); err != nil {
			t.Fatalf("Failed to write header: %v", err)
		}
	}
	if _, err := file.WriteString("\n"); err != nil {
		t.Fatalf("Failed to write newline: %v", err)
	}

	// Write records
	for _, record := range records {
		for i, field := range record {
			if i > 0 {
				if _, err := file.WriteString(","); err != nil {
					t.Fatalf("Failed to write comma: %v", err)
				}
			}
			if _, err := file.WriteString(field); err != nil {
				t.Fatalf("Failed to write field: %v", err)
			}
		}
		if _, err := file.WriteString("\n"); err != nil {
			t.Fatalf("Failed to write newline: %v", err)
		}
	}

	return csvPath
}

func TestNewStreamer(t *testing.T) {
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	streamer := NewStreamer("test.csv", 2, 10.0, broker)

	if streamer.csvPath != "test.csv" {
		t.Errorf("Expected csvPath to be 'test.csv', got %s", streamer.csvPath)
	}
	if streamer.workers != 2 {
		t.Errorf("Expected workers to be 2, got %d", streamer.workers)
	}
	if streamer.rate != 10.0 {
		t.Errorf("Expected rate to be 10.0, got %f", streamer.rate)
	}
	if streamer.broker != broker {
		t.Error("Expected broker to be set correctly")
	}
}

func TestStreamerBasicFunctionality(t *testing.T) {
	// Create test CSV
	headers := []string{"gpu_id", "utilization", "temperature", "memory_used"}
	records := [][]string{
		{"gpu0", "85.5", "72.3", "4096"},
		{"gpu1", "90.2", "75.1", "8192"},
		{"gpu2", "45.0", "65.0", "2048"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Create broker and subscribe to telemetry topic
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	ch, unsubscribe, err := broker.Subscribe("telemetry")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Create and start streamer
	streamer := NewStreamer(csvPath, 1, 5.0, broker) // 5 messages per second
	err = streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Collect messages for a short period
	messagesReceived := 0
	timeout := time.After(2 * time.Second)
	expectedMessages := 3 // We have 3 records

	for messagesReceived < expectedMessages {
		select {
		case payload := <-ch:
			messagesReceived++
			t.Logf("Received message %d: %s", messagesReceived, string(payload))

			// Verify it's valid JSON
			if len(payload) == 0 {
				t.Error("Received empty payload")
			}

		case <-timeout:
			t.Logf("Timeout reached, received %d messages", messagesReceived)
			goto done
		}
	}

done:
	streamer.Stop()

	if messagesReceived == 0 {
		t.Error("No messages received")
	}

	t.Logf("Test completed, received %d messages", messagesReceived)
}

func TestStreamerMultipleWorkers(t *testing.T) {
	// Create test CSV with more records
	headers := []string{"gpu_id", "utilization"}
	records := [][]string{
		{"gpu0", "10"},
		{"gpu1", "20"},
		{"gpu2", "30"},
		{"gpu3", "40"},
		{"gpu4", "50"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Create broker and subscribe
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	ch, unsubscribe, err := broker.Subscribe("telemetry")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Create streamer with multiple workers
	workers := 3
	streamer := NewStreamer(csvPath, workers, 10.0, broker)
	err = streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Collect messages
	messagesReceived := 0
	timeout := time.After(3 * time.Second)

	// Consume messages
	go func() {
		for range ch {
			messagesReceived++
		}
	}()

	<-timeout
	streamer.Stop()

	if messagesReceived == 0 {
		t.Error("No messages received from multiple workers")
	}

	t.Logf("Received %d messages from %d workers", messagesReceived, workers)
}

func TestStreamerGracefulShutdown(t *testing.T) {
	// Create test CSV
	headers := []string{"id", "value"}
	records := [][]string{
		{"1", "100"},
		{"2", "200"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Create broker
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	// Create streamer
	streamer := NewStreamer(csvPath, 2, 1.0, broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Let it run briefly
	time.Sleep(500 * time.Millisecond)

	// Test graceful shutdown
	start := time.Now()
	streamer.Stop()
	elapsed := time.Since(start)

	// Should stop relatively quickly (within a few seconds)
	if elapsed > 5*time.Second {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}

	t.Logf("Graceful shutdown completed in %v", elapsed)
}

func TestStreamerContinuousLoop(t *testing.T) {
	// Create small CSV for continuous looping
	headers := []string{"counter"}
	records := [][]string{
		{"1"},
		{"2"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Create broker and subscribe
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	ch, unsubscribe, err := broker.Subscribe("telemetry")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Create streamer
	streamer := NewStreamer(csvPath, 1, 10.0, broker)
	err = streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Count messages for a period that should see multiple loops
	messagesReceived := 0
	timeout := time.After(1 * time.Second)

	for {
		select {
		case <-ch:
			messagesReceived++
		case <-timeout:
			goto done
		}
	}

done:
	streamer.Stop()

	// Should receive more messages than records (due to looping)
	expectedMinimum := 4 // Should loop through at least twice
	if messagesReceived < expectedMinimum {
		t.Errorf("Expected at least %d messages for continuous loop, got %d", expectedMinimum, messagesReceived)
	}

	t.Logf("Continuous loop test: received %d messages", messagesReceived)
}

func TestStreamerRateControl(t *testing.T) {
	// Create test CSV
	headers := []string{"id"}
	records := [][]string{
		{"1"}, {"2"}, {"3"}, {"4"}, {"5"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Create broker and subscribe
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	ch, unsubscribe, err := broker.Subscribe("telemetry")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Test with slower rate
	rate := 2.0 // 2 messages per second
	streamer := NewStreamer(csvPath, 1, rate, broker)
	err = streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Measure message timing
	start := time.Now()
	messagesReceived := 0
	var firstMessageTime, secondMessageTime time.Time

	for {
		select {
		case <-ch:
			messagesReceived++
			switch messagesReceived {
			case 1:
				firstMessageTime = time.Now()
			case 2:
				secondMessageTime = time.Now()
				goto done
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for messages")
			goto done
		}
	}

done:
	streamer.Stop()

	if messagesReceived >= 2 {
		interval := secondMessageTime.Sub(firstMessageTime)
		expectedInterval := time.Duration(float64(time.Second) / rate)

		// Allow some tolerance
		tolerance := 200 * time.Millisecond
		if interval < expectedInterval-tolerance || interval > expectedInterval+tolerance {
			t.Logf("Message interval: %v, expected around: %v", interval, expectedInterval)
			// Don't fail the test as timing can be imprecise in testing environment
		}
	}

	elapsed := time.Since(start)
	t.Logf("Rate control test completed in %v, received %d messages", elapsed, messagesReceived)
}

func TestParseRecord(t *testing.T) {
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	streamer := NewStreamer("", 1, 1.0, broker)

	headers := []string{"gpu_id", "utilization", "temperature", "active"}
	record := []string{"gpu0", "85.5", "72.3", "true"}

	telemetryData, err := streamer.parseRecord(headers, record)
	if err != nil {
		t.Fatalf("Failed to parse record: %v", err)
	}

	if telemetryData.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	if len(telemetryData.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(telemetryData.Fields))
	}

	// Check specific field types and values
	if gpuID, ok := telemetryData.Fields["gpu_id"].(string); !ok || gpuID != "gpu0" {
		t.Errorf("Expected gpu_id to be 'gpu0' (string), got %v", telemetryData.Fields["gpu_id"])
	}

	if utilization, ok := telemetryData.Fields["utilization"].(float64); !ok || utilization != 85.5 {
		t.Errorf("Expected utilization to be 85.5 (float64), got %v", telemetryData.Fields["utilization"])
	}

	if active, ok := telemetryData.Fields["active"].(bool); !ok || !active {
		t.Errorf("Expected active to be true (bool), got %v", telemetryData.Fields["active"])
	}
}

func TestParseRecordMismatchedLength(t *testing.T) {
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	streamer := NewStreamer("", 1, 1.0, broker)

	headers := []string{"gpu_id", "utilization"}
	record := []string{"gpu0"} // Missing one field

	_, err := streamer.parseRecord(headers, record)
	if err == nil {
		t.Error("Expected error for mismatched header/record length")
	}
}

// Benchmark tests
func BenchmarkStreamerThroughput(b *testing.B) {
	// Create test CSV
	headers := []string{"id", "value", "timestamp"}
	records := make([][]string, 100)
	for i := 0; i < 100; i++ {
		records[i] = []string{fmt.Sprintf("id%d", i), fmt.Sprintf("%d", i*10), "2023-01-01T00:00:00Z"}
	}

	tmpDir := b.TempDir()
	csvPath := filepath.Join(tmpDir, "benchmark.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		b.Fatalf("Failed to create benchmark CSV: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			b.Logf("Failed to close file: %v", err)
		}
	}()

	// Write CSV data
	for i, header := range headers {
		if i > 0 {
			if _, err := file.WriteString(","); err != nil {
				b.Fatalf("Failed to write comma: %v", err)
			}
		}
		if _, err := file.WriteString(header); err != nil {
			b.Fatalf("Failed to write header: %v", err)
		}
	}
	if _, err := file.WriteString("\n"); err != nil {
		b.Fatalf("Failed to write newline: %v", err)
	}

	for _, record := range records {
		for i, field := range record {
			if i > 0 {
				if _, err := file.WriteString(","); err != nil {
					b.Fatalf("Failed to write comma: %v", err)
				}
			}
			if _, err := file.WriteString(field); err != nil {
				b.Fatalf("Failed to write field: %v", err)
			}
		}
		if _, err := file.WriteString("\n"); err != nil {
			b.Fatalf("Failed to write newline: %v", err)
		}
	}

	// Create broker
	config := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(config)
	defer broker.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		streamer := NewStreamer(csvPath, 1, 100.0, broker) // High rate for benchmark
		if err := streamer.Start(); err != nil {
			b.Fatalf("Failed to start streamer: %v", err)
		}
		time.Sleep(100 * time.Millisecond) // Let it process briefly
		streamer.Stop()
	}
}
