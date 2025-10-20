package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// SystemTestSuite manages the full system test environment
type SystemTestSuite struct {
	t              *testing.T
	tempDir        string
	collectorCmd   *exec.Cmd
	streamerCmd    *exec.Cmd
	apiGatewayCmd  *exec.Cmd
	mqServiceCmd   *exec.Cmd
	collectorPort  string
	apiGatewayPort string
	mqServicePort  string
	streamerRate   string
	testDataFile   string
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
}

// TestData represents the expected structure of telemetry data
type TestData struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// APIResponse represents standard API responses
type APIResponse struct {
	GPUs       []string                 `json:"gpus,omitempty"`
	Total      int                      `json:"total"`
	Data       []map[string]interface{} `json:"data,omitempty"`
	Statistics map[string]interface{}   `json:"statistics,omitempty"`
}

// HealthResponse represents health check responses
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// SetupSystemTest initializes the complete system test environment
func SetupSystemTest(t *testing.T) *SystemTestSuite {
	t.Helper()

	suite := &SystemTestSuite{
		t:              t,
		collectorPort:  "18080",
		apiGatewayPort: "18081",
		mqServicePort:  "19090",
		streamerRate:   "10", // 10 messages per second for faster testing
	}

	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "telemetry_system_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	suite.tempDir = tempDir

	t.Logf("System test temp directory: %s", tempDir)

	// Create test data
	suite.createTestData()

	// Build binaries
	suite.buildBinaries()

	// Start services in order: MQ service first, then collector, then API gateway, then streamer
	suite.startMQService()
	suite.startCollector()
	suite.startAPIGateway()
	suite.startStreamer()

	// Wait for services to be ready
	suite.waitForServices()

	t.Logf("System test environment ready")
	return suite
}

// TeardownSystemTest cleans up the system test environment
func (s *SystemTestSuite) TeardownSystemTest() {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.t.Logf("Tearing down system test environment")

	// Cancel context to signal shutdown
	if s.cancel != nil {
		s.cancel()
	}

	// Stop services gracefully
	s.stopServices()

	// Clean up temp directory
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
		s.t.Logf("Cleaned up temp directory: %s", s.tempDir)
	}

	s.t.Logf("System test environment torn down")
}

// createTestData generates test CSV data for the streamer
func (s *SystemTestSuite) createTestData() {
	s.testDataFile = filepath.Join(s.tempDir, "test_telemetry.csv")

	// Create test data matching the real DCGM format
	testData := `timestamp,metric_name,gpu_id,device,uuid,modelName,Hostname,container,pod,namespace,value,labels_raw
"2025-10-17T19:17:00Z","DCGM_FI_DEV_GPU_UTIL","0","nvidia0","GPU-12345678-1234-1234-1234-123456789abc","NVIDIA H100 80GB HBM3","test-host-001","","","","85","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-12345678-1234-1234-1234-123456789abc"",__name__=""DCGM_FI_DEV_GPU_UTIL"",device=""nvidia0"",gpu=""0"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:01Z","DCGM_FI_DEV_GPU_UTIL","1","nvidia1","GPU-87654321-4321-4321-4321-cba987654321","NVIDIA H100 80GB HBM3","test-host-001","","","","90","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-87654321-4321-4321-4321-cba987654321"",__name__=""DCGM_FI_DEV_GPU_UTIL"",device=""nvidia1"",gpu=""1"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:02Z","DCGM_FI_DEV_GPU_TEMP","0","nvidia0","GPU-12345678-1234-1234-1234-123456789abc","NVIDIA H100 80GB HBM3","test-host-001","","","","75","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-12345678-1234-1234-1234-123456789abc"",__name__=""DCGM_FI_DEV_GPU_TEMP"",device=""nvidia0"",gpu=""0"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:03Z","DCGM_FI_DEV_GPU_TEMP","1","nvidia1","GPU-87654321-4321-4321-4321-cba987654321","NVIDIA H100 80GB HBM3","test-host-001","","","","72","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-87654321-4321-4321-4321-cba987654321"",__name__=""DCGM_FI_DEV_GPU_TEMP"",device=""nvidia1"",gpu=""1"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:04Z","DCGM_FI_DEV_MEM_COPY_UTIL","0","nvidia0","GPU-12345678-1234-1234-1234-123456789abc","NVIDIA H100 80GB HBM3","test-host-001","","","","65","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-12345678-1234-1234-1234-123456789abc"",__name__=""DCGM_FI_DEV_MEM_COPY_UTIL"",device=""nvidia0"",gpu=""0"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:05Z","DCGM_FI_DEV_MEM_COPY_UTIL","1","nvidia1","GPU-87654321-4321-4321-4321-cba987654321","NVIDIA H100 80GB HBM3","test-host-001","","","","78","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-87654321-4321-4321-4321-cba987654321"",__name__=""DCGM_FI_DEV_MEM_COPY_UTIL"",device=""nvidia1"",gpu=""1"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:06Z","DCGM_FI_DEV_POWER_USAGE","0","nvidia0","GPU-12345678-1234-1234-1234-123456789abc","NVIDIA H100 80GB HBM3","test-host-001","","","","250","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-12345678-1234-1234-1234-123456789abc"",__name__=""DCGM_FI_DEV_POWER_USAGE"",device=""nvidia0"",gpu=""0"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:07Z","DCGM_FI_DEV_POWER_USAGE","1","nvidia1","GPU-87654321-4321-4321-4321-cba987654321","NVIDIA H100 80GB HBM3","test-host-001","","","","275","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-87654321-4321-4321-4321-cba987654321"",__name__=""DCGM_FI_DEV_POWER_USAGE"",device=""nvidia1"",gpu=""1"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:08Z","DCGM_FI_DEV_GPU_UTIL","2","nvidia2","GPU-11111111-2222-3333-4444-555555555555","NVIDIA H100 80GB HBM3","test-host-001","","","","45","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-11111111-2222-3333-4444-555555555555"",__name__=""DCGM_FI_DEV_GPU_UTIL"",device=""nvidia2"",gpu=""2"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:09Z","DCGM_FI_DEV_GPU_TEMP","2","nvidia2","GPU-11111111-2222-3333-4444-555555555555","NVIDIA H100 80GB HBM3","test-host-001","","","","68","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-11111111-2222-3333-4444-555555555555"",__name__=""DCGM_FI_DEV_GPU_TEMP"",device=""nvidia2"",gpu=""2"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:10Z","DCGM_FI_DEV_MEM_COPY_UTIL","2","nvidia2","GPU-11111111-2222-3333-4444-555555555555","NVIDIA H100 80GB HBM3","test-host-001","","","","32","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-11111111-2222-3333-4444-555555555555"",__name__=""DCGM_FI_DEV_MEM_COPY_UTIL"",device=""nvidia2"",gpu=""2"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""
"2025-10-17T19:17:11Z","DCGM_FI_DEV_POWER_USAGE","2","nvidia2","GPU-11111111-2222-3333-4444-555555555555","NVIDIA H100 80GB HBM3","test-host-001","","","","180","DCGM_FI_DRIVER_VERSION=""535.129.03"",Hostname=""test-host-001"",UUID=""GPU-11111111-2222-3333-4444-555555555555"",__name__=""DCGM_FI_DEV_POWER_USAGE"",device=""nvidia2"",gpu=""2"",instance=""test-host-001:9400"",job=""dgx_dcgm_exporter"",modelName=""NVIDIA H100 80GB HBM3"""`

	err := os.WriteFile(s.testDataFile, []byte(testData), 0644)
	if err != nil {
		s.t.Fatalf("Failed to create test data file: %v", err)
	}

	s.t.Logf("Created test data file: %s", s.testDataFile)
}

// buildBinaries builds all required binaries for system testing
func (s *SystemTestSuite) buildBinaries() {
	s.t.Logf("Building binaries for system tests...")

	binaries := []string{"telemetry-collector", "telemetry-streamer", "api-gateway", "mq-service"}

	// Project root is parent directory of tests
	projectRoot := filepath.Join("..", "")

	for _, binary := range binaries {
		cmd := exec.Command("go", "build", "-o", filepath.Join(s.tempDir, binary), fmt.Sprintf("./cmd/%s", binary))
		cmd.Dir = projectRoot

		output, err := cmd.CombinedOutput()
		if err != nil {
			s.t.Fatalf("Failed to build %s: %v\nOutput: %s\nWorking dir: %s", binary, err, output, cmd.Dir)
		}
		s.t.Logf("Built %s successfully", binary)
	}
}

// startMQService starts the MQ service
func (s *SystemTestSuite) startMQService() {
	s.t.Logf("Starting MQ service on port %s", s.mqServicePort)

	mqServiceBinary := filepath.Join(s.tempDir, "mq-service")
	mqDataDir := filepath.Join(s.tempDir, "mq_data")

	os.MkdirAll(mqDataDir, 0755)

	s.mqServiceCmd = exec.CommandContext(s.ctx, mqServiceBinary,
		fmt.Sprintf("--http-port=%s", s.mqServicePort),
		fmt.Sprintf("--persistence-dir=%s", mqDataDir),
		"--persistence=true",
	)

	s.mqServiceCmd.Stdout = &logWriter{name: "mq-service", t: s.t}
	s.mqServiceCmd.Stderr = &logWriter{name: "mq-service", t: s.t}

	err := s.mqServiceCmd.Start()
	if err != nil {
		s.t.Fatalf("Failed to start MQ service: %v", err)
	}

	s.t.Logf("MQ service started (PID: %d)", s.mqServiceCmd.Process.Pid)

	// Wait a moment for MQ service to be ready
	time.Sleep(2 * time.Second)
}

// startCollector starts the telemetry collector service
func (s *SystemTestSuite) startCollector() {
	s.t.Logf("Starting telemetry collector on port %s", s.collectorPort)

	collectorBinary := filepath.Join(s.tempDir, "telemetry-collector")
	collectorDataDir := filepath.Join(s.tempDir, "collector_data")
	checkpointDir := filepath.Join(s.tempDir, "collector_data", "checkpoints")

	os.MkdirAll(collectorDataDir, 0755)
	os.MkdirAll(checkpointDir, 0755)

	s.collectorCmd = exec.CommandContext(s.ctx, collectorBinary,
		"--workers=2",
		fmt.Sprintf("--data-dir=%s", collectorDataDir),
		fmt.Sprintf("--checkpoint-dir=%s", checkpointDir),
		fmt.Sprintf("--health-port=%s", s.collectorPort),
		fmt.Sprintf("--mq-url=http://localhost:%s", s.mqServicePort),
		"--mq-topic=telemetry",
		"--max-entries=1000",
		"--checkpoint=true",
	)

	s.collectorCmd.Stdout = &logWriter{name: "collector", t: s.t}
	s.collectorCmd.Stderr = &logWriter{name: "collector", t: s.t}

	err := s.collectorCmd.Start()
	if err != nil {
		s.t.Fatalf("Failed to start collector: %v", err)
	}

	s.t.Logf("Telemetry collector started (PID: %d)", s.collectorCmd.Process.Pid)
}

// startAPIGateway starts the API gateway service
func (s *SystemTestSuite) startAPIGateway() {
	s.t.Logf("Starting API gateway on port %s", s.apiGatewayPort)

	apiGatewayBinary := filepath.Join(s.tempDir, "api-gateway")
	apiDataDir := filepath.Join(s.tempDir, "api_data")

	os.MkdirAll(apiDataDir, 0755)

	s.apiGatewayCmd = exec.CommandContext(s.ctx, apiGatewayBinary,
		fmt.Sprintf("--port=%s", s.apiGatewayPort),
		fmt.Sprintf("--collector-port=%s", s.collectorPort),
		fmt.Sprintf("--data-dir=%s", apiDataDir),
	)

	// Set collector URL environment variable
	s.apiGatewayCmd.Env = append(os.Environ(),
		fmt.Sprintf("COLLECTOR_URL=http://localhost:%s", s.collectorPort),
	)

	s.apiGatewayCmd.Stdout = &logWriter{name: "api-gateway", t: s.t}
	s.apiGatewayCmd.Stderr = &logWriter{name: "api-gateway", t: s.t}

	err := s.apiGatewayCmd.Start()
	if err != nil {
		s.t.Fatalf("Failed to start API gateway: %v", err)
	}

	s.t.Logf("API gateway started (PID: %d)", s.apiGatewayCmd.Process.Pid)
}

// startStreamer starts the telemetry streamer service
func (s *SystemTestSuite) startStreamer() {
	s.t.Logf("Starting telemetry streamer with rate %s msg/sec", s.streamerRate)

	streamerBinary := filepath.Join(s.tempDir, "telemetry-streamer")

	s.streamerCmd = exec.CommandContext(s.ctx, streamerBinary,
		fmt.Sprintf("--csv-file=%s", s.testDataFile),
		"--workers=1",
		fmt.Sprintf("--rate=%s", s.streamerRate),
		fmt.Sprintf("--broker-url=http://localhost:%s", s.mqServicePort),
		"--topic=telemetry",
	)

	s.streamerCmd.Stdout = &logWriter{name: "streamer", t: s.t}
	s.streamerCmd.Stderr = &logWriter{name: "streamer", t: s.t}

	err := s.streamerCmd.Start()
	if err != nil {
		s.t.Fatalf("Failed to start streamer: %v", err)
	}

	s.t.Logf("Telemetry streamer started (PID: %d)", s.streamerCmd.Process.Pid)
}

// waitForServices waits for all services to be ready
func (s *SystemTestSuite) waitForServices() {
	s.t.Logf("Waiting for services to be ready...")

	// Wait for collector health
	s.waitForService(fmt.Sprintf("http://localhost:%s/health", s.collectorPort), "Collector")

	// Wait for API gateway health
	s.waitForService(fmt.Sprintf("http://localhost:%s/health", s.apiGatewayPort), "API Gateway")

	// Give streamer a moment to start sending data
	time.Sleep(2 * time.Second)

	s.t.Logf("All services are ready")
}

// waitForService waits for a specific service to respond to health checks
func (s *SystemTestSuite) waitForService(url, serviceName string) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			s.t.Fatalf("Timeout waiting for %s to be ready at %s", serviceName, url)
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				s.t.Logf("%s is ready at %s", serviceName, url)
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// stopServices gracefully stops all running services
func (s *SystemTestSuite) stopServices() {
	services := []*exec.Cmd{s.streamerCmd, s.apiGatewayCmd, s.collectorCmd, s.mqServiceCmd}
	names := []string{"streamer", "api-gateway", "collector", "mq-service"}

	for i, cmd := range services {
		if cmd != nil && cmd.Process != nil {
			s.t.Logf("Stopping %s (PID: %d)", names[i], cmd.Process.Pid)

			// Try graceful shutdown first
			cmd.Process.Signal(os.Interrupt)

			// Wait for graceful shutdown with timeout
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case <-done:
				s.t.Logf("%s stopped gracefully", names[i])
			case <-time.After(10 * time.Second):
				s.t.Logf("%s didn't stop gracefully, killing", names[i])
				cmd.Process.Kill()
				<-done
			}
		}
	}
}

// Helper functions for making HTTP requests

// makeAPIRequest makes an HTTP request to the API gateway
func (s *SystemTestSuite) makeAPIRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("http://localhost:%s%s", s.apiGatewayPort, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// makeCollectorRequest makes an HTTP request to the collector service
func (s *SystemTestSuite) makeCollectorRequest(method, path string) (*http.Response, error) {
	url := fmt.Sprintf("http://localhost:%s%s", s.collectorPort, path)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// parseJSONResponse parses JSON response into provided interface
func (s *SystemTestSuite) parseJSONResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, v)
}

// logWriter implements io.Writer to log service output with prefixes
type logWriter struct {
	name string
	t    *testing.T
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	// Split by lines and log each line with service prefix
	lines := strings.Split(strings.TrimSuffix(string(p), "\n"), "\n")
	for _, line := range lines {
		if line != "" {
			lw.t.Logf("[%s] %s", lw.name, line)
		}
	}
	return len(p), nil
}
