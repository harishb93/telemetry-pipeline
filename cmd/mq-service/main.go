package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/harishb93/telemetry-pipeline/internal/logger"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

// MQService represents the standalone MQ service
type MQService struct {
	broker     *mq.Broker
	httpServer *http.Server
	logger     *logger.Logger
}

// NewMQService creates a new MQ service instance
func NewMQService(broker *mq.Broker, port string, logger *logger.Logger) *MQService {
	service := &MQService{
		broker: broker,
		logger: logger,
	}

	// Create HTTP router
	router := mux.NewRouter()

	// Message publishing endpoints
	router.HandleFunc("/publish/{topic}", service.handlePublish).Methods("POST")
	router.HandleFunc("/subscribe/{topic}", service.handleSubscribe).Methods("GET")

	// Admin and monitoring endpoints
	router.HandleFunc("/health", service.handleHealth).Methods("GET")
	router.HandleFunc("/stats", service.handleStats).Methods("GET")
	router.HandleFunc("/topics", service.handleTopics).Methods("GET")

	// Create HTTP server
	service.httpServer = &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return service
}

// Start starts the MQ service
func (s *MQService) Start() error {
	s.logger.Info("Starting MQ service", "address", s.httpServer.Addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the MQ service gracefully
func (s *MQService) Stop() error {
	s.logger.Info("Stopping MQ service")

	// Close the broker
	s.broker.Close()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// handlePublish handles HTTP POST requests to publish messages
func (s *MQService) handlePublish(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := vars["topic"]

	if topic == "" {
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	// Read message payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

		// Create message
	msg := mq.Message{
		Payload: body,
		Ack:     nil, // No acknowledgment function for published messages
	}
	
	// Generate a unique message ID for logging
	messageID := fmt.Sprintf("%d", time.Now().UnixNano())
	
	// Publish to broker
	if err := s.broker.Publish(topic, msg); err != nil {
		s.logger.Error("Failed to publish message", "topic", topic, "error", err)
		http.Error(w, "Failed to publish message", http.StatusInternalServerError)
		return
	}
	
	s.logger.Debug("Message published", "topic", topic, "message_id", messageID)
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "published",
		"topic":      topic,
		"message_id": messageID,
	})
}

// handleSubscribe handles HTTP requests for subscribing to topics (basic implementation)
func (s *MQService) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := vars["topic"]

	if topic == "" {
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	// For HTTP-based subscription, we could implement Server-Sent Events or WebSockets
	// For now, return a simple response indicating subscription capability
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "HTTP subscription not implemented - use direct broker connection",
		"topic":   topic,
	})
}

// handleHealth handles health check requests
func (s *MQService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "mq-service",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(startTime).String(),
	})
}

// handleStats handles statistics requests
func (s *MQService) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.broker.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleTopics handles requests for listing topics
func (s *MQService) handleTopics(w http.ResponseWriter, r *http.Request) {
	topics := s.broker.GetTopics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"topics": topics,
		"count":  len(topics),
	})
}

var startTime time.Time

func main() {
	startTime = time.Now()

	// Initialize logger
	log := logger.NewFromEnv().WithComponent("mq-service")

	// Command line flags
	var (
		port               = flag.String("port", "9090", "HTTP server port")
		persistenceEnabled = flag.Bool("persistence", true, "Enable message persistence")
		persistenceDir     = flag.String("persistence-dir", "./mq-data", "Directory for message persistence")
		ackTimeout         = flag.Duration("ack-timeout", 30*time.Second, "Message acknowledgment timeout")
		maxRetries         = flag.Int("max-retries", 3, "Maximum message delivery retries")
		adminEnabled       = flag.Bool("admin", true, "Enable admin endpoints")
	)
	flag.Parse()

	log.Info("Starting MQ Service")
	log.Info("Configuration loaded",
		"port", *port,
		"persistence_enabled", *persistenceEnabled,
		"persistence_dir", *persistenceDir,
		"ack_timeout", *ackTimeout,
		"max_retries", *maxRetries,
		"admin_enabled", *adminEnabled)

	// Validate port
	if portNum, err := strconv.Atoi(*port); err != nil || portNum < 1 || portNum > 65535 {
		log.Fatal("Invalid port number", "port", *port)
	}

	// Create broker configuration
	brokerConfig := mq.BrokerConfig{
		PersistenceEnabled: *persistenceEnabled,
		PersistenceDir:     *persistenceDir,
		AckTimeout:         *ackTimeout,
		MaxRetries:         *maxRetries,
	}

	// Create and start MQ broker
	broker := mq.NewBroker(brokerConfig)

	// Create MQ service
	service := NewMQService(broker, *port, log)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start service
	if err := service.Start(); err != nil {
		log.Fatal("Failed to start MQ service", "error", err)
	}

	log.Info("MQ Service started successfully",
		"http_endpoint", "http://localhost:"+*port,
		"health_endpoint", "http://localhost:"+*port+"/health",
		"stats_endpoint", "http://localhost:"+*port+"/stats")
	log.Info("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigCh
	log.Info("Shutdown signal received, stopping MQ service...")

	// Graceful shutdown
	if err := service.Stop(); err != nil {
		log.Error("Error during shutdown", "error", err)
	}

	log.Info("MQ Service stopped successfully")
}
