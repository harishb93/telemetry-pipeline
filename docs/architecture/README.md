# System Architecture

## Overview

The GPU Telemetry Pipeline is a modern, production-ready system that streams GPU metrics through a custom message broker to persistent collectors. The architecture follows the pattern: **CSV Streamers → Custom MQ → Collectors → Storage**.

### Design Philosophy

- **Simplicity**: No external dependencies for core functionality
- **Scalability**: Horizontally scalable components with independent scaling
- **Observability**: Built-in health checks and metrics at every layer
- **Fault Tolerance**: Graceful handling of failures with recovery mechanisms

---

## High-Level System Architecture

```mermaid
graph LR
    A[CSV Files<br/>• GPU metrics<br/>• Flexible schema<br/>• Continuous streaming] 
    B[Custom MQ<br/>• In-memory<br/>• Persistence<br/>• Pub/Sub<br/>• Acknowledgment<br/>• Admin APIs]
    C[Collectors<br/>• Type safety<br/>• Concurrency<br/>• Checkpoints<br/>• Health checks<br/>• Error handling] 
    D[Storage<br/>• File JSONL<br/>• Memory LRU<br/>• Health APIs<br/>• Statistics]
    
    A -->|Streams| B
    B -->|Delivers| C  
    C -->|Persists| D
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#e8f5e8
    style D fill:#fff3e0
```

---

## Component Architecture

### 1. Telemetry Streamer

**Purpose**: Reads GPU telemetry data from CSV files and publishes to the message queue.

**Key Characteristics**:
- Schema-agnostic CSV parsing (works with any CSV format)
- Concurrent workers for parallel data streaming
- Configurable message rate per worker
- Continuous loop with graceful shutdown support

**Data Flow**:

```mermaid
flowchart TD
    A[CSV File] --> B[Parse Record<br/>flexible schema detection]
    B --> C[Convert Types<br/>string → number conversion]
    C --> D[Create JSON Message]
    D --> E[Publish to MQ Topic]
    
    style A fill:#e3f2fd
    style E fill:#f3e5f5
```

**Scalability**:
- Multiple streamer instances can run in parallel
- Each instance has independent worker pools
- Rate limiting per worker prevents queue saturation

---

### 2. Custom Message Queue (MQ)

**Purpose**: In-memory message broker with optional persistence for reliable message delivery.

**Architecture**:

```mermaid
graph TD
    A[Publisher<br/>Streamer] --> B[Topic Manager]
    B --> C[Topic 1<br/>GPU Telemetry]
    B --> D[Topic 2, 3, ...<br/>Future topics]
    
    C --> E[In-Memory Queue]
    C --> F[Optional Disk Buffer]  
    C --> G[Subscribers<br/>Collectors]
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style G fill:#e8f5e8
```

**Key Features**:
- **In-Memory Storage**: Optional High-performance message queue in RAM
- **Message Acknowledgment**: Ensures delivery reliability
  - Message is held until acknowledged by subscriber
  - Automatic redelivery on timeout (5s default)
  - Prevents message loss with retry mechanism
- **Topic-Based Routing**: Subscribers receive messages from specific topics
- **Persistence Layer**: Disk persistence for durability(default)
- **Admin Endpoints**: Real-time statistics and health monitoring

**Message Lifecycle**:

```mermaid
sequenceDiagram
    participant P as Publisher
    participant Q as In-Memory Queue
    participant S as Subscriber
    
    P->>Q: 1. Publish Message
    Q->>Q: 2. Store in Queue
    Q->>S: 3. Deliver to Subscriber
    S->>S: 4. Process Message
    S->>Q: 5. Send Acknowledgment
    Q->>Q: 6. Remove from Queue
```

**Reliability Guarantees**:
- At-least-once delivery (messages redelivered until acknowledged)
- Configurable timeout and retry settings
- Optional persistence for zero-message-loss scenarios

---

### 3. Telemetry Collector

**Purpose**: Consumes telemetry messages and provides dual-layer storage with querying capabilities.

**Architecture**:

```mermaid
graph TD
    A[MQ Subscriber] --> B[Message Processor<br/>Worker Pool]
    B --> C[Parse JSON]
    B --> D[Validate Structure]
    B --> E[Track GPU ID]
    B --> F[Split Storage]
    
    F --> G[File Storage JSONL]
    F --> H[Memory Storage LRU Cache]
    
    G --> I[Per-GPU files<br/>data/gpu_0.jsonl, etc.]
    H --> J[Fast in-memory access]
    
    K[Checkpoint System] --> L[Recovery State File]
    
    style A fill:#f3e5f5
    style B fill:#e8f5e8
    style G fill:#fff3e0
    style H fill:#fce4ec
```

**Dual Storage Strategy**:

| Storage | Purpose | Benefits |
|---------|---------|----------|
| **File Storage (JSONL)** | Long-term persistence | Durable, searchable, auditable |
| **Memory Storage (LRU)** | Real-time queries | Fast access, reduced disk I/O |

**Scalability**:
- Configurable worker pool for concurrent message processing
- LRU cache with configurable max entries per GPU
- Automatic cache eviction prevents memory exhaustion
- Per-GPU file sharding enables parallel I/O

---

## Data Flow Diagrams

### Real-Time Data Pipeline

```mermaid
graph TD
    A[CSV Streamer<br/>gpu_0, 72.3°C<br/>Time: 0.0s] -->|Publish to MQ| B[MQ Broker]
    B -->|Route to subscribers| C[Collector 1<br/>Worker Pool]
    C --> D[Write to disk<br/>gpu_0.jsonl]
    C --> E[Update LRU cache]
    C --> F[Acknowledge message]
    
    G[Time: 0.5s<br/>Repeat cycle] -.-> A
    
    style A fill:#e1f5fe
    style B fill:#f3e5f5
    style C fill:#e8f5e8
```

### Message Processing Sequence

```mermaid
sequenceDiagram
    participant S as Streamer
    participant MQ as MQ Broker
    participant C as Collector
    participant St as Storage
    
    S->>MQ: Publish message
    MQ->>C: Deliver to subscriber
    C->>C: Process message
    C->>St: Store data
    St-->>C: Confirm storage
    C->>MQ: Acknowledge message
    MQ->>MQ: Mark as confirmed
```

---

## API Architecture

### Health & Monitoring

**API Gateway Health Endpoint** (`GET /health`):
```json
{
  "status": "healthy",
  "service": "telemetry-api-gateway",
  "timestamp": "2025-10-20T12:00:00Z",
  "version": "1.0.0",
  "collector": {
    "status": "healthy"
  }
}
```

**Service Health Aggregation**:
- API Gateway monitors all downstream services
- Single health check endpoint for all services
- Real-time status reflecting actual service state

### Data Access APIs

**API Gateway (Port 8081)**:
- **GPU List**: `GET /api/v1/gpus`
- **GPU Telemetry**: `GET /api/v1/gpus/{id}/telemetry`
- **Host List**: `GET /api/v1/hosts`
- **Host GPUs**: `GET /api/v1/hosts/{hostname}/gpus`
- **Health Status**: `GET /health`
- **API Documentation**: `GET /swagger/`

**MQ Service (Port 9090/9091)**:
- **Publish Message**: `POST /publish/{topic}`
- **Broker Stats**: `GET /stats`
- **Health Status**: `GET /health`
- **gRPC Interface**: Port 9091

---

## Deployment Patterns

### Single Machine (Development)

```mermaid
graph TB
    subgraph DC["Docker Container"]
        subgraph AC["All Components"]
            MQ[MQ Service]
            ST[Streamer]
            CO[Collectors]
        end
    end
    
    style DC fill:#e3f2fd
    style AC fill:#f5f5f5
```

### Kubernetes (Production)

```mermaid
graph TB
    subgraph K8S["Kubernetes Cluster"]
        subgraph NS["Namespace: gpu-telemetry"]
            ST[Streamer<br/>DaemonSet<br/>2 replicas]
            CO[Collector<br/>StatefulSet<br/>2 replicas]
            AG[API Gateway<br/>Deployment<br/>2 replicas]
            
            MQ[MQ Service<br/>StatefulSet<br/>• In-memory broker<br/>• Stateless scaling]
            
            PV[Persistent Volume<br/>MQ optional persistence]
        end
    end
    
    DASH[Dashboard<br/>React Frontend]
    
    ST --> MQ
    CO --> MQ
    AG --> MQ
    MQ --> PV
    K8S --> DASH
    
    style K8S fill:#e8f5e8
    style NS fill:#f5f5f5
    style DASH fill:#fce4ec
```

---

## Concurrency Model

### Worker Pool Pattern

Each component uses a configurable worker pool for concurrent processing:

```mermaid
graph TD
    TQ[Task Queue] --> WP[Worker Pool N=4]
    
    subgraph WP["Worker Pool (N=4)"]
        W1[Worker 1<br/>Processing task A]
        W2[Worker 2<br/>Processing task B]  
        W3[Worker 3<br/>Processing task C]
        W4[Worker 4<br/>Waiting]
    end
    
    WP --> R[Results]
    
    style TQ fill:#e3f2fd
    style R fill:#e8f5e8
```

**Benefits**:
- CPU parallelism on multi-core systems
- I/O concurrency without blocking
- Configurable throughput via worker count
- Graceful scaling based on workload

---

## Error Handling & Recovery

### Circuit Pattern

```mermaid
stateDiagram-v2
    [*] --> Normal
    Normal --> Error: Error Detected
    Error --> Backoff: Exponential Backoff
    Backoff --> Retry: Retry with Jitter
    Retry --> Normal: Success
    Retry --> Logging: Failure
    Logging --> [*]
```

### Graceful Shutdown

All components support clean shutdown on `SIGINT`/`SIGTERM`:

```mermaid
flowchart TD
    A[Shutdown Signal<br/>SIGINT/SIGTERM] --> B[Stop Accepting<br/>New Work]
    B --> C[Wait for In-Flight<br/>Operations]
    C --> D[Flush Pending Data]
    D --> E[Close Connections]
    E --> F[Clean Exit<br/>code 0]
    
    style A fill:#ffebee
    style F fill:#e8f5e8
```

---

## Performance Characteristics

### Throughput

- **Streamer**: 1000+ messages/second per worker
- **MQ Broker**: 10,000+ messages/second in-memory
- **Collector**: 500+ messages/second per worker

### Latency

- **Message Queue**: < 1ms end-to-end latency
- **File Persistence**: ~5-10ms per message
- **Cache Lookup**: < 1ms for LRU hits

### Resource Usage

- **Memory**: Configurable via LRU cache size
- **CPU**: Scales with worker count
- **Disk**: Dependent on telemetry volume and retention

---

## Scalability Strategies

### Horizontal Scaling

- **Multiple Streamers**: Partition CSV files across instances
- **Multiple Collectors**: MQ distributes load across subscribers
- **Multiple API Gateway**: Load-balance across API instances

### Vertical Scaling

- **Increase Workers**: Adjust worker pool size per component
- **Increase Cache**: Grow LRU cache for more telemetry retention
- **Increase Throughput**: Adjust message rate per worker

### Adaptive Scaling

- Kubernetes HPA automatically scales based on CPU/memory
- MQ handles bursty traffic without client coordination
- Collectors independently process at their own rate

---

## Design Decisions

### Why Custom MQ?

1. **Single Binary Deployment**: No Kafka, RabbitMQ, or Zookeeper needed
2. **Optimized for Telemetry**: Tailored acknowledgment and redelivery
3. **Observable**: Built-in metrics and admin endpoints
4. **Customizable**: Message routing and persistence strategies

### Why Dual Storage?

1. **Performance**: In-memory cache for real-time queries
2. **Durability**: File storage for long-term retention
3. **Cost**: LRU eviction prevents unbounded memory growth
4. **Flexibility**: Different access patterns for different use cases

### Why Kubernetes DaemonSet for Streamer?

1. **Mimic Per-Node Processing**: Collect telemetry from node where GPU resides
2. **Auto-Scaling**: Streamer runs on every node automatically
3. **Resilience**: Node failure doesn't stop entire pipeline
4. **Resource Awareness**: Streamer respects node capacity

---

## Next Steps

- See [Quickstart Guide](../quickstart/README.md) for deployment instructions
- See [Components Documentation](../components/README.md) for detailed component info
- See [Deployment Guide](../deployment/README.md) for production setup
