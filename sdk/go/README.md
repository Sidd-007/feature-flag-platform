# Feature Flags Go SDK

A powerful, production-ready Go SDK for the Feature Flag & Experimentation Platform.

## Features

- 🚀 **High Performance**: Local caching with sub-millisecond flag evaluation
- 🔄 **Real-time Updates**: Streaming config updates via Server-Sent Events
- 📊 **Analytics**: Automatic exposure tracking and custom event reporting
- 🛡️ **Resilient**: Offline mode with local configuration fallback
- 🎯 **Advanced Targeting**: User segmentation and percentage rollouts
- 🧪 **A/B Testing**: Built-in experimentation with statistical analysis
- 📈 **Sticky Bucketing**: Consistent user experiences across sessions
- ⚡ **Thread-Safe**: Concurrent flag evaluation without locks

## Installation

```bash
go get github.com/Sidd-007/feature-flag-platform/sdk/go
```

## Quick Start

### 1. Get Your API Key

First, create an API token for your environment:

```bash
# Using curl to create an API token
curl -X POST http://localhost:8080/v1/orgs/{orgId}/projects/{projectId}/environments/{envId}/tokens \
  -H "Authorization: Bearer YOUR_USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Go SDK Token",
    "description": "API token for Go SDK integration",
    "scope": "read"
  }'
```

### 2. Initialize the SDK

```go
package main

import (
    "context"
    "log"
    "time"

    featureflags "github.com/Sidd-007/feature-flag-platform/sdk/go"
)

func main() {
    // Create SDK configuration
    config := &featureflags.Config{
        APIKey:      "ff_your_api_key_here", // Your API token
        Environment: "your-env-key",         // Environment key

        // Edge evaluator endpoint
        EvaluatorEndpoint: "http://localhost:8081",
        EventsEndpoint:    "http://localhost:8083",

        // Optional configurations
        CacheEnabled:      true,
        StreamingEnabled:  true,
        EventsEnabled:     true,
        EvaluationTimeout: 200 * time.Millisecond,
    }

    // Create and start the client
    client, err := featureflags.NewClient(config)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

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
            "beta_user": true,
        },
    }

    // Evaluate flags
    showFeature, _ := client.BooleanFlag("new-feature", user, false)
    if showFeature {
        // Show new feature
    }
}
```

### 3. Environment Setup

Make sure your services are running:

```bash
# Start infrastructure services
docker-compose -f deploy/docker-compose.yml up -d

# Start control plane
go run ./cmd/control-plane

# Start edge evaluator
go run ./cmd/edge-evaluator
```

## Configuration

### SDK Configuration Options

```go
type Config struct {
    // Required
    APIKey      string // API token from your environment
    Environment string // Environment key (e.g., "development", "production")

    // Endpoints
    EvaluatorEndpoint string // Edge evaluator URL (default: "http://localhost:8081")
    EventsEndpoint    string // Event ingestor URL (default: "http://localhost:8083")

    // Caching
    CacheEnabled bool          // Enable local caching (default: true)
    CacheTTL     time.Duration // Cache time-to-live (default: 5m)
    CacheMaxSize int           // Maximum cache entries (default: 1000)

    // Streaming
    StreamingEnabled   bool          // Enable real-time updates (default: true)
    StreamingReconnect bool          // Auto-reconnect on disconnect (default: true)
    HeartbeatInterval  time.Duration // Heartbeat interval (default: 30s)

    // Events
    EventsEnabled       bool          // Enable event tracking (default: true)
    EventsBatchSize     int           // Event batch size (default: 100)
    EventsFlushInterval time.Duration // Event flush interval (default: 10s)

    // Offline Mode
    OfflineEnabled    bool   // Enable offline fallback (default: true)
    OfflineConfigPath string // Local config file path (default: "./feature-flags-config.json")

    // Timeouts
    EvaluationTimeout time.Duration // Flag evaluation timeout (default: 100ms)
    HTTPTimeout       time.Duration // HTTP request timeout (default: 5s)

    // Logging
    LogLevel string         // Log level: "debug", "info", "warn", "error"
    Logger   zerolog.Logger // Custom logger instance
}
```

### Environment Variables

You can also configure the SDK using environment variables:

```bash
export FF_API_KEY="ff_your_api_key_here"
export FF_ENVIRONMENT="development"
export FF_EVALUATOR_ENDPOINT="http://localhost:8081"
export FF_EVENTS_ENDPOINT="http://localhost:8083"
export FF_CACHE_ENABLED="true"
export FF_STREAMING_ENABLED="true"
export FF_LOG_LEVEL="info"
```

## Flag Evaluation

### Boolean Flags

```go
enabled, err := client.BooleanFlag("feature-enabled", userContext, false)
if err != nil {
    // Handle error - default value is returned
}

if enabled {
    // Feature is enabled for this user
}
```

### String Flags

```go
theme, err := client.StringFlag("ui-theme", userContext, "light")
// Returns: "dark", "light", "auto", etc.
```

### Number Flags

```go
maxItems, err := client.NumberFlag("items-per-page", userContext, 10)
// Returns: 10, 25, 50, 100, etc.
```

### JSON Flags

```go
config, err := client.JSONFlag("app-config", userContext, defaultConfig)
if configMap, ok := config.(map[string]interface{}); ok {
    theme := configMap["theme"]
    features := configMap["features"]
}
```

### Multiple Flags

Evaluate multiple flags in a single request for better performance:

```go
flagKeys := []string{"feature-a", "feature-b", "theme"}
defaults := map[string]interface{}{
    "feature-a": false,
    "feature-b": false,
    "theme":     "light",
}

results, err := client.EvaluateMultiple(ctx, flagKeys, userContext, defaults)
for flagKey, result := range results {
    fmt.Printf("%s: %v (reason: %s)\n", flagKey, result.Value, result.Reason)
}
```

## User Context

Provide rich user context for precise targeting:

```go
userContext := &featureflags.UserContext{
    UserID:    "user-123",
    SessionID: "session-456",
    Email:     "user@example.com",
    Name:      "John Doe",

    // Geographic data
    Country: "US",
    Region:  "CA",
    City:    "San Francisco",

    // Technical data
    Platform:  "web",
    Version:   "1.2.3",
    UserAgent: "Mozilla/5.0...",
    IPAddress: "192.168.1.1",

    // Custom attributes for targeting
    Attributes: map[string]interface{}{
        "plan":           "premium",
        "signup_date":    "2023-01-15",
        "beta_user":      true,
        "feature_flags":  []string{"early_access"},
        "experiment_id":  "exp-001",
        "cohort":         "power_users",
        "ltv":            299.99,
    },

    // User groups for targeting
    Groups: []string{"beta_testers", "premium_users"},
}
```

## Event Tracking

### Automatic Exposure Tracking

The SDK automatically tracks flag exposure events when flags are evaluated:

```go
// This automatically tracks an exposure event
enabled, _ := client.BooleanFlag("new-feature", userContext, false)
```

### Custom Events

Track custom business events for analysis:

```go
// Track a conversion event
err := client.TrackEvent(ctx, &featureflags.Event{
    Type:      featureflags.EventTypeCustom,
    UserID:    userContext.UserID,
    EventName: "purchase_completed",
    Properties: map[string]interface{}{
        "amount":      99.99,
        "currency":    "USD",
        "product_id":  "prod-123",
        "experiment":  "checkout-flow-v2",
    },
})
```

### Metric Events

Track metrics for A/B test analysis:

```go
// Track a numeric metric
err := client.TrackMetric(ctx, &featureflags.MetricEvent{
    UserID:     userContext.UserID,
    MetricName: "page_load_time",
    Value:      1.23, // seconds
    Unit:       "seconds",
    Properties: map[string]interface{}{
        "page": "homepage",
        "cdn":  "cloudflare",
    },
})
```

## Advanced Features

### Streaming Updates

Enable real-time flag updates without restarting your application:

```go
config := &featureflags.Config{
    // ... other config
    StreamingEnabled:   true,
    StreamingReconnect: true,
    HeartbeatInterval:  30 * time.Second,
}
```

### Offline Mode

Configure offline fallback for network resilience:

```go
config := &featureflags.Config{
    // ... other config
    OfflineEnabled:    true,
    OfflineConfigPath: "./flags-config.json",
}
```

Create a local configuration file:

```json
{
  "flags": {
    "new-feature": {
      "key": "new-feature",
      "enabled": true,
      "default_value": false,
      "variations": [
        { "key": "on", "value": true },
        { "key": "off", "value": false }
      ]
    }
  }
}
```

### Error Handling

The SDK is designed for graceful degradation:

```go
enabled, err := client.BooleanFlag("feature", userContext, false)
if err != nil {
    // Log the error but continue with default value
    log.Printf("Flag evaluation failed: %v", err)
    // 'enabled' will be the default value (false)
}

// Your application continues normally
if enabled {
    // Show feature
}
```

### Performance Optimization

#### Caching

Local caching reduces latency and API calls:

```go
config := &featureflags.Config{
    CacheEnabled: true,
    CacheTTL:     5 * time.Minute,
    CacheMaxSize: 1000,
}
```

#### Batch Evaluation

Evaluate multiple flags in one request:

```go
// Instead of multiple individual calls
feature1, _ := client.BooleanFlag("feature-1", user, false)
feature2, _ := client.BooleanFlag("feature-2", user, false)
feature3, _ := client.BooleanFlag("feature-3", user, false)

// Use batch evaluation
results, _ := client.EvaluateMultiple(ctx,
    []string{"feature-1", "feature-2", "feature-3"},
    user,
    map[string]interface{}{
        "feature-1": false,
        "feature-2": false,
        "feature-3": false,
    })
```

## Examples

See the `examples/` directory for complete examples:

- [Basic Usage](examples/basic-usage/main.go) - Simple flag evaluation
- [Web Application](examples/web-app/main.go) - HTTP server integration
- [Worker Service](examples/worker/main.go) - Background service usage
- [A/B Testing](examples/ab-testing/main.go) - Experimentation patterns

## Production Deployment

### Environment Configuration

```bash
# Production environment variables
export FF_API_KEY="ff_prod_key_abc123..."
export FF_ENVIRONMENT="production"
export FF_EVALUATOR_ENDPOINT="https://flags.yourcompany.com"
export FF_EVENTS_ENDPOINT="https://events.yourcompany.com"
export FF_CACHE_ENABLED="true"
export FF_STREAMING_ENABLED="true"
export FF_LOG_LEVEL="warn"
```

### Health Checks

Monitor SDK health in your application:

```go
// Check if client is healthy
if client.IsHealthy() {
    // SDK is operational
} else {
    // SDK may be in offline mode or experiencing issues
}

// Get detailed status
status := client.GetStatus()
fmt.Printf("Cache hit ratio: %.2f%%\n", status.CacheHitRatio)
fmt.Printf("Streaming connected: %t\n", status.StreamingConnected)
fmt.Printf("Events queue size: %d\n", status.EventsQueueSize)
```

### Metrics and Monitoring

The SDK provides metrics for monitoring:

- Flag evaluation latency
- Cache hit/miss ratio
- Network request success/failure rates
- Event processing metrics
- Streaming connection status

Integrate with your monitoring system:

```go
import "github.com/prometheus/client_golang/prometheus"

// Custom metrics collection
client.OnFlagEvaluated(func(result *featureflags.EvaluationResult) {
    evaluationDuration.Observe(float64(result.Duration.Nanoseconds()) / 1e6)
    if result.CacheHit {
        cacheHits.Inc()
    } else {
        cacheMisses.Inc()
    }
})
```

## Best Practices

### 1. Use Meaningful Flag Names

```go
// Good
enabled := client.BooleanFlag("checkout-redesign-v2", user, false)
theme := client.StringFlag("mobile-app-theme", user, "light")

// Avoid
enabled := client.BooleanFlag("flag1", user, false)
theme := client.StringFlag("f2", user, "light")
```

### 2. Provide Sensible Defaults

```go
// Good - safe defaults
showPremiumFeatures := client.BooleanFlag("premium-features", user, false)
maxRetries := client.NumberFlag("api-retry-count", user, 3)

// Risky - could break functionality
showPremiumFeatures := client.BooleanFlag("premium-features", user, true)
maxRetries := client.NumberFlag("api-retry-count", user, 0)
```

### 3. Handle Errors Gracefully

```go
feature, err := client.BooleanFlag("new-feature", user, false)
if err != nil {
    // Log for debugging but don't crash
    logger.Warn("Flag evaluation failed", "error", err, "flag", "new-feature")
    // Use default value and continue
}

if feature {
    // Feature logic
}
```

### 4. Use Rich User Context

```go
// Include relevant targeting attributes
user := &featureflags.UserContext{
    UserID: userID,
    Email:  email,
    Attributes: map[string]interface{}{
        "subscription_tier": tier,
        "signup_date":       signupDate,
        "country":          country,
        "app_version":      appVersion,
        "platform":         "ios",
    },
}
```

### 5. Batch Related Evaluations

```go
// Evaluate related flags together
uiFlags := []string{"dark-mode", "new-navigation", "beta-features"}
uiConfig, _ := client.EvaluateMultiple(ctx, uiFlags, user, uiDefaults)
```

### 6. Cache User Context

```go
// Cache user context for multiple evaluations
user := buildUserContext(userID)

// Use same context for multiple flags
feature1, _ := client.BooleanFlag("feature-1", user, false)
feature2, _ := client.BooleanFlag("feature-2", user, false)
config, _ := client.JSONFlag("app-config", user, defaultConfig)
```

## Troubleshooting

### Common Issues

#### 1. Authentication Errors

```
Error: authentication failed for environment
```

**Solution**: Verify your API key and environment:

```go
// Check your configuration
config := &featureflags.Config{
    APIKey:      "ff_your_actual_key_here", // Must start with "ff_"
    Environment: "development",              // Must match your environment key
}
```

#### 2. Network Timeouts

```
Error: evaluation request failed: context deadline exceeded
```

**Solution**: Increase timeout or check network connectivity:

```go
config := &featureflags.Config{
    EvaluationTimeout: 500 * time.Millisecond, // Increase timeout
    HTTPTimeout:       10 * time.Second,
}
```

#### 3. Flag Not Found

```
Error: flag not found
```

**Solution**: Verify flag exists and is published:

```bash
# Check if flag exists in your environment
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/v1/orgs/{orgId}/projects/{projectId}/environments/{envId}/flags

# Publish flag configuration
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/v1/orgs/{orgId}/projects/{projectId}/environments/{envId}/flags/{flagKey}/publish
```

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
config := &featureflags.Config{
    LogLevel: "debug",
}
```

### Health Checks

Check SDK status:

```go
status := client.GetStatus()
fmt.Printf("Healthy: %t\n", status.Healthy)
fmt.Printf("Cache entries: %d\n", status.CacheSize)
fmt.Printf("Last evaluation: %v\n", status.LastEvaluation)
```

## License

This SDK is part of the Feature Flag & Experimentation Platform.

## Support

- 📖 [Documentation](https://github.com/Sidd-007/feature-flag-platform/docs)
- 🐛 [Issue Tracker](https://github.com/Sidd-007/feature-flag-platform/issues)
- 💬 [Discord Community](https://discord.gg/feature-flags)
- 📧 [Email Support](mailto:support@featureflags.com)
