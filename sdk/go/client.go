package featureflags

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Client provides the main interface for the Feature Flags SDK
type Client struct {
	config     *Config
	httpClient *http.Client
	cache      *Cache
	events     *EventProcessor
	streaming  *StreamingClient
	offline    *OfflineHandler
	evaluator  *Evaluator
	logger     zerolog.Logger

	mu        sync.RWMutex
	closed    bool
	closeChan chan struct{}
}

// Config holds the configuration for the SDK client
type Config struct {
	// Required
	APIKey      string `json:"api_key"`
	Environment string `json:"environment"`

	// Endpoints
	EvaluatorEndpoint string `json:"evaluator_endpoint"`
	EventsEndpoint    string `json:"events_endpoint"`

	// Caching
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	CacheMaxSize int           `json:"cache_max_size"`

	// Streaming
	StreamingEnabled   bool          `json:"streaming_enabled"`
	StreamingReconnect bool          `json:"streaming_reconnect"`
	HeartbeatInterval  time.Duration `json:"heartbeat_interval"`

	// Events
	EventsEnabled       bool          `json:"events_enabled"`
	EventsBatchSize     int           `json:"events_batch_size"`
	EventsFlushInterval time.Duration `json:"events_flush_interval"`

	// Offline
	OfflineEnabled    bool   `json:"offline_enabled"`
	OfflineConfigPath string `json:"offline_config_path"`

	// Timeouts
	EvaluationTimeout time.Duration `json:"evaluation_timeout"`
	HTTPTimeout       time.Duration `json:"http_timeout"`

	// Logging
	LogLevel string         `json:"log_level"`
	Logger   zerolog.Logger `json:"-"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		// Endpoints (will be set based on environment)
		EvaluatorEndpoint: "http://localhost:8081",
		EventsEndpoint:    "http://localhost:8083",

		// Caching
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 1000,

		// Streaming
		StreamingEnabled:   true,
		StreamingReconnect: true,
		HeartbeatInterval:  30 * time.Second,

		// Events
		EventsEnabled:       true,
		EventsBatchSize:     100,
		EventsFlushInterval: 10 * time.Second,

		// Offline
		OfflineEnabled:    true,
		OfflineConfigPath: "./feature-flags-config.json",

		// Timeouts
		EvaluationTimeout: 100 * time.Millisecond,
		HTTPTimeout:       5 * time.Second,

		// Logging
		LogLevel: "info",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if c.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	if c.EvaluatorEndpoint == "" {
		return fmt.Errorf("evaluator endpoint is required")
	}

	if c.EventsEndpoint == "" {
		return fmt.Errorf("events endpoint is required")
	}

	if c.CacheTTL <= 0 {
		c.CacheTTL = 5 * time.Minute
	}

	if c.CacheMaxSize <= 0 {
		c.CacheMaxSize = 1000
	}

	if c.EventsBatchSize <= 0 {
		c.EventsBatchSize = 100
	}

	if c.EventsFlushInterval <= 0 {
		c.EventsFlushInterval = 10 * time.Second
	}

	if c.EvaluationTimeout <= 0 {
		c.EvaluationTimeout = 100 * time.Millisecond
	}

	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 5 * time.Second
	}

	return nil
}

// NewClient creates a new feature flags client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Setup logger
	logger := config.Logger
	if config.Logger.GetLevel() == zerolog.Disabled {
		level, _ := zerolog.ParseLevel(config.LogLevel)
		logger = log.With().Str("component", "feature-flags-sdk").Logger().Level(level)
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
	}

	client := &Client{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		closeChan:  make(chan struct{}),
	}

	// Initialize components
	if err := client.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}

	logger.Info().
		Str("environment", config.Environment).
		Str("evaluator_endpoint", config.EvaluatorEndpoint).
		Bool("cache_enabled", config.CacheEnabled).
		Bool("streaming_enabled", config.StreamingEnabled).
		Bool("events_enabled", config.EventsEnabled).
		Bool("offline_enabled", config.OfflineEnabled).
		Msg("Feature flags client initialized")

	return client, nil
}

// initialize initializes all client components
func (c *Client) initialize() error {
	var err error

	// Initialize cache
	if c.config.CacheEnabled {
		c.cache = NewCache(c.config.CacheMaxSize, c.config.CacheTTL, c.logger)
	}

	// Initialize offline handler
	if c.config.OfflineEnabled {
		c.offline, err = NewOfflineHandler(c.config.OfflineConfigPath, c.logger)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to initialize offline handler")
		}
	}

	// Initialize event processor
	if c.config.EventsEnabled {
		c.events, err = NewEventProcessor(&EventProcessorConfig{
			EventsEndpoint: c.config.EventsEndpoint,
			APIKey:         c.config.APIKey,
			BatchSize:      c.config.EventsBatchSize,
			FlushInterval:  c.config.EventsFlushInterval,
			HTTPClient:     c.httpClient,
		}, c.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize event processor: %w", err)
		}
	}

	// Initialize evaluator
	c.evaluator = NewEvaluator(&EvaluatorConfig{
		EvaluatorEndpoint: c.config.EvaluatorEndpoint,
		Environment:       c.config.Environment,
		APIKey:            c.config.APIKey,
		Timeout:           c.config.EvaluationTimeout,
		HTTPClient:        c.httpClient,
		Cache:             c.cache,
		Offline:           c.offline,
		Events:            c.events,
	}, c.logger)

	// Initialize streaming client
	if c.config.StreamingEnabled {
		c.streaming, err = NewStreamingClient(&StreamingConfig{
			EvaluatorEndpoint: c.config.EvaluatorEndpoint,
			APIKey:            c.config.APIKey,
			Environment:       c.config.Environment,
			Reconnect:         c.config.StreamingReconnect,
			HeartbeatInterval: c.config.HeartbeatInterval,
			Cache:             c.cache,
			Offline:           c.offline,
		}, c.logger)
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to initialize streaming client")
		}
	}

	return nil
}

// Start starts the client and its background services
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	c.logger.Info().Msg("Starting feature flags client")

	// Start event processor
	if c.events != nil {
		if err := c.events.Start(ctx); err != nil {
			return fmt.Errorf("failed to start event processor: %w", err)
		}
	}

	// Start streaming client
	if c.streaming != nil {
		if err := c.streaming.Start(ctx); err != nil {
			c.logger.Warn().Err(err).Msg("Failed to start streaming client")
		}
	}

	c.logger.Info().Msg("Feature flags client started")
	return nil
}

// EvaluateFlag evaluates a feature flag for the given user context
func (c *Client) EvaluateFlag(ctx context.Context, flagKey string, userContext *UserContext, defaultValue interface{}) (*EvaluationResult, error) {
	if c.isClosed() {
		return nil, fmt.Errorf("client is closed")
	}

	if flagKey == "" {
		return nil, fmt.Errorf("flag key is required")
	}

	if userContext == nil {
		userContext = &UserContext{}
	}

	// Use evaluator to get the result
	result, err := c.evaluator.Evaluate(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		c.logger.Warn().
			Err(err).
			Str("flag_key", flagKey).
			Str("user_id", userContext.UserID).
			Msg("Flag evaluation failed")

		// Return default value on error
		return &EvaluationResult{
			FlagKey:     flagKey,
			Value:       defaultValue,
			VariationID: "",
			Reason:      ReasonError,
			Error:       err,
			DefaultUsed: true,
		}, nil
	}

	c.logger.Debug().
		Str("flag_key", flagKey).
		Str("user_id", userContext.UserID).
		Str("variation_id", result.VariationID).
		Interface("value", result.Value).
		Str("reason", string(result.Reason)).
		Msg("Flag evaluated")

	return result, nil
}

// EvaluateBoolFlag evaluates a boolean feature flag
func (c *Client) EvaluateBoolFlag(ctx context.Context, flagKey string, userContext *UserContext, defaultValue bool) (bool, error) {
	result, err := c.EvaluateFlag(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	if boolValue, ok := result.Value.(bool); ok {
		return boolValue, nil
	}

	c.logger.Warn().
		Str("flag_key", flagKey).
		Interface("value", result.Value).
		Msg("Flag value is not boolean, returning default")

	return defaultValue, fmt.Errorf("flag value is not boolean")
}

// EvaluateStringFlag evaluates a string feature flag
func (c *Client) EvaluateStringFlag(ctx context.Context, flagKey string, userContext *UserContext, defaultValue string) (string, error) {
	result, err := c.EvaluateFlag(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	if stringValue, ok := result.Value.(string); ok {
		return stringValue, nil
	}

	c.logger.Warn().
		Str("flag_key", flagKey).
		Interface("value", result.Value).
		Msg("Flag value is not string, returning default")

	return defaultValue, fmt.Errorf("flag value is not string")
}

// EvaluateIntFlag evaluates an integer feature flag
func (c *Client) EvaluateIntFlag(ctx context.Context, flagKey string, userContext *UserContext, defaultValue int) (int, error) {
	result, err := c.EvaluateFlag(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	// Handle different numeric types
	switch v := result.Value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		c.logger.Warn().
			Str("flag_key", flagKey).
			Interface("value", result.Value).
			Msg("Flag value is not integer, returning default")

		return defaultValue, fmt.Errorf("flag value is not integer")
	}
}

// EvaluateFloatFlag evaluates a float feature flag
func (c *Client) EvaluateFloatFlag(ctx context.Context, flagKey string, userContext *UserContext, defaultValue float64) (float64, error) {
	result, err := c.EvaluateFlag(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	// Handle different numeric types
	switch v := result.Value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		c.logger.Warn().
			Str("flag_key", flagKey).
			Interface("value", result.Value).
			Msg("Flag value is not float, returning default")

		return defaultValue, fmt.Errorf("flag value is not float")
	}
}

// EvaluateJSONFlag evaluates a JSON feature flag
func (c *Client) EvaluateJSONFlag(ctx context.Context, flagKey string, userContext *UserContext, defaultValue interface{}) (interface{}, error) {
	result, err := c.EvaluateFlag(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	return result.Value, nil
}

// TrackEvent tracks a custom event
func (c *Client) TrackEvent(ctx context.Context, eventName string, userContext *UserContext, properties map[string]interface{}) error {
	if c.isClosed() {
		return fmt.Errorf("client is closed")
	}

	if c.events == nil {
		c.logger.Debug().Msg("Event tracking is disabled")
		return nil
	}

	if eventName == "" {
		return fmt.Errorf("event name is required")
	}

	if userContext == nil {
		userContext = &UserContext{}
	}

	event := &CustomEvent{
		EventName:  eventName,
		UserID:     userContext.UserID,
		Timestamp:  time.Now(),
		Properties: properties,
	}

	return c.events.TrackEvent(ctx, event)
}

// TrackMetric tracks a metric event
func (c *Client) TrackMetric(ctx context.Context, metricName string, value float64, userContext *UserContext, properties map[string]interface{}) error {
	if c.isClosed() {
		return fmt.Errorf("client is closed")
	}

	if c.events == nil {
		c.logger.Debug().Msg("Event tracking is disabled")
		return nil
	}

	if metricName == "" {
		return fmt.Errorf("metric name is required")
	}

	if userContext == nil {
		userContext = &UserContext{}
	}

	event := &MetricEvent{
		MetricName: metricName,
		Value:      value,
		UserID:     userContext.UserID,
		Timestamp:  time.Now(),
		Properties: properties,
	}

	return c.events.TrackMetric(ctx, event)
}

// Flush flushes any pending events
func (c *Client) Flush(ctx context.Context) error {
	if c.events == nil {
		return nil
	}

	return c.events.Flush(ctx)
}

// GetConfig returns the current client configuration
func (c *Client) GetConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.config
}

// IsOffline returns true if the client is in offline mode
func (c *Client) IsOffline() bool {
	if c.offline == nil {
		return false
	}

	return c.offline.IsOffline()
}

// SetOffline sets the offline mode
func (c *Client) SetOffline(offline bool) {
	if c.offline == nil {
		return
	}

	c.offline.SetOffline(offline)

	c.logger.Info().
		Bool("offline", offline).
		Msg("Offline mode changed")
}

// Close closes the client and all its resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.logger.Info().Msg("Closing feature flags client")

	c.closed = true
	close(c.closeChan)

	// Close streaming client
	if c.streaming != nil {
		c.streaming.Close()
	}

	// Close event processor
	if c.events != nil {
		c.events.Close()
	}

	// Close cache
	if c.cache != nil {
		c.cache.Close()
	}

	// Close offline handler
	if c.offline != nil {
		c.offline.Close()
	}

	c.logger.Info().Msg("Feature flags client closed")
	return nil
}

// isClosed checks if the client is closed
func (c *Client) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

// WaitForReady waits for the client to be ready (cache populated, streaming connected)
func (c *Client) WaitForReady(ctx context.Context) error {
	if c.isClosed() {
		return fmt.Errorf("client is closed")
	}

	// If streaming is enabled, wait for it to be connected
	if c.streaming != nil {
		return c.streaming.WaitForReady(ctx)
	}

	// Otherwise, just ensure cache is populated by doing a dummy evaluation
	_, err := c.evaluator.Evaluate(ctx, "__dummy__", &UserContext{}, false)
	if err != nil {
		c.logger.Debug().Err(err).Msg("Cache population check failed (expected for dummy flag)")
	}

	return nil
}

// Convenience Flag Evaluation Methods

// BooleanFlag evaluates a boolean flag
func (c *Client) BooleanFlag(flagKey string, userContext *UserContext, defaultValue bool) (bool, error) {
	result, err := c.evaluator.Evaluate(context.Background(), flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	if boolValue, ok := result.Value.(bool); ok {
		return boolValue, nil
	}

	return defaultValue, fmt.Errorf("flag value is not a boolean")
}

// StringFlag evaluates a string flag
func (c *Client) StringFlag(flagKey string, userContext *UserContext, defaultValue string) (string, error) {
	result, err := c.evaluator.Evaluate(context.Background(), flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	if stringValue, ok := result.Value.(string); ok {
		return stringValue, nil
	}

	return defaultValue, fmt.Errorf("flag value is not a string")
}

// NumberFlag evaluates a number flag
func (c *Client) NumberFlag(flagKey string, userContext *UserContext, defaultValue float64) (float64, error) {
	result, err := c.evaluator.Evaluate(context.Background(), flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	if numberValue, ok := result.Value.(float64); ok {
		return numberValue, nil
	}

	// Handle integer values
	if intValue, ok := result.Value.(int); ok {
		return float64(intValue), nil
	}

	return defaultValue, fmt.Errorf("flag value is not a number")
}

// JSONFlag evaluates a JSON flag
func (c *Client) JSONFlag(flagKey string, userContext *UserContext, defaultValue interface{}) (interface{}, error) {
	result, err := c.evaluator.Evaluate(context.Background(), flagKey, userContext, defaultValue)
	if err != nil {
		return defaultValue, err
	}

	return result.Value, nil
}

// EvaluateMultiple evaluates multiple flags at once
func (c *Client) EvaluateMultiple(ctx context.Context, flagKeys []string, userContext *UserContext, defaults map[string]interface{}) (map[string]*EvaluationResult, error) {
	return c.evaluator.EvaluateMultiple(ctx, flagKeys, userContext, defaults)
}

// TrackEvent tracks a custom event
func (c *Client) TrackEvent(ctx context.Context, eventName string, userContext *UserContext, properties map[string]interface{}) error {
	if c.events == nil {
		return fmt.Errorf("events are not enabled")
	}

	event := &Event{
		Type:       EventTypeCustom,
		UserID:     userContext.UserID,
		EventName:  eventName,
		Properties: properties,
		Timestamp:  time.Now(),
	}

	return c.events.Track(ctx, event)
}
