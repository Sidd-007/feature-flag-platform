package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/pkg/bucketing"
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

// ConfigService handles environment configuration compilation and distribution
type ConfigService struct {
	repos  *repository.Repositories
	redis  *redis.Client
	logger zerolog.Logger
}

// NewConfigService creates a new config service
func NewConfigService(repos *repository.Repositories, redis *redis.Client, logger zerolog.Logger) *ConfigService {
	return &ConfigService{
		repos:  repos,
		redis:  redis,
		logger: logger.With().Str("service", "config").Logger(),
	}
}

// CompileEnvironmentConfig compiles all flags and segments for an environment
func (s *ConfigService) CompileEnvironmentConfig(ctx context.Context, envID uuid.UUID) (*EnvironmentConfig, error) {
	// Get environment details
	env, err := s.repos.Environment.GetByID(ctx, envID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// Get all flags for the environment
	flags, _, err := s.repos.Flag.List(ctx, envID, 1000, 0) // Get all flags
	if err != nil {
		return nil, fmt.Errorf("failed to get flags: %w", err)
	}

	// TODO: Get all segments for the environment when segments are implemented

	// Convert flags to bucketing format
	flagConfigs := make(map[string]*bucketing.FlagConfig)
	for _, flag := range flags {
		flagConfig := s.convertFlagToBucketingConfig(flag)
		flagConfigs[flag.Key] = flagConfig
	}

	// Convert segments to bucketing format (empty for now)
	segmentConfigs := make(map[string]*bucketing.SegmentConfig)
	// TODO: Implement segment conversion when segments are ready

	// Create environment config
	config := &EnvironmentConfig{
		EnvKey:    env.Key,
		Version:   env.Version,
		Salt:      env.Salt,
		Flags:     flagConfigs,
		Segments:  segmentConfigs,
		UpdatedAt: time.Now(),
	}

	// Generate ETag based on version and update time
	config.ETag = fmt.Sprintf(`"%d-%d"`, config.Version, config.UpdatedAt.Unix())

	return config, nil
}

// PublishEnvironmentConfig publishes config to Redis and increments version
func (s *ConfigService) PublishEnvironmentConfig(ctx context.Context, envID uuid.UUID) (*EnvironmentConfig, error) {
	// Increment environment version first
	err := s.repos.Environment.IncrementVersion(ctx, envID)
	if err != nil {
		return nil, fmt.Errorf("failed to increment environment version: %w", err)
	}

	// Compile the config with new version
	config, err := s.CompileEnvironmentConfig(ctx, envID)
	if err != nil {
		return nil, fmt.Errorf("failed to compile environment config: %w", err)
	}

	// Store in Redis
	err = s.StoreConfigInRedis(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to store config in Redis: %w", err)
	}

	s.logger.Info().
		Str("env_key", config.EnvKey).
		Int("version", config.Version).
		Int("flags_count", len(config.Flags)).
		Int("segments_count", len(config.Segments)).
		Msg("Environment config published")

	return config, nil
}

// GetEnvironmentConfig retrieves config from Redis or compiles if not found
func (s *ConfigService) GetEnvironmentConfig(ctx context.Context, envKey string) (*EnvironmentConfig, error) {
	// Try to load from Redis first
	config, err := s.LoadConfigFromRedis(ctx, envKey)
	if err != nil {
		s.logger.Debug().Err(err).Str("env_key", envKey).Msg("Failed to load config from Redis")
	}

	if config != nil {
		return config, nil
	}

	// Config not found in Redis, need to compile it
	// First, find the environment by key
	env, err := s.repos.Environment.GetByKey(ctx, envKey)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Compile and publish the config
	config, err = s.CompileEnvironmentConfig(ctx, env.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to compile config: %w", err)
	}

	// Store in Redis for future requests
	if err := s.StoreConfigInRedis(ctx, config); err != nil {
		s.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to store compiled config in Redis")
	}

	return config, nil
}

// GetEnvironmentByID retrieves environment by ID (helper method for config handler)
func (s *ConfigService) GetEnvironmentByID(ctx context.Context, envID uuid.UUID) (*repository.Environment, error) {
	return s.repos.Environment.GetByID(ctx, envID)
}

// StoreConfigInRedis stores environment config in Redis
func (s *ConfigService) StoreConfigInRedis(ctx context.Context, config *EnvironmentConfig) error {
	key := s.redisKey(config.EnvKey)

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Store with TTL of 24 hours
	err = s.redis.Set(ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store config in Redis: %w", err)
	}

	s.logger.Debug().Str("env_key", config.EnvKey).Msg("Config stored in Redis")
	return nil
}

// LoadConfigFromRedis loads environment config from Redis
func (s *ConfigService) LoadConfigFromRedis(ctx context.Context, envKey string) (*EnvironmentConfig, error) {
	key := s.redisKey(envKey)

	data, err := s.redis.Get(ctx, key).Result()
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

	s.logger.Debug().Str("env_key", envKey).Msg("Config loaded from Redis")
	return &config, nil
}

// InvalidateEnvironmentConfig removes config from Redis
func (s *ConfigService) InvalidateEnvironmentConfig(ctx context.Context, envKey string) error {
	key := s.redisKey(envKey)
	err := s.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate config: %w", err)
	}

	s.logger.Info().Str("env_key", envKey).Msg("Config invalidated")
	return nil
}

// Helper methods

func (s *ConfigService) redisKey(envKey string) string {
	return fmt.Sprintf("ff:config:%s", envKey)
}

func (s *ConfigService) convertFlagToBucketingConfig(flag *repository.Flag) *bucketing.FlagConfig {
	// Create default variations based on flag type for now
	// TODO: Parse actual variations from database when field is added
	variations := s.createDefaultVariations(flag.Type, flag.DefaultVariation)

	// Parse rules from JSON if available
	var rules []bucketing.Rule
	if flag.RulesJSON != nil {
		if rulesBytes, ok := flag.RulesJSON.([]byte); ok {
			if err := json.Unmarshal(rulesBytes, &rules); err != nil {
				s.logger.Warn().Err(err).Str("flag_key", flag.Key).Msg("Failed to parse rules JSON, using empty rules")
				rules = []bucketing.Rule{}
			}
		} else {
			s.logger.Warn().Str("flag_key", flag.Key).Msg("RulesJSON is not []byte, using empty rules")
			rules = []bucketing.Rule{}
		}
	}

	return &bucketing.FlagConfig{
		Key:               flag.Key,
		Type:              flag.Type,
		Variations:        variations,
		DefaultVariation:  flag.DefaultVariation,
		Rules:             rules,
		Status:            flag.Status,
		TrafficAllocation: 1.0, // Default to 100% traffic
	}
}

// convertSegmentToBucketingConfig will be implemented when segments are ready
// func (s *ConfigService) convertSegmentToBucketingConfig(segment *repository.Segment) *bucketing.SegmentConfig {
//     // TODO: Implement segment conversion
//     return nil
// }

func (s *ConfigService) createDefaultVariations(flagType, defaultVariation string) []bucketing.Variation {
	switch strings.ToLower(flagType) {
	case "boolean":
		return []bucketing.Variation{
			{Key: "true", Name: "True", Value: true, Description: "Flag enabled"},
			{Key: "false", Name: "False", Value: false, Description: "Flag disabled"},
		}
	case "string":
		return []bucketing.Variation{
			{Key: "default", Name: "Default", Value: defaultVariation, Description: "Default value"},
		}
	case "number":
		return []bucketing.Variation{
			{Key: "default", Name: "Default", Value: 0, Description: "Default value"},
		}
	case "json":
		return []bucketing.Variation{
			{Key: "default", Name: "Default", Value: map[string]interface{}{}, Description: "Default JSON value"},
		}
	default:
		return []bucketing.Variation{
			{Key: "default", Name: "Default", Value: defaultVariation, Description: "Default value"},
		}
	}
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
