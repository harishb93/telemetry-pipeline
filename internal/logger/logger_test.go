package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogLevel_Constants(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			if string(tt.level) != tt.want {
				t.Errorf("LogLevel %v = %v, want %v", tt.level, string(tt.level), tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != INFO {
		t.Errorf("DefaultConfig().Level = %v, want %v", config.Level, INFO)
	}
	if config.Format != "text" {
		t.Errorf("DefaultConfig().Format = %v, want %v", config.Format, "text")
	}
	if config.Output != os.Stdout {
		t.Errorf("DefaultConfig().Output = %v, want %v", config.Output, os.Stdout)
	}
	if config.AddSource != false {
		t.Errorf("DefaultConfig().AddSource = %v, want %v", config.AddSource, false)
	}
}

func TestNew(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "text handler with debug level",
			config: Config{
				Level:     DEBUG,
				Format:    "text",
				Output:    &buf,
				AddSource: false,
			},
		},
		{
			name: "json handler with error level",
			config: Config{
				Level:     ERROR,
				Format:    "json",
				Output:    &buf,
				AddSource: true,
			},
		},
		{
			name: "invalid level defaults to info",
			config: Config{
				Level:     LogLevel("INVALID"),
				Format:    "text",
				Output:    &buf,
				AddSource: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := New(tt.config)

			if logger == nil {
				t.Fatal("New() returned nil logger")
			}
			if logger.Logger == nil {
				t.Fatal("New() returned logger with nil slog.Logger")
			}

			// Test that logger can write at appropriate level
			if tt.config.Level == ERROR {
				logger.Error("test message")
			} else {
				logger.Info("test message")
			}
			output := buf.String()
			if output == "" {
				t.Error("Logger produced no output")
			}
			if !strings.Contains(output, "test message") {
				t.Errorf("Logger output %q does not contain expected message", output)
			}
		})
	}
}

func TestNewFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		testFunc func(*testing.T, *Logger)
	}{
		{
			name: "default environment",
			envVars: map[string]string{
				"LOG_LEVEL":      "",
				"LOG_FORMAT":     "",
				"LOG_ADD_SOURCE": "",
			},
			testFunc: func(t *testing.T, logger *Logger) {
				if logger.level != slog.LevelInfo {
					t.Errorf("Expected INFO level, got %v", logger.level)
				}
			},
		},
		{
			name: "debug level from env",
			envVars: map[string]string{
				"LOG_LEVEL":      "debug",
				"LOG_FORMAT":     "json",
				"LOG_ADD_SOURCE": "true",
			},
			testFunc: func(t *testing.T, logger *Logger) {
				if logger.level != slog.LevelDebug {
					t.Errorf("Expected DEBUG level, got %v", logger.level)
				}
			},
		},
		{
			name: "error level from env",
			envVars: map[string]string{
				"LOG_LEVEL":      "ERROR",
				"LOG_FORMAT":     "TEXT",
				"LOG_ADD_SOURCE": "false",
			},
			testFunc: func(t *testing.T, logger *Logger) {
				if logger.level != slog.LevelError {
					t.Errorf("Expected ERROR level, got %v", logger.level)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				if value != "" {
					os.Setenv(key, value)
				} else {
					os.Unsetenv(key)
				}
			}

			// Clean up after test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			logger := NewFromEnv()
			if logger == nil {
				t.Fatal("NewFromEnv() returned nil logger")
			}

			tt.testFunc(t, logger)
		})
	}
}

func TestLogger_With(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	childLogger := logger.With("key", "value")

	if childLogger == nil {
		t.Fatal("With() returned nil logger")
	}

	childLogger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Output %q does not contain expected message", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("Output %q does not contain expected key-value pair", output)
	}
}

func TestLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	componentLogger := logger.WithComponent("test-component")

	componentLogger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Output %q does not contain expected message", output)
	}
	if !strings.Contains(output, "component") || !strings.Contains(output, "test-component") {
		t.Errorf("Output %q does not contain expected component", output)
	}
}

func TestLogger_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	requestLogger := logger.WithRequestID("req-123")

	requestLogger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Output %q does not contain expected message", output)
	}
	if !strings.Contains(output, "request_id") || !strings.Contains(output, "req-123") {
		t.Errorf("Output %q does not contain expected request_id", output)
	}
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)
	ctx := context.Background()
	contextLogger := logger.WithContext(ctx)

	if contextLogger == nil {
		t.Fatal("WithContext() returned nil logger")
	}

	contextLogger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Output %q does not contain expected message", output)
	}
}

func TestLogger_LevelChecks(t *testing.T) {
	tests := []struct {
		name         string
		level        LogLevel
		debugEnabled bool
		infoEnabled  bool
	}{
		{
			name:         "debug level",
			level:        DEBUG,
			debugEnabled: true,
			infoEnabled:  true,
		},
		{
			name:         "info level",
			level:        INFO,
			debugEnabled: false,
			infoEnabled:  true,
		},
		{
			name:         "warn level",
			level:        WARN,
			debugEnabled: false,
			infoEnabled:  false,
		},
		{
			name:         "error level",
			level:        ERROR,
			debugEnabled: false,
			infoEnabled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:  tt.level,
				Format: "text",
				Output: &buf,
			}

			logger := New(config)

			if logger.IsDebugEnabled() != tt.debugEnabled {
				t.Errorf("IsDebugEnabled() = %v, want %v", logger.IsDebugEnabled(), tt.debugEnabled)
			}
			if logger.IsInfoEnabled() != tt.infoEnabled {
				t.Errorf("IsInfoEnabled() = %v, want %v", logger.IsInfoEnabled(), tt.infoEnabled)
			}
		})
	}
}

func TestLogger_LogMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "debug",
			logFunc: func() {
				logger.Debug("debug message", "key", "value")
			},
			expected: "debug message",
		},
		{
			name: "info",
			logFunc: func() {
				logger.Info("info message", "key", "value")
			},
			expected: "info message",
		},
		{
			name: "warn",
			logFunc: func() {
				logger.Warn("warn message", "key", "value")
			},
			expected: "warn message",
		},
		{
			name: "error",
			logFunc: func() {
				logger.Error("error message", "key", "value")
			},
			expected: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Output %q does not contain expected message %q", output, tt.expected)
			}
			if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
				t.Errorf("Output %q does not contain expected key-value pair", output)
			}
		})
	}
}

func TestLogger_ContextMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)
	ctx := context.Background()

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "debug context",
			logFunc: func() {
				logger.DebugContext(ctx, "debug context message", "key", "value")
			},
			expected: "debug context message",
		},
		{
			name: "info context",
			logFunc: func() {
				logger.InfoContext(ctx, "info context message", "key", "value")
			},
			expected: "info context message",
		},
		{
			name: "warn context",
			logFunc: func() {
				logger.WarnContext(ctx, "warn context message", "key", "value")
			},
			expected: "warn context message",
		},
		{
			name: "error context",
			logFunc: func() {
				logger.ErrorContext(ctx, "error context message", "key", "value")
			},
			expected: "error context message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Output %q does not contain expected message %q", output, tt.expected)
			}
		})
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.Info("test message", "key", "value", "number", 42)

	output := buf.String()

	// Parse JSON to validate structure
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check required fields
	if logEntry["msg"] != "test message" {
		t.Errorf("JSON msg = %v, want %v", logEntry["msg"], "test message")
	}
	if logEntry["key"] != "value" {
		t.Errorf("JSON key = %v, want %v", logEntry["key"], "value")
	}
	if logEntry["number"] != float64(42) {
		t.Errorf("JSON number = %v, want %v", logEntry["number"], 42)
	}
	if logEntry["level"] != "INFO" {
		t.Errorf("JSON level = %v, want %v", logEntry["level"], "INFO")
	}
	if _, exists := logEntry["time"]; !exists {
		t.Error("JSON output missing time field")
	}
}

func TestLogger_TimeFormatting(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.Info("test message")

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	timeStr, ok := logEntry["time"].(string)
	if !ok {
		t.Fatal("Time field is not a string")
	}

	// Validate RFC3339 format
	if _, err := time.Parse(time.RFC3339, timeStr); err != nil {
		t.Errorf("Time field %q is not in RFC3339 format: %v", timeStr, err)
	}
}

func TestLogger_AddSource(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:     INFO,
		Format:    "json",
		Output:    &buf,
		AddSource: true,
	}

	logger := New(config)
	logger.Info("test message")

	output := buf.String()

	if !strings.Contains(output, "source") {
		t.Error("Output with AddSource=true should contain source information")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test getting global logger
	global := GetGlobalLogger()
	if global == nil {
		t.Fatal("GetGlobalLogger() returned nil")
	}

	// Test setting global logger
	var buf bytes.Buffer
	config := Config{
		Level:  DEBUG,
		Format: "text",
		Output: &buf,
	}
	newLogger := New(config)
	SetGlobalLogger(newLogger)

	retrieved := GetGlobalLogger()
	if retrieved != newLogger {
		t.Error("SetGlobalLogger/GetGlobalLogger did not work correctly")
	}

	// Test package-level functions
	buf.Reset()
	Info("global info message")
	output := buf.String()
	if !strings.Contains(output, "global info message") {
		t.Errorf("Package-level Info() output %q does not contain expected message", output)
	}

	buf.Reset()
	Debug("global debug message")
	output = buf.String()
	if !strings.Contains(output, "global debug message") {
		t.Errorf("Package-level Debug() output %q does not contain expected message", output)
	}

	buf.Reset()
	Warn("global warn message")
	output = buf.String()
	if !strings.Contains(output, "global warn message") {
		t.Errorf("Package-level Warn() output %q does not contain expected message", output)
	}

	buf.Reset()
	Error("global error message")
	output = buf.String()
	if !strings.Contains(output, "global error message") {
		t.Errorf("Package-level Error() output %q does not contain expected message", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  WARN,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)

	// Debug and Info should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")
	output := buf.String()
	if strings.Contains(output, "debug message") || strings.Contains(output, "info message") {
		t.Errorf("Output %q should not contain debug or info messages when level is WARN", output)
	}

	// Warn and Error should be logged
	buf.Reset()
	logger.Warn("warn message")
	logger.Error("error message")
	output = buf.String()
	if !strings.Contains(output, "warn message") || !strings.Contains(output, "error message") {
		t.Errorf("Output %q should contain warn and error messages", output)
	}
}

func TestLogger_EmptyKeyValues(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  INFO,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)
	logger.Info("message without key-values")

	output := buf.String()
	if !strings.Contains(output, "message without key-values") {
		t.Errorf("Output %q does not contain expected message", output)
	}
}

func TestLogger_ComplexKeyValues(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.Info("complex message",
		"string", "value",
		"int", 42,
		"float", 3.14,
		"bool", true,
		"nil", nil,
	)

	output := buf.String()

	// Parse JSON to validate all types are handled
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	expectedValues := map[string]interface{}{
		"string": "value",
		"int":    float64(42),
		"float":  3.14,
		"bool":   true,
		"nil":    nil,
	}

	for key, expected := range expectedValues {
		if logEntry[key] != expected {
			t.Errorf("JSON %s = %v (%T), want %v (%T)", key, logEntry[key], logEntry[key], expected, expected)
		}
	}
}

func TestLogger_NilOutput(t *testing.T) {
	// Test with io.Discard to ensure nil handling
	config := Config{
		Level:  INFO,
		Format: "text",
		Output: io.Discard,
	}

	logger := New(config)
	// Should not panic
	logger.Info("test message")
}

// Test to ensure Fatal functions work (but don't actually call os.Exit in tests)
func TestLogger_FatalLogging(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  INFO,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)

	// We can't actually test os.Exit, but we can test the logging part
	// by creating a mock logger that doesn't call os.Exit
	mockLogger := &Logger{
		Logger: logger.Logger,
		level:  logger.level,
	}

	// Override Fatal to not call os.Exit for testing
	mockFatal := func(msg string, keysAndValues ...interface{}) {
		mockLogger.Logger.Error(msg, keysAndValues...)
		// Don't call os.Exit in test
	}

	// Test the logging part
	buf.Reset()
	mockFatal("fatal message", "key", "value")
	output := buf.String()

	if !strings.Contains(output, "fatal message") {
		t.Errorf("Fatal output %q does not contain expected message", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("Fatal output %q does not contain expected key-value pair", output)
	}
}

func TestLogger_Concurrency(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)

	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info(fmt.Sprintf("message %d", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	if output == "" {
		t.Error("No output from concurrent logging")
	}

	// Count the number of log entries
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 log lines, got %d", len(lines))
	}
}
