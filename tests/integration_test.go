package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestSystemIntegration tests integration scenarios between different components
func TestSystemIntegration(t *testing.T) {
	suite := SetupSystemTest(t)
	defer suite.TeardownSystemTest()

	// Wait for initial system setup
	time.Sleep(8 * time.Second)

	t.Run("StreamerToCollectorFlow", suite.testStreamerToCollectorFlow)
	t.Run("CollectorToAPIFlow", suite.testCollectorToAPIFlow)
	t.Run("EndToEndDataJourney", suite.testEndToEndDataJourney)
	t.Run("ServiceInterconnectivity", suite.testServiceInterconnectivity)
	t.Run("DataConsistencyAcrossServices", suite.testDataConsistencyAcrossServices)
}

// testStreamerToCollectorFlow verifies data flows from streamer to collector
func (s *SystemTestSuite) testStreamerToCollectorFlow(t *testing.T) {
	t.Log("Testing data flow from streamer to collector")

	// Get baseline collector stats
	resp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get baseline collector stats: %v", err)
	}

	var baselineStats map[string]interface{}
	if err := s.parseJSONResponse(resp, &baselineStats); err != nil {
		t.Fatalf("Failed to parse baseline collector stats: %v", err)
	}

	baselineEntries, _ := baselineStats["total_entries"].(float64)
	t.Logf("Baseline entries: %.0f", baselineEntries)

	// Wait for streamer to send more data
	time.Sleep(10 * time.Second)

	// Check updated collector stats
	resp, err = s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get updated collector stats: %v", err)
	}

	var updatedStats map[string]interface{}
	if err := s.parseJSONResponse(resp, &updatedStats); err != nil {
		t.Fatalf("Failed to parse updated collector stats: %v", err)
	}

	updatedEntries, _ := updatedStats["total_entries"].(float64)
	entriesAdded := updatedEntries - baselineEntries

	t.Logf("Updated entries: %.0f", updatedEntries)
	t.Logf("Entries added: %.0f", entriesAdded)

	if entriesAdded > 0 {
		t.Logf("✅ Streamer to collector flow verified: %.0f entries added", entriesAdded)
	} else {
		t.Error("❌ No entries added - streamer to collector flow may be broken")
	}

	// Verify GPU entry distribution
	if gpuEntryCounts, ok := updatedStats["gpu_entry_counts"].(map[string]interface{}); ok && len(gpuEntryCounts) > 0 {
		t.Logf("✅ GPU data distribution verified: %d GPUs have data", len(gpuEntryCounts))
		for gpuID, count := range gpuEntryCounts {
			if entryCount, ok := count.(float64); ok {
				t.Logf("  GPU %s: %.0f entries", gpuID, entryCount)
			}
		}
	} else {
		t.Error("❌ No GPU entry distribution found")
	}
}

// testCollectorToAPIFlow verifies data flows from collector to API
func (s *SystemTestSuite) testCollectorToAPIFlow(t *testing.T) {
	t.Log("Testing data flow from collector to API")

	// Get collector stats
	collectorResp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get collector stats: %v", err)
	}

	var collectorStats map[string]interface{}
	if err := s.parseJSONResponse(collectorResp, &collectorStats); err != nil {
		t.Fatalf("Failed to parse collector stats: %v", err)
	}

	// Get API GPU list
	apiResp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get API GPU list: %v", err)
	}

	var apiGPUResponse APIResponse
	if err := s.parseJSONResponse(apiResp, &apiGPUResponse); err != nil {
		t.Fatalf("Failed to parse API GPU response: %v", err)
	}

	collectorGPUs, _ := collectorStats["total_gpus"].(float64)
	apiGPUs := float64(apiGPUResponse.Total)

	t.Logf("Collector GPUs: %.0f", collectorGPUs)
	t.Logf("API GPUs: %.0f", apiGPUs)

	if collectorGPUs > 0 && apiGPUs > 0 {
		if collectorGPUs == apiGPUs {
			t.Logf("✅ Collector to API flow verified: same GPU count (%.0f)", collectorGPUs)
		} else {
			t.Logf("⚠️  Different GPU counts: collector %.0f, API %.0f", collectorGPUs, apiGPUs)
		}
	} else {
		t.Error("❌ No GPUs found in collector or API")
	}

	// Test specific GPU telemetry retrieval
	if len(apiGPUResponse.GPUs) > 0 {
		testGPUID := apiGPUResponse.GPUs[0]
		telemetryPath := fmt.Sprintf("/api/v1/gpus/%s/telemetry", testGPUID)

		telemetryResp, err := s.makeAPIRequest("GET", telemetryPath, nil)
		if err != nil {
			t.Fatalf("Failed to get telemetry for GPU %s: %v", testGPUID, err)
		}

		var telemetryResponse APIResponse
		if err := s.parseJSONResponse(telemetryResp, &telemetryResponse); err != nil {
			t.Fatalf("Failed to parse telemetry response: %v", err)
		}

		if telemetryResponse.Total > 0 {
			t.Logf("✅ Telemetry retrieval verified: %d entries for GPU %s", telemetryResponse.Total, testGPUID)
		} else {
			t.Logf("⚠️  No telemetry data found for GPU %s", testGPUID)
		}
	}
}

// testEndToEndDataJourney traces data from streamer through collector to API
func (s *SystemTestSuite) testEndToEndDataJourney(t *testing.T) {
	t.Log("Testing end-to-end data journey")

	// Record initial state
	t.Log("Recording initial state...")

	collectorResp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get initial collector stats: %v", err)
	}

	var initialCollectorStats map[string]interface{}
	if err := s.parseJSONResponse(collectorResp, &initialCollectorStats); err != nil {
		t.Fatalf("Failed to parse initial collector stats: %v", err)
	}

	apiResp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get initial API GPU list: %v", err)
	}

	var initialAPIResponse APIResponse
	if err := s.parseJSONResponse(apiResp, &initialAPIResponse); err != nil {
		t.Fatalf("Failed to parse initial API response: %v", err)
	}

	initialCollectorEntries, _ := initialCollectorStats["total_entries"].(float64)
	initialAPIGPUs := len(initialAPIResponse.GPUs)

	t.Logf("Initial state: collector has %.0f entries, API has %d GPUs", initialCollectorEntries, initialAPIGPUs)

	// Wait for data processing
	processingTime := 15 * time.Second
	t.Logf("Waiting %v for data processing...", processingTime)
	time.Sleep(processingTime)

	// Record final state
	t.Log("Recording final state...")

	collectorResp, err = s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get final collector stats: %v", err)
	}

	var finalCollectorStats map[string]interface{}
	if err := s.parseJSONResponse(collectorResp, &finalCollectorStats); err != nil {
		t.Fatalf("Failed to parse final collector stats: %v", err)
	}

	apiResp, err = s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get final API GPU list: %v", err)
	}

	var finalAPIResponse APIResponse
	if err := s.parseJSONResponse(apiResp, &finalAPIResponse); err != nil {
		t.Fatalf("Failed to parse final API response: %v", err)
	}

	finalCollectorEntries, _ := finalCollectorStats["total_entries"].(float64)
	finalAPIGPUs := len(finalAPIResponse.GPUs)

	collectorEntriesAdded := finalCollectorEntries - initialCollectorEntries
	apiGPUsAdded := finalAPIGPUs - initialAPIGPUs

	t.Logf("Final state: collector has %.0f entries (+%.0f), API has %d GPUs (+%d)",
		finalCollectorEntries, collectorEntriesAdded, finalAPIGPUs, apiGPUsAdded)

	// Verify end-to-end flow
	if collectorEntriesAdded > 0 {
		t.Logf("✅ Data processing verified: %.0f entries added to collector", collectorEntriesAdded)
	} else {
		t.Error("❌ No entries added to collector during test period")
	}

	if finalAPIGPUs > 0 {
		t.Logf("✅ API data availability verified: %d GPUs available", finalAPIGPUs)
	} else {
		t.Error("❌ No GPUs available through API")
	}

	// Test specific data retrieval
	if len(finalAPIResponse.GPUs) > 0 {
		testGPUID := finalAPIResponse.GPUs[0]
		telemetryPath := fmt.Sprintf("/api/v1/gpus/%s/telemetry?limit=5", testGPUID)

		telemetryResp, err := s.makeAPIRequest("GET", telemetryPath, nil)
		if err != nil {
			t.Fatalf("Failed to get telemetry sample: %v", err)
		}

		var telemetryResponse APIResponse
		if err := s.parseJSONResponse(telemetryResp, &telemetryResponse); err != nil {
			t.Fatalf("Failed to parse telemetry sample: %v", err)
		}

		if len(telemetryResponse.Data) > 0 {
			t.Logf("✅ End-to-end data retrieval verified: retrieved %d telemetry entries", len(telemetryResponse.Data))

			// Verify data structure
			entry := telemetryResponse.Data[0]
			if gpuID, exists := entry["gpu_id"]; exists {
				t.Logf("  Sample GPU ID: %v", gpuID)
			}
			if timestamp, exists := entry["timestamp"]; exists {
				t.Logf("  Sample timestamp: %v", timestamp)
			}
			if metrics, exists := entry["metrics"]; exists {
				if metricsMap, ok := metrics.(map[string]interface{}); ok {
					t.Logf("  Sample metrics: %d metric fields", len(metricsMap))
				}
			}
		} else {
			t.Error("❌ No telemetry data retrieved for end-to-end test")
		}
	}
}

// testServiceInterconnectivity tests the connectivity between all services
func (s *SystemTestSuite) testServiceInterconnectivity(t *testing.T) {
	t.Log("Testing service interconnectivity")

	// Test collector health
	t.Run("CollectorHealth", func(t *testing.T) {
		resp, err := s.makeCollectorRequest("GET", "/health")
		if err != nil {
			t.Errorf("Collector health check failed: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Collector health check returned status %d", resp.StatusCode)
		} else {
			t.Log("✅ Collector health check passed")
		}
		resp.Body.Close()
	})

	// Test API gateway health
	t.Run("APIGatewayHealth", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/health", nil)
		if err != nil {
			t.Errorf("API gateway health check failed: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("API gateway health check returned status %d", resp.StatusCode)
		} else {
			t.Log("✅ API gateway health check passed")
		}
		resp.Body.Close()
	})

	// Test API to collector connectivity
	t.Run("APIToCollectorConnectivity", func(t *testing.T) {
		resp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
		if err != nil {
			t.Errorf("API to collector connectivity test failed: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("API to collector connectivity returned status %d", resp.StatusCode)
		} else {
			t.Log("✅ API to collector connectivity verified")
		}
		resp.Body.Close()
	})

	// Test collector statistics endpoint
	t.Run("CollectorStatsEndpoint", func(t *testing.T) {
		resp, err := s.makeCollectorRequest("GET", "/stats")
		if err != nil {
			t.Errorf("Collector stats endpoint test failed: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Collector stats endpoint returned status %d", resp.StatusCode)
		} else {
			t.Log("✅ Collector stats endpoint accessible")
		}
		resp.Body.Close()
	})
}

// testDataConsistencyAcrossServices verifies data consistency between services
func (s *SystemTestSuite) testDataConsistencyAcrossServices(t *testing.T) {
	t.Log("Testing data consistency across services")

	// Wait for data stabilization
	time.Sleep(5 * time.Second)

	// Get collector data
	collectorResp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get collector data: %v", err)
	}

	var collectorStats map[string]interface{}
	if err := s.parseJSONResponse(collectorResp, &collectorStats); err != nil {
		t.Fatalf("Failed to parse collector data: %v", err)
	}

	// Get API data
	apiResp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get API data: %v", err)
	}

	var apiResponse APIResponse
	if err := s.parseJSONResponse(apiResp, &apiResponse); err != nil {
		t.Fatalf("Failed to parse API data: %v", err)
	}

	// Compare GPU counts
	collectorGPUs, _ := collectorStats["total_gpus"].(float64)
	apiGPUs := float64(len(apiResponse.GPUs))

	t.Logf("GPU count comparison:")
	t.Logf("  Collector: %.0f GPUs", collectorGPUs)
	t.Logf("  API: %.0f GPUs", apiGPUs)

	if collectorGPUs == apiGPUs && collectorGPUs > 0 {
		t.Log("✅ GPU count consistency verified")
	} else if collectorGPUs == 0 && apiGPUs == 0 {
		t.Log("⚠️  No GPUs found in either service")
	} else {
		t.Errorf("❌ GPU count inconsistency: collector %.0f, API %.0f", collectorGPUs, apiGPUs)
	}

	// Compare GPU IDs if available
	if len(apiResponse.GPUs) > 0 {
		if gpuEntryCounts, ok := collectorStats["gpu_entry_counts"].(map[string]interface{}); ok {
			collectorGPUIDs := make(map[string]bool)
			for gpuID := range gpuEntryCounts {
				collectorGPUIDs[gpuID] = true
			}

			apiGPUIDs := make(map[string]bool)
			for _, gpuID := range apiResponse.GPUs {
				apiGPUIDs[gpuID] = true
			}

			// Check if GPU IDs match
			matches := 0
			for gpuID := range apiGPUIDs {
				if collectorGPUIDs[gpuID] {
					matches++
				}
			}

			consistency := float64(matches) / float64(len(apiResponse.GPUs)) * 100
			t.Logf("GPU ID consistency: %.0f%% (%d/%d matching)", consistency, matches, len(apiResponse.GPUs))

			if consistency >= 80 {
				t.Log("✅ GPU ID consistency verified")
			} else {
				t.Errorf("❌ Low GPU ID consistency: %.0f%%", consistency)
			}
		}
	}

	// Test telemetry data consistency for a specific GPU
	if len(apiResponse.GPUs) > 0 {
		testGPUID := apiResponse.GPUs[0]

		// Get telemetry from collector
		collectorTelemetryResp, err := s.makeCollectorRequest("GET", fmt.Sprintf("/api/v1/gpus/%s/telemetry", testGPUID))
		if err != nil {
			t.Logf("Could not get telemetry from collector for %s: %v", testGPUID, err)
		} else {
			var collectorTelemetry map[string]interface{}
			if err := s.parseJSONResponse(collectorTelemetryResp, &collectorTelemetry); err == nil {
				// Get telemetry from API
				apiTelemetryResp, err := s.makeAPIRequest("GET", fmt.Sprintf("/api/v1/gpus/%s/telemetry", testGPUID), nil)
				if err != nil {
					t.Errorf("Failed to get telemetry from API for %s: %v", testGPUID, err)
				} else {
					var apiTelemetry APIResponse
					if err := s.parseJSONResponse(apiTelemetryResp, &apiTelemetry); err == nil {
						collectorTotal, _ := collectorTelemetry["total"].(float64)
						apiTotal := float64(apiTelemetry.Total)

						t.Logf("Telemetry consistency for GPU %s:", testGPUID)
						t.Logf("  Collector: %.0f entries", collectorTotal)
						t.Logf("  API: %.0f entries", apiTotal)

						if collectorTotal == apiTotal && collectorTotal > 0 {
							t.Log("✅ Telemetry data consistency verified")
						} else if collectorTotal == 0 && apiTotal == 0 {
							t.Log("⚠️  No telemetry data found")
						} else {
							t.Logf("⚠️  Telemetry count difference: collector %.0f, API %.0f", collectorTotal, apiTotal)
						}
					}
				}
			}
		}
	}
}
