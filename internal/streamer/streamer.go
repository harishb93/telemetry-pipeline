package streamer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// TelemetryData represents a flexible telemetry data point
type TelemetryData struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// Streamer handles streaming CSV data to MQ
type Streamer struct {
	csvPath string
	workers int
	rate    float64
	broker  *mq.Broker
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewStreamer creates a new streamer instance
func NewStreamer(csvPath string, workers int, rate float64, broker *mq.Broker) *Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Streamer{
		csvPath: csvPath,
		workers: workers,
		rate:    rate,
		broker:  broker,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins streaming CSV data to MQ with specified number of workers
func (s *Streamer) Start() error {
	log.Printf("Streamer starting with %d workers at rate %.2f msg/sec per worker", s.workers, s.rate)

	// Read CSV headers first
	headers, err := s.readHeaders()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	log.Printf("CSV headers: %v", headers)

	// Start workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i, headers)
	}

	return nil
}

// Stop gracefully stops the streamer
func (s *Streamer) Stop() {
	log.Println("Streamer stopping...")
	s.cancel()
	s.wg.Wait()
	log.Println("All workers stopped")
}

// readHeaders reads the CSV file headers
func (s *Streamer) readHeaders() ([]string, error) {
	file, err := os.Open(s.csvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
	log.Printf("Worker %d started", workerID)

	// Calculate rate interval
	var rateInterval time.Duration
	if s.rate > 0 {
		rateInterval = time.Duration(float64(time.Second) / s.rate)
	}

	recordsProcessed := 0

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Worker %d stopping after processing %d records", workerID, recordsProcessed)
			return
		default:
			// Open CSV file for this worker's loop iteration
			if err := s.processCSVLoop(workerID, headers, &recordsProcessed, rateInterval); err != nil {
				log.Printf("Worker %d: Error processing CSV: %v", workerID, err)
				// Continue to next iteration after a brief pause
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processCSVLoop processes the entire CSV file once
func (s *Streamer) processCSVLoop(workerID int, headers []string, recordsProcessed *int, rateInterval time.Duration) error {
	file, err := os.Open(s.csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

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
					log.Printf("Worker %d: Reached end of CSV, restarting from beginning", workerID)
					return nil // Return to restart the loop
				}
				return err
			}

			// Parse record into flexible format
			telemetryData, err := s.parseRecord(headers, record)
			if err != nil {
				log.Printf("Worker %d: Error parsing record: %v", workerID, err)
				continue
			}

			// Convert to JSON
			jsonData, err := json.Marshal(telemetryData)
			if err != nil {
				log.Printf("Worker %d: Error marshaling to JSON: %v", workerID, err)
				continue
			}

			// Create MQ message
			msg := mq.Message{
				Payload: jsonData,
				Ack:     func() {}, // Will be overridden by broker
			}

			// Publish to MQ
			if err := s.broker.Publish("telemetry", msg); err != nil {
				log.Printf("Worker %d: Error publishing message: %v", workerID, err)
			} else {
				*recordsProcessed++
				if *recordsProcessed%100 == 0 {
					log.Printf("Worker %d: Processed %d records", workerID, *recordsProcessed)
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
