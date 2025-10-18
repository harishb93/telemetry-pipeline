package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/harishb93/telemetry-pipeline/internal/logger"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
	"github.com/harishb93/telemetry-pipeline/internal/streamer"
)

func main() {
	// Initialize logger
	log := logger.NewFromEnv().WithComponent("streamer")

	log.Info("Telemetry Streamer starting...")

	// Define CLI flags
	csvPath := flag.String("csv-file", "", "Path to the CSV file containing telemetry data")
	workers := flag.Int("workers", 1, "Number of worker goroutines")
	rate := flag.Float64("rate", 1.0, "Messages per second per worker (fractional values allowed)")
	persistence := flag.Bool("persistence", false, "Enable message persistence")
	persistenceDir := flag.String("persistence-dir", "/tmp/mq-data", "Directory for message persistence")
	brokerURL := flag.String("broker-url", "", "URL of remote MQ broker (if not set, uses local broker)")
	flag.Parse()

	if *csvPath == "" {
		log.Fatal("--csv-file flag is required")
	}

	// Validate inputs
	if *workers <= 0 {
		log.Fatal("--workers must be greater than 0")
	}
	if *rate <= 0 {
		log.Fatal("--rate must be greater than 0")
	}

	log.Info("Configuration loaded",
		"csv_file", *csvPath,
		"workers", *workers,
		"rate", *rate,
		"persistence", *persistence,
		"persistence_dir", *persistenceDir,
		"broker_url", *brokerURL)

	// Initialize the message broker with configuration
	var broker mq.BrokerInterface

	if *brokerURL != "" {
		// Use HTTP broker to connect to remote collector
		log.Info("Connecting to remote broker", "url", *brokerURL)
		broker = mq.NewHTTPBroker(*brokerURL)
	} else {
		// Use local broker (original behavior)
		log.Info("Using local message broker")
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
		log.Fatal("Failed to start streamer", "error", err)
	}

	// Handle graceful shutdown
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Streamer running",
		"workers", *workers,
		"rate_per_worker", *rate,
		"total_rate", float64(*workers)*(*rate))
	log.Info("Press Ctrl+C to stop...")

	<-signalCh
	log.Info("Received shutdown signal, stopping streamer...")

	s.Stop()
	log.Info("Streamer stopped gracefully")
}
