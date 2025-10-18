package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/harishb93/telemetry-pipeline/internal/collector"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// MockCollector implements a mock collector for testing
type MockCollector struct {
	memoryStats   map[string]interface{}
	telemetryData map[string][]*collector.Telemetry
}

func NewMockCollector() *MockCollector {
	return &MockCollector{
		memoryStats: map[string]interface{}{
			"total_entries": 6,
			"total_gpus":    2,
			"gpu_entry_counts": map[string]int{
				"gpu_0": 3,
				"gpu_1": 3,
			},
		},
		telemetryData: map[string][]*collector.Telemetry{
			"gpu_0": {
				{
					GPUId: "gpu_0",
					Metrics: map[string]float64{
						"temperature": 72.3,
						"utilization": 85.5,
						"memory_used": 4096.0,
					},
					Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					GPUId: "gpu_0",
					Metrics: map[string]float64{
						"temperature": 74.1,
						"utilization": 87.2,
						"memory_used": 4200.0,
					},
					Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
				},
				{
					GPUId: "gpu_0",
					Metrics: map[string]float64{
						"temperature": 75.8,
						"utilization": 89.1,
						"memory_used": 4300.0,
					},
					Timestamp: time.Date(2024, 1, 1, 12, 2, 0, 0, time.UTC),
				},
			},
			"gpu_1": {
				{
					GPUId: "gpu_1",
					Metrics: map[string]float64{
						"temperature": 68.9,
						"utilization": 92.3,
						"memory_used": 8192.0,
					},
					Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					GPUId: "gpu_1",
					Metrics: map[string]float64{
						"temperature": 70.2,
						"utilization": 94.1,
						"memory_used": 8300.0,
					},
					Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
				},
				{
					GPUId: "gpu_1",
					Metrics: map[string]float64{
						"temperature": 71.5,
						"utilization": 95.8,
						"memory_used": 8400.0,
					},
					Timestamp: time.Date(2024, 1, 1, 12, 2, 0, 0, time.UTC),
				},
			},
		},
	}
}

func (m *MockCollector) GetMemoryStats() map[string]interface{} {
	return m.memoryStats
}

func (m *MockCollector) GetTelemetryForGPU(gpuID string, limit int) []*collector.Telemetry {
	data, exists := m.telemetryData[gpuID]
	if !exists {
		return []*collector.Telemetry{}
	}

	if limit == 0 || limit >= len(data) {
		return data
	}

	return data[:limit]
}

// Helper function to create a test collector
func createTestCollector() *collector.Collector {
	brokerConfig := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(brokerConfig)

	config := collector.CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "8899", // Use a fixed test port
	}

	return collector.NewCollector(broker, config)
}

// createTestHandlers creates handlers with a running collector service for testing
func createTestHandlers(t *testing.T) (*Handlers, func()) {
	coll := createTestCollector()

	// Start the collector in background
	go func() {
		if err := coll.Start(); err != nil {
			t.Logf("Failed to start test collector: %v", err)
		}
	}()

	// Give it a moment to start
	time.Sleep(200 * time.Millisecond)

	// Set the collector URL to point to our test collector
	if err := os.Setenv("COLLECTOR_URL", "http://localhost:8899"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	handlers := NewHandlers(coll)

	// Return cleanup function
	cleanup := func() {
		coll.Stop()
		if err := os.Unsetenv("COLLECTOR_URL"); err != nil {
			t.Logf("Failed to unset environment variable: %v", err)
		}
	}

	return handlers, cleanup
}

func TestGetGPUs(t *testing.T) {
	// Create a real collector with some test data
	handlers, cleanup := createTestHandlers(t)
	defer cleanup()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, response GPUResponse)
	}{
		{
			name:           "Get GPUs without pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response GPUResponse) {
				if response.Total < 0 {
					t.Errorf("Expected non-negative total, got %d", response.Total)
				}
				if response.Pagination.Limit != 100 {
					t.Errorf("Expected default limit 100, got %d", response.Pagination.Limit)
				}
				if response.Pagination.Offset != 0 {
					t.Errorf("Expected default offset 0, got %d", response.Pagination.Offset)
				}
			},
		},
		{
			name:           "Get GPUs with pagination",
			queryParams:    "?limit=5&offset=0",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response GPUResponse) {
				if response.Pagination.Limit != 5 {
					t.Errorf("Expected limit 5, got %d", response.Pagination.Limit)
				}
				if response.Pagination.Offset != 0 {
					t.Errorf("Expected offset 0, got %d", response.Pagination.Offset)
				}
			},
		},
		{
			name:           "Get GPUs with invalid limit",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/gpus"+tt.queryParams, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlers.GetGPUs)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var response GPUResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Could not parse response: %v", err)
				}
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestGetTelemetry(t *testing.T) {
	handlers, cleanup := createTestHandlers(t)
	defer cleanup()

	tests := []struct {
		name           string
		gpuID          string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, response TelemetryResponse)
	}{
		{
			name:           "Get telemetry for existing GPU",
			gpuID:          "gpu_0",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response TelemetryResponse) {
				if response.Total < 0 {
					t.Errorf("Expected non-negative total, got %d", response.Total)
				}
				if response.Pagination.Limit != 100 {
					t.Errorf("Expected default limit 100, got %d", response.Pagination.Limit)
				}
			},
		},
		{
			name:           "Get telemetry with pagination",
			gpuID:          "gpu_0",
			queryParams:    "?limit=2&offset=0",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response TelemetryResponse) {
				if response.Pagination.Limit != 2 {
					t.Errorf("Expected limit 2, got %d", response.Pagination.Limit)
				}
				if response.Pagination.Offset != 0 {
					t.Errorf("Expected offset 0, got %d", response.Pagination.Offset)
				}
			},
		},
		{
			name:           "Get telemetry with time range",
			gpuID:          "gpu_0",
			queryParams:    "?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response TelemetryResponse) {
				// Just verify structure, actual filtering depends on data
				if response.Pagination.Limit != 100 {
					t.Errorf("Expected default limit 100, got %d", response.Pagination.Limit)
				}
			},
		},
		{
			name:           "Get telemetry with invalid time format",
			gpuID:          "gpu_0",
			queryParams:    "?start_time=invalid-time",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "Get telemetry with invalid limit",
			gpuID:          "gpu_0",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/gpus/"+tt.gpuID+"/telemetry"+tt.queryParams, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			// Create router to handle path variables
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/gpus/{id}/telemetry", handlers.GetTelemetry).Methods("GET")
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var response TelemetryResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Could not parse response: %v", err)
				}
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestHealth(t *testing.T) {
	handlers, cleanup := createTestHandlers(t)
	defer cleanup()

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.Health)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Could not parse response: %v", err)
	}

	// Check required fields
	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["service"] != "telemetry-api-gateway" {
		t.Errorf("Expected service 'telemetry-api-gateway', got %v", response["service"])
	}

	if response["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", response["version"])
	}

	// Check timestamp is present and valid
	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected timestamp field in response")
	}

	// Check collector status is present
	if _, ok := response["collector"]; !ok {
		t.Error("Expected collector field in response")
	}
}

func TestParsePagination(t *testing.T) {
	handlers := &Handlers{}

	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedLimit  int
		expectedOffset int
		expectError    bool
	}{
		{
			name:           "Default values",
			queryParams:    map[string]string{},
			expectedLimit:  100,
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "Valid pagination",
			queryParams:    map[string]string{"limit": "50", "offset": "10"},
			expectedLimit:  50,
			expectedOffset: 10,
			expectError:    false,
		},
		{
			name:           "Invalid limit",
			queryParams:    map[string]string{"limit": "invalid"},
			expectedLimit:  0,
			expectedOffset: 0,
			expectError:    true,
		},
		{
			name:           "Invalid offset",
			queryParams:    map[string]string{"offset": "invalid"},
			expectedLimit:  0,
			expectedOffset: 0,
			expectError:    true,
		},
		{
			name:           "Limit too high",
			queryParams:    map[string]string{"limit": "2000"},
			expectedLimit:  100, // Should default to 100
			expectedOffset: 0,
			expectError:    false,
		},
		{
			name:           "Negative values",
			queryParams:    map[string]string{"limit": "-1", "offset": "-5"},
			expectedLimit:  100, // Should default
			expectedOffset: 0,   // Should default
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with query parameters
			req, err := http.NewRequest("GET", "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			limit, offset, err := handlers.parsePagination(req)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if limit != tt.expectedLimit {
					t.Errorf("Expected limit %d, got %d", tt.expectedLimit, limit)
				}

				if offset != tt.expectedOffset {
					t.Errorf("Expected offset %d, got %d", tt.expectedOffset, offset)
				}
			}
		})
	}
}

func TestParseTimeRange(t *testing.T) {
	handlers := &Handlers{}

	tests := []struct {
		name        string
		queryParams map[string]string
		expectError bool
	}{
		{
			name:        "No time parameters",
			queryParams: map[string]string{},
			expectError: false,
		},
		{
			name: "Valid time range",
			queryParams: map[string]string{
				"start_time": "2024-01-01T00:00:00Z",
				"end_time":   "2024-01-01T23:59:59Z",
			},
			expectError: false,
		},
		{
			name:        "Invalid start time",
			queryParams: map[string]string{"start_time": "invalid-time"},
			expectError: true,
		},
		{
			name:        "Invalid end time",
			queryParams: map[string]string{"end_time": "invalid-time"},
			expectError: true,
		},
		{
			name:        "Only start time",
			queryParams: map[string]string{"start_time": "2024-01-01T00:00:00Z"},
			expectError: false,
		},
		{
			name:        "Only end time",
			queryParams: map[string]string{"end_time": "2024-01-01T23:59:59Z"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with query parameters
			req, err := http.NewRequest("GET", "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			startTime, endTime, err := handlers.parseTimeRange(req)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// If start_time was provided, verify it's parsed
				if _, exists := tt.queryParams["start_time"]; exists && startTime == nil {
					t.Error("Expected start time to be parsed")
				}

				// If end_time was provided, verify it's parsed
				if _, exists := tt.queryParams["end_time"]; exists && endTime == nil {
					t.Error("Expected end time to be parsed")
				}
			}
		})
	}
}

// Benchmark tests
func TestGetHosts(t *testing.T) {
	handlers, cleanup := createTestHandlers(t)
	defer cleanup()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, response HostsResponse)
	}{
		{
			name:           "Get all hosts",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response HostsResponse) {
				if response.Total < 0 {
					t.Errorf("Expected non-negative total, got %d", response.Total)
				}
				if response.Pagination.Limit != 100 {
					t.Errorf("Expected default limit 100, got %d", response.Pagination.Limit)
				}
				// Check that Hosts is a slice of strings
				if response.Hosts == nil {
					t.Error("Expected hosts slice, got nil")
				}
			},
		},
		{
			name:           "Get hosts with pagination",
			queryParams:    "?limit=10&offset=0",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response HostsResponse) {
				if response.Pagination.Limit != 10 {
					t.Errorf("Expected limit 10, got %d", response.Pagination.Limit)
				}
				if response.Pagination.Offset != 0 {
					t.Errorf("Expected offset 0, got %d", response.Pagination.Offset)
				}
			},
		},
		{
			name:           "Get hosts with invalid limit",
			queryParams:    "?limit=invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/hosts"+tt.queryParams, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlers.GetHosts)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var response HostsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Could not parse response: %v", err)
				}
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestGetHostGPUs(t *testing.T) {
	handlers, cleanup := createTestHandlers(t)
	defer cleanup()

	tests := []struct {
		name           string
		hostname       string
		expectedStatus int
		checkResponse  func(t *testing.T, response HostGPUsResponse)
	}{
		{
			name:           "Get GPUs for existing host",
			hostname:       "test-host",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response HostGPUsResponse) {
				if response.Hostname != "test-host" {
					t.Errorf("Expected hostname 'test-host', got '%s'", response.Hostname)
				}
				if response.Total < 0 {
					t.Errorf("Expected non-negative total, got %d", response.Total)
				}
				if response.GPUs == nil {
					t.Error("Expected GPUs slice, got nil")
				}
			},
		},
		{
			name:           "Get GPUs for non-existent host",
			hostname:       "non-existent-host",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response HostGPUsResponse) {
				if response.Hostname != "non-existent-host" {
					t.Errorf("Expected hostname 'non-existent-host', got '%s'", response.Hostname)
				}
				if response.Total != 0 {
					t.Errorf("Expected total 0 for non-existent host, got %d", response.Total)
				}
				if len(response.GPUs) != 0 {
					t.Errorf("Expected empty GPUs slice for non-existent host, got %d GPUs", len(response.GPUs))
				}
			},
		},
		{
			name:           "Get GPUs with empty hostname gets redirected",
			hostname:       "",
			expectedStatus: http.StatusMovedPermanently,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/hosts/"+tt.hostname+"/gpus", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			// Use mux router to handle path parameters
			router := mux.NewRouter()
			router.HandleFunc("/api/v1/hosts/{hostname}/gpus", handlers.GetHostGPUs).Methods("GET")
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var response HostGPUsResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Could not parse response: %v", err)
				}
				tt.checkResponse(t, response)
			}
		})
	}
}

func BenchmarkGetGPUs(b *testing.B) {
	coll := createTestCollector()
	handlers := NewHandlers(coll)

	req, _ := http.NewRequest("GET", "/api/v1/gpus", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetGPUs(rr, req)
	}
}

func BenchmarkGetTelemetry(b *testing.B) {
	coll := createTestCollector()
	handlers := NewHandlers(coll)

	req, _ := http.NewRequest("GET", "/api/v1/gpus/gpu_0/telemetry", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/api/v1/gpus/{id}/telemetry", handlers.GetTelemetry).Methods("GET")
		router.ServeHTTP(rr, req)
	}
}
