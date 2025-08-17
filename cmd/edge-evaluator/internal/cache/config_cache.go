package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/pkg/bucketing"
)

// EnvironmentConfig represents the configuration for an environment
type EnvironmentConfig struct {
	EnvKey    string                              `json:"env_key"`
	Version   int                                 `json:"version"`
	Salt      string                              `json:"salt"`
	Flags     map[string]*bucketing.FlagConfig    `json:"flags"`
	Segments  map[string]*bucketing.SegmentConfig `json:"segments"`
	UpdatedAt time.Time                           `json:"updated_at"`
	ETag      string                              `json:"etag"`
}

// ConfigCache manages flag configurations in memory and Redis
type ConfigCache struct {
	redis  *redis.Client
	logger zerolog.Logger

	// In-memory cache with read-write mutex for concurrent access
	mu      sync.RWMutex
	configs map[string]*EnvironmentConfig

	// Cache statistics
	stats CacheStats
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	Size        int       `json:"size"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewConfigCache creates a new configuration cache
func NewConfigCache(redisClient *redis.Client, logger zerolog.Logger) *ConfigCache {
	return &ConfigCache{
		redis:   redisClient,
		logger:  logger.With().Str("component", "config_cache").Logger(),
		configs: make(map[string]*EnvironmentConfig),
	}
}

// ConfigLoader interface for loading configs when not in cache
type ConfigLoader interface {
	FetchConfig(ctx context.Context, envKey string) error
}

// GetConfig retrieves configuration for an environment
func (c *ConfigCache) GetConfig(ctx context.Context, envKey string) (*EnvironmentConfig, error) {
	c.mu.RLock()
	config, exists := c.configs[envKey]
	c.mu.RUnlock()

	if exists {
		c.recordHit()
		c.logger.Debug().Str("env_key", envKey).Msg("Config cache hit")
		return config, nil
	}

	c.recordMiss()
	c.logger.Debug().Str("env_key", envKey).Msg("Config cache miss, loading from Redis")

	// Load from Redis
	config, err := c.loadFromRedis(ctx, envKey)
	if err != nil {
		return nil, err
	}

	if config != nil {
		c.setConfig(envKey, config)
	}

	return config, nil
}

// GetConfigWithLoader retrieves configuration with fallback to external loader
func (c *ConfigCache) GetConfigWithLoader(ctx context.Context, envKey string, loader ConfigLoader) (*EnvironmentConfig, error) {
	// Try normal cache lookup first
	config, err := c.GetConfig(ctx, envKey)
	if err != nil || config != nil {
		return config, err
	}

	// Config not found, try to fetch from external source
	c.logger.Debug().Str("env_key", envKey).Msg("Config not found in cache or Redis, trying external fetch")

	if loader != nil {
		if err := loader.FetchConfig(ctx, envKey); err != nil {
			c.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to fetch config from external source")
			return nil, err
		}

		// Try cache again after fetch
		return c.GetConfig(ctx, envKey)
	}

	return nil, nil // Not found
}

// SetConfig updates configuration for an environment
func (c *ConfigCache) SetConfig(envKey string, config *EnvironmentConfig) {
	c.setConfig(envKey, config)

	// Store in Redis asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.storeInRedis(ctx, envKey, config); err != nil {
			c.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to store config in Redis")
		}
	}()
}

// InvalidateConfig removes configuration for an environment
func (c *ConfigCache) InvalidateConfig(envKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.configs[envKey]; exists {
		delete(c.configs, envKey)
		c.recordEviction()
		c.logger.Info().Str("env_key", envKey).Msg("Config invalidated")
	}

	// Remove from Redis asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.redis.Del(ctx, c.redisKey(envKey)).Err(); err != nil {
			c.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to delete config from Redis")
		}
	}()
}

// ListCachedEnvironments returns list of cached environment keys
func (c *ConfigCache) ListCachedEnvironments() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.configs))
	for key := range c.configs {
		keys = append(keys, key)
	}

	return keys
}

// GetStats returns cache statistics
func (c *ConfigCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.configs)
	return stats
}

// WarmupCache preloads configurations for specified environments
func (c *ConfigCache) WarmupCache(ctx context.Context, envKeys []string) error {
	c.logger.Info().Int("count", len(envKeys)).Msg("Starting cache warmup")

	for _, envKey := range envKeys {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if _, err := c.GetConfig(ctx, envKey); err != nil {
				c.logger.Warn().Err(err).Str("env_key", envKey).Msg("Failed to warmup config")
			}
		}
	}

	c.logger.Info().Msg("Cache warmup completed")
	return nil
}

// Private methods

func (c *ConfigCache) setConfig(envKey string, config *EnvironmentConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs[envKey] = config
	c.stats.LastUpdated = time.Now()

	c.logger.Info().
		Str("env_key", envKey).
		Int("version", config.Version).
		Int("flags_count", len(config.Flags)).
		Int("segments_count", len(config.Segments)).
		Msg("Config updated in cache")
}

func (c *ConfigCache) loadFromRedis(ctx context.Context, envKey string) (*EnvironmentConfig, error) {
	key := c.redisKey(envKey)

	data, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to load config from Redis: %w", err)
	}

	var config EnvironmentConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	c.logger.Debug().Str("env_key", envKey).Msg("Config loaded from Redis")
	return &config, nil
}

func (c *ConfigCache) storeInRedis(ctx context.Context, envKey string, config *EnvironmentConfig) error {
	key := c.redisKey(envKey)

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Store with TTL of 1 hour (configs should be refreshed regularly)
	err = c.redis.Set(ctx, key, data, time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store config in Redis: %w", err)
	}

	c.logger.Debug().Str("env_key", envKey).Msg("Config stored in Redis")
	return nil
}

func (c *ConfigCache) redisKey(envKey string) string {
	return fmt.Sprintf("ff:config:%s", envKey)
}

func (c *ConfigCache) recordHit() {
	c.stats.Hits++
}

func (c *ConfigCache) recordMiss() {
	c.stats.Misses++
}

func (c *ConfigCache) recordEviction() {
	c.stats.Evictions++
}

// GetCacheHitRatio returns the cache hit ratio as a percentage
func (c *ConfigCache) GetCacheHitRatio() float64 {
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total) * 100
}
