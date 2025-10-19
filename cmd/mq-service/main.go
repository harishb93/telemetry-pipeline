package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/harishb93/telemetry-pipeline/internal/logger"
	"github.com/harishb93/telemetry-pipeline/internal/mq"
	pb "github.com/harishb93/telemetry-pipeline/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// gRPCMQService implements the gRPC MQ service
type gRPCMQService struct {
	pb.UnimplementedMQServiceServer
	broker *mq.Broker
	logger *logger.Logger
}

// NewgRPCMQService creates a new gRPC MQ service
func NewgRPCMQService(broker *mq.Broker, logger *logger.Logger) *gRPCMQService {
	return &gRPCMQService{
		broker: broker,
		logger: logger,
	}
}

// Publish implements the Publish gRPC method
func (s *gRPCMQService) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	messageID := fmt.Sprintf("%d", time.Now().UnixNano())

	msg := mq.Message{
		Payload: req.Payload,
		Ack:     nil, // No acknowledgment function for published messages
	}

	if err := s.broker.Publish(req.Topic, msg); err != nil {
		s.logger.Error("Failed to publish message", "topic", req.Topic, "error", err)
		return &pb.PublishResponse{
			MessageId: messageID,
			Success:   false,
			Error:     err.Error(),
		}, nil
	}

	s.logger.Debug("Message published via gRPC", "topic", req.Topic, "message_id", messageID)

	return &pb.PublishResponse{
		MessageId: messageID,
		Success:   true,
	}, nil
}

// Subscribe implements the Subscribe gRPC streaming method
func (s *gRPCMQService) Subscribe(req *pb.SubscribeRequest, stream pb.MQService_SubscribeServer) error {
	s.logger.Info("Starting gRPC subscription", "topic", req.Topic, "consumer_group", req.ConsumerGroup)

	// Subscribe to the topic
	msgCh, unsubscribe, err := s.broker.SubscribeWithAck(req.Topic)
	if err != nil {
		s.logger.Error("Failed to subscribe to topic", "topic", req.Topic, "error", err)
		return fmt.Errorf("failed to subscribe to topic %s: %w", req.Topic, err)
	}
	defer unsubscribe()

	// Handle context cancellation
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("gRPC subscription cancelled", "topic", req.Topic)
			return ctx.Err()
		case msg := <-msgCh:
			// Create protobuf message
			pbMsg := &pb.Message{
				Id:        fmt.Sprintf("%d", time.Now().UnixNano()),
				Topic:     req.Topic,
				Payload:   msg.Payload,
				Timestamp: time.Now().Unix(),
				Headers:   make(map[string]string),
			}

			// Send message to client
			if err := stream.Send(pbMsg); err != nil {
				s.logger.Error("Failed to send message to gRPC client", "topic", req.Topic, "error", err)
				return err
			}

			// Acknowledge the message
			if msg.Ack != nil {
				msg.Ack()
			}

			s.logger.Debug("Message sent via gRPC stream", "topic", req.Topic, "message_id", pbMsg.Id)
		}
	}
}

// Health implements the Health gRPC method
func (s *gRPCMQService) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Service:   "mq-service",
		Version:   "1.0.0",
	}, nil
}

// GetStats implements the GetStats gRPC method
func (s *gRPCMQService) GetStats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	stats := s.broker.GetStats()

	pbStats := &pb.StatsResponse{
		Topics:        make(map[string]*pb.TopicStats),
		TotalMessages: 0,
		Timestamp:     time.Now().Unix(),
	}

	for topicName, topicStats := range stats.Topics {
		pbTopicStats := &pb.TopicStats{
			Topic:             topicName,
			QueueSize:         int64(topicStats.QueueSize),
			SubscriberCount:   int32(topicStats.SubscriberCount),
			PendingMessages:   int64(topicStats.PendingMessages),
			PublishedMessages: 0, // Would need to track this in broker
			ConsumedMessages:  0, // Would need to track this in broker
		}
		pbStats.Topics[topicName] = pbTopicStats
		pbStats.TotalMessages += pbTopicStats.QueueSize
	}

	return pbStats, nil
}

// HTTPMQService provides HTTP endpoints (for backward compatibility)
type HTTPMQService struct {
	broker     *mq.Broker
	httpServer *http.Server
	logger     *logger.Logger
}

// NewHTTPMQService creates a new HTTP MQ service
func NewHTTPMQService(broker *mq.Broker, port string, logger *logger.Logger) *HTTPMQService {
	service := &HTTPMQService{
		broker: broker,
		logger: logger,
	}

	router := mux.NewRouter()
	router.HandleFunc("/publish/{topic}", service.handlePublish).Methods("POST", "OPTIONS")
	router.HandleFunc("/health", service.handleHealth).Methods("GET", "OPTIONS")
	router.HandleFunc("/stats", service.handleStats).Methods("GET", "OPTIONS")

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

func (s *HTTPMQService) handlePublish(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	topic := vars["topic"]

	if topic == "" {
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	msg := mq.Message{
		Payload: body,
		Ack:     nil,
	}

	messageID := fmt.Sprintf("%d", time.Now().UnixNano())

	if err := s.broker.Publish(topic, msg); err != nil {
		s.logger.Error("Failed to publish message", "topic", topic, "error", err)
		http.Error(w, "Failed to publish message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":     "published",
		"topic":      topic,
		"message_id": messageID,
	})
}

func (s *HTTPMQService) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "mq-service",
		"timestamp": time.Now().UTC(),
	})
}

func (s *HTTPMQService) handleStats(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	stats := s.broker.GetStats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *HTTPMQService) Start() error {
	s.logger.Info("Starting HTTP MQ service", "address", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()
	return nil
}

func (s *HTTPMQService) Stop() error {
	s.logger.Info("Stopping HTTP MQ service")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func main() {
	// Initialize logger
	log := logger.NewFromEnv().WithComponent("mq-service")

	// Command line flags
	var (
		grpcPort           = flag.String("grpc-port", "9091", "gRPC server port")
		httpPort           = flag.String("http-port", "9090", "HTTP server port")
		persistenceEnabled = flag.Bool("persistence", true, "Enable message persistence")
		persistenceDir     = flag.String("persistence-dir", "./mq-data", "Directory for message persistence")
		ackTimeout         = flag.Duration("ack-timeout", 30*time.Second, "Message acknowledgment timeout")
		maxRetries         = flag.Int("max-retries", 3, "Maximum message delivery retries")
	)
	flag.Parse()

	log.Info("Starting MQ Service")
	log.Info("Configuration loaded",
		"grpc_port", *grpcPort,
		"http_port", *httpPort,
		"persistence_enabled", *persistenceEnabled,
		"persistence_dir", *persistenceDir,
		"ack_timeout", *ackTimeout,
		"max_retries", *maxRetries)

	// Validate ports
	if portNum, err := strconv.Atoi(*grpcPort); err != nil || portNum < 1 || portNum > 65535 {
		log.Fatal("Invalid gRPC port number", "port", *grpcPort)
	}
	if portNum, err := strconv.Atoi(*httpPort); err != nil || portNum < 1 || portNum > 65535 {
		log.Fatal("Invalid HTTP port number", "port", *httpPort)
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

	// Create gRPC server
	grpcServer := grpc.NewServer()
	grpcService := NewgRPCMQService(broker, log)
	pb.RegisterMQServiceServer(grpcServer, grpcService)
	reflection.Register(grpcServer)

	// Create HTTP service (for backward compatibility)
	httpService := NewHTTPMQService(broker, *httpPort, log)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", ":"+*grpcPort)
	if err != nil {
		log.Fatal("Failed to listen on gRPC port", "port", *grpcPort, "error", err)
	}

	go func() {
		log.Info("Starting gRPC server", "port", *grpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	// Start HTTP server
	if err := httpService.Start(); err != nil {
		log.Fatal("Failed to start HTTP service", "error", err)
	}

	log.Info("MQ Service started successfully",
		"grpc_endpoint", "localhost:"+*grpcPort,
		"http_endpoint", "http://localhost:"+*httpPort,
		"health_endpoint", "http://localhost:"+*httpPort+"/health")
	log.Info("Press Ctrl+C to stop...")

	// Wait for shutdown signal
	<-sigCh
	log.Info("Shutdown signal received, stopping MQ service...")

	// Graceful shutdown
	grpcServer.GracefulStop()
	if err := httpService.Stop(); err != nil {
		log.Error("Error during HTTP service shutdown", "error", err)
	}
	broker.Close()

	log.Info("MQ Service stopped successfully")
}
