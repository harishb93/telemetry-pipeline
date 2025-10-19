package streamer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/logger"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// TelemetryData represents a flexible telemetry data point
type TelemetryData struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// PreProcessCSVByHostNames filters the CSV file by the provided hostnames and creates a new filtered CSV file
func PreProcessCSVByHostNames(csvPath, hostList string) (string, error) {
	log := logger.NewFromEnv().WithComponent("preprocessor")

	if hostList == "" {
		return csvPath, nil // No filtering needed
	}

	// Parse comma-separated host list
	hostnames := strings.Split(strings.TrimSpace(hostList), ",")
	for i, hostname := range hostnames {
		hostnames[i] = strings.TrimSpace(hostname)
	}

	log.Info("Starting CSV preprocessing",
		"source_file", csvPath,
		"hostnames", hostnames,
		"hostname_count", len(hostnames))

	// Open source CSV file
	sourceFile, err := os.Open(csvPath)
	if err != nil {
		return csvPath, fmt.Errorf("failed to open source CSV file: %w", err)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Warn("Failed to close source file", "error", err)
		}
	}()

	// Create CSV reader
	reader := csv.NewReader(sourceFile)

	// Read headers
	headers, err := reader.Read()
	if err != nil {
		return csvPath, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Find hostname column index
	hostnameIndex := -1
	for i, header := range headers {
		if strings.ToLower(strings.TrimSpace(header)) == "hostname" {
			hostnameIndex = i
			break
		}
	}

	// If hostname column not found, return original file path
	if hostnameIndex == -1 {
		log.Debug("Hostname column not found in CSV headers", "headers", headers)
		return csvPath, nil
	}

	// Create temporary filtered file
	tempFile, err := os.CreateTemp("", "filtered_telemetry_*.csv")
	if err != nil {
		return csvPath, fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()

	defer func() {
		if err := tempFile.Close(); err != nil {
			log.Warn("Failed to close temp file", "error", err)
		}
	}()

	// Create CSV writer
	writer := csv.NewWriter(tempFile)
	defer writer.Flush()

	// Write headers to filtered file
	if err := writer.Write(headers); err != nil {
		os.Remove(tempFilePath) // Clean up on error
		return csvPath, fmt.Errorf("failed to write headers to filtered file: %w", err)
	}

	// Create hostname lookup map for efficient filtering
	hostnameMap := make(map[string]bool)
	for _, hostname := range hostnames {
		if hostname != "" {
			hostnameMap[hostname] = true
		}
	}

	// Filter and write matching records
	recordsRead := 0
	recordsWritten := 0

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Warn("Error reading CSV record", "error", err, "records_read", recordsRead)
			continue
		}

		recordsRead++

		// Check if record has enough columns
		if len(record) <= hostnameIndex {
			log.Warn("Record has insufficient columns", "record_index", recordsRead, "columns", len(record), "expected_hostname_index", hostnameIndex)
			continue
		}

		// Check if hostname matches any in the filter list
		recordHostname := strings.TrimSpace(record[hostnameIndex])
		if hostnameMap[recordHostname] {
			if err := writer.Write(record); err != nil {
				log.Warn("Failed to write filtered record", "error", err, "record_index", recordsRead)
				continue
			}
			recordsWritten++
		}
	}

	// Flush writer to ensure all data is written
	writer.Flush()
	if err := writer.Error(); err != nil {
		os.Remove(tempFilePath) // Clean up on error
		return csvPath, fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	log.Info("CSV preprocessing completed",
		"source_file", csvPath,
		"filtered_file", tempFilePath,
		"records_read", recordsRead,
		"records_written", recordsWritten,
		"hostname_column_index", hostnameIndex)

	// If no records were written, return original file path
	if recordsWritten == 0 {
		log.Warn("No records matched the hostname filter, proceeding with original file")
		os.Remove(tempFilePath) // Clean up empty file
		return csvPath, nil
	}

	return tempFilePath, nil
}

// Streamer handles streaming CSV data to MQ
type Streamer struct {
	csvPath string
	workers int
	rate    float64
	topic   string
	broker  mq.BrokerInterface
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *logger.Logger
}

// NewStreamer creates a new streamer instance
func NewStreamer(csvPath string, workers int, rate float64, topic string, broker mq.BrokerInterface) *Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Streamer{
		csvPath: csvPath,
		workers: workers,
		rate:    rate,
		topic:   topic,
		broker:  broker,
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger.NewFromEnv().WithComponent("streamer"),
	}
}

// Start begins streaming CSV data to MQ with specified number of workers
func (s *Streamer) Start() error {
	s.logger.Info("Streamer starting",
		"workers", s.workers,
		"rate_per_worker", s.rate,
		"csv_file", s.csvPath)

	// Check if CSV file is accessible
	if _, err := os.Stat(s.csvPath); err != nil {
		s.logger.Error("CSV file not accessible", "file", s.csvPath, "error", err)
		return fmt.Errorf("failed to access CSV file: %w", err)
	}

	// Read CSV headers first
	headers, err := s.readHeaders()
	if err != nil {
		s.logger.Error("Failed to read CSV headers", "error", err)
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	s.logger.Info("CSV headers parsed", "headers", headers, "count", len(headers))

	// Start workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i, headers)
	}

	s.logger.Info("All workers started successfully")
	return nil
}

// Stop gracefully stops the streamer
func (s *Streamer) Stop() {
	s.logger.Info("Streamer stopping...")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("All workers stopped")
}

// readHeaders reads the CSV file headers
func (s *Streamer) readHeaders() ([]string, error) {
	file, err := os.Open(s.csvPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Can't return error from defer in this context
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	return headers, nil
}

// worker runs a single worker goroutine
func (s *Streamer) worker(workerID int, headers []string) {
	defer s.wg.Done()
	workerLogger := s.logger.WithComponent("worker").With("worker_id", workerID)
	workerLogger.Info("Worker started")

	// Calculate rate interval
	var rateInterval time.Duration
	if s.rate > 0 {
		rateInterval = time.Duration(float64(time.Second) / s.rate)
		workerLogger.Debug("Rate limiting configured", "interval", rateInterval)
	}

	recordsProcessed := 0

	for {
		select {
		case <-s.ctx.Done():
			workerLogger.Info("Worker stopping", "records_processed", recordsProcessed)
			return
		default:
			// Open CSV file for this worker's loop iteration
			if err := s.processCSVLoop(workerID, headers, &recordsProcessed, rateInterval, workerLogger); err != nil {
				workerLogger.Error("Error processing CSV", "error", err)
				// Continue to next iteration after a brief pause
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processCSVLoop processes the entire CSV file once
func (s *Streamer) processCSVLoop(_ int, headers []string, recordsProcessed *int, rateInterval time.Duration, workerLogger *logger.Logger) error {
	file, err := os.Open(s.csvPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	reader := csv.NewReader(file)

	// Skip headers
	if _, err := reader.Read(); err != nil {
		return err
	}

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			record, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					workerLogger.Debug("Reached end of CSV, restarting from beginning")
					return nil // Return to restart the loop
				}
				return err
			}

			// Parse record into flexible format
			telemetryData, err := s.parseRecord(headers, record)
			if err != nil {
				workerLogger.Warn("Error parsing record", "error", err, "record", record)
				continue
			}

			// Convert to JSON
			jsonData, err := json.Marshal(telemetryData)
			if err != nil {
				workerLogger.Error("Error marshaling to JSON", "error", err)
				continue
			}

			// Create MQ message
			msg := mq.Message{
				Payload: jsonData,
				Ack:     func() {}, // Will be overridden by broker
			}

			// Publish to MQ
			if err := s.broker.Publish(s.topic, msg); err != nil {
				workerLogger.Error("Error publishing message", "error", err)
			} else {
				*recordsProcessed++
				if *recordsProcessed%100 == 0 {
					workerLogger.Info("Processed records", "count", *recordsProcessed)
				}
			}

			// Rate limiting
			if rateInterval > 0 {
				time.Sleep(rateInterval)
			}
		}
	}
}

// parseRecord converts CSV record to flexible telemetry data format
func (s *Streamer) parseRecord(headers, record []string) (*TelemetryData, error) {
	if len(headers) != len(record) {
		return nil, fmt.Errorf("header count (%d) doesn't match record count (%d)", len(headers), len(record))
	}

	telemetryData := &TelemetryData{
		Timestamp: time.Now(), // Use current processing time as timestamp
		Fields:    make(map[string]interface{}),
	}

	// Parse all CSV fields into the flexible fields map
	for i, header := range headers {
		if header == "" {
			continue // Skip empty headers
		}

		value := record[i]

		// Try to parse as different types for better JSON representation
		if parsedFloat, err := parseFloat(value); err == nil {
			telemetryData.Fields[header] = parsedFloat
		} else if parsedBool, err := parseBool(value); err == nil {
			telemetryData.Fields[header] = parsedBool
		} else {
			// Keep as string
			telemetryData.Fields[header] = value
		}
	}

	return telemetryData, nil
}

// Helper functions for type parsing
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	// Simple float parsing - could be more sophisticated
	var f float64
	n, err := fmt.Sscanf(s, "%f", &f)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("not a float")
	}
	return f, nil
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true", "True", "TRUE", "1", "yes", "Yes", "YES":
		return true, nil
	case "false", "False", "FALSE", "0", "no", "No", "NO":
		return false, nil
	default:
		return false, fmt.Errorf("not a boolean")
	}
}
