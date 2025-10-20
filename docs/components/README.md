# Components Reference

Detailed overview of each component in the GPU Telemetry Pipeline.

---

## Table of Contents

1. [Telemetry Streamer](#telemetry-streamer)
2. [Custom Message Queue](#custom-message-queue)
3. [Telemetry Collector](#telemetry-collector)
4. [API Gateway](#api-gateway)
5. [Dashboard](#dashboard)

---

## Telemetry Streamer

**Location**: `cmd/telemetry-streamer/`

**Purpose**: Reads GPU telemetry from CSV files and publishes to the message queue at a configurable rate.

### What It Does

The streamer is responsible for:
- Reading CSV data continuously
- Publishing messages to the MQ at a controlled rate
- Managing multiple concurrent workers
- Graceful shutdown on signals

### Architecture

```
CSV File
   ↓
[Worker Pool]
├─ Worker 1: Read rows 0-1000
├─ Worker 2: Read rows 1000-2000
├─ Worker 3: Read rows 2000-3000
└─ Worker 4: Read rows 3000-4000
   ↓
[Rate Limiter] (5 msg/sec per worker)
   ↓
MQ Publisher
   ↓
Message Queue
```

### Key Configuration

| Parameter | Default | Purpose |
|-----------|---------|---------|
| `--csv` | `data.csv` | Path to CSV file |
| `--workers` | `4` | Number of concurrent workers |
| `--rate` | `1.0` | Messages/second per worker |
| `--duration` | `0` | How long to stream (0 = infinite) |
| `--mq-url` | `http://localhost:9090` | MQ broker URL |
| `--log-level` | `info` | Logging level |

### Usage Example

```bash
# Run streamer with 4 workers at 10 msg/sec per worker for 1 hour
./telemetry-streamer \
  --csv data.csv \
  --workers 4 \
  --rate 10 \
  --duration 1h

# Stream continuously at 1 msg/sec
./telemetry-streamer --csv data.csv --rate 1

# Stream at high throughput
./telemetry-streamer --csv data.csv --workers 8 --rate 50
```

### Performance Characteristics

- **Throughput**: 1000+ messages/second per worker
- **Latency**: < 10ms per message
- **Memory**: ~50MB base + buffer for workers
- **CPU**: Scales with worker count

### How It Works

1. **CSV Parsing**
   - Reads CSV line by line
   - Flexible schema detection
   - Automatic type conversion (string → number)

2. **Worker Pool**
   - Each worker processes independently
   - Rows distributed round-robin across workers
   - Configurable concurrency level

3. **Rate Limiting**
   - Per-worker rate limiter
   - Token bucket algorithm
   - Prevents MQ overload

4. **Message Publishing**
   - Convert CSV row to JSON
   - Add timestamp
   - Publish to MQ topic
   - Handle failures gracefully

### Data Format

**Input (CSV)**:
```csv
gpu_id,temperature,utilization,memory_used,power_draw
gpu_0,72.3,85.5,4096,250.5
gpu_1,75.1,90.2,8192,275.8
```

**Output (JSON)**:
```json
{
  "timestamp": "2025-10-20T12:00:00Z",
  "fields": {
    "gpu_id": "gpu_0",
    "temperature": 72.3,
    "utilization": 85.5,
    "memory_used": 4096.0,
    "power_draw": 250.5
  }
}
```

---

## Custom Message Queue

**Location**: `internal/mq/`

**Purpose**: In-memory message broker with acknowledgment semantics for reliable message delivery.

### What It Does

The MQ provides:
- Publish/Subscribe messaging
- Message acknowledgment with timeout
- Automatic redelivery
- Topic-based routing
- Optional disk persistence
- Admin APIs for monitoring

### Architecture

```
┌─────────────────────────────────────┐
│      Message Queue Service           │
├─────────────────────────────────────┤
│                                     │
│  Topic Manager                      │
│  ├─ gpu-telemetry (default)         │
│  │  ├─ In-Memory Queue              │
│  │  ├─ Message Index                │
│  │  └─ Subscriber Tracking          │
│  │                                  │
│  └─ Other topics (future)           │
│                                     │
│  Optional: Disk Persistence         │
│  └─ WAL (Write-Ahead Log)           │
│                                     │
│  Admin Endpoints                    │
│  ├─ GET /health                     │
│  ├─ GET /stats                      │
│  └─ GET /topics                     │
└─────────────────────────────────────┘
```

### Message Lifecycle

```
Publisher                         Subscriber
   │                                 │
   ├─ Publish Message ──────────────→ MQ
   │                                 │
   │                  MQ ────────────→ Delivers to all subscribers
   │                                 │
   │                              [Process Message]
   │                                 │
   │                  ←────── Acknowledge
   │
   [Message stored until ACK received]
   [After 5s timeout, redelivery triggered]
```

### Configuration

| Parameter | Default | Purpose |
|-----------|---------|---------|
| `--port` | `9090` | Admin API port |
| `--persistence` | `false` | Enable disk persistence |
| `--persistence-path` | `/data/mq` | Where to store messages |
| `--ack-timeout` | `5s` | Timeout before redelivery |
| `--max-retries` | `3` | Max redelivery attempts |

### Admin Endpoints

**Health Check**:
```bash
curl http://localhost:9090/health
# Response:
# {"status":"healthy","timestamp":"2025-10-20T12:00:00Z"}
```

**Queue Statistics**:
```bash
curl http://localhost:9090/stats
# Response:
# {
#   "topics": {
#     "gpu-telemetry": {
#       "messages": 15000,
#       "subscribers": 2,
#       "throughput_msg_per_sec": 250
#     }
#   },
#   "total_processed": 1000000,
#   "uptime_seconds": 3600
# }
```

**List Topics**:
```bash
curl http://localhost:9090/topics
# Response:
# {
#   "topics": ["gpu-telemetry"]
# }
```

### Reliability Features

1. **Message Acknowledgment**
   - Subscriber must acknowledge after processing
   - Message redelivered if not acknowledged within timeout
   - Prevents silent failures

2. **Automatic Redelivery**
   - Configurable timeout (default: 5 seconds)
   - Exponential backoff between retries
   - Max retry limit prevents infinite loops

3. **Persistence (Optional)**
   - Write-ahead log (WAL) format
   - Survives broker restart
   - Recovers unacknowledged messages

4. **Monitoring**
   - Real-time message count
   - Throughput metrics
   - Subscriber tracking
   - Health status endpoint

### Performance

- **Throughput**: 10,000+ messages/second
- **Latency**: < 1ms per publish
- **Memory**: ~100MB for 1M messages in queue
- **Persistence**: ~50MB per 1M messages on disk

---

## Telemetry Collector

**Location**: `cmd/telemetry-collector/`

**Purpose**: Consumes messages from the queue and provides dual-layer storage with querying capabilities.

### What It Does

The collector:
- Subscribes to MQ messages
- Parses and validates telemetry data
- Stores data in two layers (file + memory)
- Provides REST APIs for data access
- Tracks processing checkpoints

### Architecture

```
MQ Subscriber
   ↓
[Worker Pool] (4 workers)
   ├─ Worker 1: Process messages
   ├─ Worker 2: Process messages
   ├─ Worker 3: Process messages
   └─ Worker 4: Process messages
   ↓
[Data Router]
   ├─→ File Storage
   │   └─ Persistent (JSONL format)
   │      ├─ data/gpu_0.jsonl
   │      ├─ data/gpu_1.jsonl
   │      └─ data/gpu_N.jsonl
   │
   └─→ Memory Storage
       └─ LRU Cache (fast access)
          ├─ GPU 0: [last 1000 entries]
          ├─ GPU 1: [last 1000 entries]
          └─ GPU N: [last 1000 entries]

Checkpoint System
   └─ Saves processing offset
```

### Configuration

| Parameter | Default | Purpose |
|-----------|---------|---------|
| `--workers` | `4` | Message processing workers |
| `--data-dir` | `./data` | Directory for file storage |
| `--max-entries` | `1000` | Max cache entries per GPU |
| `--checkpoint` | `true` | Enable recovery checkpoints |
| `--api-port` | `8080` | REST API port |
| `--broker-port` | `9090` | MQ broker port |

### Data Storage

**File Storage** (`JSONL Format`):
```
data/gpu_0.jsonl:
{"gpu_id":"gpu_0","metrics":{"temperature":72.3,"utilization":85.5},"timestamp":"2025-10-20T12:00:00Z"}
{"gpu_id":"gpu_0","metrics":{"temperature":72.5,"utilization":85.7},"timestamp":"2025-10-20T12:00:01Z"}

data/gpu_1.jsonl:
{"gpu_id":"gpu_1","metrics":{"temperature":75.1,"utilization":90.2},"timestamp":"2025-10-20T12:00:00Z"}
```

**Memory Storage** (LRU Cache):
```
GPU 0 Cache: [Entry 9995, Entry 9996, Entry 9997, Entry 9998, Entry 9999]
GPU 1 Cache: [Entry 4995, Entry 4996, Entry 4997, Entry 4998, Entry 4999]
```

### REST APIs

**Health Check**:
```bash
curl http://localhost:8080/health
# {"status":"healthy","timestamp":"2025-10-20T12:00:00Z"}
```

**Statistics**:
```bash
curl http://localhost:8080/stats
# {
#   "total_processed": 150000,
#   "active_gpus": 4,
#   "file_entries": 150000,
#   "cache_entries": 4000,
#   "uptime_seconds": 3600
# }
```

**Get Telemetry**:
```bash
# Latest entries for a GPU
curl "http://localhost:8080/telemetry/gpu_0?limit=10"

# Reverse order (oldest first)
curl "http://localhost:8080/telemetry/gpu_0?reverse=true&limit=10"

# Response:
# [
#   {
#     "gpu_id": "gpu_0",
#     "metrics": {
#       "temperature": 72.5,
#       "utilization": 85.7,
#       "memory_used": 4096,
#       "power_draw": 250.8
#     },
#     "timestamp": "2025-10-20T12:00:01Z"
#   },
#   ...
# ]
```

### Scalability

1. **Horizontal**: Run multiple collectors as separate pods
2. **Vertical**: Increase worker count per collector
3. **Storage**: Per-GPU file sharding enables parallel I/O
4. **Cache**: LRU eviction prevents memory exhaustion

### Performance

- **Throughput**: 500+ messages/second per worker
- **Latency**: ~5-10ms per message with file persistence
- **Memory**: Configurable via LRU cache size
- **Disk I/O**: Batched writes for efficiency

---

## API Gateway

**Location**: `cmd/api-gateway/` / `internal/api/`

**Purpose**: Central REST API for accessing telemetry data and health information.

### What It Does

The API Gateway:
- Provides unified REST API
- Aggregates health information
- Routes requests to collectors
- Caches responses
- Provides OpenAPI documentation

### Architecture

```
HTTP Client
   ↓
┌─────────────────────────┐
│   API Gateway           │
├─────────────────────────┤
│                         │
│ Routing Layer           │
│ ├─ /api/v1/gpus         │
│ ├─ /api/v1/telemetry    │
│ ├─ /health              │
│ └─ /docs (Swagger)      │
│                         │
│ Service Discovery       │
│ ├─ Collector services   │
│ ├─ MQ service           │
│ └─ Health checks        │
│                         │
│ Caching Layer           │
│ └─ Response cache       │
│                         │
│ Middleware              │
│ ├─ Logging              │
│ ├─ CORS                 │
│ └─ Error handling       │
└─────────────────────────┘
   ↓
Collector Services / MQ
```

### REST Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health status of all services |
| `/api/v1/gpus` | GET | List available GPUs |
| `/api/v1/gpus/{id}/telemetry` | GET | GPU telemetry data |
| `/stats` | GET | MQ statistics |
| `/docs` | GET | API documentation (Swagger) |

### Health Aggregation

The `/health` endpoint returns:
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

This endpoint checks:
- API Gateway status
- Collector connectivity
- MQ broker status

### Performance

- **Throughput**: 1000+ requests/second
- **Latency**: < 50ms average
- **Cache Hit Rate**: 80%+ for typical workloads

---

## Dashboard

**Location**: `dashboard/`

**Purpose**: Web-based UI for monitoring GPU telemetry in real-time.

### What It Does

The dashboard:
- Displays GPU list and status
- Shows real-time telemetry charts
- Monitors service health
- Provides statistics
- Responsive design for mobile

### Tech Stack

- **Framework**: React 19 with TypeScript
- **Bundler**: Vite (fast development)
- **Styling**: Tailwind CSS
- **Charts**: Recharts
- **Icons**: Lucide React

### Key Features

1. **GPU Monitoring**
   - List all available GPUs
   - Select GPU for detailed view
   - Real-time metric charts

2. **Health Dashboard**
   - API Gateway status
   - Collector status
   - MQ service status
   - Last update timestamp

3. **Statistics**
   - Total messages processed
   - Messages per second
   - Cache hit rate
   - Uptime

4. **Responsive Design**
   - Works on desktop
   - Works on tablets
   - Mobile-friendly layout

### Architecture

```
Browser
   ↓
React Application
   ├─ Components
   │  ├─ GPUList
   │  ├─ TelemetryChart
   │  ├─ HealthStatus
   │  └─ Statistics
   │
   ├─ API Client
   │  └─ Fetch from API Gateway
   │
   └─ State Management
      ├─ GPU list
      ├─ Telemetry data
      └─ Health status
   ↓
API Gateway
   ↓
Backend Services
```

### Configuration

Environment variables (see `.env` file):

```env
VITE_API_BASE_URL=http://api-gateway:8081
VITE_MQ_BASE_URL=http://mq-service:9090
VITE_REFRESH_INTERVAL=5000
```

### Data Flow

```
1. Dashboard loads
   ↓
2. Fetch GPU list from API
   ↓
3. Display GPU list
   ↓
4. User selects GPU
   ↓
5. Poll telemetry data every 5 seconds
   ↓
6. Update charts with new data
   ↓
7. Poll health status every 10 seconds
   ↓
8. Update health indicators
```

### Performance Optimization

- **Polling Strategy**: Staggered requests to avoid thundering herd
- **Caching**: Client-side cache for GPU list (rarely changes)
- **Chart Optimization**: Limit number of data points displayed
- **Lazy Loading**: Load GPU data on demand

---

## Component Interactions

### Full Data Flow

```
Streamer → MQ → Collector → API Gateway → Dashboard
    ↓        ↓       ↓           ↓             ↓
  CSV     Routes  Storage    Queries      Visualize
  Data    Msgs    2-Layer    Aggregate    Monitor
         ...             Data

Health Flow:
Dashboard → API Gateway → Collector (health)
                       → MQ (health)
```

### Network Communication

**In Docker**:
```
streamer:5000 → mq-service:9090 → collector:5000 → api-gateway:8081 → dashboard:80
```

**In Kubernetes**:
```
streamer → mq-service.gpu-telemetry → collector.gpu-telemetry → api-gateway.gpu-telemetry → dashboard.gpu-telemetry
```

---

## Monitoring & Observability

### Health Checks

```bash
# API Gateway health (includes all services)
curl http://localhost:8081/health

# MQ service stats
curl http://localhost:9090/stats

# Collector stats
curl http://localhost:8080/stats
```

### Logs

Each component logs to stdout with structured JSON format:

```json
{"timestamp":"2025-10-20T12:00:00Z","level":"info","service":"collector","message":"Processed message","gpu_id":"gpu_0"}
```

### Metrics

Available via endpoints:
- Message throughput (msg/sec)
- Latency (ms)
- Error rate (%)
- Cache hit rate (%)
- Active connections

---

## Troubleshooting

### Streamer Issues

```bash
# Check connectivity to MQ
curl http://localhost:9090/stats

# Check logs
docker logs telemetry-streamer

# Verify CSV file
head -5 data.csv
```

### MQ Issues

```bash
# Check MQ health
curl http://localhost:9090/health

# Check topics
curl http://localhost:9090/topics

# Check persistence (if enabled)
ls -lh /data/mq/
```

### Collector Issues

```bash
# Check health
curl http://localhost:8080/health

# Check stats
curl http://localhost:8080/stats

# Check storage
ls -lh data/
```

### API Gateway Issues

```bash
# Test endpoint directly
curl http://localhost:8080/telemetry/gpu_0

# Check logs
docker logs api-gateway
```

### Dashboard Issues

```bash
# Check console for errors (F12)
# Check network tab for failed requests
# Verify API Gateway is accessible
curl http://localhost:8081/health
```

---

For detailed setup and deployment instructions, see:
- [Quickstart Guide](../quickstart/README.md)
- [Deployment Guide](../deployment/README.md)
- [Architecture Guide](../architecture/README.md)
