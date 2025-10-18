package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

// Config holds the logger configuration
type Config struct {
	Level     LogLevel
	Format    string // "json" or "text"
	Output    io.Writer
	AddSource bool
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:     INFO,
		Format:    "text",
		Output:    os.Stdout,
		AddSource: false,
	}
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	level slog.Level
}

// New creates a new logger with the given configuration
func New(config Config) *Logger {
	var level slog.Level
	switch strings.ToUpper(string(config.Level)) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Format timestamp
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
				}
			}
			return a
		},
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(config.Output, opts)
	} else {
		handler = slog.NewTextHandler(config.Output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		level:  level,
	}
}

// NewFromEnv creates a logger from environment variables
func NewFromEnv() *Logger {
	config := DefaultConfig()

	// Read log level from environment
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		config.Level = LogLevel(strings.ToUpper(levelStr))
	}

	// Read log format from environment
	if formatStr := os.Getenv("LOG_FORMAT"); formatStr != "" {
		config.Format = strings.ToLower(formatStr)
	}

	// Add source if requested
	if os.Getenv("LOG_ADD_SOURCE") == "true" {
		config.AddSource = true
	}

	return New(config)
}

// With adds key-value pairs to all log messages
func (l *Logger) With(keysAndValues ...interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(keysAndValues...),
		level:  l.level,
	}
}

// WithComponent adds a component field to all log messages
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		level:  l.level,
	}
}

// WithRequestID adds a request ID field to all log messages
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With("request_id", requestID),
		level:  l.level,
	}
}

// WithContext creates a logger that uses the given context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.Logger.With(),
		level:  l.level,
	}
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= slog.LevelDebug
}

// IsInfoEnabled returns true if info logging is enabled
func (l *Logger) IsInfoEnabled() bool {
	return l.level <= slog.LevelInfo
}

// Debug logs a debug message with optional key-value pairs
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, keysAndValues...)
}

// Info logs an info message with optional key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}

// Warn logs a warning message with optional key-value pairs
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.Warn(msg, keysAndValues...)
}

// Error logs an error message with optional key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, keysAndValues...)
}

// DebugContext logs a debug message with context
func (l *Logger) DebugContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.Logger.DebugContext(ctx, msg, keysAndValues...)
}

// InfoContext logs an info message with context
func (l *Logger) InfoContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.Logger.InfoContext(ctx, msg, keysAndValues...)
}

// WarnContext logs a warning message with context
func (l *Logger) WarnContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.Logger.WarnContext(ctx, msg, keysAndValues...)
}

// ErrorContext logs an error message with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.Logger.ErrorContext(ctx, msg, keysAndValues...)
}

// Fatal logs a fatal error and calls os.Exit(1)
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, keysAndValues...)
	os.Exit(1)
}

// FatalContext logs a fatal error with context and calls os.Exit(1)
func (l *Logger) FatalContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	l.Logger.ErrorContext(ctx, msg, keysAndValues...)
	os.Exit(1)
}

// Global logger instance
var globalLogger *Logger

func init() {
	globalLogger = NewFromEnv()
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// Package-level convenience functions

// Debug logs a debug message using the global logger
func Debug(msg string, keysAndValues ...interface{}) {
	globalLogger.Debug(msg, keysAndValues...)
}

// Info logs an info message using the global logger
func Info(msg string, keysAndValues ...interface{}) {
	globalLogger.Info(msg, keysAndValues...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, keysAndValues ...interface{}) {
	globalLogger.Warn(msg, keysAndValues...)
}

// Error logs an error message using the global logger
func Error(msg string, keysAndValues ...interface{}) {
	globalLogger.Error(msg, keysAndValues...)
}

// Fatal logs a fatal error and calls os.Exit(1)
func Fatal(msg string, keysAndValues ...interface{}) {
	globalLogger.Fatal(msg, keysAndValues...)
}
