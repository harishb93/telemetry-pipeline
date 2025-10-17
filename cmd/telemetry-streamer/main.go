package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
	"github.com/harishb93/telemetry-pipeline/internal/streamer"
)

func main() {
	log.Println("Telemetry Streamer starting...")

	// Define CLI flags
	csvPath := flag.String("csv-file", "", "Path to the CSV file containing telemetry data")
	workers := flag.Int("workers", 1, "Number of worker goroutines")
	rate := flag.Float64("rate", 1.0, "Messages per second per worker (fractional values allowed)")
	persistence := flag.Bool("persistence", false, "Enable message persistence")
	persistenceDir := flag.String("persistence-dir", "/tmp/mq-data", "Directory for message persistence")
	brokerURL := flag.String("broker-url", "", "URL of remote MQ broker (if not set, uses local broker)")
	flag.Parse()

	if *csvPath == "" {
		log.Fatal("--csv flag is required")
	}

	// Validate inputs
	if *workers <= 0 {
		log.Fatal("--workers must be greater than 0")
	}
	if *rate <= 0 {
		log.Fatal("--rate must be greater than 0")
	}

	// Initialize the message broker with configuration
	var broker mq.BrokerInterface

	if *brokerURL != "" {
		// Use HTTP broker to connect to remote collector
		log.Printf("Connecting to remote broker at: %s", *brokerURL)
		broker = mq.NewHTTPBroker(*brokerURL)
	} else {
		// Use local broker (original behavior)
		log.Println("Using local message broker")
		config := mq.DefaultBrokerConfig()
		config.PersistenceEnabled = *persistence
		config.PersistenceDir = *persistenceDir
		localBroker := mq.NewBroker(config)
		defer localBroker.Close()
		broker = localBroker
	}

	// Create the streamer
	s := streamer.NewStreamer(*csvPath, *workers, *rate, broker)

	// Start the streamer
	if err := s.Start(); err != nil {
		log.Fatalf("Failed to start streamer: %v", err)
	}

	// Handle graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Streamer running with %d workers at %.2f msg/sec per worker", *workers, *rate)
	log.Println("Press Ctrl+C to stop...")

	<-signalCh
	log.Println("Received shutdown signal, stopping streamer...")

	s.Stop()
	log.Println("Streamer stopped gracefully")
}
