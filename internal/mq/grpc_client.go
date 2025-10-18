package mq

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	pb "github.com/harishb93/telemetry-pipeline/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCBrokerClient is a gRPC client for the MQ service
type GRPCBrokerClient struct {
	conn          *grpc.ClientConn
	client        pb.MQServiceClient
	serverAddr    string
	ctx           context.Context
	cancel        context.CancelFunc
	subscriptions map[string]*grpcSubscription
	mu            sync.RWMutex
}

type grpcSubscription struct {
	topic  string
	msgCh  chan Message
	stream pb.MQService_SubscribeClient
	cancel context.CancelFunc
	stopCh chan struct{}
}

// NewGRPCBrokerClient creates a new gRPC broker client
func NewGRPCBrokerClient(serverAddr string) (*GRPCBrokerClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Connect to gRPC server
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", serverAddr, err)
	}

	client := pb.NewMQServiceClient(conn)

	return &GRPCBrokerClient{
		conn:          conn,
		client:        client,
		serverAddr:    serverAddr,
		ctx:           ctx,
		cancel:        cancel,
		subscriptions: make(map[string]*grpcSubscription),
	}, nil
}

// Publish publishes a message to a topic via gRPC
func (g *GRPCBrokerClient) Publish(topic string, msg Message) error {
	req := &pb.PublishRequest{
		Topic:   topic,
		Payload: msg.Payload,
		Headers: make(map[string]string),
	}

	resp, err := g.client.Publish(g.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to publish message via gRPC: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("publish failed: %s", resp.Error)
	}

	return nil
}

// Subscribe subscribes to a topic (not implemented for gRPC - use SubscribeWithAck)
func (g *GRPCBrokerClient) Subscribe(topic string) (chan []byte, func(), error) {
	return nil, nil, fmt.Errorf("Subscribe not supported in gRPC broker - use SubscribeWithAck")
}

// SubscribeWithAck subscribes to a topic with acknowledgment via gRPC streaming
func (g *GRPCBrokerClient) SubscribeWithAck(topic string) (chan Message, func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Create a unique subscription key for this topic + consumer
	subscriptionKey := fmt.Sprintf("%s-%d", topic, time.Now().UnixNano())

	// Check if already subscribed to this topic (allowing multiple subscriptions)
	if len(g.subscriptions) > 10 { // Limit concurrent subscriptions
		return nil, nil, fmt.Errorf("too many subscriptions active")
	}

	// Create subscription context
	subCtx, subCancel := context.WithCancel(g.ctx)

	// Create subscription request
	req := &pb.SubscribeRequest{
		Topic:          topic,
		ConsumerGroup:  "default",
		BatchSize:      10,
		TimeoutSeconds: 30,
	}

	// Start gRPC stream
	stream, err := g.client.Subscribe(subCtx, req)
	if err != nil {
		subCancel()
		return nil, nil, fmt.Errorf("failed to create gRPC subscription for topic %s: %w", topic, err)
	}

	// Create message channel and subscription
	msgCh := make(chan Message, 100)
	stopCh := make(chan struct{})

	subscription := &grpcSubscription{
		topic:  topic,
		msgCh:  msgCh,
		stream: stream,
		cancel: subCancel,
		stopCh: stopCh,
	}

	g.subscriptions[subscriptionKey] = subscription

	// Start message receiver goroutine
	go g.receiveMessages(subscription)

	// Unsubscribe function
	unsubscribe := func() {
		g.mu.Lock()
		defer g.mu.Unlock()

		if sub, exists := g.subscriptions[subscriptionKey]; exists {
			close(sub.stopCh)
			sub.cancel()
			close(sub.msgCh)
			delete(g.subscriptions, subscriptionKey)
		}
	}

	return msgCh, unsubscribe, nil
}

// receiveMessages handles receiving messages from gRPC stream
func (g *GRPCBrokerClient) receiveMessages(sub *grpcSubscription) {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic in goroutine
			fmt.Printf("Panic in gRPC message receiver for topic %s: %v\n", sub.topic, r)
		}
	}()

	for {
		select {
		case <-sub.stopCh:
			return
		default:
			// Receive message from stream
			pbMsg, err := sub.stream.Recv()
			if err != nil {
				if err == io.EOF {
					// Stream ended normally
					return
				}
				// Stream error - could attempt reconnection here
				fmt.Printf("gRPC stream error for topic %s: %v\n", sub.topic, err)
				return
			}

			// Convert protobuf message to internal message
			msg := Message{
				Payload: pbMsg.Payload,
				Ack:     func() {}, // gRPC acknowledgment is handled automatically
			}

			// Send to message channel
			select {
			case sub.msgCh <- msg:
				// Message sent successfully
			case <-sub.stopCh:
				return
			default:
				// Channel full, skip message (or implement buffering)
				fmt.Printf("Message channel full for topic %s, skipping message\n", sub.topic)
			}
		}
	}
}

// Close closes the gRPC connection and all subscriptions
func (g *GRPCBrokerClient) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Cancel all subscriptions
	for _, sub := range g.subscriptions {
		close(sub.stopCh)
		sub.cancel()
		close(sub.msgCh)
	}
	g.subscriptions = make(map[string]*grpcSubscription)

	// Close connection
	g.cancel()
	if g.conn != nil {
		_ = g.conn.Close()
	}
}

// Health checks the health of the gRPC service
func (g *GRPCBrokerClient) Health() error {
	ctx, cancel := context.WithTimeout(g.ctx, 5*time.Second)
	defer cancel()

	_, err := g.client.Health(ctx, &pb.HealthRequest{})
	return err
}

// GetStats gets statistics from the gRPC service
func (g *GRPCBrokerClient) GetStats() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(g.ctx, 5*time.Second)
	defer cancel()

	resp, err := g.client.GetStats(ctx, &pb.StatsRequest{})
	if err != nil {
		return nil, err
	}

	// Convert protobuf response to map
	stats := map[string]interface{}{
		"total_messages": resp.TotalMessages,
		"timestamp":      resp.Timestamp,
		"topics":         make(map[string]interface{}),
	}

	topics := make(map[string]interface{})
	for topicName, topicStats := range resp.Topics {
		topics[topicName] = map[string]interface{}{
			"queue_size":         topicStats.QueueSize,
			"subscriber_count":   topicStats.SubscriberCount,
			"pending_messages":   topicStats.PendingMessages,
			"published_messages": topicStats.PublishedMessages,
			"consumed_messages":  topicStats.ConsumedMessages,
		}
	}
	stats["topics"] = topics

	return stats, nil
}

// Ensure GRPCBrokerClient implements BrokerInterface
var _ BrokerInterface = (*GRPCBrokerClient)(nil)
