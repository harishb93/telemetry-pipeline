package mq

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Broker configuration
type BrokerConfig struct {
	PersistenceEnabled bool
	PersistenceDir     string
	AckTimeout         time.Duration
	MaxRetries         int
}

// DefaultBrokerConfig returns a default configuration
func DefaultBrokerConfig() BrokerConfig {
	return BrokerConfig{
		PersistenceEnabled: false,
		PersistenceDir:     "/data/mq",
		AckTimeout:         30 * time.Second,
		MaxRetries:         3,
	}
}

// PendingMessage represents a message awaiting acknowledgment
type PendingMessage struct {
	Message   Message
	Timestamp time.Time
	Retries   int
	TopicName string
	MessageID string
}

// TopicData holds topic-specific data
type TopicData struct {
	subscribers    map[chan []byte]struct{}
	ackSubscribers map[chan Message]struct{} // Subscribers that support acknowledgment
	messageQueue   []Message
	pendingMsgs    map[string]*PendingMessage // messageID -> PendingMessage
}

// Broker implements the message broker
type Broker struct {
	mu       sync.RWMutex
	topics   map[string]*TopicData
	config   BrokerConfig
	closed   bool
	stopChan chan struct{}
}

// NewBroker creates a new message broker with the given configuration
func NewBroker(config BrokerConfig) *Broker {
	b := &Broker{
		topics:   make(map[string]*TopicData),
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Create persistence directory if needed
	if config.PersistenceEnabled {
		os.MkdirAll(config.PersistenceDir, 0755)
	}

	// Start background goroutine for handling acknowledgment timeouts
	go b.handleAckTimeouts()

	return b
}

// Publish publishes a message to the specified topic
func (b *Broker) Publish(topic string, msg Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("broker is closed")
	}

	// Get or create topic
	topicData, exists := b.topics[topic]
	if !exists {
		topicData = &TopicData{
			subscribers:    make(map[chan []byte]struct{}),
			ackSubscribers: make(map[chan Message]struct{}),
			messageQueue:   make([]Message, 0),
			pendingMsgs:    make(map[string]*PendingMessage),
		}
		b.topics[topic] = topicData
	}

	// Persist message if enabled
	if b.config.PersistenceEnabled {
		if err := b.persistMessage(topic, msg); err != nil {
			return fmt.Errorf("failed to persist message: %w", err)
		}
	}

	// Add to message queue
	topicData.messageQueue = append(topicData.messageQueue, msg)

	// Generate message ID for acknowledgment tracking
	msgID := fmt.Sprintf("%s-%d", topic, time.Now().UnixNano())

	// Create acknowledgment function
	ackFunc := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if td, exists := b.topics[topic]; exists {
			delete(td.pendingMsgs, msgID)
		}
	}

	// Update message with acknowledgment function
	msg.Ack = ackFunc

	// Track pending message for acknowledgment timeout
	topicData.pendingMsgs[msgID] = &PendingMessage{
		Message:   msg,
		Timestamp: time.Now(),
		Retries:   0,
		TopicName: topic,
		MessageID: msgID,
	}

	// Send to regular subscribers (payload only)
	for ch := range topicData.subscribers {
		select {
		case ch <- msg.Payload:
		default:
			// Channel is full, skip this subscriber
		}
	}

	// Send to acknowledgment subscribers (full message with ack function)
	for ch := range topicData.ackSubscribers {
		select {
		case ch <- msg:
		default:
			// Channel is full, skip this subscriber
		}
	}

	return nil
}

// Subscribe subscribes to a topic and returns a channel for receiving messages
func (b *Broker) Subscribe(topic string) (chan []byte, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, nil, fmt.Errorf("broker is closed")
	}

	// Get or create topic
	topicData, exists := b.topics[topic]
	if !exists {
		topicData = &TopicData{
			subscribers:    make(map[chan []byte]struct{}),
			ackSubscribers: make(map[chan Message]struct{}),
			messageQueue:   make([]Message, 0),
			pendingMsgs:    make(map[string]*PendingMessage),
		}
		b.topics[topic] = topicData
	}

	// Create channel for subscriber
	ch := make(chan []byte, 100) // Buffered channel
	topicData.subscribers[ch] = struct{}{}

	// Send any existing messages in the queue
	for _, msg := range topicData.messageQueue {
		select {
		case ch <- msg.Payload:
		default:
			// Channel is full, skip
		}
	}

	// Unsubscribe function
	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if topicData, exists := b.topics[topic]; exists {
			if _, exists := topicData.subscribers[ch]; exists {
				delete(topicData.subscribers, ch)
				close(ch)
			}
		}
	}

	return ch, unsubscribe, nil
}

// SubscribeWithAck subscribes to a topic and returns a channel for receiving messages with acknowledgment support
func (b *Broker) SubscribeWithAck(topic string) (chan Message, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, nil, fmt.Errorf("broker is closed")
	}

	// Get or create topic
	topicData, exists := b.topics[topic]
	if !exists {
		topicData = &TopicData{
			subscribers:    make(map[chan []byte]struct{}),
			ackSubscribers: make(map[chan Message]struct{}),
			messageQueue:   make([]Message, 0),
			pendingMsgs:    make(map[string]*PendingMessage),
		}
		b.topics[topic] = topicData
	}

	// Create channel for subscriber and register it
	ch := make(chan Message, 100) // Buffered channel
	topicData.ackSubscribers[ch] = struct{}{}

	// Send any existing messages in the queue with acknowledgment tracking
	for _, msg := range topicData.messageQueue {
		msgID := fmt.Sprintf("%s-%d", topic, time.Now().UnixNano())

		// Create acknowledgment function
		ackFunc := func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if topicData, exists := b.topics[topic]; exists {
				delete(topicData.pendingMsgs, msgID)
			}
		}

		// Create message with acknowledgment
		msgWithAck := Message{
			Payload: msg.Payload,
			Ack:     ackFunc,
		}

		// Track pending message
		topicData.pendingMsgs[msgID] = &PendingMessage{
			Message:   msgWithAck,
			Timestamp: time.Now(),
			Retries:   0,
			TopicName: topic,
			MessageID: msgID,
		}

		select {
		case ch <- msgWithAck:
		default:
			// Channel is full, skip
		}
	}

	// Unsubscribe function
	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if topicData, exists := b.topics[topic]; exists {
			if _, exists := topicData.ackSubscribers[ch]; exists {
				delete(topicData.ackSubscribers, ch)
				close(ch)
			}
		}
	}

	return ch, unsubscribe, nil
}

// Close closes the broker and all its resources
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	close(b.stopChan)

	// Close all subscriber channels
	for _, topicData := range b.topics {
		for ch := range topicData.subscribers {
			close(ch)
		}
		for ch := range topicData.ackSubscribers {
			close(ch)
		}
		// Clear subscribers maps to prevent double closing
		topicData.subscribers = make(map[chan []byte]struct{})
		topicData.ackSubscribers = make(map[chan Message]struct{})
	}
}

// GetQueueSize returns the number of messages in a topic's queue
func (b *Broker) GetQueueSize(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if topicData, exists := b.topics[topic]; exists {
		return len(topicData.messageQueue)
	}
	return 0
}

// GetSubscriberCount returns the number of subscribers for a topic
func (b *Broker) GetSubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if topicData, exists := b.topics[topic]; exists {
		return len(topicData.subscribers) + len(topicData.ackSubscribers)
	}
	return 0
}

// GetTopics returns all topic names
func (b *Broker) GetTopics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topics := make([]string, 0, len(b.topics))
	for topic := range b.topics {
		topics = append(topics, topic)
	}
	return topics
}

// AdminStats represents broker statistics for the admin endpoint
type AdminStats struct {
	Topics map[string]TopicStats `json:"topics"`
}

// TopicStats represents statistics for a single topic
type TopicStats struct {
	QueueSize       int `json:"queue_size"`
	SubscriberCount int `json:"subscriber_count"`
	PendingMessages int `json:"pending_messages"`
}

// GetStats returns comprehensive broker statistics
func (b *Broker) GetStats() AdminStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := AdminStats{
		Topics: make(map[string]TopicStats),
	}

	for topicName, topicData := range b.topics {
		stats.Topics[topicName] = TopicStats{
			QueueSize:       len(topicData.messageQueue),
			SubscriberCount: len(topicData.subscribers) + len(topicData.ackSubscribers),
			PendingMessages: len(topicData.pendingMsgs),
		}
	}

	return stats
}

// StartAdminServer starts an HTTP server for admin endpoints
func (b *Broker) StartAdminServer(port string) error {
	mux := http.NewServeMux()

	// Stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats := b.GetStats()
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(stats); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		b.mu.RLock()
		closed := b.closed
		b.mu.RUnlock()

		if closed {
			http.Error(w, "Broker is closed", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Topic-specific stats endpoint
	mux.HandleFunc("/stats/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		topicName := r.URL.Path[len("/stats/"):]
		if topicName == "" {
			http.Error(w, "Topic name required", http.StatusBadRequest)
			return
		}

		b.mu.RLock()
		topicData, exists := b.topics[topicName]
		if !exists {
			b.mu.RUnlock()
			http.Error(w, "Topic not found", http.StatusNotFound)
			return
		}

		stats := TopicStats{
			QueueSize:       len(topicData.messageQueue),
			SubscriberCount: len(topicData.subscribers) + len(topicData.ackSubscribers),
			PendingMessages: len(topicData.pendingMsgs),
		}
		b.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	return http.ListenAndServe(":"+port, mux)
}

// persistMessage writes a message to the persistence file for the topic
func (b *Broker) persistMessage(topic string, msg Message) error {
	if !b.config.PersistenceEnabled {
		return nil
	}

	topicDir := filepath.Join(b.config.PersistenceDir, topic)
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(topicDir, "messages.log")
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write message as JSON line
	msgData := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"payload":   msg.Payload,
	}

	jsonData, err := json.Marshal(msgData)
	if err != nil {
		return err
	}

	_, err = file.Write(append(jsonData, '\n'))
	return err
}

// handleAckTimeouts runs in background to handle message acknowledgment timeouts
func (b *Broker) handleAckTimeouts() {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-b.stopChan:
			return
		case <-ticker.C:
			b.processAckTimeouts()
		}
	}
}

// processAckTimeouts checks for messages that haven't been acknowledged and redelivers them
func (b *Broker) processAckTimeouts() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	for _, topicData := range b.topics {
		for msgID, pendingMsg := range topicData.pendingMsgs {
			if now.Sub(pendingMsg.Timestamp) > b.config.AckTimeout {
				if pendingMsg.Retries < b.config.MaxRetries {
					// Redeliver message
					pendingMsg.Retries++
					pendingMsg.Timestamp = now

					// Send to regular subscribers (payload only)
					for ch := range topicData.subscribers {
						select {
						case ch <- pendingMsg.Message.Payload:
						default:
							// Channel is full, skip
						}
					}

					// Send to acknowledgment subscribers (full message with ack function)
					for ch := range topicData.ackSubscribers {
						select {
						case ch <- pendingMsg.Message:
						default:
							// Channel is full, skip
						}
					}
				} else {
					// Max retries exceeded, remove from pending
					delete(topicData.pendingMsgs, msgID)
				}
			}
		}
	}
}
