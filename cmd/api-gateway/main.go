package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/harishb93/telemetry-pipeline/api" // Swagger docs
	"github.com/harishb93/telemetry-pipeline/internal/api"
	"github.com/harishb93/telemetry-pipeline/internal/collector"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// @title Telemetry API Gateway
// @version 1.0
// @description API Gateway for GPU Telemetry Pipeline
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8081
// @BasePath /api/v1

func main() {
	// Command line flags
	var (
		port          = flag.String("port", "8081", "Port for API server")
		collectorPort = flag.String("collector-port", "8080", "Port of the collector health endpoint")
		dataDir       = flag.String("data-dir", "./data", "Directory where telemetry data is stored")
	)
	flag.Parse()

	log.Printf("Starting Telemetry API Gateway")
	log.Printf("API Port: %s", *port)
	log.Printf("Data Directory: %s", *dataDir)

	// Create a minimal collector instance for data access
	// In a real deployment, this would connect to the actual collector service
	brokerConfig := mq.DefaultBrokerConfig()
	broker := mq.NewBroker(brokerConfig)

	collectorConfig := collector.CollectorConfig{
		Workers:           1,
		DataDir:           *dataDir,
		MaxEntriesPerGPU:  1000,
		CheckpointEnabled: false,
		HealthPort:        *collectorPort,
	}

	coll := collector.NewCollector(broker, collectorConfig)

	// Create API server
	serverConfig := api.ServerConfig{
		Port: *port,
	}

	server := api.NewServer(coll, serverConfig)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	log.Printf("API Gateway started successfully on port %s", *port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/", *port)
	log.Printf("Health endpoint: http://localhost:%s/health", *port)
	log.Printf("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigCh
	log.Printf("Shutdown signal received, stopping API Gateway...")

	// Graceful shutdown
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	log.Printf("API Gateway stopped successfully")
}
