package mq

import (
	"testing"
	"time"
)

func TestBrokerPublishSubscribe(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	topic := "test-topic"

	// Create a message
	msg := Message{
		Payload: []byte("test message"),
		Ack:     func() {},
	}

	// Subscribe to the topic
	ch, unsubscribe, err := broker.Subscribe(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Publish a message
	err = broker.Publish(topic, msg)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Receive the message
	select {
	case received := <-ch:
		if string(received) != string(msg.Payload) {
			t.Errorf("Expected %s, got %s", string(msg.Payload), string(received))
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestBrokerMultipleSubscribers(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	topic := "test-topic"
	numSubscribers := 3

	// Create multiple subscribers
	subscribers := make([]chan []byte, numSubscribers)
	unsubscribes := make([]func(), numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		ch, unsubscribe, err := broker.Subscribe(topic)
		if err != nil {
			t.Fatalf("Failed to subscribe %d: %v", i, err)
		}
		subscribers[i] = ch
		unsubscribes[i] = unsubscribe
	}

	// Clean up
	defer func() {
		for _, unsubscribe := range unsubscribes {
			unsubscribe()
		}
	}()

	// Create a message
	msg := Message{
		Payload: []byte("broadcast message"),
		Ack:     func() {},
	}

	// Publish a message
	err := broker.Publish(topic, msg)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// All subscribers should receive the message
	for i, ch := range subscribers {
		select {
		case received := <-ch:
			if string(received) != string(msg.Payload) {
				t.Errorf("Subscriber %d: Expected %s, got %s", i, string(msg.Payload), string(received))
			}
		case <-time.After(1 * time.Second):
			t.Errorf("Subscriber %d: Timeout waiting for message", i)
		}
	}
}

func TestBrokerMultipleTopics(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	topic1 := "topic-1"
	topic2 := "topic-2"

	// Subscribe to both topics
	ch1, unsubscribe1, err := broker.Subscribe(topic1)
	if err != nil {
		t.Fatalf("Failed to subscribe to topic1: %v", err)
	}
	defer unsubscribe1()

	ch2, unsubscribe2, err := broker.Subscribe(topic2)
	if err != nil {
		t.Fatalf("Failed to subscribe to topic2: %v", err)
	}
	defer unsubscribe2()

	// Create messages for each topic
	msg1 := Message{
		Payload: []byte("message for topic 1"),
		Ack:     func() {},
	}
	msg2 := Message{
		Payload: []byte("message for topic 2"),
		Ack:     func() {},
	}

	// Publish to both topics
	err = broker.Publish(topic1, msg1)
	if err != nil {
		t.Fatalf("Failed to publish to topic1: %v", err)
	}

	err = broker.Publish(topic2, msg2)
	if err != nil {
		t.Fatalf("Failed to publish to topic2: %v", err)
	}

	// Check topic1 subscriber receives only topic1 message
	select {
	case received := <-ch1:
		if string(received) != string(msg1.Payload) {
			t.Errorf("Topic1: Expected %s, got %s", string(msg1.Payload), string(received))
		}
	case <-time.After(1 * time.Second):
		t.Error("Topic1: Timeout waiting for message")
	}

	// Check topic2 subscriber receives only topic2 message
	select {
	case received := <-ch2:
		if string(received) != string(msg2.Payload) {
			t.Errorf("Topic2: Expected %s, got %s", string(msg2.Payload), string(received))
		}
	case <-time.After(1 * time.Second):
		t.Error("Topic2: Timeout waiting for message")
	}

	// Ensure no cross-topic message delivery
	select {
	case <-ch1:
		t.Error("Topic1 subscriber received unexpected message")
	case <-ch2:
		t.Error("Topic2 subscriber received unexpected message")
	case <-time.After(100 * time.Millisecond):
		// Expected - no more messages
	}
}

func TestBrokerClose(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)

	topic := "test-topic"

	// Subscribe to the topic
	ch, unsubscribe, err := broker.Subscribe(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Close the broker
	broker.Close()

	// Try to publish after close (should fail)
	msg := Message{
		Payload: []byte("test message"),
		Ack:     func() {},
	}
	err = broker.Publish(topic, msg)
	if err == nil {
		t.Error("Expected error when publishing to closed broker")
	}

	// Subscribe channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Expected channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Channel was not closed")
	}
}

func TestBrokerGetQueueSize(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	topic := "test-topic"

	// Initially, queue should be empty
	if size := broker.GetQueueSize(topic); size != 0 {
		t.Errorf("Expected queue size 0, got %d", size)
	}

	// Publish some messages
	for i := 0; i < 3; i++ {
		msg := Message{
			Payload: []byte("test message"),
			Ack:     func() {},
		}
		err := broker.Publish(topic, msg)
		if err != nil {
			t.Fatalf("Failed to publish message %d: %v", i, err)
		}
	}

	// Queue size should be 3
	if size := broker.GetQueueSize(topic); size != 3 {
		t.Errorf("Expected queue size 3, got %d", size)
	}
}

func TestBrokerGetSubscriberCount(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	topic := "test-topic"

	// Initially, no subscribers
	if count := broker.GetSubscriberCount(topic); count != 0 {
		t.Errorf("Expected subscriber count 0, got %d", count)
	}

	// Add subscribers
	ch1, unsubscribe1, err := broker.Subscribe(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe1()

	ch2, unsubscribe2, err := broker.Subscribe(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe2()

	// Should have 2 subscribers
	if count := broker.GetSubscriberCount(topic); count != 2 {
		t.Errorf("Expected subscriber count 2, got %d", count)
	}

	// Close channels to drain them
	go func() {
		for range ch1 {
		}
	}()
	go func() {
		for range ch2 {
		}
	}()
}

func TestBrokerGetTopics(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	// Initially, no topics
	if topics := broker.GetTopics(); len(topics) != 0 {
		t.Errorf("Expected 0 topics, got %d", len(topics))
	}

	// Subscribe to different topics
	_, unsubscribe1, err := broker.Subscribe("topic-1")
	if err != nil {
		t.Fatalf("Failed to subscribe to topic-1: %v", err)
	}
	defer unsubscribe1()

	_, unsubscribe2, err := broker.Subscribe("topic-2")
	if err != nil {
		t.Fatalf("Failed to subscribe to topic-2: %v", err)
	}
	defer unsubscribe2()

	// Should have 2 topics
	topics := broker.GetTopics()
	if len(topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(topics))
	}

	// Check topic names (order may vary)
	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}

	if !topicMap["topic-1"] {
		t.Error("Expected topic-1 to be present")
	}
	if !topicMap["topic-2"] {
		t.Error("Expected topic-2 to be present")
	}
}

func TestBrokerPersistence(t *testing.T) {
	config := DefaultBrokerConfig()
	config.PersistenceEnabled = true
	config.PersistenceDir = "/tmp/mq-test"

	broker := NewBroker(config)
	defer broker.Close()

	topic := "persistent-topic"
	msg := Message{
		Payload: []byte("persistent message"),
		Ack:     func() {},
	}

	// Publish a message
	err := broker.Publish(topic, msg)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Check if persistence file was created
	// Note: This is a basic test - in production, you'd want to verify file contents
	// For now, we just check that publishing doesn't fail with persistence enabled
}

func TestBrokerConcurrency(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	topic := "concurrent-topic"
	numPublishers := 5
	numSubscribers := 3
	messagesPerPublisher := 10

	// Create subscribers
	subscribers := make([]chan []byte, numSubscribers)
	unsubscribes := make([]func(), numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		ch, unsubscribe, err := broker.Subscribe(topic)
		if err != nil {
			t.Fatalf("Failed to subscribe %d: %v", i, err)
		}
		subscribers[i] = ch
		unsubscribes[i] = unsubscribe

		// Start goroutine to consume messages
		go func(ch chan []byte) {
			for range ch {
				// Just consume messages
			}
		}(ch)
	}

	// Clean up
	defer func() {
		for _, unsubscribe := range unsubscribes {
			unsubscribe()
		}
	}()

	// Create publishers
	done := make(chan bool, numPublishers)

	for p := 0; p < numPublishers; p++ {
		go func(publisherID int) {
			defer func() { done <- true }()

			for m := 0; m < messagesPerPublisher; m++ {
				msg := Message{
					Payload: []byte("concurrent message"),
					Ack:     func() {},
				}

				err := broker.Publish(topic, msg)
				if err != nil {
					t.Errorf("Publisher %d failed to publish message %d: %v", publisherID, m, err)
					return
				}
			}
		}(p)
	}

	// Wait for all publishers to finish
	for i := 0; i < numPublishers; i++ {
		<-done
	}

	// Give some time for messages to be delivered
	time.Sleep(100 * time.Millisecond)

	// Test passed if no race conditions occurred
}

func TestBrokerAcknowledgment(t *testing.T) {
	config := DefaultBrokerConfig()
	config.AckTimeout = 2 * time.Second
	broker := NewBroker(config)
	defer broker.Close()

	topic := "ack-topic"

	// Subscribe with acknowledgment support
	ch, unsubscribe, err := broker.SubscribeWithAck(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe with ack: %v", err)
	}
	defer unsubscribe()

	// Publish a message
	msg := Message{
		Payload: []byte("test ack message"),
		Ack:     func() {}, // Will be overridden by Publish
	}

	err = broker.Publish(topic, msg)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Receive and acknowledge the message
	select {
	case received := <-ch:
		if string(received.Payload) != string(msg.Payload) {
			t.Errorf("Expected %s, got %s", string(msg.Payload), string(received.Payload))
		}

		// Acknowledge the message
		received.Ack()

		// Wait a bit to ensure acknowledgment is processed
		time.Sleep(100 * time.Millisecond)

		// Check that pending messages count is 0 after acknowledgment
		stats := broker.GetStats()
		if pendingCount, exists := stats.Topics[topic]; exists {
			if pendingCount.PendingMessages != 0 {
				t.Errorf("Expected 0 pending messages after ack, got %d", pendingCount.PendingMessages)
			}
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestBrokerRedelivery(t *testing.T) {
	config := DefaultBrokerConfig()
	config.AckTimeout = 500 * time.Millisecond
	config.MaxRetries = 2
	broker := NewBroker(config)
	defer broker.Close()

	topic := "redelivery-topic"

	// Subscribe with acknowledgment support
	ch, unsubscribe, err := broker.SubscribeWithAck(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe with ack: %v", err)
	}
	defer unsubscribe()

	// Publish a message
	msg := Message{
		Payload: []byte("test redelivery message"),
		Ack:     func() {}, // Will be overridden by Publish
	}

	err = broker.Publish(topic, msg)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	messagesReceived := 0
	timeout := time.After(8 * time.Second) // Give enough time for redeliveries (ack timeout processing runs every 5s)
	firstMessage := true

	// Don't acknowledge the first message to trigger redelivery
	for {
		select {
		case received := <-ch:
			messagesReceived++
			if string(received.Payload) != string(msg.Payload) {
				t.Errorf("Expected %s, got %s", string(msg.Payload), string(received.Payload))
			}

			if firstMessage {
				t.Logf("Received original message %d, not acknowledging", messagesReceived)
				firstMessage = false
			} else {
				t.Logf("Received redelivered message %d, acknowledging", messagesReceived)
				received.Ack() // Acknowledge redelivered messages to prevent infinite loop
			}

		case <-timeout:
			// Should have received the message multiple times due to redelivery
			if messagesReceived < 2 {
				t.Errorf("Expected at least 2 message deliveries (original + redelivery), got %d", messagesReceived)
			} else {
				t.Logf("Test passed: received %d message deliveries", messagesReceived)
			}
			return
		}
	}
}

func TestBrokerPendingMessagesNotDuplicatedOnResubscribe(t *testing.T) {
	config := DefaultBrokerConfig()
	config.AckTimeout = time.Hour // Avoid redelivery interference during the test
	broker := NewBroker(config)
	defer broker.Close()

	topic := "dedupe-topic"
	msg := Message{
		Payload: []byte("dedupe"),
		Ack:     func() {},
	}

	if err := broker.Publish(topic, msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	ch1, unsubscribe1, err := broker.SubscribeWithAck(topic)
	if err != nil {
		t.Fatalf("failed to subscribe first consumer: %v", err)
	}

	var firstDelivery Message
	select {
	case firstDelivery = <-ch1:
		if firstDelivery.Ack == nil {
			t.Fatal("expected ack function on first delivery")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first delivery")
	}

	// Keep message pending and unsubscribe the first consumer
	unsubscribe1()

	stats := broker.GetStats()
	if pending := stats.Topics[topic].PendingMessages; pending != 1 {
		t.Fatalf("expected 1 pending message after first unsubscribe, got %d", pending)
	}

	ch2, unsubscribe2, err := broker.SubscribeWithAck(topic)
	if err != nil {
		t.Fatalf("failed to subscribe second consumer: %v", err)
	}
	defer unsubscribe2()

	var secondDelivery Message
	select {
	case secondDelivery = <-ch2:
		if secondDelivery.Ack == nil {
			t.Fatal("expected ack function on second delivery")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for second delivery")
	}

	stats = broker.GetStats()
	if pending := stats.Topics[topic].PendingMessages; pending != 1 {
		t.Fatalf("expected pending messages to remain at 1 after resubscribe, got %d", pending)
	}

	// Acknowledge to clean up state for future tests
	secondDelivery.Ack()

	stats = broker.GetStats()
	if pending := stats.Topics[topic].PendingMessages; pending != 0 {
		t.Fatalf("expected 0 pending messages after ack, got %d", pending)
	}
}

func TestBrokerAdminEndpoint(t *testing.T) {
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	// Subscribe to a topic to create some stats
	topic := "admin-test-topic"
	ch, unsubscribe, err := broker.Subscribe(topic)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Drain messages from channel
	go func() {
		for range ch {
		}
	}()

	// Publish some messages
	for i := 0; i < 3; i++ {
		msg := Message{
			Payload: []byte("admin test message"),
			Ack:     func() {},
		}
		err := broker.Publish(topic, msg)
		if err != nil {
			t.Fatalf("Failed to publish message %d: %v", i, err)
		}
	}

	// Test GetStats method directly
	stats := broker.GetStats()
	if len(stats.Topics) != 1 {
		t.Errorf("Expected 1 topic in stats, got %d", len(stats.Topics))
	}

	topicStats, exists := stats.Topics[topic]
	if !exists {
		t.Error("Topic not found in stats")
	}

	if topicStats.QueueSize != 3 {
		t.Errorf("Expected queue size 3, got %d", topicStats.QueueSize)
	}

	if topicStats.SubscriberCount != 1 {
		t.Errorf("Expected subscriber count 1, got %d", topicStats.SubscriberCount)
	}

	t.Logf("Admin stats test passed: %+v", topicStats)
}
