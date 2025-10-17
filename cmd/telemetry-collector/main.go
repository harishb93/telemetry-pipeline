package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/collector"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

func main() {
	// Command line flags
	var (
		workers            = flag.Int("workers", 4, "Number of worker goroutines")
		dataDir            = flag.String("data-dir", "./data", "Directory for file storage")
		maxEntriesPerGPU   = flag.Int("max-entries", 1000, "Maximum entries per GPU in memory storage")
		checkpointEnabled  = flag.Bool("checkpoint", true, "Enable checkpoint persistence")
		checkpointDir      = flag.String("checkpoint-dir", "./checkpoints", "Directory for checkpoint files")
		healthPort         = flag.String("health-port", "8080", "Port for health check server")
		brokerPort         = flag.String("broker-port", "9090", "MQ broker admin port")
		persistenceEnabled = flag.Bool("persistence", false, "Enable MQ persistence")
		persistenceDir     = flag.String("persistence-dir", "./mq-data", "Directory for MQ persistence")
	)
	flag.Parse()

	log.Printf("Starting Telemetry Collector")
	log.Printf("Workers: %d", *workers)
	log.Printf("Data Directory: %s", *dataDir)
	log.Printf("Max Entries per GPU: %d", *maxEntriesPerGPU)
	log.Printf("Checkpoint Enabled: %t", *checkpointEnabled)
	log.Printf("Health Port: %s", *healthPort)

	// Create broker configuration
	brokerConfig := mq.BrokerConfig{
		PersistenceEnabled: *persistenceEnabled,
		PersistenceDir:     *persistenceDir,
		AckTimeout:         30 * time.Second,
		MaxRetries:         3,
	}

	// Create and start MQ broker
	broker := mq.NewBroker(brokerConfig)

	// Start broker admin server
	go func() {
		if err := broker.StartAdminServer(*brokerPort); err != nil {
			log.Printf("Failed to start broker admin server: %v", err)
		}
	}()

	// Create collector configuration
	collectorConfig := collector.CollectorConfig{
		Workers:           *workers,
		DataDir:           *dataDir,
		MaxEntriesPerGPU:  *maxEntriesPerGPU,
		CheckpointEnabled: *checkpointEnabled,
		CheckpointDir:     *checkpointDir,
		HealthPort:        *healthPort,
	}

	// Create collector
	coll := collector.NewCollector(broker, collectorConfig)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start collector in background
	go func() {
		if err := coll.Start(); err != nil {
			log.Fatalf("Failed to start collector: %v", err)
		}
	}()

	log.Printf("Collector started successfully. Health endpoint: http://localhost:%s/health", *healthPort)
	log.Printf("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigCh
	log.Printf("Shutdown signal received, stopping collector...")

	// Graceful shutdown
	coll.Stop()

	log.Printf("Collector stopped successfully")
}
