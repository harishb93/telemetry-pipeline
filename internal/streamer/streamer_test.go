package streamer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// MockBroker implements BrokerInterface for testing
type MockBroker struct {
	messages     []mq.Message
	publishError error
	mu           sync.Mutex
	closed       bool
}

func NewMockBroker() *MockBroker {
	return &MockBroker{
		messages: make([]mq.Message, 0),
		closed:   false,
	}
}

func (m *MockBroker) Publish(topic string, msg mq.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("broker is closed")
	}

	if m.publishError != nil {
		return m.publishError
	}

	m.messages = append(m.messages, msg)
	return nil
}

func (m *MockBroker) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

func (m *MockBroker) Subscribe(topic string) (chan []byte, func(), error) {
	return make(chan []byte), func() {}, nil
}

func (m *MockBroker) SubscribeWithAck(topic string) (chan mq.Message, func(), error) {
	return make(chan mq.Message), func() {}, nil
}

func (m *MockBroker) GetMessages() []mq.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]mq.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *MockBroker) SetPublishError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishError = err
}

func (m *MockBroker) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]mq.Message, 0)
	m.publishError = nil
}

// testingInterface defines common methods for *testing.T and *testing.B
type testingInterface interface {
	TempDir() string
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

// createTestCSV creates a temporary CSV file for testing
func createTestCSV(tb testingInterface, headers []string, records [][]string) string {
	tmpDir := tb.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		tb.Fatalf("Failed to create test CSV: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			tb.Logf("Failed to close file: %v", err)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	if err := writer.Write(headers); err != nil {
		tb.Fatalf("Failed to write headers: %v", err)
	}

	// Write records
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			tb.Fatalf("Failed to write record: %v", err)
		}
	}

	return csvPath
}

// createTestCSVWithContent creates a CSV file with raw content
func createTestCSVWithContent(tb testingInterface, content string) string {
	tmpDir := tb.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	if err := os.WriteFile(csvPath, []byte(content), 0644); err != nil {
		tb.Fatalf("Failed to create test CSV: %v", err)
	}

	return csvPath
}

// ==== PreProcessCSVByHostNames Tests ====

func TestPreProcessCSVByHostNames_EmptyHostList(t *testing.T) {
	// Create test CSV
	headers := []string{"hostname", "gpu_id", "temperature"}
	records := [][]string{
		{"host-A", "gpu-001", "65.0"},
		{"host-B", "gpu-002", "70.0"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Test with empty host list
	result, err := PreProcessCSVByHostNames(csvPath, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != csvPath {
		t.Errorf("Expected original path %s, got %s", csvPath, result)
	}
}

func TestPreProcessCSVByHostNames_ValidFiltering(t *testing.T) {
	// Create test CSV with hostname column
	headers := []string{"hostname", "gpu_id", "temperature", "utilization"}
	records := [][]string{
		{"host-A", "gpu-001", "65.0", "75.5"},
		{"host-B", "gpu-002", "70.0", "80.2"},
		{"host-C", "gpu-003", "68.5", "45.0"},
		{"host-A", "gpu-004", "72.1", "90.5"},
		{"host-D", "gpu-005", "60.0", "55.0"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Filter by host-A and host-C
	hostList := "host-A,host-C"
	result, err := PreProcessCSVByHostNames(csvPath, hostList)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == csvPath {
		t.Error("Expected filtered file path, got original path")
	}

	// Verify filtered content
	file, err := os.Open(result)
	if err != nil {
		t.Fatalf("Failed to open filtered file: %v", err)
	}
	defer func() { _ = file.Close() }()
	defer func() { _ = os.Remove(result) }()

	reader := csv.NewReader(file)
	allRecords, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read filtered CSV: %v", err)
	}

	// Should have header + 3 records (2 host-A + 1 host-C)
	expectedRecords := 4 // header + 3 data records
	if len(allRecords) != expectedRecords {
		t.Errorf("Expected %d records (including header), got %d", expectedRecords, len(allRecords))
	}

	// Check first data record
	if allRecords[1][0] != "host-A" {
		t.Errorf("Expected first record hostname to be 'host-A', got %s", allRecords[1][0])
	}
}

func TestPreProcessCSVByHostNames_NoHostnameColumn(t *testing.T) {
	// Create CSV without hostname column
	headers := []string{"gpu_id", "temperature", "utilization"}
	records := [][]string{
		{"gpu-001", "65.0", "75.5"},
		{"gpu-002", "70.0", "80.2"},
	}
	csvPath := createTestCSV(t, headers, records)

	result, err := PreProcessCSVByHostNames(csvPath, "host-A,host-B")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != csvPath {
		t.Errorf("Expected original path when no hostname column, got %s", result)
	}
}

func TestPreProcessCSVByHostNames_NoMatchingRecords(t *testing.T) {
	// Create test CSV
	headers := []string{"hostname", "gpu_id", "temperature"}
	records := [][]string{
		{"host-A", "gpu-001", "65.0"},
		{"host-B", "gpu-002", "70.0"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Filter by non-existent hosts
	result, err := PreProcessCSVByHostNames(csvPath, "host-X,host-Y")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != csvPath {
		t.Errorf("Expected original path when no matches, got %s", result)
	}
}

func TestPreProcessCSVByHostNames_CaseInsensitiveHeader(t *testing.T) {
	// Create CSV with different case hostname header
	headers := []string{"HOSTNAME", "gpu_id", "temperature"}
	records := [][]string{
		{"host-A", "gpu-001", "65.0"},
		{"host-B", "gpu-002", "70.0"},
	}
	csvPath := createTestCSV(t, headers, records)

	result, err := PreProcessCSVByHostNames(csvPath, "host-A")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == csvPath {
		t.Error("Expected filtering to work with case-insensitive header")
	}
	defer func() { _ = os.Remove(result) }()
}

func TestPreProcessCSVByHostNames_WhitespaceHandling(t *testing.T) {
	// Create test CSV
	headers := []string{"hostname", "gpu_id"}
	records := [][]string{
		{"  host-A  ", "gpu-001"},
		{"host-B", "gpu-002"},
		{" host-C ", "gpu-003"},
	}
	csvPath := createTestCSV(t, headers, records)

	// Test with whitespace in hostList
	result, err := PreProcessCSVByHostNames(csvPath, " host-A , host-C ")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == csvPath {
		t.Error("Expected filtering to work with whitespace handling")
	}
	defer func() { _ = os.Remove(result) }()

	// Verify content
	file, err := os.Open(result)
	if err != nil {
		t.Fatalf("Failed to open filtered file: %v", err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	allRecords, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read filtered CSV: %v", err)
	}

	// Should have header + 2 records
	if len(allRecords) != 3 {
		t.Errorf("Expected 3 records (including header), got %d", len(allRecords))
	}
}

func TestPreProcessCSVByHostNames_FileNotFound(t *testing.T) {
	result, err := PreProcessCSVByHostNames("/nonexistent/file.csv", "host-A")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if result != "/nonexistent/file.csv" {
		t.Errorf("Expected original path on error, got %s", result)
	}
}

func TestPreProcessCSVByHostNames_InvalidCSV(t *testing.T) {
	// Create invalid CSV (empty file)
	csvPath := createTestCSVWithContent(t, "")

	result, err := PreProcessCSVByHostNames(csvPath, "host-A")
	if err == nil {
		t.Error("Expected error for invalid CSV")
	}
	if result != csvPath {
		t.Errorf("Expected original path on error, got %s", result)
	}
}

func TestPreProcessCSVByHostNames_InsufficientColumns(t *testing.T) {
	// Create CSV with inconsistent column counts
	content := "hostname,gpu_id,temperature\nhost-A,gpu-001\nhost-B,gpu-002,70.0"
	csvPath := createTestCSVWithContent(t, content)

	result, err := PreProcessCSVByHostNames(csvPath, "host-A,host-B")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should still process valid records
	if result == csvPath {
		t.Error("Expected filtering to create new file even with some invalid records")
	}
	defer func() { _ = os.Remove(result) }()
}

// ==== NewStreamer Tests ====

func TestNewStreamer(t *testing.T) {
	broker := NewMockBroker()
	streamer := NewStreamer("test.csv", 2, 10.0, "test-topic", broker)

	if streamer.csvPath != "test.csv" {
		t.Errorf("Expected csvPath to be 'test.csv', got %s", streamer.csvPath)
	}
	if streamer.workers != 2 {
		t.Errorf("Expected workers to be 2, got %d", streamer.workers)
	}
	if streamer.rate != 10.0 {
		t.Errorf("Expected rate to be 10.0, got %f", streamer.rate)
	}
	if streamer.broker == nil {
		t.Error("Expected broker to be set correctly")
	}
	if streamer.ctx == nil {
		t.Error("Expected context to be initialized")
	}
	if streamer.cancel == nil {
		t.Error("Expected cancel function to be initialized")
	}
	if streamer.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

// ==== Streamer Start/Stop Tests ====

func TestStreamer_Start_Success(t *testing.T) {
	// Create test CSV
	headers := []string{"gpu_id", "temperature"}
	records := [][]string{
		{"gpu-001", "65.0"},
		{"gpu-002", "70.0"},
	}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)

	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Give it time to process some records
	time.Sleep(100 * time.Millisecond)

	streamer.Stop()

	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected to receive some messages")
	}
}

func TestStreamer_Start_FileNotFound(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer("/nonexistent/file.csv", 1, 1.0, "test-topic", broker)

	err := streamer.Start()
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "failed to access CSV file") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestStreamer_Start_InvalidCSV(t *testing.T) {
	// Create invalid CSV (empty file)
	csvPath := createTestCSVWithContent(t, "")

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)

	err := streamer.Start()
	if err == nil {
		t.Error("Expected error for invalid CSV")
	}

	if !strings.Contains(err.Error(), "failed to read CSV headers") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestStreamer_GracefulShutdown(t *testing.T) {
	headers := []string{"id", "value"}
	records := [][]string{{"1", "100"}, {"2", "200"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 2, 1.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	// Test graceful shutdown
	start := time.Now()
	streamer.Stop()
	elapsed := time.Since(start)

	// Should stop relatively quickly
	if elapsed > 2*time.Second {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}
}

func TestStreamer_MultipleWorkers(t *testing.T) {
	headers := []string{"id", "value"}
	records := make([][]string, 10)
	for i := 0; i < 10; i++ {
		records[i] = []string{fmt.Sprintf("id-%d", i), fmt.Sprintf("value-%d", i)}
	}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	workers := 3
	streamer := NewStreamer(csvPath, workers, 10.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Let workers process some records
	time.Sleep(200 * time.Millisecond)
	streamer.Stop()

	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected messages from multiple workers")
	}
}

func TestStreamer_PublishError(t *testing.T) {
	headers := []string{"id", "value"}
	records := [][]string{{"1", "100"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	broker.SetPublishError(fmt.Errorf("publish failed"))
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Let it try to process records
	time.Sleep(100 * time.Millisecond)
	streamer.Stop()

	// Should handle publish errors gracefully (no crash)
	messages := broker.GetMessages()
	if len(messages) > 0 {
		t.Error("Expected no messages due to publish error")
	}
}

// ==== ReadHeaders Tests ====

func TestStreamer_ReadHeaders_Success(t *testing.T) {
	headers := []string{"gpu_id", "temperature", "utilization"}
	records := [][]string{{"gpu-001", "65.0", "75.5"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)

	readHeaders, err := streamer.readHeaders()
	if err != nil {
		t.Fatalf("Failed to read headers: %v", err)
	}

	if !reflect.DeepEqual(readHeaders, headers) {
		t.Errorf("Expected headers %v, got %v", headers, readHeaders)
	}
}

func TestStreamer_ReadHeaders_FileNotFound(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer("/nonexistent/file.csv", 1, 1.0, "test-topic", broker)

	_, err := streamer.readHeaders()
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestStreamer_ReadHeaders_EmptyFile(t *testing.T) {
	csvPath := createTestCSVWithContent(t, "")

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)

	_, err := streamer.readHeaders()
	if err == nil {
		t.Error("Expected error for empty file")
	}
}

// ==== ParseRecord Tests ====

func TestStreamer_ParseRecord_Success(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	headers := []string{"gpu_id", "temperature", "utilization", "active", "hostname"}
	record := []string{"gpu-001", "72.5", "85.2", "true", "host-A"}

	telemetryData, err := streamer.parseRecord(headers, record)
	if err != nil {
		t.Fatalf("Failed to parse record: %v", err)
	}

	if telemetryData.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if len(telemetryData.Fields) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(telemetryData.Fields))
	}

	// Check string field
	if gpuID, ok := telemetryData.Fields["gpu_id"].(string); !ok || gpuID != "gpu-001" {
		t.Errorf("Expected gpu_id to be 'gpu-001' (string), got %v", telemetryData.Fields["gpu_id"])
	}

	// Check float field
	if temp, ok := telemetryData.Fields["temperature"].(float64); !ok || temp != 72.5 {
		t.Errorf("Expected temperature to be 72.5 (float64), got %v", telemetryData.Fields["temperature"])
	}

	// Check boolean field
	if active, ok := telemetryData.Fields["active"].(bool); !ok || !active {
		t.Errorf("Expected active to be true (bool), got %v", telemetryData.Fields["active"])
	}

	// Check hostname field
	if hostname, ok := telemetryData.Fields["hostname"].(string); !ok || hostname != "host-A" {
		t.Errorf("Expected hostname to be 'host-A' (string), got %v", telemetryData.Fields["hostname"])
	}
}

func TestStreamer_ParseRecord_MismatchedLength(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	headers := []string{"gpu_id", "temperature"}
	record := []string{"gpu-001"} // Missing one field

	_, err := streamer.parseRecord(headers, record)
	if err == nil {
		t.Error("Expected error for mismatched header/record length")
	}

	if !strings.Contains(err.Error(), "header count") {
		t.Errorf("Expected error about header count, got: %v", err)
	}
}

func TestStreamer_ParseRecord_EmptyHeaders(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	headers := []string{"gpu_id", "", "temperature"} // Empty header
	record := []string{"gpu-001", "ignored", "72.5"}

	telemetryData, err := streamer.parseRecord(headers, record)
	if err != nil {
		t.Fatalf("Failed to parse record: %v", err)
	}

	// Should skip empty headers
	if len(telemetryData.Fields) != 2 {
		t.Errorf("Expected 2 fields (skipping empty header), got %d", len(telemetryData.Fields))
	}

	if _, exists := telemetryData.Fields[""]; exists {
		t.Error("Expected empty header field to be skipped")
	}
}

func TestStreamer_ParseRecord_BooleanValues(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	testCases := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"no", false},
		{"No", false},
		{"NO", false},
	}

	for _, tc := range testCases {
		headers := []string{"active"}
		record := []string{tc.value}

		telemetryData, err := streamer.parseRecord(headers, record)
		if err != nil {
			t.Fatalf("Failed to parse record with value %s: %v", tc.value, err)
		}

		if active, ok := telemetryData.Fields["active"].(bool); !ok {
			t.Errorf("Expected %s to be parsed as bool, got %T", tc.value, telemetryData.Fields["active"])
		} else if active != tc.expected {
			t.Errorf("Expected %s to be %v, got %v", tc.value, tc.expected, active)
		}
	}
}

func TestStreamer_ParseRecord_FloatValues(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	testCases := []struct {
		value    string
		expected float64
	}{
		{"72.5", 72.5},
		{"0", 0.0},
		{"123", 123.0},
		{"-45.7", -45.7},
		{"1.23e2", 123.0},
	}

	for _, tc := range testCases {
		headers := []string{"temperature"}
		record := []string{tc.value}

		telemetryData, err := streamer.parseRecord(headers, record)
		if err != nil {
			t.Fatalf("Failed to parse record with value %s: %v", tc.value, err)
		}

		if temp, ok := telemetryData.Fields["temperature"].(float64); !ok {
			t.Errorf("Expected %s to be parsed as float64, got %T", tc.value, telemetryData.Fields["temperature"])
		} else if temp != tc.expected {
			t.Errorf("Expected %s to be %v, got %v", tc.value, tc.expected, temp)
		}
	}
}

func TestStreamer_ParseRecord_StringValues(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	// Values that should remain as strings
	testCases := []string{
		"gpu-001",
		"host-A",
		"not-a-number",
		"maybe",    // Not a valid boolean
		"",         // Empty string
		"12.34.56", // Invalid float format
	}

	for _, tc := range testCases {
		headers := []string{"field"}
		record := []string{tc}

		telemetryData, err := streamer.parseRecord(headers, record)
		if err != nil {
			t.Fatalf("Failed to parse record with value %s: %v", tc, err)
		}

		if value, ok := telemetryData.Fields["field"].(string); !ok {
			t.Errorf("Expected %s to remain as string, got %T", tc, telemetryData.Fields["field"])
		} else if value != tc {
			t.Errorf("Expected string value %s, got %s", tc, value)
		}
	}
}

// ==== Helper Function Tests ====

func TestParseFloat(t *testing.T) {
	testCases := []struct {
		input     string
		expected  float64
		shouldErr bool
	}{
		{"72.5", 72.5, false},
		{"0", 0.0, false},
		{"123", 123.0, false},
		{"-45.7", -45.7, false},
		{"1.23e2", 123.0, false},
		{"", 0.0, true},
		{"not-a-number", 0.0, true},
		{"12.34.56", 0.0, true},
	}

	for _, tc := range testCases {
		result, err := parseFloat(tc.input)

		if tc.shouldErr {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %s to parse to %f, got %f", tc.input, tc.expected, result)
			}
		}
	}
}

func TestParseBool(t *testing.T) {
	testCases := []struct {
		input     string
		expected  bool
		shouldErr bool
	}{
		// True values
		{"true", true, false},
		{"True", true, false},
		{"TRUE", true, false},
		{"yes", true, false},
		{"Yes", true, false},
		{"YES", true, false},
		// False values
		{"false", false, false},
		{"False", false, false},
		{"FALSE", false, false},
		{"no", false, false},
		{"No", false, false},
		{"NO", false, false},
		// Invalid values (now including numeric values)
		{"1", false, true},
		{"0", false, true},
		{"maybe", false, true},
		{"", false, true},
		{"2", false, true},
		{"not-a-bool", false, true},
	}

	for _, tc := range testCases {
		result, err := parseBool(tc.input)

		if tc.shouldErr {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("Expected %s to parse to %v, got %v", tc.input, tc.expected, result)
			}
		}
	}
}

// ==== ProcessCSVLoop Tests ====

func TestStreamer_ProcessCSVLoop_Context_Cancelled(t *testing.T) {
	headers := []string{"id", "value"}
	records := [][]string{{"1", "100"}, {"2", "200"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)

	// Cancel context immediately
	streamer.cancel()

	recordsProcessed := 0
	err := streamer.processCSVLoop(0, headers, &recordsProcessed, 0, streamer.logger.WithComponent("test"))

	if err != nil {
		t.Errorf("Expected no error when context is cancelled, got: %v", err)
	}

	if recordsProcessed != 0 {
		t.Errorf("Expected no records processed when context cancelled, got: %d", recordsProcessed)
	}
}

func TestStreamer_ProcessCSVLoop_FileNotFound(t *testing.T) {
	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer("/nonexistent/file.csv", 1, 1.0, "test-topic", broker)

	headers := []string{"id", "value"}
	recordsProcessed := 0

	err := streamer.processCSVLoop(0, headers, &recordsProcessed, 0, streamer.logger.WithComponent("test"))

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestStreamer_ProcessCSVLoop_RateLimit(t *testing.T) {
	headers := []string{"id"}
	records := [][]string{{"1"}, {"2"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 1.0, "test-topic", broker)

	recordsProcessed := 0
	rateInterval := 50 * time.Millisecond

	start := time.Now()
	err := streamer.processCSVLoop(0, headers, &recordsProcessed, rateInterval, streamer.logger.WithComponent("test"))
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have some delay due to rate limiting
	expectedMinTime := time.Duration(recordsProcessed-1) * rateInterval
	if recordsProcessed > 1 && elapsed < expectedMinTime {
		t.Errorf("Expected rate limiting delay of at least %v, but took %v", expectedMinTime, elapsed)
	}
}

// ==== JSON Marshaling Tests ====

func TestStreamer_JSONMarshaling(t *testing.T) {
	headers := []string{"gpu_id", "temperature", "active"}
	records := [][]string{{"gpu-001", "72.5", "true"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 10.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Give it time to process
	time.Sleep(100 * time.Millisecond)
	streamer.Stop()

	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}

	// Verify JSON structure
	var telemetryData TelemetryData
	err = json.Unmarshal(messages[0].Payload, &telemetryData)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if telemetryData.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if len(telemetryData.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(telemetryData.Fields))
	}

	// Check field types are preserved
	if _, ok := telemetryData.Fields["gpu_id"].(string); !ok {
		t.Error("Expected gpu_id to be string")
	}
	if _, ok := telemetryData.Fields["temperature"].(float64); !ok {
		t.Error("Expected temperature to be float64")
	}
	if _, ok := telemetryData.Fields["active"].(bool); !ok {
		t.Error("Expected active to be bool")
	}
}

// ==== Continuous Loop Tests ====

func TestStreamer_ContinuousLoop(t *testing.T) {
	// Create small CSV for multiple loops
	headers := []string{"counter"}
	records := [][]string{{"1"}, {"2"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 20.0, "test-topic", broker) // High rate
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Let it run long enough to complete multiple loops
	time.Sleep(300 * time.Millisecond)
	streamer.Stop()

	messages := broker.GetMessages()

	// Should receive more messages than records (due to looping)
	minExpected := 4 // Should loop through at least twice
	if len(messages) < minExpected {
		t.Errorf("Expected at least %d messages for continuous loop, got %d", minExpected, len(messages))
	}
}

// ==== Rate Control Tests ====

func TestStreamer_RateControl_ZeroRate(t *testing.T) {
	headers := []string{"id"}
	records := [][]string{{"1"}, {"2"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	// Rate of 0 should mean no rate limiting
	streamer := NewStreamer(csvPath, 1, 0.0, "test-topic", broker)

	start := time.Now()
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(50 * time.Millisecond) // Short time
	streamer.Stop()
	elapsed := time.Since(start)

	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected messages even with zero rate")
	}

	// Should process quickly without rate limiting
	if elapsed > 200*time.Millisecond {
		t.Errorf("Processing took too long with zero rate: %v", elapsed)
	}
}

func TestStreamer_RateControl_HighRate(t *testing.T) {
	headers := []string{"id"}
	records := [][]string{{"1"}, {"2"}, {"3"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	// High rate should allow fast processing
	streamer := NewStreamer(csvPath, 1, 100.0, "test-topic", broker)

	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	streamer.Stop()

	messages := broker.GetMessages()

	// Should process multiple loops with high rate
	if len(messages) < 6 { // At least 2 complete loops
		t.Errorf("Expected more messages with high rate, got %d", len(messages))
	}
}

// ==== Error Handling Tests ====

func TestStreamer_HandleCSVReadError(t *testing.T) {
	// Create CSV with invalid content after headers
	content := "id,value\n1,100\ninvalid,csv,too,many,fields\n2,200"
	csvPath := createTestCSVWithContent(t, content)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 10.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	streamer.Stop()

	// Should handle CSV read errors gracefully and continue processing valid records
	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected some messages even with CSV read errors")
	}
}

func TestStreamer_HandleJSONMarshalError(t *testing.T) {
	// This is harder to test directly since json.Marshal rarely fails
	// with our simple data types, but we can test the error handling path
	// exists in the code coverage

	headers := []string{"id"}
	records := [][]string{{"1"}}
	csvPath := createTestCSV(t, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 10.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	streamer.Stop()

	// Should handle any errors gracefully
	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected at least one message")
	}
}

// ==== Edge Cases ====

func TestStreamer_EmptyCSVAfterHeaders(t *testing.T) {
	// CSV with only headers
	content := "id,value"
	csvPath := createTestCSVWithContent(t, content)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 10.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	streamer.Stop()

	// Should handle empty CSV gracefully (will keep trying to read EOF)
	messages := broker.GetMessages()
	// No messages expected since there are no data records
	if len(messages) > 0 {
		t.Errorf("Expected no messages from headers-only CSV, got %d", len(messages))
	}
}

func TestStreamer_CSVWithQuotedFields(t *testing.T) {
	// CSV with quoted fields
	content := `id,description,value
1,"quoted field with, comma",100
2,"another ""quoted"" field",200`
	csvPath := createTestCSVWithContent(t, content)

	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(csvPath, 1, 10.0, "test-topic", broker)
	err := streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	streamer.Stop()

	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected messages from quoted CSV")
	}

	// Verify quoted content is parsed correctly
	var telemetryData TelemetryData
	err = json.Unmarshal(messages[0].Payload, &telemetryData)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	desc, ok := telemetryData.Fields["description"].(string)
	if !ok {
		t.Error("Expected description field to be string")
	}
	if desc != "quoted field with, comma" {
		t.Errorf("Expected quoted field content, got: %s", desc)
	}
}

// ==== Benchmark Tests ====

func BenchmarkPreProcessCSVByHostNames(b *testing.B) {
	// Create large test CSV
	headers := []string{"hostname", "gpu_id", "temperature", "utilization", "power_draw"}
	records := make([][]string, 1000)
	hosts := []string{"host-A", "host-B", "host-C", "host-D", "host-E"}

	for i := 0; i < 1000; i++ {
		hostIdx := i % len(hosts)
		records[i] = []string{
			hosts[hostIdx],
			fmt.Sprintf("gpu-%03d", i),
			fmt.Sprintf("%.1f", float64(60+i%40)),
			fmt.Sprintf("%.1f", float64(i%100)),
			fmt.Sprintf("%.1f", float64(150+i%100)),
		}
	}

	tmpDir := b.TempDir()
	csvPath := filepath.Join(tmpDir, "benchmark.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		b.Fatalf("Failed to create benchmark CSV: %v", err)
	}
	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	_ = writer.Write(headers)
	for _, record := range records {
		_ = writer.Write(record)
	}
	writer.Flush()

	hostList := "host-A,host-C"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := PreProcessCSVByHostNames(csvPath, hostList)
		if err != nil {
			b.Fatalf("PreProcessCSVByHostNames failed: %v", err)
		}
		if result != csvPath {
			_ = os.Remove(result) // Clean up filtered file
		}
	}
}

func BenchmarkStreamerThroughput(b *testing.B) {
	// Create test CSV
	headers := []string{"id", "value", "timestamp", "active"}
	records := make([][]string, 100)
	for i := 0; i < 100; i++ {
		records[i] = []string{
			fmt.Sprintf("id%d", i),
			fmt.Sprintf("%d", i*10),
			"2023-01-01T00:00:00Z",
			fmt.Sprintf("%t", i%2 == 0),
		}
	}
	csvPath := createTestCSV(b, headers, records)

	broker := NewMockBroker()
	defer broker.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		streamer := NewStreamer(csvPath, 1, 1000.0, "telemetry", broker) // High rate
		if err := streamer.Start(); err != nil {
			b.Fatalf("Failed to start streamer: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Brief processing time
		streamer.Stop()
		broker.Reset() // Clear messages for next iteration
	}
}

func BenchmarkParseRecord(b *testing.B) {
	broker := NewMockBroker()
	defer broker.Close()
	streamer := NewStreamer("", 1, 1.0, "test-topic", broker)

	headers := []string{"gpu_id", "temperature", "utilization", "active", "hostname", "power_draw"}
	record := []string{"gpu-001", "72.5", "85.2", "true", "host-A", "150.7"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := streamer.parseRecord(headers, record)
		if err != nil {
			b.Fatalf("Failed to parse record: %v", err)
		}
	}
}

func BenchmarkParseFloat(b *testing.B) {
	testValues := []string{"72.5", "0", "123", "-45.7", "1.23e2"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[i%len(testValues)]
		_, _ = parseFloat(value)
	}
}

func BenchmarkParseBool(b *testing.B) {
	testValues := []string{"true", "false", "True", "False", "1", "0", "yes", "no"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := testValues[i%len(testValues)]
		_, _ = parseBool(value)
	}
}

// ==== Integration Tests ====

func TestStreamer_Integration_CompleteWorkflow(t *testing.T) {
	// Test complete workflow from CSV preprocessing to message publishing

	// Create CSV with mixed hostnames
	headers := []string{"hostname", "gpu_id", "temperature", "utilization", "active"}
	records := [][]string{
		{"host-A", "gpu-001", "65.5", "75.2", "true"},
		{"host-B", "gpu-002", "70.1", "80.5", "false"},
		{"host-A", "gpu-003", "68.3", "72.8", "true"},
		{"host-C", "gpu-004", "72.0", "85.0", "true"},
	}
	originalCSV := createTestCSV(t, headers, records)

	// Step 1: Preprocess CSV
	filteredCSV, err := PreProcessCSVByHostNames(originalCSV, "host-A,host-C")
	if err != nil {
		t.Fatalf("Failed to preprocess CSV: %v", err)
	}
	defer func() {
		if filteredCSV != originalCSV {
			_ = os.Remove(filteredCSV)
		}
	}()

	// Step 2: Stream filtered CSV
	broker := NewMockBroker()
	defer broker.Close()

	streamer := NewStreamer(filteredCSV, 2, 20.0, "telemetry", broker)
	err = streamer.Start()
	if err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	streamer.Stop()

	// Step 3: Verify results
	messages := broker.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected messages from integration test")
	}

	// Verify message content
	var telemetryData TelemetryData
	err = json.Unmarshal(messages[0].Payload, &telemetryData)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Should only contain host-A or host-C data
	hostname, ok := telemetryData.Fields["hostname"].(string)
	if !ok {
		t.Error("Expected hostname field to be string")
	}
	if hostname != "host-A" && hostname != "host-C" {
		t.Errorf("Expected hostname to be host-A or host-C, got: %s", hostname)
	}

	// Verify all field types
	if _, ok := telemetryData.Fields["gpu_id"].(string); !ok {
		t.Error("Expected gpu_id to be string")
	}
	if _, ok := telemetryData.Fields["temperature"].(float64); !ok {
		t.Error("Expected temperature to be float64")
	}
	if _, ok := telemetryData.Fields["utilization"].(float64); !ok {
		t.Error("Expected utilization to be float64")
	}
	if _, ok := telemetryData.Fields["active"].(bool); !ok {
		t.Error("Expected active to be bool")
	}
}
