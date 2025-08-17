package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/cache"
	"github.com/Sidd-007/feature-flag-platform/pkg/config"
)

// ConfigService manages configuration updates from the control plane
type ConfigService struct {
	cache  *cache.ConfigCache
	nats   *nats.Conn
	config *config.Config
	logger zerolog.Logger

	// HTTP client for polling control plane
	httpClient *http.Client

	// NATS subscriptions
	subscription *nats.Subscription

	// Polling control
	pollInterval time.Duration
	stopChan     chan struct{}
}

// ConfigUpdateMessage represents a configuration update message
type ConfigUpdateMessage struct {
	Type      string                   `json:"type"` // "full_refresh", "incremental", "invalidate"
	EnvKey    string                   `json:"env_key"`
	Version   int                      `json:"version"`
	Config    *cache.EnvironmentConfig `json:"config,omitempty"`
	Timestamp int64                    `json:"timestamp"`
}

// NewConfigService creates a new configuration service
func NewConfigService(configCache *cache.ConfigCache, natsConn *nats.Conn, cfg *config.Config, logger zerolog.Logger) *ConfigService {
	return &ConfigService{
		cache:  configCache,
		nats:   natsConn,
		config: cfg,
		logger: logger.With().Str("service", "config").Logger(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		pollInterval: 30 * time.Second, // Poll every 30 seconds
		stopChan:     make(chan struct{}),
	}
}

// Start begins listening for configuration updates
func (s *ConfigService) Start() error {
	// Subscribe to configuration updates via NATS
	subject := "ff.config.updates"

	var err error
	s.subscription, err = s.nats.Subscribe(subject, s.handleConfigUpdate)
	if err != nil {
		return fmt.Errorf("failed to subscribe to config updates: %w", err)
	}

	// Start HTTP polling goroutine as fallback
	go s.startPolling()

	s.logger.Info().Str("subject", subject).Msg("Subscribed to configuration updates and started polling")
	return nil
}

// Close stops the configuration service
func (s *ConfigService) Close() error {
	// Stop polling
	close(s.stopChan)

	if s.subscription != nil {
		if err := s.subscription.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe from config updates: %w", err)
		}
	}

	s.logger.Info().Msg("Configuration service stopped")
	return nil
}

// GetConfig retrieves configuration for an environment
func (s *ConfigService) GetConfig(ctx context.Context, envKey string) (*cache.EnvironmentConfig, error) {
	return s.cache.GetConfig(ctx, envKey)
}

// InvalidateConfig invalidates configuration for an environment
func (s *ConfigService) InvalidateConfig(envKey string) {
	s.cache.InvalidateConfig(envKey)
}

// GetCacheStats returns cache statistics
func (s *ConfigService) GetCacheStats() cache.CacheStats {
	return s.cache.GetStats()
}

// Private methods

func (s *ConfigService) handleConfigUpdate(msg *nats.Msg) {
	var update ConfigUpdateMessage
	if err := json.Unmarshal(msg.Data, &update); err != nil {
		s.logger.Error().Err(err).Msg("Failed to unmarshal config update message")
		return
	}

	s.logger.Info().
		Str("env_key", update.EnvKey).
		Str("type", update.Type).
		Int("version", update.Version).
		Msg("Received config update")

	switch update.Type {
	case "full_refresh":
		if update.Config != nil {
			s.cache.SetConfig(update.EnvKey, update.Config)
		}
	case "incremental":
		// For incremental updates, we might need to merge with existing config
		// For now, treat as full refresh
		if update.Config != nil {
			s.cache.SetConfig(update.EnvKey, update.Config)
		}
	case "invalidate":
		s.cache.InvalidateConfig(update.EnvKey)
	default:
		s.logger.Warn().Str("type", update.Type).Msg("Unknown config update type")
	}

	// Acknowledge the message
	if err := msg.Ack(); err != nil {
		s.logger.Error().Err(err).Msg("Failed to acknowledge config update message")
	}
}

// startPolling starts HTTP polling of the control plane
func (s *ConfigService) startPolling() {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	s.logger.Info().Dur("interval", s.pollInterval).Msg("Started config polling")

	for {
		select {
		case <-s.stopChan:
			s.logger.Info().Msg("Stopping config polling")
			return
		case <-ticker.C:
			s.pollAllConfigs()
		}
	}
}

// pollAllConfigs polls configurations for all cached environments
func (s *ConfigService) pollAllConfigs() {
	envKeys := s.cache.ListCachedEnvironments()

	for _, envKey := range envKeys {
		if err := s.pollConfig(context.Background(), envKey); err != nil {
			s.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to poll config")
		}
	}
}

// pollConfig polls configuration for a specific environment
func (s *ConfigService) pollConfig(ctx context.Context, envKey string) error {
	// Get current config to check ETag
	currentConfig, _ := s.cache.GetConfig(ctx, envKey)

	// Build request URL
	url := fmt.Sprintf("%s/v1/configs/%s", s.config.ControlPlane.URL, envKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add Authorization header with API key if available
	if s.config.EdgeEvaluator.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.EdgeEvaluator.APIKey)
	}

	// Add If-None-Match header for ETag support
	if currentConfig != nil && currentConfig.ETag != "" {
		req.Header.Set("If-None-Match", currentConfig.ETag)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// New config available
		var config cache.EnvironmentConfig
		if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
			return fmt.Errorf("failed to decode config: %w", err)
		}

		s.cache.SetConfig(envKey, &config)
		s.logger.Info().
			Str("env_key", envKey).
			Int("version", config.Version).
			Str("etag", config.ETag).
			Msg("Updated config from polling")

	case http.StatusNotModified:
		// Config hasn't changed
		s.logger.Debug().Str("env_key", envKey).Msg("Config not modified")

	case http.StatusNotFound:
		// Environment doesn't exist anymore
		s.cache.InvalidateConfig(envKey)
		s.logger.Info().Str("env_key", envKey).Msg("Environment not found, invalidating cache")

	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed for environment %s", envKey)

	default:
		return fmt.Errorf("unexpected response status %d for environment %s", resp.StatusCode, envKey)
	}

	return nil
}

// FetchConfig explicitly fetches configuration for an environment from control plane
func (s *ConfigService) FetchConfig(ctx context.Context, envKey string) error {
	return s.pollConfig(ctx, envKey)
}
