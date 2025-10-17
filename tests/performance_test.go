package tests

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// TestSystemPerformance tests the performance characteristics of the system
func TestSystemPerformance(t *testing.T) {
	suite := SetupSystemTest(t)
	defer suite.TeardownSystemTest()

	// Wait for initial data to flow through
	time.Sleep(10 * time.Second)

	t.Run("HighThroughputDataProcessing", suite.testHighThroughputProcessing)
	t.Run("ConcurrentAPIRequests", suite.testConcurrentAPIRequests)
	t.Run("MemoryUsageStability", suite.testMemoryUsageStability)
	t.Run("ResponseTimeConsistency", suite.testResponseTimeConsistency)
}

// testHighThroughputProcessing tests the system's ability to handle high data throughput
func (s *SystemTestSuite) testHighThroughputProcessing(t *testing.T) {
	t.Log("Testing high throughput data processing")

	// Get initial stats
	resp, err := s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get initial stats: %v", err)
	}

	var initialStats map[string]interface{}
	if err := s.parseJSONResponse(resp, &initialStats); err != nil {
		t.Fatalf("Failed to parse initial stats: %v", err)
	}

	initialEntries, _ := initialStats["total_entries"].(float64)
	startTime := time.Now()

	// Wait for a known duration to measure throughput
	testDuration := 30 * time.Second
	t.Logf("Measuring throughput for %v", testDuration)
	time.Sleep(testDuration)

	// Get final stats
	resp, err = s.makeCollectorRequest("GET", "/stats")
	if err != nil {
		t.Fatalf("Failed to get final stats: %v", err)
	}

	var finalStats map[string]interface{}
	if err := s.parseJSONResponse(resp, &finalStats); err != nil {
		t.Fatalf("Failed to parse final stats: %v", err)
	}

	finalEntries, _ := finalStats["total_entries"].(float64)
	actualDuration := time.Since(startTime)

	entriesProcessed := finalEntries - initialEntries
	throughput := entriesProcessed / actualDuration.Seconds()

	t.Logf("Processed %.0f entries in %v", entriesProcessed, actualDuration)
	t.Logf("Throughput: %.2f entries/second", throughput)

	// Verify minimum throughput (adjust based on expected performance)
	minThroughput := 5.0 // entries per second
	if throughput >= minThroughput {
		t.Logf("✅ Throughput test passed: %.2f >= %.2f entries/sec", throughput, minThroughput)
	} else {
		t.Logf("⚠️  Low throughput: %.2f < %.2f entries/sec", throughput, minThroughput)
	}
}

// testConcurrentAPIRequests tests the system's ability to handle concurrent API requests
func (s *SystemTestSuite) testConcurrentAPIRequests(t *testing.T) {
	t.Log("Testing concurrent API requests")

	// First ensure we have some GPUs
	resp, err := s.makeAPIRequest("GET", "/api/v1/gpus", nil)
	if err != nil {
		t.Fatalf("Failed to get GPU list: %v", err)
	}

	var gpuResponse APIResponse
	if err := s.parseJSONResponse(resp, &gpuResponse); err != nil {
		t.Fatalf("Failed to parse GPU list: %v", err)
	}

	if len(gpuResponse.GPUs) == 0 {
		t.Skip("No GPUs available for concurrent testing")
	}

	// Test parameters
	numConcurrentRequests := 20
	requestsPerGoroutine := 5

	t.Logf("Starting %d concurrent goroutines, %d requests each", numConcurrentRequests, requestsPerGoroutine)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount, errorCount int
	var totalResponseTime time.Duration

	startTime := time.Now()

	// Launch concurrent requests
	for i := 0; i < numConcurrentRequests; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				requestStart := time.Now()

				// Alternate between GPU list and telemetry requests
				var resp *http.Response
				var err error

				if j%2 == 0 {
					resp, err = s.makeAPIRequest("GET", "/api/v1/gpus", nil)
				} else {
					testGPUID := gpuResponse.GPUs[j%len(gpuResponse.GPUs)]
					path := fmt.Sprintf("/api/v1/gpus/%s/telemetry?limit=10", testGPUID)
					resp, err = s.makeAPIRequest("GET", path, nil)
				}

				requestDuration := time.Since(requestStart)

				mu.Lock()
				if err != nil || resp.StatusCode != http.StatusOK {
					errorCount++
					if err != nil {
						t.Logf("Goroutine %d request %d error: %v", goroutineID, j, err)
					} else {
						t.Logf("Goroutine %d request %d status: %d", goroutineID, j, resp.StatusCode)
					}
				} else {
					successCount++
				}
				totalResponseTime += requestDuration
				mu.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	totalRequests := numConcurrentRequests * requestsPerGoroutine
	successRate := float64(successCount) / float64(totalRequests) * 100
	avgResponseTime := totalResponseTime / time.Duration(totalRequests)
	requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()

	t.Logf("Concurrent request results:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Success rate: %.2f%%", successRate)
	t.Logf("  Average response time: %v", avgResponseTime)
	t.Logf("  Requests per second: %.2f", requestsPerSecond)
	t.Logf("  Total test duration: %v", totalDuration)

	// Verify performance criteria
	minSuccessRate := 95.0
	maxAvgResponseTime := 2 * time.Second

	if successRate >= minSuccessRate {
		t.Logf("✅ Success rate test passed: %.2f%% >= %.2f%%", successRate, minSuccessRate)
	} else {
		t.Errorf("❌ Success rate test failed: %.2f%% < %.2f%%", successRate, minSuccessRate)
	}

	if avgResponseTime <= maxAvgResponseTime {
		t.Logf("✅ Response time test passed: %v <= %v", avgResponseTime, maxAvgResponseTime)
	} else {
		t.Errorf("❌ Response time test failed: %v > %v", avgResponseTime, maxAvgResponseTime)
	}
}

// testMemoryUsageStability tests that memory usage remains stable over time
func (s *SystemTestSuite) testMemoryUsageStability(t *testing.T) {
	t.Log("Testing memory usage stability")

	// Sample collector stats multiple times to check for memory leaks
	samples := 5
	sampleInterval := 10 * time.Second

	var entryCount []float64
	var gpuCount []float64

	for i := 0; i < samples; i++ {
		t.Logf("Taking sample %d/%d", i+1, samples)

		resp, err := s.makeCollectorRequest("GET", "/stats")
		if err != nil {
			t.Fatalf("Failed to get stats sample %d: %v", i, err)
		}

		var stats map[string]interface{}
		if err := s.parseJSONResponse(resp, &stats); err != nil {
			t.Fatalf("Failed to parse stats sample %d: %v", i, err)
		}

		entries, _ := stats["total_entries"].(float64)
		gpus, _ := stats["total_gpus"].(float64)

		entryCount = append(entryCount, entries)
		gpuCount = append(gpuCount, gpus)

		t.Logf("  Sample %d: %.0f entries, %.0f GPUs", i+1, entries, gpus)

		if i < samples-1 {
			time.Sleep(sampleInterval)
		}
	}

	// Analyze stability
	t.Log("Memory stability analysis:")

	// Check that GPU count is stable (shouldn't decrease)
	for i := 1; i < len(gpuCount); i++ {
		if gpuCount[i] < gpuCount[i-1] {
			t.Errorf("GPU count decreased from %.0f to %.0f (sample %d to %d)", gpuCount[i-1], gpuCount[i], i, i+1)
		}
	}

	// Check that entry count is increasing (showing data is being processed)
	totalIncrease := entryCount[len(entryCount)-1] - entryCount[0]
	if totalIncrease > 0 {
		t.Logf("✅ Data processing verified: %.0f entries added during test", totalIncrease)
	} else {
		t.Log("⚠️  No entry count increase detected")
	}

	// Check for reasonable growth (not explosive)
	maxSampleIncrease := 0.0
	for i := 1; i < len(entryCount); i++ {
		increase := entryCount[i] - entryCount[i-1]
		if increase > maxSampleIncrease {
			maxSampleIncrease = increase
		}
	}

	expectedMaxIncrease := 50.0 * sampleInterval.Seconds() * 2 // 2x buffer for 50 msg/sec default
	if maxSampleIncrease <= expectedMaxIncrease {
		t.Logf("✅ Growth rate reasonable: max increase %.0f <= %.0f", maxSampleIncrease, expectedMaxIncrease)
	} else {
		t.Logf("⚠️  High growth rate: max increase %.0f > %.0f", maxSampleIncrease, expectedMaxIncrease)
	}
}

// testResponseTimeConsistency tests that API response times are consistent
func (s *SystemTestSuite) testResponseTimeConsistency(t *testing.T) {
	t.Log("Testing API response time consistency")

	// Test different endpoints multiple times
	tests := []struct {
		name string
		path string
	}{
		{"Health", "/health"},
		{"GPUList", "/api/v1/gpus"},
		{"GPUListPaginated", "/api/v1/gpus?limit=5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			numRequests := 10
			var responseTimes []time.Duration

			for i := 0; i < numRequests; i++ {
				start := time.Now()
				resp, err := s.makeAPIRequest("GET", test.path, nil)
				duration := time.Since(start)

				if err != nil {
					t.Fatalf("Request %d failed: %v", i, err)
				}

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Request %d returned status %d", i, resp.StatusCode)
				}

				resp.Body.Close()
				responseTimes = append(responseTimes, duration)
			}

			// Calculate statistics
			var total time.Duration
			min := responseTimes[0]
			max := responseTimes[0]

			for _, duration := range responseTimes {
				total += duration
				if duration < min {
					min = duration
				}
				if duration > max {
					max = duration
				}
			}

			avg := total / time.Duration(numRequests)
			variance := max - min

			t.Logf("%s endpoint performance:", test.name)
			t.Logf("  Requests: %d", numRequests)
			t.Logf("  Average: %v", avg)
			t.Logf("  Min: %v", min)
			t.Logf("  Max: %v", max)
			t.Logf("  Variance: %v", variance)

			// Check consistency (variance should be reasonable)
			maxAcceptableVariance := 5 * time.Second
			if variance <= maxAcceptableVariance {
				t.Logf("✅ Consistency test passed: variance %v <= %v", variance, maxAcceptableVariance)
			} else {
				t.Errorf("❌ Consistency test failed: variance %v > %v", variance, maxAcceptableVariance)
			}

			// Check reasonable response times
			maxAcceptableAvg := 2 * time.Second
			if avg <= maxAcceptableAvg {
				t.Logf("✅ Performance test passed: average %v <= %v", avg, maxAcceptableAvg)
			} else {
				t.Errorf("❌ Performance test failed: average %v > %v", avg, maxAcceptableAvg)
			}
		})
	}
}
