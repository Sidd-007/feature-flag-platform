# Feature Flag & Experimentation Platform

A production-grade feature flag and experimentation platform built with Go, designed for high performance, reliability, and scalability.

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Admin UI      │    │   Control       │    │   Edge          │
│   (Next.js)     │◄──►│   Plane API     │◄──►│   Evaluator     │
│                 │    │   (Go)          │    │   (Go)          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Event         │    │   Analytics     │    │   SDKs          │
│   Ingestor      │◄──►│   Engine        │    │   (Go/Node)     │
│   (Go)          │    │   (Go)          │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                        │
        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐
│   NATS          │    │   ClickHouse    │
│   JetStream     │    │   Analytics     │
└─────────────────┘    └─────────────────┘
        │
        ▼
┌─────────────────┐    ┌─────────────────┐
│   PostgreSQL    │    │   Redis Cache   │
│   Control DB    │    │                 │
└─────────────────┘    └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### Local Development

1. **Clone and setup**

   ```bash
   git clone <repository-url>
   cd feature-flag-platform
   make install
   ```

2. **Start development environment**

   ```bash
   make up
   ```

3. **Run database migrations**

   ```bash
   make migrate-up
   ```

4. **Seed demo data**

   ```bash
   make seed
   ```

5. **Access the platform**
   - Admin UI: http://localhost:3000
   - Control Plane API: http://localhost:8080
   - Edge Evaluator: http://localhost:8081
   - Event Ingestor: http://localhost:8082
   - Analytics Engine: http://localhost:8083

## 📊 Core Components

### Control Plane API

- **Purpose**: CRUD operations for organizations, projects, environments, flags, segments, and experiments
- **Features**: Authentication, authorization (RBAC), config publishing, audit logging
- **Port**: 8080

### Edge Evaluator

- **Purpose**: High-performance flag evaluation with sub-10ms latency
- **Features**: In-memory rule engine, real-time config updates, local caching
- **Port**: 8081

### Event Ingestor

- **Purpose**: Collects exposure and metric events from SDKs
- **Features**: High-throughput ingestion, validation, batching to analytics storage
- **Port**: 8082

### Analytics Engine

- **Purpose**: Experiment analysis and statistical computations
- **Features**: A/B test results, statistical significance, CUPED support
- **Port**: 8083

### Admin UI

- **Purpose**: Web interface for managing flags and experiments
- **Features**: Flag editor, experiment designer, real-time dashboards
- **Port**: 3000

## 🔧 Configuration

Configuration is handled via environment variables and YAML files. See `.env.example` for all available options.

Key environment variables:

```bash
# Database
FF_DATABASE_HOST=localhost
FF_DATABASE_NAME=feature_flags
FF_DATABASE_USER=postgres
FF_DATABASE_PASSWORD=password

# Redis
FF_REDIS_HOST=localhost
FF_REDIS_PORT=6379

# NATS
FF_NATS_URL=nats://localhost:4222

# Authentication
FF_AUTH_JWT_SECRET=your-secret-key
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Generate coverage report
make test-coverage
```

## 📈 Performance Targets

- **Edge Evaluator**: p99 < 10ms @ 2k RPS per pod
- **Control Plane**: p95 < 150ms for CRUD operations
- **Event Ingestion**: 10k events/sec with ≤ 1min freshness
- **Config Propagation**: < 5s edge convergence

## 🔐 Security

- **Authentication**: JWT tokens, OIDC integration, bcrypt password hashing
- **Authorization**: Role-based access control (RBAC) with Casbin
- **Data Protection**: PII hashing, encrypted connections, secret rotation
- **API Security**: Rate limiting, CORS, request validation

## 🗄️ Data Model

### Core Entities

- **Organization**: Top-level tenant isolation
- **Project**: Groups related flags and experiments
- **Environment**: Deployment stage (dev, staging, prod)
- **Flag**: Feature flag with targeting rules
- **Segment**: User targeting segments
- **Experiment**: A/B test configuration
- **Metric**: Success metrics for experiments

### Analytics Schema

- **events_exposure**: Flag evaluation events
- **events_metric**: Custom metric events
- **experiments_snapshot**: Pre-aggregated experiment data

## 🔄 Rule Evaluation

### Bucketing Algorithm

- Uses SHA-256 deterministic hashing
- Consistent assignment across restarts
- 10,000 buckets for precise percentage control
- Formula: `bucket = SHA256(env_salt + flag_key + user_key) % 10000`

### Rule DSL

```json
{
  "if": {
    "and": [
      { "attribute": "user_id", "operator": "in", "value": ["user1", "user2"] },
      { "attribute": "country", "operator": "eq", "value": "US" }
    ]
  },
  "then": {
    "rollout": {
      "variations": [
        { "key": "control", "weight": 50 },
        { "key": "treatment", "weight": 50 }
      ]
    }
  }
}
```

## 📊 Experimentation

### Statistical Methods

- **Binary metrics**: Chi-square test
- **Continuous metrics**: Welch's t-test
- **Optional**: Sequential monitoring (Pocock boundaries)
- **Variance reduction**: CUPED support

### Results Interpretation

- Lift calculation with confidence intervals
- Statistical significance (p-values)
- Minimum detectable effect (MDE)
- Guardrail monitoring

## 🛠️ Available Commands

```bash
# Development
make dev          # Start development environment
make down         # Stop development environment
make clean        # Clean up containers and volumes

# Building
make build        # Build all services
make build-docker # Build Docker images

# Testing
make test         # Run all tests
make lint         # Run linter
make fmt          # Format code

# Database
make migrate-up   # Run migrations
make migrate-down # Rollback migrations
make seed         # Seed demo data

# Documentation
make docs-generate # Generate API docs
make proto-generate # Generate protobuf code

# Monitoring
make logs         # Show service logs
make ps           # Show running containers
```

## 🔧 SDK Usage

### Go SDK

```go
import "github.com/feature-flag-platform/sdk/go"

client := ff.NewClient(ff.Config{
    EnvKey:   "your-env-key",
    Endpoint: "http://localhost:8081",
})

// Boolean flag
enabled := client.IsEnabled("new-feature", user)

// Multivariate flag
variation, _ := client.Variation("button-color", user, "blue")

// Track metrics
client.TrackMetric("conversion", 1.0, user)
```

### Node.js SDK

```javascript
import { FeatureFlagClient } from "@feature-flag-platform/sdk";

const client = new FeatureFlagClient({
  envKey: "your-env-key",
  endpoint: "http://localhost:8081",
});

// Boolean flag
const enabled = await client.isEnabled("new-feature", user);

// Multivariate flag
const variation = await client.variation("button-color", user, "blue");

// Track metrics
await client.trackMetric("conversion", 1.0, user);
```

## 📚 API Documentation

- **OpenAPI Spec**: Available at `/api/docs` when running locally
- **gRPC**: Protocol buffer definitions in `/proto`

## 🚀 Deployment

### Local Development

```bash
make up    # Docker Compose
```

### Production

- **Kubernetes**: Manifests in `/deploy/k8s`
- **Helm**: Charts available for customization
- **Dependencies**: PostgreSQL, Redis, NATS, ClickHouse

## 📋 Roadmap

- [ ] Mobile SDK support (iOS/Android)
- [ ] Visual experiment editor
- [ ] Advanced segmentation (SQL-based)
- [ ] Multi-region deployment
- [ ] CDP integrations
- [ ] Machine learning recommendations

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make ci` to verify
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

- **Documentation**: [docs/](./docs/)
- **Issues**: GitHub Issues
- **Discord**: [Feature Flag Platform Discord](#)

## 🔧 Troubleshooting

### Common Issues

**Services won't start**

```bash
make down && make clean && make up
```

**Database migration errors**

```bash
make migrate-down && make migrate-up
```

**Permission denied errors**

```bash
sudo chown -R $USER:$USER .
```

**Out of memory**

```bash
docker system prune -f
```

For more detailed troubleshooting, see [docs/troubleshooting.md](./docs/troubleshooting.md).
