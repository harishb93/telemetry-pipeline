# Message Queue (MQ) Implementation

This is a custom message queue implementation for the telemetry pipeline that supports multiple topics, message acknowledgment, persistence, and horizontal scaling.

## Features

### 1. Core API
- **`Publish(topic string, msg Message) error`**: Publishes a message to a topic
- **`Subscribe(topic string) (chan []byte, unsubscribe func(), error)`**: Subscribes to a topic and returns a channel for receiving message payloads
- **`SubscribeWithAck(topic string) (chan Message, unsubscribe func(), error)`**: Subscribes with acknowledgment support
- **`Close()`**: Closes the broker and all resources

### 2. Multiple Topics
- Support for multiple topics using simple topic strings (e.g., "gpu-telemetry")
- Each topic maintains its own subscribers and message queue

### 3. Persistence
- **In-memory storage** with optional persistence
- **Configurable persistence**: When enabled, messages are written to rotating files under `/data/mq/{topic}`
- **Append-only log format**: Messages are stored as JSON lines

### 4. Message Acknowledgment
- **At-least-once delivery**: Messages are redelivered if not acknowledged within timeout
- **Configurable timeout**: Default 30 seconds
- **Max retries**: Configurable maximum retry attempts (default 3)
- **Acknowledgment function**: `Message{Payload []byte, Ack func()}`

### 5. Concurrency Support
- **Thread-safe**: Safe for up to 10+ streamer/collector instances
- **Proper synchronization**: Uses RWMutex for concurrent access
- **Worker pools**: Supports concurrent publishers and subscribers

### 6. HTTP Admin Endpoint
- **`GET /health`**: Health check endpoint
- **`GET /stats`**: Overall broker statistics
- **`GET /stats/{topic}`**: Topic-specific statistics
- **No Prometheus/Grafana dependency**: Simple JSON responses

## Configuration

```go
type BrokerConfig struct {
    PersistenceEnabled bool          // Enable message persistence
    PersistenceDir     string        // Directory for persistence files
    AckTimeout         time.Duration // Acknowledgment timeout
    MaxRetries         int           // Maximum retry attempts
}

// Default configuration
config := mq.DefaultBrokerConfig()
config.PersistenceEnabled = true
config.PersistenceDir = "/data/mq"
config.AckTimeout = 30 * time.Second
config.MaxRetries = 3
```

## Usage Examples

### Basic Publish/Subscribe

```go
broker := mq.NewBroker(mq.DefaultBrokerConfig())
defer broker.Close()

// Subscribe to a topic
ch, unsubscribe, err := broker.Subscribe("gpu-telemetry")
if err != nil {
    log.Fatal(err)
}
defer unsubscribe()

// Consume messages
go func() {
    for payload := range ch {
        fmt.Printf("Received: %s\n", string(payload))
    }
}()

// Publish a message
msg := mq.Message{
    Payload: []byte("GPU utilization: 85%"),
    Ack:     func() {}, // Will be overridden by Publish
}
broker.Publish("gpu-telemetry", msg)
```

### Acknowledgment-Aware Subscription

```go
// Subscribe with acknowledgment support
ch, unsubscribe, err := broker.SubscribeWithAck("gpu-telemetry")
if err != nil {
    log.Fatal(err)
}
defer unsubscribe()

// Consume messages with acknowledgment
go func() {
    for msg := range ch {
        // Process the message
        fmt.Printf("Processing: %s\n", string(msg.Payload))
        
        // Acknowledge when done
        msg.Ack()
    }
}()
```

### Admin Endpoints

```go
// Start admin server
go func() {
    broker.StartAdminServer("8080")
}()

// Query endpoints:
// curl http://localhost:8080/health
// curl http://localhost:8080/stats
// curl http://localhost:8080/stats/gpu-telemetry
```

## Kubernetes Deployment Considerations

### Service Discovery
The broker supports Kubernetes service discovery patterns:

```go
// Use environment variables for configuration
mqEndpoint := os.Getenv("MQ_SERVICE_ENDPOINT") // e.g., "mq-service.default.svc.cluster.local:8080"
```

### Horizontal Scaling
- **Streamers**: Multiple streamer instances can publish to the same topics
- **Collectors**: Multiple collector instances can subscribe to the same topics
- **Load Distribution**: Messages are distributed to all subscribers (fan-out pattern)

### Resource Management
- **Memory usage**: Configurable buffer sizes for channels
- **Persistence**: Optional file-based persistence for message durability
- **Graceful shutdown**: Proper cleanup of resources on pod termination

## Testing

The implementation includes comprehensive unit tests:

- **Basic publish/subscribe functionality**
- **Multiple subscribers and topics**
- **Message acknowledgment and redelivery**
- **Concurrency safety**
- **Persistence functionality** 
- **Admin endpoint statistics**

Run tests:
```bash
go test ./internal/mq -v
```

## Performance Characteristics

- **Memory usage**: O(messages in queue + subscribers)
- **Throughput**: Optimized for moderate throughput (suitable for telemetry data)
- **Latency**: Low latency message delivery within process boundaries
- **Persistence**: Optional file I/O for durability vs. performance trade-off

## Files

- **`internal/mq/mq.go`**: Main broker implementation
- **`internal/mq/message.go`**: Message and data structures
- **`internal/mq/mq_test.go`**: Comprehensive unit tests
- **`examples/mq_demo.go`**: Usage demonstration