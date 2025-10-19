package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
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
	brokerURL := flag.String("broker-url", "http://localhost:9090", "URL of MQ service (default: http://localhost:9090)")
	topic := flag.String("topic", "telemetry", "Topic to publish messages to")
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
		"broker_url", *brokerURL,
		"topic", *topic)

	// Initialize the message broker with configuration
	var broker mq.BrokerInterface

	// Always use HTTP broker to connect to MQ service
	log.Info("Connecting to MQ service", "url", *brokerURL)
	broker = mq.NewHTTPBroker(*brokerURL)

	// Check if list of HostNames are provided and pre-process csv file with HostNames
	hostList := os.Getenv("HOSTNAME_LIST")
	finalCSVPath := *csvPath

	if hostList != "" && strings.TrimSpace(hostList) != "" {
		log.Info("HOSTNAME_LIST environment variable found, preprocessing CSV file",
			"hostname_list", hostList,
			"original_csv", *csvPath)

		processedCSVPath, err := streamer.PreProcessCSVByHostNames(*csvPath, hostList)
		if err != nil {
			log.Info("Failed to preprocess CSV file, continuing with original file",
				"error", err,
				"original_csv", *csvPath)
		} else if processedCSVPath != *csvPath {
			log.Info("CSV preprocessing successful, using filtered file",
				"original_csv", *csvPath,
				"processed_csv", processedCSVPath)
			finalCSVPath = processedCSVPath
		} else {
			log.Info("CSV preprocessing completed, continuing with original file",
				"csv", *csvPath)
		}
	} else {
		log.Debug("No HOSTNAME_LIST environment variable found, using original CSV file",
			"csv", *csvPath)
	}

	// Create the streamer with the final CSV path (either original or filtered)
	s := streamer.NewStreamer(finalCSVPath, *workers, *rate, *topic, broker)

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
