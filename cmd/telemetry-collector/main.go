package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
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
		workers           = flag.Int("workers", 1, "Number of worker goroutines")
		dataDir           = flag.String("data-dir", "./data", "Directory for file storage")
		maxEntriesPerGPU  = flag.Int("max-entries", 1000, "Maximum entries per GPU in memory storage")
		checkpointEnabled = flag.Bool("checkpoint", true, "Enable checkpoint persistence")
		checkpointDir     = flag.String("checkpoint-dir", "./checkpoints", "Directory for checkpoint files")
		healthPort        = flag.String("health-port", "9090", "Port for health check server")
		mqGrpcPort        = flag.String("mq-grpc-port", "9091", "Port for gRPC server")
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
		"grpc_port", *mqGrpcPort,
		"mq_service_url", *mqServiceURL,
		"mq_topic", *mqTopic)

	// Connect to external MQ service via gRPC
	// Parse the MQ URL to get the gRPC address
	grpcAddr := *mqServiceURL
	// Default to localhost if URL is not provided
	if grpcAddr == "http://localhost:9090" {
		grpcAddr = "localhost:" + *mqGrpcPort
	} else {
		// Remove http:// prefix
		grpcAddr = strings.TrimPrefix(grpcAddr, "http://")
		// Remove any existing port and replace with mqGrpcPort
		if idx := strings.LastIndex(grpcAddr, ":"); idx != -1 {
			grpcAddr = grpcAddr[:idx]
		}
		grpcAddr = grpcAddr + ":" + *mqGrpcPort
	}

	broker, err := mq.NewGRPCBrokerClient(grpcAddr)
	if err != nil {
		log.Fatal("Failed to connect to MQ service via gRPC", "address", grpcAddr, "error", err)
	}

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
	broker.Close()

	log.Info("Collector stopped successfully")
}
