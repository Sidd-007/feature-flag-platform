# Feature Flag Platform - Quick Start Guide

Get your feature flag platform up and running in under 10 minutes.

## Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Node.js 18+ (for Admin UI)
- PostgreSQL 15+
- Redis 7+

## 1. Start Infrastructure Services

```bash
# Clone the repository
git clone https://github.com/feature-flag-platform/platform
cd platform

# Start PostgreSQL, Redis, NATS, and ClickHouse
docker-compose -f deploy/docker-compose.yml up -d

# Verify services are running
docker-compose -f deploy/docker-compose.yml ps
```

## 2. Set Up Environment

```bash
# Copy environment configuration
cp config.env.example .env

# Edit .env and set your JWT secret
# FF_AUTH_JWT_SECRET=your-super-secret-jwt-key-at-least-32-chars
```

## 3. Run Database Migrations

```bash
# Connect to PostgreSQL and run migrations
docker exec -i ff-postgres psql -U postgres -d feature_flags < db/migrations/postgres/001_initial_schema.up.sql
docker exec -i ff-postgres psql -U postgres -d feature_flags < db/migrations/postgres/002_api_tokens_update.up.sql

# Run ClickHouse migrations
docker exec -i ff-clickhouse clickhouse-client --database feature_flags < db/migrations/clickhouse/001_analytics_schema.up.sql
```

## 4. Start Core Services

### Control Plane (Port 8080)

```bash
# In terminal 1
go run ./cmd/control-plane
```

### Edge Evaluator (Port 8081)

```bash
# In terminal 2
go run ./cmd/edge-evaluator
```

### Event Ingestor (Port 8083)

```bash
# In terminal 3
go run ./cmd/event-ingestor
```

## 5. Start Admin UI (Port 3000)

```bash
# In terminal 4
cd web/admin
npm install
npm run dev
```

## 6. Initial Setup via Admin UI

1. **Open Admin UI**: http://localhost:3000
2. **Login**:
   - Email: `test@example.com`
   - Password: `password`
3. **Create Organization**: Click "Create Organization"
   - Name: "My Company"
   - The slug will be auto-generated
4. **Create Project**: Click "Manage" on your organization
   - Name: "My App"
   - Key: "my-app"
5. **Create Environment**:
   - Name: "Development"
   - Key: "development"
6. **Create Feature Flag**:
   - Key: "new-feature"
   - Name: "New Feature Toggle"
   - Type: "boolean"
   - Default Value: `false`
   - Click "Create Flag"
7. **Publish Configuration**: Click "Publish" on your flag

## 7. Create API Token

```bash
# Get your auth token first (login via UI and check browser dev tools for the token)
# Or use the API directly:

curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password"}'

# Use the returned access_token to create an API token:
curl -X POST http://localhost:8080/v1/orgs/{ORG_ID}/projects/{PROJECT_ID}/environments/{ENV_ID}/tokens \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Development API Key",
    "description": "API key for development environment",
    "scope": "read"
  }'

# Save the returned "plain_key" - this is your API token!
```

## 8. Test with Go SDK

Create a test Go application:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	featureflags "github.com/feature-flag-platform/sdk/go"
)

func main() {
	// Configure the SDK
	config := &featureflags.Config{
		APIKey:      "ff_your_api_key_here", // Replace with your API key
		Environment: "development",          // Your environment key

		// Local endpoints
		EvaluatorEndpoint: "http://localhost:8081",
		EventsEndpoint:    "http://localhost:8083",

		// Optional optimizations
		CacheEnabled:      true,
		StreamingEnabled:  true,
		EventsEnabled:     true,
		EvaluationTimeout: 200 * time.Millisecond,
	}

	// Create client
	client, err := featureflags.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Start client
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Create user context
	user := &featureflags.UserContext{
		UserID: "user-123",
		Email:  "user@example.com",
		Attributes: map[string]interface{}{
			"plan": "premium",
		},
	}

	// Evaluate feature flag
	enabled, err := client.BooleanFlag("new-feature", user, false)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Printf("Feature 'new-feature' is enabled: %t\n", enabled)

	if enabled {
		fmt.Println("🎉 New feature is ON!")
	} else {
		fmt.Println("📴 New feature is OFF")
	}
}
```

Run your test application:

```bash
# Initialize go module
go mod init test-flags
go mod tidy

# Run the test
go run main.go
```

## 9. Toggle Your Flag

1. Go back to the Admin UI: http://localhost:3000
2. Navigate to your flag
3. Toggle the flag on/off
4. Re-run your Go application to see the change!

## 10. View Analytics (Coming Soon)

Analytics data is being collected in ClickHouse. Access the analytics engine at:

```bash
# Start analytics engine (Port 8084)
go run ./cmd/analytics-engine
```

## API Endpoints Summary

| Service          | Port | Purpose                    | Example Endpoint            |
| ---------------- | ---- | -------------------------- | --------------------------- |
| Control Plane    | 8080 | Flag management, user auth | `POST /v1/auth/login`       |
| Edge Evaluator   | 8081 | Flag evaluation            | `POST /v1/evaluate`         |
| Event Ingestor   | 8083 | Event collection           | `POST /v1/events`           |
| Analytics Engine | 8084 | Analytics queries          | `GET /v1/analytics/metrics` |
| Admin UI         | 3000 | Web interface              | http://localhost:3000       |

## Configuration Endpoints

```bash
# Health check
curl http://localhost:8080/health

# Get environment config (used by evaluators)
curl -H "Authorization: Bearer API_KEY" \
  http://localhost:8080/v1/configs/development

# Evaluate flags
curl -X POST http://localhost:8081/v1/evaluate \
  -H "Authorization: Bearer API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "env_key": "development",
    "context": {
      "user_key": "user-123",
      "attributes": {"plan": "premium"}
    }
  }'
```

## Common Issues & Solutions

### 1. Database Connection Failed

```bash
# Check if PostgreSQL is running
docker-compose -f deploy/docker-compose.yml ps

# Check logs
docker-compose -f deploy/docker-compose.yml logs postgres
```

### 2. JWT Secret Error

```bash
# Ensure .env file has a strong JWT secret
echo "FF_AUTH_JWT_SECRET=$(openssl rand -base64 32)" >> .env
```

### 3. API Token Invalid

- Ensure you're using the correct API token format (starts with `ff_`)
- Check that the token scope matches your usage (read/write)
- Verify the token hasn't expired

### 4. Flag Not Found

- Ensure flag is created and published via Admin UI
- Check that environment key matches your SDK configuration
- Verify API token has access to the correct environment

### 5. Evaluation Timeout

- Check that Edge Evaluator is running on port 8081
- Increase timeout in SDK configuration
- Verify network connectivity

## Next Steps

1. **Explore Advanced Features**:

   - User segmentation and targeting rules
   - A/B testing and experimentation
   - Real-time streaming updates
   - Custom event tracking

2. **Production Deployment**:

   - Set up proper secrets management
   - Configure load balancing
   - Set up monitoring and alerting
   - Implement proper security policies

3. **Integrate with Your Application**:

   - Add feature flags to your existing codebase
   - Set up CI/CD integration
   - Configure different environments (staging, production)
   - Implement gradual rollouts

4. **Monitoring & Analytics**:
   - Set up dashboards for flag usage
   - Monitor performance metrics
   - Analyze A/B test results
   - Track business impact

## Help & Support

- 📖 **Documentation**: See individual service READMEs
- 🐛 **Issues**: Check the issue tracker
- 💬 **Community**: Join our Discord
- 📧 **Support**: Contact the team

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Admin UI      │    │  Your App +     │    │   Analytics     │
│   (Port 3000)   │    │   Go SDK        │    │   (Port 8084)   │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Control Plane   │    │ Edge Evaluator  │    │ Event Ingestor  │
│ (Port 8080)     │    │ (Port 8081)     │    │ (Port 8083)     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                        ┌─────────▼───────┐
                        │  Infrastructure │
                        │ PostgreSQL,     │
                        │ Redis,          │
                        │ NATS,           │
                        │ ClickHouse      │
                        └─────────────────┘
```

You now have a fully functional feature flag platform! Start experimenting with flags and see how they can improve your development workflow. 🚀
