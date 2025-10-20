package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/harishb93/telemetry-pipeline/internal/collector"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// Test middleware functionality
func TestCORSMiddleware(t *testing.T) {
	config := ServerConfig{Port: "8091"}
	coll := createTestCollector()
	server := NewServer(coll, config)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkHeaders   func(t *testing.T, headers http.Header)
	}{
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, headers http.Header) {
				if headers.Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Expected CORS origin header to be '*'")
				}
				if !strings.Contains(headers.Get("Access-Control-Allow-Methods"), "GET") {
					t.Error("Expected CORS methods to include GET")
				}
				if !strings.Contains(headers.Get("Access-Control-Allow-Headers"), "Content-Type") {
					t.Error("Expected CORS headers to include Content-Type")
				}
			},
		},
		{
			name:           "GET request",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, headers http.Header) {
				if headers.Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Expected CORS origin header to be '*'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/health", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			// Create router with middleware
			router := mux.NewRouter()
			handlers := NewHandlers(coll)
			router.HandleFunc("/health", handlers.Health).Methods("GET", "OPTIONS")
			router.Use(server.corsMiddleware)

			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			tt.checkHeaders(t, rr.Header())
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	config := ServerConfig{Port: "8092"}
	coll := createTestCollector()
	server := NewServer(coll, config)

	// Capture logs by redirecting to a buffer
	var logBuffer bytes.Buffer
	// Note: In a real implementation, you'd need to redirect log.Printf
	// For this test, we'll just verify the middleware doesn't break functionality

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Create router with middleware
	router := mux.NewRouter()
	handlers := NewHandlers(coll)
	router.HandleFunc("/health", handlers.Health).Methods("GET")
	router.Use(server.loggingMiddleware)

	start := time.Now()
	router.ServeHTTP(rr, req)
	duration := time.Since(start)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify response time is reasonable (allow up to 15 seconds due to collector startup time)
	if duration > 15*time.Second {
		t.Errorf("Request took too long: %v", duration)
	}

	// Verify that the response was captured (status code should be available)
	// The actual logging is tested by ensuring the middleware doesn't break functionality
	_ = logBuffer
}

func TestResponseWriter(t *testing.T) {
	originalWriter := httptest.NewRecorder()
	wrapper := &responseWriter{
		ResponseWriter: originalWriter,
		statusCode:     http.StatusOK,
	}

	// Test default status code
	if wrapper.statusCode != http.StatusOK {
		t.Errorf("Expected default status code 200, got %d", wrapper.statusCode)
	}

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusNotFound)
	if wrapper.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", wrapper.statusCode)
	}

	if originalWriter.Code != http.StatusNotFound {
		t.Errorf("Expected underlying writer status code 404, got %d", originalWriter.Code)
	}
}

func TestServerLifecycle(t *testing.T) {
	config := ServerConfig{Port: "8093"}
	coll := createTestCollector()
	server := NewServer(coll, config)

	// Test server creation
	if server.port != "8093" {
		t.Errorf("Expected port 8093, got %s", server.port)
	}

	if server.collector != coll {
		t.Error("Expected server to have the provided collector")
	}

	// Test Stop before Start (should not panic)
	err := server.Stop()
	if err != nil {
		t.Errorf("Stop should not error when server not started: %v", err)
	}

	// Start server in background
	serverError := make(chan error, 1)
	go func() {
		serverError <- server.Start()
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Test that server is running by making a request
	resp, err := http.Get("http://localhost:8093/health")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected health check to return 200, got %d", resp.StatusCode)
	}

	// Stop server
	err = server.Stop()
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Verify server stopped
	select {
	case err := <-serverError:
		if err != http.ErrServerClosed {
			t.Errorf("Expected server to close gracefully, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}

func TestErrorResponseHandling(t *testing.T) {
	handlers := &Handlers{}

	tests := []struct {
		name       string
		statusCode int
		message    string
		details    string
		checkJSON  func(t *testing.T, response ErrorResponse)
	}{
		{
			name:       "Basic error response",
			statusCode: http.StatusBadRequest,
			message:    "Invalid request",
			details:    "",
			checkJSON: func(t *testing.T, response ErrorResponse) {
				if response.Code != http.StatusBadRequest {
					t.Errorf("Expected code 400, got %d", response.Code)
				}
				if response.Message != "Invalid request" {
					t.Errorf("Expected message 'Invalid request', got %s", response.Message)
				}
				if response.Error != "Bad Request" {
					t.Errorf("Expected error 'Bad Request', got %s", response.Error)
				}
			},
		},
		{
			name:       "Error with details",
			statusCode: http.StatusInternalServerError,
			message:    "Database error",
			details:    "connection timeout",
			checkJSON: func(t *testing.T, response ErrorResponse) {
				if response.Code != http.StatusInternalServerError {
					t.Errorf("Expected code 500, got %d", response.Code)
				}
				if !strings.Contains(response.Message, "Database error") {
					t.Errorf("Expected message to contain 'Database error', got %s", response.Message)
				}
				if !strings.Contains(response.Message, "connection timeout") {
					t.Errorf("Expected message to contain details, got %s", response.Message)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handlers.writeErrorResponse(rr, tt.statusCode, tt.message, tt.details)

			if status := rr.Code; status != tt.statusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.statusCode)
			}

			var response ErrorResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("Could not parse error response: %v", err)
			}

			tt.checkJSON(t, response)
		})
	}
}

func TestJSONResponseHandling(t *testing.T) {
	handlers := &Handlers{}

	tests := []struct {
		name       string
		data       interface{}
		statusCode int
		expectErr  bool
	}{
		{
			name: "Valid JSON response",
			data: map[string]string{
				"message": "success",
				"status":  "ok",
			},
			statusCode: http.StatusOK,
			expectErr:  false,
		},
		{
			name: "Complex struct response",
			data: GPUResponse{
				GPUs:  []string{"gpu-0", "gpu-1"},
				Total: 2,
				Pagination: PaginationMetadata{
					Limit:   10,
					Offset:  0,
					HasNext: false,
				},
			},
			statusCode: http.StatusAccepted,
			expectErr:  false,
		},
		{
			name: "Function response (should fail)",
			data: func() {}, // Functions can't be marshaled to JSON
			statusCode: http.StatusOK,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handlers.writeJSONResponse(rr, tt.statusCode, tt.data)

			if !tt.expectErr {
				if status := rr.Code; status != tt.statusCode {
					t.Errorf("handler returned wrong status code: got %v want %v", status, tt.statusCode)
				}

				contentType := rr.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}

				// Verify JSON is valid
				var result interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
					t.Fatalf("Response is not valid JSON: %v", err)
				}
			} else {
				// For error cases, the status code is already written, so it stays as the original
				// But the response body should contain the error message
				if status := rr.Code; status != tt.statusCode {
					t.Errorf("Expected status %d for JSON error, got %v", tt.statusCode, status)
				}
				
				// Check that the body contains error message due to JSON encoding failure
				body := rr.Body.String()
				if !strings.Contains(body, "Internal server error") {
					t.Errorf("Expected error message in response body, got: %s", body)
				}
			}
		})
	}
}

func TestConcurrentAPIRequests(t *testing.T) {
	handlers, cleanup := createTestHandlers(t)
	defer cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/gpus", handlers.GetGPUs).Methods("GET")
	router.HandleFunc("/health", handlers.Health).Methods("GET")

	var wg sync.WaitGroup
	numRequests := 10
	errors := make(chan error, numRequests*2)

	// Concurrent GPU requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req, err := http.NewRequest("GET", "/api/v1/gpus", nil)
			if err != nil {
				errors <- fmt.Errorf("request creation failed: %v", err)
				return
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				errors <- fmt.Errorf("request %d failed with status %d", id, rr.Code)
			}
		}(i)
	}

	// Concurrent health requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req, err := http.NewRequest("GET", "/health", nil)
			if err != nil {
				errors <- fmt.Errorf("health request creation failed: %v", err)
				return
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				errors <- fmt.Errorf("health request %d failed with status %d", id, rr.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}

func TestHTTPClientErrorScenarios(t *testing.T) {
	// Test what happens when collector service is unavailable
	brokerConfig := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(brokerConfig)
	defer broker.Close()

	config := collector.CollectorConfig{
		Workers:           1,
		DataDir:           "/tmp/test-unavailable",
		MaxEntriesPerGPU:  100,
		CheckpointEnabled: false,
		HealthPort:        "9999", // Use a port that's not running
	}

	coll := collector.NewCollector(broker, config)
	handlers := NewHandlers(coll)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{
			name:           "Get GPUs with unavailable collector",
			endpoint:       "/api/v1/gpus",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Health check with unavailable collector",
			endpoint:       "/health",
			expectedStatus: http.StatusOK, // Health should still work, just report unhealthy collector
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/api/v1/gpus", handlers.GetGPUs).Methods("GET")
			router.HandleFunc("/health", handlers.Health).Methods("GET")
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.endpoint == "/health" && tt.expectedStatus == http.StatusOK {
				// Verify health response shows unhealthy collector
				var response map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Could not parse health response: %v", err)
				}

				if collector, ok := response["collector"].(map[string]interface{}); ok {
					if collector["status"] != "unhealthy" {
						t.Error("Expected collector status to be unhealthy")
					}
				} else {
					t.Error("Expected collector health info in response")
				}
			}
		})
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	handlers := &Handlers{}

	tests := []struct {
		name        string
		queryParams map[string]string
		expectError bool
		checkValues func(t *testing.T, limit, offset int)
	}{
		{
			name:        "Very large limit",
			queryParams: map[string]string{"limit": "999999"},
			expectError: false,
			checkValues: func(t *testing.T, limit, offset int) {
				if limit != 100 { // Should be capped at default
					t.Errorf("Expected limit to be capped at 100, got %d", limit)
				}
			},
		},
		{
			name:        "Zero limit",
			queryParams: map[string]string{"limit": "0"},
			expectError: false,
			checkValues: func(t *testing.T, limit, offset int) {
				if limit != 100 { // Should default
					t.Errorf("Expected limit to default to 100, got %d", limit)
				}
			},
		},
		{
			name:        "Very large offset",
			queryParams: map[string]string{"offset": "999999"},
			expectError: false,
			checkValues: func(t *testing.T, limit, offset int) {
				if offset != 999999 { // Should accept large offsets
					t.Errorf("Expected offset 999999, got %d", offset)
				}
			},
		},
		{
			name:        "Float values",
			queryParams: map[string]string{"limit": "10.5", "offset": "5.7"},
			expectError: true,
		},
		{
			name:        "Special characters",
			queryParams: map[string]string{"limit": "10@#$", "offset": "abc"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && tt.checkValues != nil {
				tt.checkValues(t, limit, offset)
			}
		})
	}
}

func TestTimeRangeEdgeCases(t *testing.T) {
	handlers := &Handlers{}

	tests := []struct {
		name        string
		queryParams map[string]string
		expectError bool
		checkTimes  func(t *testing.T, start, end *time.Time)
	}{
		{
			name: "End time before start time",
			queryParams: map[string]string{
				"start_time": "2024-01-02T00:00:00Z",
				"end_time":   "2024-01-01T00:00:00Z",
			},
			expectError: false, // Parser doesn't validate logic, just format
			checkTimes: func(t *testing.T, start, end *time.Time) {
				if start == nil || end == nil {
					t.Error("Expected both times to be parsed")
				}
				if start != nil && end != nil && !end.Before(*start) {
					t.Error("Expected end time to be before start time (for this test)")
				}
			},
		},
		{
			name:        "Invalid time format",
			queryParams: map[string]string{"start_time": "2024/01/01 00:00:00"},
			expectError: true,
		},
		{
			name:        "Empty time values",
			queryParams: map[string]string{"start_time": "", "end_time": ""},
			expectError: false,
			checkTimes: func(t *testing.T, start, end *time.Time) {
				if start != nil || end != nil {
					t.Error("Expected nil times for empty values")
				}
			},
		},
		{
			name: "Future timestamps",
			queryParams: map[string]string{
				"start_time": "2030-01-01T00:00:00Z",
				"end_time":   "2030-12-31T23:59:59Z",
			},
			expectError: false,
			checkTimes: func(t *testing.T, start, end *time.Time) {
				if start == nil || end == nil {
					t.Error("Expected both future times to be parsed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			start, end, err := handlers.parseTimeRange(req)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && tt.checkTimes != nil {
				tt.checkTimes(t, start, end)
			}
		})
	}
}

func TestServerConfigValidation(t *testing.T) {
	coll := createTestCollector()

	configs := []struct {
		name   string
		config ServerConfig
		valid  bool
	}{
		{
			name:   "Valid port",
			config: ServerConfig{Port: "8080"},
			valid:  true,
		},
		{
			name:   "Empty port",
			config: ServerConfig{Port: ""},
			valid:  true, // Should work, will use default
		},
		{
			name:   "Very high port",
			config: ServerConfig{Port: "65535"},
			valid:  true,
		},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer(coll, tc.config)
			if server == nil && tc.valid {
				t.Error("Expected server to be created")
			}
			if server != nil {
				if server.port != tc.config.Port {
					t.Errorf("Expected port %s, got %s", tc.config.Port, server.port)
				}
			}
		})
	}
}
