package proto

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

// Test PublishRequest serialization and deserialization
func TestPublishRequest_SerializationRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		request *PublishRequest
	}{
		{
			name: "Basic publish request",
			request: &PublishRequest{
				Topic:   "test-topic",
				Payload: []byte("test message"),
				Headers: map[string]string{
					"content-type": "application/json",
					"source":       "test-service",
				},
			},
		},
		{
			name: "Empty payload",
			request: &PublishRequest{
				Topic:   "empty-topic",
				Payload: []byte{},
				Headers: map[string]string{},
			},
		},
		{
			name: "Binary payload",
			request: &PublishRequest{
				Topic:   "binary-topic",
				Payload: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
				Headers: map[string]string{
					"encoding": "binary",
				},
			},
		},
		{
			name: "Unicode content",
			request: &PublishRequest{
				Topic:   "unicode-topic",
				Payload: []byte("Hello 世界! 🌍🚀"),
				Headers: map[string]string{
					"language": "multi-language",
					"emoji":    "🚀",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to bytes
			data, err := proto.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal PublishRequest: %v", err)
			}

			// Unmarshal back
			unmarshaled := &PublishRequest{}
			err = proto.Unmarshal(data, unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal PublishRequest: %v", err)
			}

			// Verify fields
			if unmarshaled.Topic != tt.request.Topic {
				t.Errorf("Topic mismatch: expected %s, got %s", tt.request.Topic, unmarshaled.Topic)
			}

			if !bytes.Equal(unmarshaled.Payload, tt.request.Payload) {
				t.Errorf("Payload mismatch: expected %v, got %v", tt.request.Payload, unmarshaled.Payload)
			}

			if len(unmarshaled.Headers) != len(tt.request.Headers) {
				t.Errorf("Headers length mismatch: expected %d, got %d", len(tt.request.Headers), len(unmarshaled.Headers))
			}

			for key, expectedValue := range tt.request.Headers {
				if actualValue, exists := unmarshaled.Headers[key]; !exists || actualValue != expectedValue {
					t.Errorf("Header mismatch for key %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// Test PublishResponse message validation
func TestPublishResponse_Validation(t *testing.T) {
	tests := []struct {
		name     string
		response *PublishResponse
		valid    bool
	}{
		{
			name: "Successful response",
			response: &PublishResponse{
				MessageId: "msg-123456",
				Success:   true,
				Error:     "",
			},
			valid: true,
		},
		{
			name: "Error response",
			response: &PublishResponse{
				MessageId: "",
				Success:   false,
				Error:     "Topic not found",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := proto.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal PublishResponse: %v", err)
			}

			// Test deserialization
			unmarshaled := &PublishResponse{}
			err = proto.Unmarshal(data, unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal PublishResponse: %v", err)
			}

			// Verify fields
			if unmarshaled.MessageId != tt.response.MessageId {
				t.Errorf("MessageId mismatch: expected %s, got %s", tt.response.MessageId, unmarshaled.MessageId)
			}
			if unmarshaled.Success != tt.response.Success {
				t.Errorf("Success mismatch: expected %t, got %t", tt.response.Success, unmarshaled.Success)
			}
			if unmarshaled.Error != tt.response.Error {
				t.Errorf("Error mismatch: expected %s, got %s", tt.response.Error, unmarshaled.Error)
			}
		})
	}
}

// Test SubscribeRequest message
func TestSubscribeRequest_SerializationRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		request *SubscribeRequest
	}{
		{
			name: "Basic subscribe request",
			request: &SubscribeRequest{
				Topic:          "events",
				ConsumerGroup:  "service-a",
				BatchSize:      10,
				TimeoutSeconds: 30,
			},
		},
		{
			name: "High throughput subscription",
			request: &SubscribeRequest{
				Topic:          "high-volume",
				ConsumerGroup:  "processor-pool",
				BatchSize:      1000,
				TimeoutSeconds: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal and unmarshal
			data, err := proto.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal SubscribeRequest: %v", err)
			}

			unmarshaled := &SubscribeRequest{}
			err = proto.Unmarshal(data, unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal SubscribeRequest: %v", err)
			}

			// Verify all fields
			if unmarshaled.Topic != tt.request.Topic {
				t.Errorf("Topic mismatch: expected %s, got %s", tt.request.Topic, unmarshaled.Topic)
			}
			if unmarshaled.ConsumerGroup != tt.request.ConsumerGroup {
				t.Errorf("ConsumerGroup mismatch: expected %s, got %s", tt.request.ConsumerGroup, unmarshaled.ConsumerGroup)
			}
			if unmarshaled.BatchSize != tt.request.BatchSize {
				t.Errorf("BatchSize mismatch: expected %d, got %d", tt.request.BatchSize, unmarshaled.BatchSize)
			}
			if unmarshaled.TimeoutSeconds != tt.request.TimeoutSeconds {
				t.Errorf("TimeoutSeconds mismatch: expected %d, got %d", tt.request.TimeoutSeconds, unmarshaled.TimeoutSeconds)
			}
		})
	}
}

// Test Message serialization with timestamps
func TestMessage_SerializationRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
	}{
		{
			name: "JSON message",
			message: &Message{
				Id:        "msg-001",
				Topic:     "user-events",
				Payload:   []byte(`{"user_id": 123, "action": "login"}`),
				Timestamp: 1640995200, // Fixed timestamp
				Headers: map[string]string{
					"content-type": "application/json",
					"user-agent":   "test-client/1.0",
				},
			},
		},
		{
			name: "Binary message",
			message: &Message{
				Id:        "msg-002",
				Topic:     "image-uploads",
				Payload:   []byte{0x89, 0x50, 0x4E, 0x47}, // PNG header
				Timestamp: 1640991600,                     // Fixed timestamp
				Headers: map[string]string{
					"content-type": "image/png",
					"size":         "1024",
				},
			},
		},
		{
			name: "Empty message",
			message: &Message{
				Id:        "msg-003",
				Topic:     "heartbeat",
				Payload:   []byte{},
				Timestamp: 1640995200,
				Headers:   map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := proto.Marshal(tt.message)
			if err != nil {
				t.Fatalf("Failed to marshal Message: %v", err)
			}

			// Deserialize
			unmarshaled := &Message{}
			err = proto.Unmarshal(data, unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal Message: %v", err)
			}

			// Verify fields
			if unmarshaled.Id != tt.message.Id {
				t.Errorf("ID mismatch: expected %s, got %s", tt.message.Id, unmarshaled.Id)
			}
			if unmarshaled.Topic != tt.message.Topic {
				t.Errorf("Topic mismatch: expected %s, got %s", tt.message.Topic, unmarshaled.Topic)
			}
			if !bytes.Equal(unmarshaled.Payload, tt.message.Payload) {
				t.Errorf("Payload mismatch: expected %v, got %v", tt.message.Payload, unmarshaled.Payload)
			}
			if unmarshaled.Timestamp != tt.message.Timestamp {
				t.Errorf("Timestamp mismatch: expected %d, got %d", tt.message.Timestamp, unmarshaled.Timestamp)
			}

			// Verify headers
			if len(unmarshaled.Headers) != len(tt.message.Headers) {
				t.Errorf("Headers length mismatch: expected %d, got %d", len(tt.message.Headers), len(unmarshaled.Headers))
			}
			for key, expectedValue := range tt.message.Headers {
				if actualValue, exists := unmarshaled.Headers[key]; !exists || actualValue != expectedValue {
					t.Errorf("Header mismatch for key %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// Test StatsResponse with nested TopicStats
func TestStatsResponse_SerializationRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		response *StatsResponse
	}{
		{
			name: "Multiple topics stats",
			response: &StatsResponse{
				Topics: map[string]*TopicStats{
					"user-events": {
						Topic:             "user-events",
						QueueSize:         150,
						SubscriberCount:   3,
						PendingMessages:   25,
						PublishedMessages: 1000,
						ConsumedMessages:  975,
					},
					"system-logs": {
						Topic:             "system-logs",
						QueueSize:         500,
						SubscriberCount:   1,
						PendingMessages:   500,
						PublishedMessages: 2000,
						ConsumedMessages:  1500,
					},
				},
				TotalMessages: 3000,
				Timestamp:     1640995200,
			},
		},
		{
			name: "Empty stats",
			response: &StatsResponse{
				Topics:        map[string]*TopicStats{},
				TotalMessages: 0,
				Timestamp:     1640995200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := proto.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal StatsResponse: %v", err)
			}

			// Deserialize
			unmarshaled := &StatsResponse{}
			err = proto.Unmarshal(data, unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal StatsResponse: %v", err)
			}

			// Verify basic fields
			if unmarshaled.TotalMessages != tt.response.TotalMessages {
				t.Errorf("TotalMessages mismatch: expected %d, got %d", tt.response.TotalMessages, unmarshaled.TotalMessages)
			}
			if unmarshaled.Timestamp != tt.response.Timestamp {
				t.Errorf("Timestamp mismatch: expected %d, got %d", tt.response.Timestamp, unmarshaled.Timestamp)
			}

			// Verify topics map
			if len(unmarshaled.Topics) != len(tt.response.Topics) {
				t.Errorf("Topics length mismatch: expected %d, got %d", len(tt.response.Topics), len(unmarshaled.Topics))
			}

			for topicName, expectedStats := range tt.response.Topics {
				actualStats, exists := unmarshaled.Topics[topicName]
				if !exists {
					t.Errorf("Topic %s not found in unmarshaled response", topicName)
					continue
				}

				if actualStats.Topic != expectedStats.Topic {
					t.Errorf("Topic name mismatch: expected %s, got %s", expectedStats.Topic, actualStats.Topic)
				}
				if actualStats.QueueSize != expectedStats.QueueSize {
					t.Errorf("QueueSize mismatch for %s: expected %d, got %d", topicName, expectedStats.QueueSize, actualStats.QueueSize)
				}
				if actualStats.SubscriberCount != expectedStats.SubscriberCount {
					t.Errorf("SubscriberCount mismatch for %s: expected %d, got %d", topicName, expectedStats.SubscriberCount, actualStats.SubscriberCount)
				}
				if actualStats.PendingMessages != expectedStats.PendingMessages {
					t.Errorf("PendingMessages mismatch for %s: expected %d, got %d", topicName, expectedStats.PendingMessages, actualStats.PendingMessages)
				}
				if actualStats.PublishedMessages != expectedStats.PublishedMessages {
					t.Errorf("PublishedMessages mismatch for %s: expected %d, got %d", topicName, expectedStats.PublishedMessages, actualStats.PublishedMessages)
				}
				if actualStats.ConsumedMessages != expectedStats.ConsumedMessages {
					t.Errorf("ConsumedMessages mismatch for %s: expected %d, got %d", topicName, expectedStats.ConsumedMessages, actualStats.ConsumedMessages)
				}
			}
		})
	}
}

// Test message reset functionality
func TestProtoReset(t *testing.T) {
	req := &PublishRequest{
		Topic:   "test-topic",
		Payload: []byte("test-payload"),
		Headers: map[string]string{"key": "value"},
	}

	// Verify initial state
	if req.Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got %s", req.Topic)
	}

	// Reset message
	req.Reset()

	// Verify reset state
	if req.Topic != "" {
		t.Errorf("Expected empty topic after reset, got %s", req.Topic)
	}
	if len(req.Payload) != 0 {
		t.Errorf("Expected empty payload after reset, got %v", req.Payload)
	}
	if len(req.Headers) != 0 {
		t.Errorf("Expected empty headers after reset, got %v", req.Headers)
	}
}

// Test string representation
func TestProtoString(t *testing.T) {
	req := &PublishRequest{
		Topic:   "test-topic",
		Payload: []byte("hello"),
		Headers: map[string]string{"type": "greeting"},
	}

	str := req.String()
	if str == "" {
		t.Error("String() should not return empty string")
	}
}

// Benchmark serialization performance
func BenchmarkPublishRequest_Marshal(b *testing.B) {
	req := &PublishRequest{
		Topic:   "benchmark-topic",
		Payload: make([]byte, 1024), // 1KB payload
		Headers: map[string]string{
			"content-type": "application/octet-stream",
			"timestamp":    "2024-01-01T00:00:00Z",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(req)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

func BenchmarkPublishRequest_Unmarshal(b *testing.B) {
	req := &PublishRequest{
		Topic:   "benchmark-topic",
		Payload: make([]byte, 1024), // 1KB payload
		Headers: map[string]string{
			"content-type": "application/octet-stream",
			"timestamp":    "2024-01-01T00:00:00Z",
		},
	}

	data, err := proto.Marshal(req)
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		unmarshaled := &PublishRequest{}
		err := proto.Unmarshal(data, unmarshaled)
		if err != nil {
			b.Fatalf("Unmarshal failed: %v", err)
		}
	}
}
