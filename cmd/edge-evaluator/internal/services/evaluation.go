package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/cache"
	"github.com/Sidd-007/feature-flag-platform/pkg/bucketing"
)

// EvaluationService handles flag evaluation
type EvaluationService struct {
	cache        *cache.ConfigCache
	bucketer     *bucketing.Bucketer
	configLoader cache.ConfigLoader
	eventService *EventService
	logger       zerolog.Logger
}

// EvaluationRequest represents a flag evaluation request
type EvaluationRequest struct {
	EnvKey        string             `json:"env_key"`
	FlagKeys      []string           `json:"flag_keys,omitempty"` // If empty, evaluate all flags
	Context       *bucketing.Context `json:"context"`
	IncludeReason bool               `json:"include_reason,omitempty"`
}

// EvaluationResponse represents the response containing evaluated flags
type EvaluationResponse struct {
	Flags         map[string]*bucketing.EvaluationResult `json:"flags"`
	ConfigVersion int                                    `json:"config_version"`
	EvaluatedAt   time.Time                              `json:"evaluated_at"`
	RequestID     string                                 `json:"request_id,omitempty"`
}

// NewEvaluationService creates a new evaluation service
func NewEvaluationService(configCache *cache.ConfigCache, bucketer *bucketing.Bucketer, configLoader cache.ConfigLoader, eventService *EventService, logger zerolog.Logger) *EvaluationService {
	return &EvaluationService{
		cache:        configCache,
		bucketer:     bucketer,
		configLoader: configLoader,
		eventService: eventService,
		logger:       logger.With().Str("service", "evaluation").Logger(),
	}
}

// EvaluateFlags evaluates multiple flags for a user context
func (s *EvaluationService) EvaluateFlags(ctx context.Context, req *EvaluationRequest) (*EvaluationResponse, error) {
	start := time.Now()

	// Get environment configuration with fallback
	envConfig, err := s.cache.GetConfigWithLoader(ctx, req.EnvKey, s.configLoader)
	if err != nil {
		s.logger.Error().Err(err).Str("env_key", req.EnvKey).Msg("Failed to get environment config")
		return nil, fmt.Errorf("failed to retrieve environment configuration")
	}

	if envConfig == nil {
		return nil, fmt.Errorf("environment configuration not found")
	}

	// Determine which flags to evaluate
	flagKeys := req.FlagKeys
	if len(flagKeys) == 0 {
		// Evaluate all active flags
		flagKeys = make([]string, 0, len(envConfig.Flags))
		for key, flag := range envConfig.Flags {
			if flag.Status == "active" {
				flagKeys = append(flagKeys, key)
			}
		}
	}

	// Evaluate each flag
	results := make(map[string]*bucketing.EvaluationResult)

	for _, flagKey := range flagKeys {
		flagConfig, exists := envConfig.Flags[flagKey]
		if !exists {
			s.logger.Debug().Str("flag_key", flagKey).Msg("Flag not found, skipping")
			continue
		}

		// Only evaluate active flags
		if flagConfig.Status != "active" {
			s.logger.Debug().Str("flag_key", flagKey).Str("status", flagConfig.Status).Msg("Flag not active, skipping")
			continue
		}

		result, err := s.bucketer.EvaluateFlag(flagConfig, req.Context, envConfig.Salt, envConfig.Segments)
		if err != nil {
			s.logger.Error().Err(err).Str("flag_key", flagKey).Msg("Failed to evaluate flag")
			// Create error result instead of failing the entire request
			result = &bucketing.EvaluationResult{
				FlagKey:      flagKey,
				VariationKey: flagConfig.DefaultVariation,
				Reason:       fmt.Sprintf("evaluation error: %s", err.Error()),
			}

			// Try to get the default variation value
			if variation := s.findVariation(flagConfig.Variations, flagConfig.DefaultVariation); variation != nil {
				result.Value = variation.Value
			}
		}

		// Clear reason if not requested
		if !req.IncludeReason {
			result.Reason = ""
		}

		results[flagKey] = result
	}

	response := &EvaluationResponse{
		Flags:         results,
		ConfigVersion: envConfig.Version,
		EvaluatedAt:   time.Now(),
	}

	// Track exposure events for successfully evaluated flags
	if s.eventService != nil {
		for flagKey, result := range results {
			// Track all successful evaluations (result is not nil)
			go func(fk string, r *bucketing.EvaluationResult) {
				err := s.eventService.TrackExposure(context.Background(), req.EnvKey, fk, r, req.Context, envConfig.Version)
				if err != nil {
					s.logger.Error().Err(err).
						Str("flag_key", fk).
						Str("env_key", req.EnvKey).
						Msg("Failed to track exposure event")
				}
			}(flagKey, result)
		}
	}

	// Log evaluation metrics
	duration := time.Since(start)
	s.logger.Info().
		Str("env_key", req.EnvKey).
		Str("user_key", req.Context.UserKey).
		Int("flags_count", len(results)).
		Dur("duration", duration).
		Msg("Flags evaluated")

	return response, nil
}

// EvaluateFlag evaluates a single flag for a user context
func (s *EvaluationService) EvaluateFlag(ctx context.Context, envKey, flagKey string, userContext *bucketing.Context) (*bucketing.EvaluationResult, error) {
	start := time.Now()

	// Get environment configuration with fallback
	envConfig, err := s.cache.GetConfigWithLoader(ctx, envKey, s.configLoader)
	if err != nil {
		s.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to get environment config")
		return nil, fmt.Errorf("failed to retrieve environment configuration")
	}

	if envConfig == nil {
		return nil, fmt.Errorf("environment configuration not found")
	}

	// Get flag configuration
	flagConfig, exists := envConfig.Flags[flagKey]
	if !exists {
		return nil, fmt.Errorf("flag not found")
	}

	// Check if flag is active
	if flagConfig.Status != "active" {
		// Return default variation for inactive flags
		result := &bucketing.EvaluationResult{
			FlagKey:      flagKey,
			VariationKey: flagConfig.DefaultVariation,
			Reason:       "flag is not active",
		}

		if variation := s.findVariation(flagConfig.Variations, flagConfig.DefaultVariation); variation != nil {
			result.Value = variation.Value
		}

		return result, nil
	}

	// Evaluate the flag
	result, err := s.bucketer.EvaluateFlag(flagConfig, userContext, envConfig.Salt, envConfig.Segments)
	if err != nil {
		s.logger.Error().Err(err).Str("flag_key", flagKey).Msg("Failed to evaluate flag")
		return nil, fmt.Errorf("flag evaluation failed")
	}

	// Track exposure event for successful flag evaluation
	if s.eventService != nil {
		go func() {
			err := s.eventService.TrackExposure(context.Background(), envKey, flagKey, result, userContext, envConfig.Version)
			if err != nil {
				s.logger.Error().Err(err).
					Str("flag_key", flagKey).
					Str("env_key", envKey).
					Msg("Failed to track exposure event")
			}
		}()
	}

	// Log evaluation metrics
	duration := time.Since(start)
	s.logger.Debug().
		Str("env_key", envKey).
		Str("flag_key", flagKey).
		Str("user_key", userContext.UserKey).
		Str("variation", result.VariationKey).
		Dur("duration", duration).
		Msg("Flag evaluated")

	return result, nil
}

// GetEnvironmentInfo returns basic information about an environment
func (s *EvaluationService) GetEnvironmentInfo(ctx context.Context, envKey string) (map[string]interface{}, error) {
	envConfig, err := s.cache.GetConfigWithLoader(ctx, envKey, s.configLoader)
	if err != nil {
		return nil, err
	}

	if envConfig == nil {
		return nil, fmt.Errorf("environment not found")
	}

	activeFlags := 0
	for _, flag := range envConfig.Flags {
		if flag.Status == "active" {
			activeFlags++
		}
	}

	return map[string]interface{}{
		"env_key":      envConfig.EnvKey,
		"version":      envConfig.Version,
		"flags_total":  len(envConfig.Flags),
		"flags_active": activeFlags,
		"segments":     len(envConfig.Segments),
		"updated_at":   envConfig.UpdatedAt,
	}, nil
}

// Private helper methods

func (s *EvaluationService) findVariation(variations []bucketing.Variation, key string) *bucketing.Variation {
	for _, variation := range variations {
		if variation.Key == key {
			return &variation
		}
	}
	return nil
}
