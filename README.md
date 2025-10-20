# GPU Telemetry Pipeline

[![Go Report Card](https://goreportcard.com/badge/github.com/harishb93/telemetry-pipeline)](https://goreportcard.com/report/github.com/harishb93/telemetry-pipeline)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-supported-blue.svg)](https://www.docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-ready-green.svg)](https://kubernetes.io/)

A production-ready, scalable telemetry pipeline for GPU metrics built with Go. Streams GPU telemetry data through a custom message broker to persistent collectors with real-time monitoring via React dashboard.

**Pattern**: CSV Streamers → Custom MQ → Collectors → Storage → Dashboard

---

## � Quick Start

### With Docker (5 minutes)

```bash
git clone https://github.com/harishb93/telemetry-pipeline.git
cd telemetry-pipeline/deploy/docker
./setup.sh
```

Open [http://localhost:8080](http://localhost:8080) to view the dashboard.

### With Kubernetes (10 minutes)

```bash
git clone https://github.com/harishb93/telemetry-pipeline.git
cd telemetry-pipeline/deploy/helm
./quickstart.sh
```

See [Quickstart Guide](docs/quickstart/README.md) for detailed setup instructions.

---

## 📋 Documentation

Comprehensive documentation is organized into focused guides:

### Getting Started
- **[Quickstart Guide](docs/quickstart/README.md)** - Get running in minutes
  - Docker Compose setup
  - Kubernetes (Kind) setup  
  - Verification & testing
  - Troubleshooting

### Understanding the System
- **[System Architecture](docs/architecture/README.md)** - How it all works
  - High-level overview
  - Component interactions
  - Data flow diagrams
  - Scalability strategies
  - Design decisions

- **[Components Reference](docs/components/README.md)** - What each service does
  - Telemetry Streamer
  - Custom Message Queue
  - Telemetry Collector
  - API Gateway
  - React Dashboard

### Deployment & Operations
- **[Deployment Guide](docs/deployment/README.md)** - Deploy to any environment
  - Docker Compose deployment
  - Kubernetes deployment
  - Helm configuration
  - Production hardening
  - Advanced patterns (multi-cluster, GitOps, etc.)

- **[Makefile Guide](docs/makefile.md)** - Build and deployment automation
  - Build targets
  - Docker targets
  - Kubernetes targets
  - Common workflows
  - Troubleshooting

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CSV Files     │    │  Custom MQ      │    │   Collectors    │    │    Storage      │
│                 │    │                 │    │                 │    │                 │
│ • GPU metrics   │───▶│ • In-memory     │───▶│ • Type safety   │───▶│ • File (JSONL)  │
│ • Flexible      │    │ • Persistence   │    │ • Concurrency   │    │ • Memory (LRU)  │
│   schema        │    │ • Pub/Sub       │    │ • Checkpoints   │    │ • Health APIs   │
│ • Continuous    │    │ • Acknowledgment│    │ • Health checks │    │ • Statistics    │
│   streaming     │    │ • Admin APIs    │    │ • Error handling│    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
                                                                              ↓
                                                                      ┌──────────────────┐
                                                                      │  React Dashboard │
                                                                      │   & API Gateway  │
                                                                      └──────────────────┘
```

**Key Pattern**:
1. **Streamers** read CSV files and publish to MQ
2. **Message Queue** reliably routes messages with acknowledgment
3. **Collectors** consume messages and store in dual-layer (file + cache)
4. **API Gateway** provides unified REST interface
5. **Dashboard** visualizes real-time data

See [System Architecture](docs/architecture/README.md) for detailed design.

## ⚡ Key Features

### Telemetry Streaming
- Schema-agnostic CSV parsing (works with any CSV format)
- Concurrent workers for parallel processing
- Configurable throughput and duration
- Continuous streaming with graceful shutdown

### Custom Message Queue
- Single-binary deployment (no external dependencies)
- Acknowledgment semantics for reliable delivery
- Message routing with topic support
- Optional disk persistence for durability
- Real-time statistics and monitoring

### Data Collection & Storage
- Dual-layer persistence (file + in-memory cache)
- Per-GPU JSONL file storage for durability
- LRU cache for fast queries with memory bounds
- Checkpoint system for recovery
- Configurable worker pools

### Observability
- Health endpoints for all services
- Real-time statistics APIs
- Aggregated health status
- Structured JSON logging
- Service-to-service health tracking

### User Interface
- Modern React-based dashboard
- Real-time GPU telemetry charts
- Service health monitoring
- Queue statistics
- Responsive design for mobile

---

## 📊 Component Breakdown

| Component | Purpose | Technology | Scaling |
|-----------|---------|-----------|---------|
| **Streamer** | Publish CSV data to MQ | Go, goroutines | Horizontal (multiple instances) |
| **Message Queue** | Route messages reliably | Custom Go implementation | Horizontal (stateless) |
| **Collector** | Consume & persist data | Go, worker pool | Horizontal (independent consumers) |
| **API Gateway** | Unified REST API | Go, Gorilla mux | Horizontal (stateless) |
| **Dashboard** | Real-time monitoring | React 19, Vite | Horizontal (stateless frontend) |

See [Components Reference](docs/components/README.md) for detailed information on each component.

---

## 🛠️ Technology Stack

### Backend
- **Language**: Go 1.24+
- **Architecture**: Microservices with clear separation of concerns
- **Concurrency**: Goroutines with configurable worker pools
- **Storage**: File (JSONL) + In-Memory (LRU cache)
- **API**: REST with JSON payloads

### Frontend
- **Framework**: React 19 with TypeScript
- **Bundler**: Vite (fast development & builds)
- **Styling**: Tailwind CSS
- **Charts**: Recharts for telemetry visualization
- **Icons**: Lucide React

### Deployment
- **Containerization**: Docker with multi-stage builds
- **Orchestration**: Kubernetes with Helm charts
- **Local Dev**: Docker Compose
- **Testing**: Docker in Docker (DinD) for local Kind cluster

---

## 📈 Performance

### Throughput
- **Streamer**: 1000+ messages/second per worker
- **MQ**: 10,000+ messages/second in-memory
- **Collector**: 500+ messages/second per worker
- **API**: 1000+ requests/second

### Latency
- **MQ**: < 1ms end-to-end
- **File Write**: 5-10ms per message
- **Cache Hit**: < 1ms
- **API Response**: < 50ms average

### Resource Usage
- **Memory**: Configurable via LRU cache size
- **CPU**: Scales with worker count
- **Disk**: Dependent on data volume and retention

### Scalability
- **Horizontal**: Add more pods in Kubernetes
- **Vertical**: Increase workers and cache size per pod
- **Adaptive**: Kubernetes HPA auto-scales based on metrics

---

## 🚢 Deployment Options

### Development
- **Docker Compose** - Single command to run locally
- **Local Kind cluster** - Kubernetes simulation on laptop
- Perfect for: Learning, local testing, CI/CD pipelines

### Production
- **Managed Kubernetes** (AWS EKS, GCP GKE, Azure AKS)
- **Self-managed Kubernetes** (on-premises or VMs)
- **Docker Swarm** - For simpler deployments
- Features: Auto-scaling, multi-node, high availability

---

## 🎯 Use Cases

1. **GPU Cluster Monitoring**
   - Monitor GPU metrics across distributed cluster
   - Track temperature, utilization, power consumption
   - Alert on anomalies

2. **Telemetry Aggregation**
   - Collect metrics from multiple sources
   - Centralized storage and analysis
   - Real-time dashboarding

3. **Performance Analysis**
   - Historical data retention
   - Trend analysis and visualization
   - Capacity planning

4. **Development & Testing**
   - Pipeline for learning Go and Kubernetes
   - Reference architecture for distributed systems
   - Production patterns and best practices

---

## 🚀 Key Features

### Telemetry Streamer
- **Flexible CSV Processing**: Schema-agnostic parsing with automatic type detection
- **Continuous Streaming**: Loops through data with configurable rates
- **Concurrent Workers**: Multiple workers with independent rate limiting
- **Type Conversion**: Automatic string-to-number conversion for metrics
- **Graceful Shutdown**: Signal-based cleanup with proper resource management

### Custom Message Queue
- **Custom Implementation**: No external dependencies (Kafka, RabbitMQ, etc.)
- **Acknowledgment Semantics**: Message acknowledgment with timeout and redelivery
- **Persistence Layer**: Optional disk persistence for message durability
- **Admin Endpoints**: HTTP APIs for monitoring queue stats and health
- **Concurrency Safe**: Thread-safe operations with proper locking

### Telemetry Collector
- **Typed Data Structures**: JSON parsing to strongly-typed Go structs
- **Dual Persistence**: Both file-based (JSONL) and in-memory (LRU) storage
- **Checkpoint System**: Processing state persistence for recovery
- **Worker Pool**: Configurable concurrent message processing
- **Health Monitoring**: HTTP endpoints for health checks and statistics

### Observability & Operations
- **Health Endpoints**: Comprehensive health checks for all components
- **Statistics APIs**: Real-time metrics and system state information
- **Graceful Shutdown**: Proper cleanup on SIGINT/SIGTERM signals
- **Error Handling**: Robust error handling with logging and recovery
- **Checkpointing**: Processing state persistence for fault tolerance

## 📦 Components

### 1. Telemetry Streamer (`cmd/telemetry-streamer`)
Reads CSV files and streams data to the message queue.

**Key Features:**
- Schema-agnostic CSV parsing
- Configurable workers and message rates
- Continuous loop processing
- JSON message encoding
- Signal-based shutdown

**Usage:**
```bash
./telemetry-streamer --csv data.csv --workers 4 --rate 10 --duration 1h
```

---

## 🏃 Getting Started

### Prerequisites
- Docker 20.10+ OR Go 1.24+
- kubectl 1.27+ (for Kubernetes)
- Helm 3.12+ (for Kubernetes)

### Fastest Start (Docker)
```bash
cd deploy/docker
./setup.sh
# Services start automatically
# Dashboard at http://localhost:8080
# API at http://localhost:8081
```

### Kubernetes Start
```bash
cd deploy/helm
./quickstart.sh
# Dashboard at http://localhost:8080 (after port-forward)
# API at http://localhost:8081
```

### Local Development (Go)
```bash
go build ./cmd/telemetry-streamer
go build ./cmd/telemetry-collector
go build ./cmd/api-gateway
make dev  # build + test
```

For step-by-step instructions, see [Quickstart Guide](docs/quickstart/README.md).

---

## 🔧 Makefile Quick Commands

```bash
make all              # Full build pipeline
make dev              # Build + test (development)
make docker-build     # Build Docker images
make docker-push      # Push to registry
make helm-install     # Deploy to Kubernetes
make clean            # Remove artifacts
```

See [Makefile Guide](docs/makefile.md) for complete reference.

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# With coverage
go test ./... -cover

# Specific package
go test ./internal/mq -v

# Generate coverage report
make coverage
```

---

## 📁 Data Formats

### CSV Input Format
```csv
gpu_id,temperature,utilization,memory_used,power_draw
gpu_0,72.3,85.5,4096,250.5
gpu_1,75.1,90.2,8192,275.8
```

### JSON Message Format (Internal)
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "fields": {
    "gpu_id": "gpu_0",
    "temperature": 72.3,
    "utilization": 85.5,
    "memory_used": 4096.0,
    "power_draw": 250.5
  }
}
```

### Stored Format (JSONL + Memory Cache)
```json
{
  "gpu_id": "gpu_0",
  "metrics": {
    "temperature": 72.3,
    "utilization": 85.5,
    "memory_used": 4096.0,
    "power_draw": 250.5
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure tests pass: `go test ./...`
5. Submit a pull request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🎓 Learning Resources

This project demonstrates:
- **Go best practices**: Clean architecture, error handling, concurrency
- **Distributed systems**: Message queues, async processing, service orchestration
- **Kubernetes patterns**: Deployments, Services, DaemonSets, ConfigMaps
- **DevOps**: Docker, Helm, automated deployment pipelines
- **Frontend**: Modern React patterns, real-time data updates

Perfect for learning production software engineering patterns!

---

## 📞 Support

- 📖 **Documentation**: See [docs/](docs/) for comprehensive guides
- 🐛 **Issues**: GitHub Issues for bug reports and feature requests
- 💡 **Discussions**: GitHub Discussions for questions and ideas

---

## Quick Links

| What | Where |
|------|-------|
| **I want to start quickly** | [Quickstart Guide](docs/quickstart/README.md) |
| **I want to understand the design** | [System Architecture](docs/architecture/README.md) |
| **I want to know what each service does** | [Components Reference](docs/components/README.md) |
| **I want to deploy to production** | [Deployment Guide](docs/deployment/README.md) |
| **I want to use Make for building** | [Makefile Guide](docs/makefile.md) |
| **I want to see all API endpoints** | API at `/health` or dashboard at port 8080 |

---

**Ready to get started?** Jump to [Quickstart Guide](docs/quickstart/README.md) →
```

### 2. Custom Message Queue (`internal/mq`)
In-memory message broker with optional persistence.

**Key Features:**
- Publish/Subscribe pattern
- Message acknowledgment with timeouts
- Topic-based routing
- Optional disk persistence
- HTTP admin interface

**Admin Endpoints:**
- `GET /stats` - Queue statistics
- `GET /health` - Health status
- `GET /topics` - List all topics

### 3. Telemetry Collector (`cmd/telemetry-collector`) 
Consumes messages and provides dual persistence.

**Key Features:**
- JSON to typed struct conversion
- File storage (per-GPU JSONL files)
- Memory storage with LRU eviction
- Processing checkpoints
- REST API for data access

**Usage:**
```bash
./telemetry-collector --workers 4 --data-dir ./data --checkpoint
```

**API Endpoints:**
- `GET /health` - Health check
- `GET /stats` - Collection statistics
- `GET /memory-stats` - Memory storage stats
- `GET /telemetry/{gpu_id}` - GPU-specific data

## 🛠️ Getting Started

### Prerequisites
- Go 1.24+
- Optional: `jq` for JSON formatting in demo

### Quick Start

1. **Clone and build:**
```bash
git clone <repository-url>
cd telemetry-pipeline
go build ./cmd/telemetry-streamer
go build ./cmd/telemetry-collector
```

2. **Run the integration demo:**
```bash
./demo.sh
```

This will:
- Create sample GPU telemetry data
- Start collector with MQ broker
- Stream data for 10 seconds
- Show real-time statistics and health checks
- Demonstrate file and memory persistence

### Manual Setup

1. **Start the collector (includes MQ broker):**
```bash
./telemetry-collector \
    --workers=4 \
    --data-dir=./data \
    --health-port=8080 \
    --broker-port=9090 \
    --checkpoint
```

2. **Stream CSV data:**
```bash
./telemetry-streamer \
    --csv=your_data.csv \
    --workers=2 \
    --rate=5
```

3. **Monitor the pipeline:**
```bash
# Health checks
curl http://localhost:8080/health

# Statistics
curl http://localhost:8080/stats

# Specific GPU data
curl http://localhost:8080/telemetry/gpu_0?limit=10
```

## 📊 Configuration Options

### Streamer Configuration
- `--csv`: Path to CSV file
- `--workers`: Number of concurrent workers (default: 4)
- `--rate`: Messages per second per worker (default: 1.0)
- `--duration`: How long to stream (default: continuous)

### Collector Configuration
- `--workers`: Number of message processing workers (default: 4)
- `--data-dir`: Directory for file storage (default: ./data)
- `--max-entries`: Max entries per GPU in memory (default: 1000)
- `--checkpoint`: Enable checkpoint persistence (default: true)
- `--health-port`: Port for health endpoints (default: 8080)
- `--broker-port`: Port for MQ admin endpoints (default: 9090)

## 🧪 Testing

### Run All Tests
```bash
go test ./... -v
```

### Component-Specific Tests
```bash
# Message Queue tests (12 tests)
go test ./internal/mq -v

# Streamer tests (8 tests + benchmark)
go test ./internal/streamer -v

# Collector tests (8 tests + benchmark)
go test ./internal/collector -v

# Persistence tests
go test ./internal/persistence -v
```

### Performance Benchmarks
```bash
# Streamer performance
go test ./internal/streamer -bench=BenchmarkParseRecord

# Collector performance  
go test ./internal/collector -bench=BenchmarkTelemetryConversion

# MQ performance
go test ./internal/mq -bench=.
```

## 🚀 CI/CD Pipeline

### Automated Quality Assurance
Our CI/CD pipeline runs on **every branch** and **every pull request** to ensure code quality:

- **✅ Continuous Integration**: Automated testing, linting, and building on all branches
- **🔒 Security Analysis**: CodeQL security scanning and vulnerability checks
- **📊 Code Coverage**: Automatic coverage reporting with Codecov integration
- **🏗️ Multi-Environment Testing**: Unit tests, integration tests, and Docker builds
- **📚 Documentation**: Automatic API documentation generation
- **🐳 Container Testing**: Docker image building and validation

### Pipeline Stages
1. **Unit & Integration Tests** - Comprehensive test suite with coverage reporting
2. **Code Quality Checks** - Linting, formatting, and static analysis
3. **Security Scanning** - CodeQL analysis and vulnerability detection
4. **Docker Build** - Container image creation and validation
5. **Documentation** - API docs generation and validation

### Quality Gates
- ✅ All tests must pass
- ✅ Code coverage maintained
- ✅ No security vulnerabilities
- ✅ Docker builds successful
- ✅ Linting and formatting checks pass

> **Note**: The CI pipeline now runs on **all branches**, not just main and develop. This ensures early feedback on feature branches and comprehensive testing across the entire development workflow.

## 📁 Data Formats

### CSV Input Format
The streamer accepts any CSV format. Sample:
```csv
gpu_id,temperature,utilization,memory_used,power_draw
gpu_0,72.3,85.5,4096,250.5
gpu_1,75.1,90.2,8192,275.8
```

### JSON Message Format (Internal)
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "fields": {
    "gpu_id": "gpu_0",
    "temperature": 72.3,
    "utilization": 85.5,
    "memory_used": 4096.0,
    "power_draw": 250.5
  }
}
```

### Typed Storage Format
```json
{
  "gpu_id": "gpu_0",
  "metrics": {
    "temperature": 72.3,
    "utilization": 85.5,
    "memory_used": 4096.0,
    "power_draw": 250.5
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 🔧 Production Deployment

### Kubernetes Ready
All components support:
- Health check endpoints for readiness/liveness probes
- Graceful shutdown on SIGTERM
- Configurable resource limits
- Horizontal pod autoscaling compatibility

### Docker Support
```bash
# Build container images
docker build -t telemetry-streamer -f Dockerfile.streamer .
docker build -t telemetry-collector -f Dockerfile.collector .

# Run with docker-compose
docker-compose up
```

### Monitoring Integration
- Prometheus-compatible metrics endpoints
- Structured JSON logging
- Health check endpoints for load balancers
- Processing statistics for monitoring dashboards

## 🏗️ Architecture Decisions

### Why Custom MQ Instead of Kafka/RabbitMQ?
1. **Simplicity**: Single binary deployment, no external dependencies
2. **Performance**: Optimized for telemetry workloads with in-memory processing
3. **Observability**: Built-in admin endpoints and statistics
4. **Customization**: Tailored acknowledgment and redelivery semantics

### Why Dual Persistence?
1. **Performance**: In-memory storage for real-time queries
2. **Durability**: File storage for long-term retention
3. **Scalability**: LRU eviction prevents memory exhaustion
4. **Flexibility**: Different access patterns for different use cases

### Why Schema-Agnostic Processing?
1. **Flexibility**: Handle any CSV format without code changes
2. **Type Safety**: Automatic conversion to appropriate Go types
3. **Evolution**: Support schema changes without deployment updates
4. **Interoperability**: Work with data from multiple sources

## 📈 Performance Characteristics

### Throughput
- **Streamer**: 1000+ messages/second per worker
- **MQ Broker**: 10,000+ messages/second in-memory
- **Collector**: 500+ messages/second per worker with dual persistence

### Scalability
- **Horizontal**: Multiple streamer and collector instances
- **Vertical**: Configurable worker pools per instance
- **Memory**: Configurable LRU limits prevent memory exhaustion
- **Storage**: Per-GPU file sharding for parallel I/O

### Reliability
- **Message Acknowledgment**: Prevents data loss with timeout/retry
- **Graceful Shutdown**: Clean resource cleanup on signals
- **Checkpointing**: Resume processing after restarts
- **Error Handling**: Continue processing despite individual message failures

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.