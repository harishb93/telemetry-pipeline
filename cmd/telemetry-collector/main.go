package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/harishb93/telemetry-pipeline/internal/collector"
	"github.com/harishb93/telemetry-pipeline/internal/logger"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

func main() {
	// Initialize logger
	log := logger.NewFromEnv().WithComponent("collector")

	// Command line flags
	var (
		workers           = flag.Int("workers", 4, "Number of worker goroutines")
		dataDir           = flag.String("data-dir", "./data", "Directory for file storage")
		maxEntriesPerGPU  = flag.Int("max-entries", 1000, "Maximum entries per GPU in memory storage")
		checkpointEnabled = flag.Bool("checkpoint", true, "Enable checkpoint persistence")
		checkpointDir     = flag.String("checkpoint-dir", "./checkpoints", "Directory for checkpoint files")
		healthPort        = flag.String("health-port", "8080", "Port for health check server")
		mqServiceURL      = flag.String("mq-url", "http://localhost:9090", "URL of the MQ service")
		mqTopic           = flag.String("mq-topic", "telemetry", "MQ topic to subscribe to")
	)
	flag.Parse()

	log.Info("Starting Telemetry Collector")
	log.Info("Configuration loaded",
		"workers", *workers,
		"data_dir", *dataDir,
		"max_entries_per_gpu", *maxEntriesPerGPU,
		"checkpoint_enabled", *checkpointEnabled,
		"checkpoint_dir", *checkpointDir,
		"health_port", *healthPort,
		"mq_service_url", *mqServiceURL,
		"mq_topic", *mqTopic)

	// For now, create a local broker instance  
	// TODO: Connect to external MQ service once subscription mechanism is implemented
	brokerConfig := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(brokerConfig)

	// Create collector configuration
	collectorConfig := collector.CollectorConfig{
		Workers:           *workers,
		DataDir:           *dataDir,
		MaxEntriesPerGPU:  *maxEntriesPerGPU,
		CheckpointEnabled: *checkpointEnabled,
		CheckpointDir:     *checkpointDir,
		HealthPort:        *healthPort,
		MQTopic:           *mqTopic,
	}

	// Create collector  
	coll := collector.NewCollector(broker, collectorConfig)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start collector in background
	go func() {
		if err := coll.Start(); err != nil {
			log.Fatal("Failed to start collector", "error", err)
		}
	}()

	log.Info("Collector started successfully",
		"health_endpoint", "http://localhost:"+*healthPort+"/health",
		"mq_service_url", *mqServiceURL)
	log.Info("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigCh
	log.Info("Shutdown signal received, stopping collector...")

	// Graceful shutdown
	coll.Stop()

	log.Info("Collector stopped successfully")
}
