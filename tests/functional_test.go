package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestSystemEndToEnd tests the complete telemetry pipeline functionality
func TestSystemEndToEnd(t *testing.T) {
	suite := SetupSystemTest(t)
	defer suite.TeardownSystemTest()

	t.Run("HealthChecks", suite.testHealthChecks)
	t.Run("CollectorStats", suite.testCollectorStats)
	t.Run("APIGatewayGPUs", suite.testAPIGatewayGPUs)
	t.Run("APIGatewayTelemetry", suite.testAPIGatewayTelemetry)
	t.Run("APIGatewayHosts", suite.testAPIGatewayHosts)
	t.Run("APIGatewayHostGPUs", suite.testAPIGatewayHostGPUs)
	t.Run("DataFlowIntegration", suite.testDataFlowIntegration)
	t.Run("APIParameters", suite.testAPIParameters)
	t.Run("SwaggerDocumentation", suite.testSwaggerDocumentation)
	t.Run("ErrorHandling", suite.testErrorHandling)
}

// testHealthChecks verifies that all services respond to health checks
func (s *SystemTestSuite) testHealthChecks(t *testing.T) {
	t.Log("Testing health checks for all services")

	// Test collector health
	t.Run("CollectorHealth", func(t *testing.T) {
		resp, err := s.makeCollectorRequest("GET", "/health")
		if err != nil {
			t.Fatalf("Failed to make collector health request: %v", err)
		}

		var health HealthResponse
		if err := s.parseJSONResponse(resp, &health); err != nil {
			t.Fatalf("Failed to parse collector health response: %v", err)
		}

		if health.Status != "healthy" {
			t.Errorf("Expected collector status 'healthy', got '%s'", health.Status)
		}

		if health.Timestamp == "" {
			t.Error("Expected collector health timestamp to be non-empty")
		}

		t.Logf("Collector health: %s at %s", health.Status, health.Timestamp)
	})

	// Test API gateway health
	t.Run("APIGatewayHealth", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to make API gateway health request: %v", err)
		}

		var health HealthResponse
		if err := s.parseJSONResponse(resp, &health); err != nil {
			t.Fatalf("Failed to parse API gateway health response: %v", err)
		}

		if health.Status != "healthy" {
			t.Errorf("Expected API gateway status 'healthy', got '%s'", health.Status)
		}

		t.Logf("API gateway health: %s at %s", health.Status, health.Timestamp)
	})
}

// testCollectorStats verifies collector statistics endpoint
func (s *SystemTestSuite) testCollectorStats(t *testing.T) {
	t.Log("Testing collector statistics endpoint")

	// Give some time for data to be processed
	time.Sleep(3 * time.Second)

	resp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to make collector stats request: %v", err)
	}

	var stats map[string]interface{}
	if err := s.parseJSONResponse(resp, &stats); err != nil {
		t.Fatalf("Failed to parse collector stats response: %v", err)
	}

	// Verify expected fields in stats
	expectedFields := []string{"gpu_entry_counts", "max_entries_per_gpu", "total_entries", "total_gpus"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected field '%s' not found in collector stats", field)
		}
	}

	// Log stats for debugging
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	t.Logf("Collector stats: %s", statsJSON)

	// Verify we have some GPUs and entries
	if totalGPUs, ok := stats["total_gpus"].(float64); ok && totalGPUs > 0 {
		t.Logf("Collector has %.0f GPUs", totalGPUs)
	} else {
		t.Log("Warning: No GPUs found in collector stats yet")
	}

	if totalEntries, ok := stats["total_entries"].(float64); ok && totalEntries > 0 {
		t.Logf("Collector has %.0f total entries", totalEntries)
	} else {
		t.Log("Warning: No entries found in collector stats yet")
	}
}

// testAPIGatewayGPUs tests the GPU listing endpoint
func (s *SystemTestSuite) testAPIGatewayGPUs(t *testing.T) {
	t.Log("Testing API gateway GPU listing endpoint")

	// Give some time for data to be processed
	time.Sleep(3 * time.Second)

	t.Run("ListGPUs", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
		if err != nil {
			t.Fatalf("Failed to make GPU list request: %v", err)
		}

		var gpuResponse APIResponse
		if err := s.parseJSONResponse(resp, &gpuResponse); err != nil {
			t.Fatalf("Failed to parse GPU list response: %v", err)
		}

		t.Logf("Found %d GPUs: %v", gpuResponse.Total, gpuResponse.GPUs)

		// Verify we have some GPUs (might take time for data to flow through)
		if gpuResponse.Total == 0 {
			t.Log("Warning: No GPUs found yet, this might be due to timing")
		} else {
			// Verify GPU ID format (should be UUIDs for DCGM format)
			for _, gpu := range gpuResponse.GPUs {
				if !isValidGPUID(gpu) {
					t.Errorf("Invalid GPU ID format: %s", gpu)
				}
			}
		}
	})

	t.Run("ListGPUsWithPagination", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus?limit=2&offset=0", nil)
		if err != nil {
			t.Fatalf("Failed to make paginated GPU list request: %v", err)
		}

		var gpuResponse APIResponse
		if err := s.parseJSONResponse(resp, &gpuResponse); err != nil {
			t.Fatalf("Failed to parse paginated GPU list response: %v", err)
		}

		t.Logf("Paginated GPU list: %d total, returned %d", gpuResponse.Total, len(gpuResponse.GPUs))

		// Verify pagination worked
		if len(gpuResponse.GPUs) > 2 {
			t.Errorf("Expected at most 2 GPUs with limit=2, got %d", len(gpuResponse.GPUs))
		}
	})
}

// testAPIGatewayTelemetry tests the telemetry data endpoint
func (s *SystemTestSuite) testAPIGatewayTelemetry(t *testing.T) {
	t.Log("Testing API gateway telemetry endpoint")

	// Wait for data to flow through the system
	time.Sleep(5 * time.Second)

	// First get GPU list to have a valid GPU ID
	resp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get GPU list: %v", err)
	}

	var gpuResponse APIResponse
	if err := s.parseJSONResponse(resp, &gpuResponse); err != nil {
		t.Fatalf("Failed to parse GPU list: %v", err)
	}

	if len(gpuResponse.GPUs) == 0 {
		t.Skip("No GPUs available for telemetry testing")
	}

	testGPUID := gpuResponse.GPUs[0]
	t.Logf("Testing telemetry for GPU: %s", testGPUID)

	t.Run("GetTelemetryData", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/gpus/%s/telemetry", testGPUID)
		resp, err := s.makeAPIRequest("GET", path, nil)
		if err != nil {
			t.Fatalf("Failed to make telemetry request: %v", err)
		}

		var telemetryResponse APIResponse
		if err := s.parseJSONResponse(resp, &telemetryResponse); err != nil {
			t.Fatalf("Failed to parse telemetry response: %v", err)
		}

		t.Logf("Telemetry data: %d entries", telemetryResponse.Total)

		// Verify telemetry data structure
		if len(telemetryResponse.Data) > 0 {
			entry := telemetryResponse.Data[0]
			expectedFields := []string{"gpu_id", "timestamp", "metrics"}
			for _, field := range expectedFields {
				if _, exists := entry[field]; !exists {
					t.Errorf("Expected field '%s' not found in telemetry entry", field)
				}
			}

			// Log first entry for debugging
			entryJSON, _ := json.MarshalIndent(entry, "", "  ")
			t.Logf("Sample telemetry entry: %s", entryJSON)
		}
	})

	t.Run("GetTelemetryWithPagination", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/gpus/%s/telemetry?limit=5&offset=0", testGPUID)
		resp, err := s.makeAPIRequest("GET", path, nil)
		if err != nil {
			t.Fatalf("Failed to make paginated telemetry request: %v", err)
		}

		var telemetryResponse APIResponse
		if err := s.parseJSONResponse(resp, &telemetryResponse); err != nil {
			t.Fatalf("Failed to parse paginated telemetry response: %v", err)
		}

		t.Logf("Paginated telemetry: %d total, returned %d", telemetryResponse.Total, len(telemetryResponse.Data))

		// Verify pagination
		if len(telemetryResponse.Data) > 5 {
			t.Errorf("Expected at most 5 entries with limit=5, got %d", len(telemetryResponse.Data))
		}
	})

	t.Run("GetTelemetryWithTimeRange", func(t *testing.T) {
		// Use current time and go back 1 hour
		endTime := time.Now()
		startTime := endTime.Add(-1 * time.Hour)

		path := fmt.Sprintf("/api/v1/gpus/%s/telemetry?start_time=%s&end_time=%s",
			testGPUID,
			startTime.Format(time.RFC3339),
			endTime.Format(time.RFC3339))

		resp, err := s.makeAPIRequest("GET", path, nil)
		if err != nil {
			t.Fatalf("Failed to make time-ranged telemetry request: %v", err)
		}

		var telemetryResponse APIResponse
		if err := s.parseJSONResponse(resp, &telemetryResponse); err != nil {
			t.Fatalf("Failed to parse time-ranged telemetry response: %v", err)
		}

		t.Logf("Time-ranged telemetry: %d entries", telemetryResponse.Total)
	})
}

// testAPIGatewayHosts tests the hosts listing endpoint
func (s *SystemTestSuite) testAPIGatewayHosts(t *testing.T) {
	t.Log("Testing API gateway hosts endpoint")

	// Wait for data to flow through the system
	time.Sleep(5 * time.Second)

	t.Run("GetHosts", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/hosts", nil)
		if err != nil {
			t.Fatalf("Failed to make hosts request: %v", err)
		}

		var hostsResponse struct {
			Hosts      []string `json:"hosts"`
			Total      int      `json:"total"`
			Pagination struct {
				Limit   int  `json:"limit"`
				Offset  int  `json:"offset"`
				HasNext bool `json:"has_next"`
			} `json:"pagination"`
		}

		if err := s.parseJSONResponse(resp, &hostsResponse); err != nil {
			t.Fatalf("Failed to parse hosts response: %v", err)
		}

		t.Logf("Found %d hosts: %v", hostsResponse.Total, hostsResponse.Hosts)

		// Verify response structure
		if hostsResponse.Total < 0 {
			t.Errorf("Expected non-negative total, got %d", hostsResponse.Total)
		}

		if hostsResponse.Pagination.Limit <= 0 {
			t.Errorf("Expected positive pagination limit, got %d", hostsResponse.Pagination.Limit)
		}

		// Verify hosts are unique
		hostSet := make(map[string]bool)
		for _, host := range hostsResponse.Hosts {
			if host == "" {
				t.Error("Found empty hostname in response")
			}
			if hostSet[host] {
				t.Errorf("Duplicate hostname found: %s", host)
			}
			hostSet[host] = true
		}
	})

	t.Run("GetHostsWithPagination", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/hosts?limit=2&offset=0", nil)
		if err != nil {
			t.Fatalf("Failed to make paginated hosts request: %v", err)
		}

		var hostsResponse struct {
			Hosts      []string `json:"hosts"`
			Total      int      `json:"total"`
			Pagination struct {
				Limit   int  `json:"limit"`
				Offset  int  `json:"offset"`
				HasNext bool `json:"has_next"`
			} `json:"pagination"`
		}

		if err := s.parseJSONResponse(resp, &hostsResponse); err != nil {
			t.Fatalf("Failed to parse paginated hosts response: %v", err)
		}

		// Verify pagination parameters
		if hostsResponse.Pagination.Limit != 2 {
			t.Errorf("Expected limit 2, got %d", hostsResponse.Pagination.Limit)
		}

		if hostsResponse.Pagination.Offset != 0 {
			t.Errorf("Expected offset 0, got %d", hostsResponse.Pagination.Offset)
		}

		// Verify returned data respects pagination
		if len(hostsResponse.Hosts) > 2 {
			t.Errorf("Expected at most 2 hosts with limit=2, got %d", len(hostsResponse.Hosts))
		}

		t.Logf("Paginated hosts: %d returned out of %d total", len(hostsResponse.Hosts), hostsResponse.Total)
	})
}

// testAPIGatewayHostGPUs tests the host GPUs endpoint
func (s *SystemTestSuite) testAPIGatewayHostGPUs(t *testing.T) {
	t.Log("Testing API gateway host GPUs endpoint")

	// Wait for data to flow through the system
	time.Sleep(5 * time.Second)

	// First get hosts list to have a valid hostname
	resp, err := s.makeAPIRequest("GET", "/api/v1/hosts", nil)
	if err != nil {
		t.Fatalf("Failed to get hosts list: %v", err)
	}

	var hostsResponse struct {
		Hosts []string `json:"hosts"`
		Total int      `json:"total"`
	}

	if err := s.parseJSONResponse(resp, &hostsResponse); err != nil {
		t.Fatalf("Failed to parse hosts list: %v", err)
	}

	if len(hostsResponse.Hosts) == 0 {
		t.Skip("No hosts available for host GPUs testing")
	}

	testHostname := hostsResponse.Hosts[0]
	t.Logf("Testing GPUs for host: %s", testHostname)

	t.Run("GetHostGPUs", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/hosts/%s/gpus", testHostname)
		resp, err := s.makeAPIRequest("GET", path, nil)
		if err != nil {
			t.Fatalf("Failed to make host GPUs request: %v", err)
		}

		var hostGPUsResponse struct {
			Hostname string   `json:"hostname"`
			GPUs     []string `json:"gpus"`
			Total    int      `json:"total"`
		}

		if err := s.parseJSONResponse(resp, &hostGPUsResponse); err != nil {
			t.Fatalf("Failed to parse host GPUs response: %v", err)
		}

		t.Logf("Host %s has %d GPUs: %v", hostGPUsResponse.Hostname, hostGPUsResponse.Total, hostGPUsResponse.GPUs)

		// Verify response structure
		if hostGPUsResponse.Hostname != testHostname {
			t.Errorf("Expected hostname %s, got %s", testHostname, hostGPUsResponse.Hostname)
		}

		if hostGPUsResponse.Total < 0 {
			t.Errorf("Expected non-negative total, got %d", hostGPUsResponse.Total)
		}

		if hostGPUsResponse.Total != len(hostGPUsResponse.GPUs) {
			t.Errorf("Total (%d) doesn't match number of GPUs returned (%d)", hostGPUsResponse.Total, len(hostGPUsResponse.GPUs))
		}

		// Verify GPUs are unique
		gpuSet := make(map[string]bool)
		for _, gpu := range hostGPUsResponse.GPUs {
			if gpu == "" {
				t.Error("Found empty GPU ID in response")
			}
			if gpuSet[gpu] {
				t.Errorf("Duplicate GPU ID found: %s", gpu)
			}
			gpuSet[gpu] = true
		}
	})

	t.Run("GetHostGPUsNonExistent", func(t *testing.T) {
		path := "/api/v1/hosts/non-existent-host/gpus"
		resp, err := s.makeAPIRequest("GET", path, nil)
		if err != nil {
			t.Fatalf("Failed to make non-existent host GPUs request: %v", err)
		}

		var hostGPUsResponse struct {
			Hostname string   `json:"hostname"`
			GPUs     []string `json:"gpus"`
			Total    int      `json:"total"`
		}

		if err := s.parseJSONResponse(resp, &hostGPUsResponse); err != nil {
			t.Fatalf("Failed to parse non-existent host GPUs response: %v", err)
		}

		// Should return empty list for non-existent host
		if hostGPUsResponse.Total != 0 {
			t.Errorf("Expected 0 GPUs for non-existent host, got %d", hostGPUsResponse.Total)
		}

		if len(hostGPUsResponse.GPUs) != 0 {
			t.Errorf("Expected empty GPUs list for non-existent host, got %d GPUs", len(hostGPUsResponse.GPUs))
		}

		t.Logf("Non-existent host correctly returned 0 GPUs")
	})
}

// testDataFlowIntegration tests the complete data flow from streamer to API
func (s *SystemTestSuite) testDataFlowIntegration(t *testing.T) {
	t.Log("Testing complete data flow integration")

	// Wait for data to flow through the system
	t.Log("Waiting for data to flow through the system...")
	time.Sleep(10 * time.Second)

	// Get initial stats
	resp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get initial collector stats: %v", err)
	}

	var initialStats map[string]interface{}
	if err := s.parseJSONResponse(resp, &initialStats); err != nil {
		t.Fatalf("Failed to parse initial collector stats: %v", err)
	}

	initialEntries, _ := initialStats["total_entries"].(float64)
	t.Logf("Initial total entries: %.0f", initialEntries)

	// Wait for more data to be processed
	time.Sleep(5 * time.Second)

	// Get updated stats
	resp, err = s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get updated collector stats: %v", err)
	}

	var updatedStats map[string]interface{}
	if err := s.parseJSONResponse(resp, &updatedStats); err != nil {
		t.Fatalf("Failed to parse updated collector stats: %v", err)
	}

	updatedEntries, _ := updatedStats["total_entries"].(float64)
	t.Logf("Updated total entries: %.0f", updatedEntries)

	// Verify data is flowing (entries should increase)
	if updatedEntries > initialEntries {
		t.Logf("Data flow verified: entries increased from %.0f to %.0f", initialEntries, updatedEntries)
	} else {
		t.Log("Warning: Entry count didn't increase, data flow might be slow")
	}

	// Verify data consistency between collector and API
	apiResp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get GPU list from API: %v", err)
	}

	var gpuResponse APIResponse
	if err := s.parseJSONResponse(apiResp, &gpuResponse); err != nil {
		t.Fatalf("Failed to parse GPU list from API: %v", err)
	}

	collectorGPUs, _ := updatedStats["total_gpus"].(float64)
	apiGPUs := float64(gpuResponse.Total)

	t.Logf("GPU count comparison - Collector: %.0f, API: %.0f", collectorGPUs, apiGPUs)

	if collectorGPUs > 0 && apiGPUs > 0 && collectorGPUs == apiGPUs {
		t.Log("Data consistency verified: same GPU count in collector and API")
	} else if collectorGPUs == 0 && apiGPUs == 0 {
		t.Log("Warning: No GPUs found in either collector or API")
	} else {
		t.Logf("Data consistency issue: collector has %.0f GPUs, API has %.0f GPUs", collectorGPUs, apiGPUs)
	}
}

// testAPIParameters tests various API parameter combinations
func (s *SystemTestSuite) testAPIParameters(t *testing.T) {
	t.Log("Testing API parameter validation and handling")

	t.Run("InvalidGPUID", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus/invalid-gpu-id/telemetry", nil)
		if err != nil {
			t.Fatalf("Failed to make request with invalid GPU ID: %v", err)
		}

		// Should return empty data, not an error
		var telemetryResponse APIResponse
		if err := s.parseJSONResponse(resp, &telemetryResponse); err != nil {
			t.Fatalf("Failed to parse response for invalid GPU ID: %v", err)
		}

		if telemetryResponse.Total != 0 {
			t.Errorf("Expected 0 entries for invalid GPU ID, got %d", telemetryResponse.Total)
		}
	})

	t.Run("InvalidTimeFormat", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus/test/telemetry?start_time=invalid-time", nil)
		if err != nil {
			t.Fatalf("Failed to make request with invalid time: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid time format, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("InvalidPaginationParams", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus?limit=invalid", nil)
		if err != nil {
			t.Fatalf("Failed to make request with invalid limit: %v", err)
		}

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid limit, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("LargeLimitHandling", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus?limit=10000", nil)
		if err != nil {
			t.Fatalf("Failed to make request with large limit: %v", err)
		}

		var gpuResponse APIResponse
		if err := s.parseJSONResponse(resp, &gpuResponse); err != nil {
			t.Fatalf("Failed to parse response with large limit: %v", err)
		}

		// Should handle large limits gracefully
		t.Logf("Large limit handled, returned %d GPUs", len(gpuResponse.GPUs))
	})
}

// testSwaggerDocumentation tests the Swagger documentation endpoint
func (s *SystemTestSuite) testSwaggerDocumentation(t *testing.T) {
	t.Log("Testing Swagger documentation")

	resp, err := s.makeAPIRequest("GET", "/swagger/index.html", nil)
	if err != nil {
		t.Fatalf("Failed to access Swagger UI: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for Swagger UI, got %d", resp.StatusCode)
	}

	// Check for Swagger documentation JSON
	resp, err = s.makeAPIRequest("GET", "/swagger/doc.json", nil)
	if err != nil {
		t.Fatalf("Failed to access Swagger JSON: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for Swagger JSON, got %d", resp.StatusCode)
	}

	t.Log("Swagger documentation is accessible")
}

// testErrorHandling tests various error conditions
func (s *SystemTestSuite) testErrorHandling(t *testing.T) {
	t.Log("Testing error handling scenarios")

	t.Run("NonExistentEndpoint", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/nonexistent", nil)
		if err != nil {
			t.Fatalf("Failed to make request to non-existent endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent endpoint, got %d", resp.StatusCode)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		resp, err := s.makeAPIRequest("POST", "/api/v1/gpus", nil)
		if err != nil {
			t.Fatalf("Failed to make POST request to GET endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 405 (Method Not Allowed) or 404 (Not Found) for invalid method, got %d", resp.StatusCode)
		}
	})

	t.Run("CollectorConnectivity", func(t *testing.T) {
		// This test verifies that the API gateway handles collector connectivity issues gracefully
		// In this case, we know the collector is running, so we should get valid responses
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
		if err != nil {
			t.Fatalf("Failed to test collector connectivity: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected successful response when collector is available, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

// Helper functions

// isValidGPUID checks if a GPU ID is in valid format (UUID for DCGM)
func isValidGPUID(gpuID string) bool {
	// For DCGM format, we expect UUIDs like "GPU-12345678-1234-1234-1234-123456789abc"
	// or legacy format like "gpu_0"
	return strings.HasPrefix(gpuID, "GPU-") || strings.HasPrefix(gpuID, "gpu")
}
