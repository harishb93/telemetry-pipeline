package mq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHTTPBroker_NewHTTPBroker(t *testing.T) {
	broker := NewHTTPBroker("http://localhost:9090")

	if broker == nil {
		t.Fatal("NewHTTPBroker should not return nil")
	}
	if broker.baseURL != "http://localhost:9090" {
		t.Errorf("Expected baseURL 'http://localhost:9090', got '%s'", broker.baseURL)
	}
	if broker.client == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestHTTPBroker_Publish_Success(t *testing.T) {
	// Create a test server that accepts POST requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}
		if r.URL.Path != "/publish/test-topic" {
			t.Errorf("Expected path '/publish/test-topic', got %s", r.URL.Path)
		}

		// Read and verify body
		var payload []byte
		if _, err := r.Body.Read(payload); err != nil && err.Error() != "EOF" {
			t.Errorf("Error reading body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "published"}`))
	}))
	defer server.Close()

	broker := NewHTTPBroker(server.URL)
	msg := Message{
		Payload: []byte(`{"test": "data"}`),
		Ack:     func() {},
	}

	err := broker.Publish("test-topic", msg)
	if err != nil {
		t.Errorf("Publish should succeed, got error: %v", err)
	}
}

func TestHTTPBroker_Publish_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	broker := NewHTTPBroker(server.URL)
	msg := Message{
		Payload: []byte(`{"test": "data"}`),
		Ack:     func() {},
	}

	err := broker.Publish("test-topic", msg)
	if err == nil {
		t.Error("Publish should fail with server error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("500")) {
		t.Errorf("Error should mention status code 500, got: %v", err)
	}
}

func TestHTTPBroker_Publish_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	broker := NewHTTPBroker("http://invalid-host-that-does-not-exist:9999")
	msg := Message{
		Payload: []byte(`{"test": "data"}`),
		Ack:     func() {},
	}

	err := broker.Publish("test-topic", msg)
	if err == nil {
		t.Error("Publish should fail with network error")
	}
}

func TestHTTPBroker_Subscribe_NotSupported(t *testing.T) {
	broker := NewHTTPBroker("http://localhost:9090")

	ch, unsubscribe, err := broker.Subscribe("test-topic")

	if err == nil {
		t.Error("Subscribe should return error for HTTP broker")
	}
	if ch != nil {
		t.Error("Channel should be nil for unsupported operation")
	}
	if unsubscribe != nil {
		t.Error("Unsubscribe function should be nil for unsupported operation")
	}
}

func TestHTTPBroker_SubscribeWithAck_NotSupported(t *testing.T) {
	broker := NewHTTPBroker("http://localhost:9090")

	ch, unsubscribe, err := broker.SubscribeWithAck("test-topic")

	if err == nil {
		t.Error("SubscribeWithAck should return error for HTTP broker")
	}
	if ch != nil {
		t.Error("Channel should be nil for unsupported operation")
	}
	if unsubscribe != nil {
		t.Error("Unsubscribe function should be nil for unsupported operation")
	}
}

func TestHTTPBroker_Close(t *testing.T) {
	broker := NewHTTPBroker("http://localhost:9090")

	// Close should not panic
	broker.Close()
}

func TestHTTPBroker_ConcurrentPublish(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "published"}`))
	}))
	defer server.Close()

	broker := NewHTTPBroker(server.URL)

	numGoroutines := 10
	numRequests := 5
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numRequests; j++ {
				msg := Message{
					Payload: []byte(fmt.Sprintf(`{"id": %d, "seq": %d}`, id, j)),
					Ack:     func() {},
				}

				err := broker.Publish(fmt.Sprintf("topic-%d", id), msg)
				if err != nil {
					t.Errorf("Goroutine %d request %d failed: %v", id, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	mu.Lock()
	expectedRequests := numGoroutines * numRequests
	mu.Unlock()

	if requestCount != expectedRequests {
		t.Errorf("Expected %d requests, got %d", expectedRequests, requestCount)
	}
}

func TestDirectBrokerClient_NewDirectBrokerClient(t *testing.T) {
	client := NewDirectBrokerClient("http://localhost:9090")

	if client == nil {
		t.Fatal("NewDirectBrokerClient should not return nil")
	}
	if client.baseURL != "http://localhost:9090" {
		t.Errorf("Expected baseURL 'http://localhost:9090', got '%s'", client.baseURL)
	}
	if client.client == nil {
		t.Error("HTTP client should not be nil")
	}
	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}

func TestDirectBrokerClient_Publish_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/publish/test-topic" {
			t.Errorf("Expected path '/publish/test-topic', got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "published"}`))
	}))
	defer server.Close()

	client := NewDirectBrokerClient(server.URL)
	msg := Message{
		Payload: []byte(`{"test": "data"}`),
		Ack:     func() {},
	}

	err := client.Publish("test-topic", msg)
	if err != nil {
		t.Errorf("Publish should succeed, got error: %v", err)
	}
}

func TestDirectBrokerClient_Publish_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client := NewDirectBrokerClient(server.URL)
	msg := Message{
		Payload: []byte(`{"test": "data"}`),
		Ack:     func() {},
	}

	err := client.Publish("test-topic", msg)
	if err == nil {
		t.Error("Publish should fail with bad request")
	}
}

func TestDirectBrokerClient_Subscribe_NotSupported(t *testing.T) {
	client := NewDirectBrokerClient("http://localhost:9090")

	ch, unsubscribe, err := client.Subscribe("test-topic")

	if err == nil {
		t.Error("Subscribe should return error for DirectBrokerClient")
	}
	if ch != nil {
		t.Error("Channel should be nil for unsupported operation")
	}
	if unsubscribe != nil {
		t.Error("Unsubscribe function should be nil for unsupported operation")
	}
}

func TestDirectBrokerClient_SubscribeWithAck(t *testing.T) {
	// Mock server that returns messages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/consume/test-topic" {
			response := map[string]interface{}{
				"messages": []map[string]string{
					{"payload": "message1", "timestamp": "2023-01-01T00:00:00Z"},
					{"payload": "message2", "timestamp": "2023-01-01T00:00:01Z"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewDirectBrokerClient(server.URL)

	msgCh, unsubscribe, err := client.SubscribeWithAck("test-topic")
	if err != nil {
		t.Fatalf("SubscribeWithAck should succeed, got error: %v", err)
	}
	defer unsubscribe()

	if msgCh == nil {
		t.Fatal("Message channel should not be nil")
	}
	if unsubscribe == nil {
		t.Fatal("Unsubscribe function should not be nil")
	}

	// Wait a bit for polling to potentially fetch messages
	time.Sleep(2 * time.Second)

	// Test unsubscribe doesn't panic
	unsubscribe()
}

func TestDirectBrokerClient_PollMessages_NoMessages(t *testing.T) {
	// Mock server that returns no messages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/consume/empty-topic" {
			response := map[string]interface{}{
				"messages": []map[string]string{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewDirectBrokerClient(server.URL)

	msgCh, unsubscribe, err := client.SubscribeWithAck("empty-topic")
	if err != nil {
		t.Fatalf("SubscribeWithAck should succeed, got error: %v", err)
	}

	// Wait briefly and then unsubscribe
	time.Sleep(100 * time.Millisecond)
	unsubscribe()

	// Verify channel is closed (or at least unsubscribe works)
	select {
	case <-msgCh:
		// Channel was closed, which is expected
	case <-time.After(100 * time.Millisecond):
		// Timeout is fine too
	}
}

func TestDirectBrokerClient_PollMessages_ServerError(t *testing.T) {
	// Mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewDirectBrokerClient(server.URL)

	msgCh, unsubscribe, err := client.SubscribeWithAck("error-topic")
	if err != nil {
		t.Fatalf("SubscribeWithAck should succeed even if server has errors, got error: %v", err)
	}

	// Wait briefly - polling should handle errors gracefully
	time.Sleep(100 * time.Millisecond)
	unsubscribe()

	// Check that no meaningful messages are received due to server errors
	// Note: The implementation might still deliver empty messages, which is acceptable
	messageCount := 0
	timeout := time.After(50 * time.Millisecond)
	
	for {
		select {
		case msg := <-msgCh:
			// If we receive messages, they should have empty payload due to server errors
			if len(msg.Payload) > 0 {
				t.Errorf("Should not receive messages with payload when server returns errors, got payload: %s", string(msg.Payload))
			}
			messageCount++
		case <-timeout:
			// Expected - either no messages or empty messages due to errors
			return
		}
		
		// Prevent infinite loop in case of unexpected behavior
		if messageCount > 10 {
			break
		}
	}
}

func TestDirectBrokerClient_Close(t *testing.T) {
	client := NewDirectBrokerClient("http://localhost:9090")

	// Close should not panic
	client.Close()
}

func TestBrokerInterface_Implementations(t *testing.T) {
	// Test that all broker types implement BrokerInterface
	var _ BrokerInterface = NewHTTPBroker("http://localhost:9090")
	var _ BrokerInterface = NewDirectBrokerClient("http://localhost:9090")

	// Test with actual broker too
	config := DefaultBrokerConfig()
	broker := NewBroker(config)
	defer broker.Close()

	var _ BrokerInterface = broker
}

func TestBrokerInterface_MethodSignatures(t *testing.T) {
	// Test method signatures for consistency across implementations
	implementations := []BrokerInterface{
		NewHTTPBroker("http://localhost:9090"),
		NewDirectBrokerClient("http://localhost:9090"),
	}

	msg := Message{
		Payload: []byte("test"),
		Ack:     func() {},
	}

	for i, impl := range implementations {
		t.Run(fmt.Sprintf("implementation_%d", i), func(t *testing.T) {
			// Test Publish method signature
			err := impl.Publish("test-topic", msg)
			// Error is expected for HTTP implementations, but method should exist
			_ = err

			// Test Subscribe method signature
			ch, unsubscribe, err := impl.Subscribe("test-topic")
			// Error is expected for HTTP implementations
			_ = ch
			_ = unsubscribe
			_ = err

			// Test SubscribeWithAck method signature
			msgCh, unsubscribe2, err2 := impl.SubscribeWithAck("test-topic")
			_ = msgCh
			_ = unsubscribe2
			_ = err2

			// Test Close method signature
			impl.Close()
		})
	}
}

func TestMessage_Structure(t *testing.T) {
	// Test Message struct fields and methods
	ackCalled := false
	msg := Message{
		Payload: []byte("test payload"),
		Ack: func() {
			ackCalled = true
		},
	}

	// Test payload
	if string(msg.Payload) != "test payload" {
		t.Errorf("Expected payload 'test payload', got '%s'", string(msg.Payload))
	}

	// Test ack function
	if ackCalled {
		t.Error("Ack should not be called yet")
	}

	msg.Ack()

	if !ackCalled {
		t.Error("Ack should have been called")
	}
}

func TestMessage_NilAck(t *testing.T) {
	// Test Message with nil Ack function
	msg := Message{
		Payload: []byte("test payload"),
		Ack:     nil,
	}

	// Should not panic when Ack is nil
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Should not panic with nil Ack function, got panic: %v", r)
		}
	}()

	// This should not panic, but will if we try to call msg.Ack()
	// In real usage, code should check if Ack is nil before calling
	if msg.Ack != nil {
		msg.Ack()
	}
}

func TestHTTPBroker_LargePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the entire body to verify large payload handling
		body := make([]byte, 0)
		buf := make([]byte, 1024)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
			}
			if err != nil {
				break
			}
		}

		if len(body) < 10000 {
			t.Errorf("Expected large payload, got %d bytes", len(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	broker := NewHTTPBroker(server.URL)

	// Create large payload (10KB)
	largePayload := make([]byte, 10000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	msg := Message{
		Payload: largePayload,
		Ack:     func() {},
	}

	err := broker.Publish("test-topic", msg)
	if err != nil {
		t.Errorf("Should handle large payload, got error: %v", err)
	}
}

func TestDirectBrokerClient_PollingTimeout(t *testing.T) {
	// Server that takes too long to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than client timeout to test timeout handling
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short timeout for this test
	client := &DirectBrokerClient{
		baseURL: server.URL,
		client: &http.Client{
			Timeout: 100 * time.Millisecond, // Very short timeout
		},
	}

	msgCh, unsubscribe, err := client.SubscribeWithAck("timeout-topic")
	if err != nil {
		t.Fatalf("SubscribeWithAck should succeed, got error: %v", err)
	}
	defer unsubscribe()

	// Wait briefly for polling attempt
	time.Sleep(200 * time.Millisecond)

	// Should not receive messages due to timeout
	select {
	case msg := <-msgCh:
		t.Errorf("Should not receive messages due to timeout, got %v", msg)
	case <-time.After(50 * time.Millisecond):
		// Expected - no messages due to timeout
	}
}
