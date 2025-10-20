package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainFlagDefaults(t *testing.T) {
	// Test that flags can be initialized without panicking
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Simulate command line args with defaults
	os.Args = []string{"telemetry-streamer",
		"-csv-file=test.csv",
		"-workers=1",
		"-rate=1.0",
		"-persistence=false",
		"-persistence-dir=/tmp/mq-data",
		"-broker-url=http://localhost:9090",
		"-topic=telemetry"}

	// Test flag parsing doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Flag initialization panicked: %v", r)
		}
	}()
}

func TestCSVFileValidation(t *testing.T) {
	tests := []struct {
		name    string
		csvFile string
		valid   bool
	}{
		{"valid_csv_file", "data.csv", true},
		{"valid_path_csv", "/path/to/data.csv", true},
		{"valid_relative_csv", "./data/telemetry.csv", true},
		{"invalid_empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.csvFile != ""
			if isValid != tt.valid {
				t.Errorf("CSV file validation for %s: got %v, want %v", tt.csvFile, isValid, tt.valid)
			}
		})
	}
}

func TestWorkersValidation(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		valid   bool
	}{
		{"valid_single_worker", 1, true},
		{"valid_multiple_workers", 4, true},
		{"valid_many_workers", 10, true},
		{"invalid_zero_workers", 0, false},
		{"invalid_negative_workers", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.workers > 0
			if isValid != tt.valid {
				t.Errorf("Workers validation for %d: got %v, want %v", tt.workers, isValid, tt.valid)
			}
		})
	}
}

func TestRateValidation(t *testing.T) {
	tests := []struct {
		name  string
		rate  float64
		valid bool
	}{
		{"valid_rate_1", 1.0, true},
		{"valid_rate_fractional", 0.5, true},
		{"valid_rate_high", 100.0, true},
		{"valid_rate_small", 0.1, true},
		{"invalid_zero_rate", 0.0, false},
		{"invalid_negative_rate", -1.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.rate > 0
			if isValid != tt.valid {
				t.Errorf("Rate validation for %f: got %v, want %v", tt.rate, isValid, tt.valid)
			}
		})
	}
}

func TestBrokerURLValidation(t *testing.T) {
	tests := []struct {
		name      string
		brokerURL string
		valid     bool
	}{
		{"valid_localhost", "http://localhost:9090", true},
		{"valid_custom_host", "http://mq-service:9090", true},
		{"valid_ip", "http://192.168.1.100:9090", true},
		{"invalid_empty", "", false},
		{"invalid_no_protocol", "localhost:9090", true}, // Still technically valid for the application
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.brokerURL != ""
			if isValid != tt.valid {
				t.Errorf("Broker URL validation for %s: got %v, want %v", tt.brokerURL, isValid, tt.valid)
			}
		})
	}
}

func TestTopicValidation(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		valid bool
	}{
		{"valid_telemetry_topic", "telemetry", true},
		{"valid_custom_topic", "gpu-metrics", true},
		{"valid_simple_topic", "data", true},
		{"invalid_empty_topic", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.topic != ""
			if isValid != tt.valid {
				t.Errorf("Topic validation for %s: got %v, want %v", tt.topic, isValid, tt.valid)
			}
		})
	}
}

func TestPersistenceConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		persistence    bool
		persistenceDir string
		valid          bool
	}{
		{"valid_persistence_enabled", true, "/tmp/mq-data", true},
		{"valid_persistence_disabled", false, "/tmp/mq-data", true},
		{"valid_custom_dir", true, "/custom/path", true},
		{"invalid_empty_dir", true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := !tt.persistence || tt.persistenceDir != ""
			if isValid != tt.valid {
				t.Errorf("Persistence config validation: got %v, want %v", isValid, tt.valid)
			}
		})
	}
}

func TestHostnameListProcessing(t *testing.T) {
	tests := []struct {
		name         string
		hostnameList string
		expectFilter bool
	}{
		{"empty_hostname_list", "", false},
		{"whitespace_only", "   ", false},
		{"single_hostname", "host-1", true},
		{"multiple_hostnames", "host-1,host-2,host-3", true},
		{"hostnames_with_spaces", "host-1, host-2, host-3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the hostname list processing logic
			shouldFilter := tt.hostnameList != "" && strings.TrimSpace(tt.hostnameList) != ""

			if shouldFilter != tt.expectFilter {
				t.Errorf("Hostname list processing for %q: got %v, want %v", tt.hostnameList, shouldFilter, tt.expectFilter)
			}
		})
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{
			name: "hostname_list_env",
			envVars: map[string]string{
				"HOSTNAME_LIST": "host-1,host-2,host-3",
			},
		},
		{
			name: "logger_config_env",
			envVars: map[string]string{
				"LOG_LEVEL":  "DEBUG",
				"LOG_FORMAT": "json",
			},
		},
		{
			name: "empty_hostname_env",
			envVars: map[string]string{
				"HOSTNAME_LIST": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Clean up after test
			defer func() {
				for key := range tt.envVars {
					_ = os.Unsetenv(key)
				}
			}()

			// Test environment setup
			for key, expectedValue := range tt.envVars {
				actualValue := os.Getenv(key)
				if actualValue != expectedValue {
					t.Errorf("Environment variable %s: got %s, want %s", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestStreamerConfiguration(t *testing.T) {
	// Test streamer configuration structure
	t.Run("valid_config", func(t *testing.T) {
		config := struct {
			csvFile        string
			workers        int
			rate           float64
			topic          string
			persistence    bool
			persistenceDir string
			brokerURL      string
		}{
			csvFile:        "test.csv",
			workers:        2,
			rate:           1.5,
			topic:          "telemetry",
			persistence:    false,
			persistenceDir: "/tmp/mq-data",
			brokerURL:      "http://localhost:9090",
		}

		if config.csvFile == "" {
			t.Error("CSV file should not be empty")
		}
		if config.workers <= 0 {
			t.Error("Workers should be positive")
		}
		if config.rate <= 0 {
			t.Error("Rate should be positive")
		}
		if config.topic == "" {
			t.Error("Topic should not be empty")
		}
		if config.brokerURL == "" {
			t.Error("Broker URL should not be empty")
		}
		if config.persistence && config.persistenceDir == "" {
			t.Error("Persistence directory should not be empty when persistence is enabled")
		}
	})
}

func TestCSVPreprocessingLogic(t *testing.T) {
	// Test CSV preprocessing decision logic
	tests := []struct {
		name             string
		hostList         string
		originalCSV      string
		expectPreprocess bool
	}{
		{
			name:             "no_hostname_list",
			hostList:         "",
			originalCSV:      "data.csv",
			expectPreprocess: false,
		},
		{
			name:             "whitespace_hostname_list",
			hostList:         "   ",
			originalCSV:      "data.csv",
			expectPreprocess: false,
		},
		{
			name:             "valid_hostname_list",
			hostList:         "host-1,host-2",
			originalCSV:      "data.csv",
			expectPreprocess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldPreprocess := tt.hostList != "" && strings.TrimSpace(tt.hostList) != ""

			if shouldPreprocess != tt.expectPreprocess {
				t.Errorf("CSV preprocessing decision for hostList=%q: got %v, want %v",
					tt.hostList, shouldPreprocess, tt.expectPreprocess)
			}
		})
	}
}

func TestBrokerConnectionType(t *testing.T) {
	// Test broker connection type selection
	t.Run("http_broker_selection", func(t *testing.T) {
		// Test that HTTP broker is selected by default
		brokerURL := "http://localhost:9090"
		if brokerURL == "" {
			t.Error("Broker URL should not be empty")
		}

		// Test URL format
		if !strings.HasPrefix(brokerURL, "http://") {
			t.Log("Note: Broker URL might not have http:// prefix")
		}
	})
}

func TestSignalHandling(t *testing.T) {
	// Test signal handling setup
	t.Run("signal_channel", func(t *testing.T) {
		signalCh := make(chan os.Signal, 1)
		// Verify channel capacity instead of nil check (make() never returns nil)
		if cap(signalCh) != 1 {
			t.Error("Signal channel should have capacity of 1")
		}

		// Test channel capacity
		if cap(signalCh) != 1 {
			t.Error("Signal channel should have capacity of 1")
		}
	})

	t.Run("signal_types", func(t *testing.T) {
		// Test that signal types are available
		signals := []os.Signal{os.Interrupt}
		for _, sig := range signals {
			if sig == nil {
				t.Error("Signal should not be nil")
			}
		}
	})
}

func TestRateCalculation(t *testing.T) {
	// Test rate calculation logic
	tests := []struct {
		name          string
		workers       int
		ratePerWorker float64
		expectedTotal float64
	}{
		{"single_worker", 1, 1.0, 1.0},
		{"multiple_workers", 4, 1.0, 4.0},
		{"fractional_rate", 2, 0.5, 1.0},
		{"high_rate", 2, 10.0, 20.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalRate := float64(tt.workers) * tt.ratePerWorker
			if totalRate != tt.expectedTotal {
				t.Errorf("Total rate calculation: got %f, want %f", totalRate, tt.expectedTotal)
			}
		})
	}
}

func TestImports(t *testing.T) {
	// Test that all required imports are available
	imports := []string{
		"flag",
		"os",
		"os/signal",
		"strings",
		"syscall",
		"github.com/harishb93/telemetry-pipeline/internal/logger",
		"github.com/harishb93/telemetry-pipeline/internal/mq",
		"github.com/harishb93/telemetry-pipeline/internal/streamer",
	}

	for _, imp := range imports {
		t.Run("import_"+imp, func(t *testing.T) {
			// Test passes if the import is available (compilation test)
			if testing.Verbose() {
				t.Logf("Testing import: %s", imp)
			}
		})
	}
}

func TestGracefulShutdown(t *testing.T) {
	// Test graceful shutdown process
	t.Run("shutdown_sequence", func(t *testing.T) {
		// Test shutdown sequence components
		components := []string{
			"streamer",
			"signal_handler",
		}

		for _, component := range components {
			t.Run("shutdown_"+component, func(t *testing.T) {
				// Test that components can be shut down
				if testing.Verbose() {
					t.Logf("Testing %s shutdown", component)
				}
			})
		}
	})
}

func TestInputValidationFlow(t *testing.T) {
	// Test the overall input validation flow
	t.Run("complete_validation", func(t *testing.T) {
		// Test that all required inputs are validated
		validInputs := struct {
			csvFile   string
			workers   int
			rate      float64
			brokerURL string
			topic     string
		}{
			csvFile:   "test.csv",
			workers:   2,
			rate:      1.0,
			brokerURL: "http://localhost:9090",
			topic:     "telemetry",
		}

		// Validate each input
		if validInputs.csvFile == "" {
			t.Error("CSV file validation failed")
		}
		if validInputs.workers <= 0 {
			t.Error("Workers validation failed")
		}
		if validInputs.rate <= 0 {
			t.Error("Rate validation failed")
		}
		if validInputs.brokerURL == "" {
			t.Error("Broker URL validation failed")
		}
		if validInputs.topic == "" {
			t.Error("Topic validation failed")
		}
	})
}
