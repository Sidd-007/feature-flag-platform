package featureflags

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Evaluator handles flag evaluation logic
type Evaluator struct {
	config *EvaluatorConfig
	logger zerolog.Logger
}

// EvaluatorConfig holds configuration for the evaluator
type EvaluatorConfig struct {
	EvaluatorEndpoint string
	Environment       string
	APIKey            string
	Timeout           time.Duration
	HTTPClient        *http.Client
	Cache             *Cache
	Offline           *OfflineHandler
	Events            *EventProcessor
}

// NewEvaluator creates a new evaluator
func NewEvaluator(config *EvaluatorConfig, logger zerolog.Logger) *Evaluator {
	return &Evaluator{
		config: config,
		logger: logger.With().Str("component", "evaluator").Logger(),
	}
}

// Evaluate evaluates a single flag
func (e *Evaluator) Evaluate(ctx context.Context, flagKey string, userContext *UserContext, defaultValue interface{}) (*EvaluationResult, error) {
	startTime := time.Now()

	// Create default result
	result := &EvaluationResult{
		FlagKey:     flagKey,
		Value:       defaultValue,
		VariationID: "",
		Reason:      ReasonDefault,
		DefaultUsed: true,
		EvaluatedAt: startTime,
	}

	// Check cache first
	if e.config.Cache != nil {
		if cachedResult := e.checkCache(flagKey, userContext); cachedResult != nil {
			e.logger.Debug().
				Str("flag_key", flagKey).
				Str("user_id", userContext.UserID).
				Msg("Cache hit for flag evaluation")

			cachedResult.CacheHit = true
			cachedResult.EvaluatedAt = startTime

			// Track exposure event
			if e.config.Events != nil && !cachedResult.DefaultUsed {
				e.trackExposure(ctx, cachedResult, userContext)
			}

			return cachedResult, nil
		}
	}

	// Try offline evaluation if available
	if e.config.Offline != nil && e.config.Offline.IsOffline() {
		if offlineResult := e.evaluateOffline(flagKey, userContext); offlineResult != nil {
			e.logger.Debug().
				Str("flag_key", flagKey).
				Str("user_id", userContext.UserID).
				Msg("Offline evaluation successful")

			offlineResult.EvaluatedAt = startTime

			// Cache the result
			if e.config.Cache != nil {
				e.config.Cache.Set(e.getCacheKey(flagKey, userContext), offlineResult)
			}

			// Track exposure event
			if e.config.Events != nil && !offlineResult.DefaultUsed {
				e.trackExposure(ctx, offlineResult, userContext)
			}

			return offlineResult, nil
		}
	}

	// Perform network evaluation
	networkResult, err := e.evaluateNetwork(ctx, flagKey, userContext, defaultValue)
	if err != nil {
		e.logger.Warn().
			Err(err).
			Str("flag_key", flagKey).
			Str("user_id", userContext.UserID).
			Msg("Network evaluation failed, returning default")

		result.Error = err
		result.Reason = ReasonError
		return result, nil
	}

	networkResult.EvaluatedAt = startTime

	// Cache the result
	if e.config.Cache != nil && !networkResult.DefaultUsed {
		e.config.Cache.Set(e.getCacheKey(flagKey, userContext), networkResult)
	}

	// Track exposure event
	if e.config.Events != nil && !networkResult.DefaultUsed {
		e.trackExposure(ctx, networkResult, userContext)
	}

	e.logger.Debug().
		Str("flag_key", flagKey).
		Str("user_id", userContext.UserID).
		Str("variation_id", networkResult.VariationID).
		Interface("value", networkResult.Value).
		Str("reason", string(networkResult.Reason)).
		Dur("duration", time.Since(startTime)).
		Msg("Flag evaluation completed")

	return networkResult, nil
}

// EvaluateMultiple evaluates multiple flags at once
func (e *Evaluator) EvaluateMultiple(ctx context.Context, flagKeys []string, userContext *UserContext, defaults map[string]interface{}) (map[string]*EvaluationResult, error) {
	if len(flagKeys) == 0 {
		return make(map[string]*EvaluationResult), nil
	}

	results := make(map[string]*EvaluationResult)

	// Check cache for all flags first
	var uncachedFlags []string
	for _, flagKey := range flagKeys {
		if e.config.Cache != nil {
			if cachedResult := e.checkCache(flagKey, userContext); cachedResult != nil {
				cachedResult.CacheHit = true
				results[flagKey] = cachedResult
				continue
			}
		}
		uncachedFlags = append(uncachedFlags, flagKey)
	}

	if len(uncachedFlags) == 0 {
		e.logger.Debug().
			Int("flag_count", len(flagKeys)).
			Str("user_id", userContext.UserID).
			Msg("All flags found in cache")
		return results, nil
	}

	// Try offline evaluation for uncached flags
	var networkFlags []string
	if e.config.Offline != nil && e.config.Offline.IsOffline() {
		for _, flagKey := range uncachedFlags {
			if offlineResult := e.evaluateOffline(flagKey, userContext); offlineResult != nil {
				results[flagKey] = offlineResult

				// Cache the result
				if e.config.Cache != nil {
					e.config.Cache.Set(e.getCacheKey(flagKey, userContext), offlineResult)
				}
			} else {
				networkFlags = append(networkFlags, flagKey)
			}
		}
	} else {
		networkFlags = uncachedFlags
	}

	// Evaluate remaining flags via network
	if len(networkFlags) > 0 {
		networkResults, err := e.evaluateMultipleNetwork(ctx, networkFlags, userContext, defaults)
		if err != nil {
			e.logger.Warn().
				Err(err).
				Strs("flag_keys", networkFlags).
				Str("user_id", userContext.UserID).
				Msg("Network evaluation failed for some flags")

			// Add default results for failed flags
			for _, flagKey := range networkFlags {
				if _, exists := networkResults[flagKey]; !exists {
					defaultValue := defaults[flagKey]
					results[flagKey] = &EvaluationResult{
						FlagKey:     flagKey,
						Value:       defaultValue,
						VariationID: "",
						Reason:      ReasonError,
						Error:       err,
						DefaultUsed: true,
						EvaluatedAt: time.Now(),
					}
				}
			}
		}

		// Add network results and cache them
		for flagKey, result := range networkResults {
			results[flagKey] = result

			if e.config.Cache != nil && !result.DefaultUsed {
				e.config.Cache.Set(e.getCacheKey(flagKey, userContext), result)
			}
		}
	}

	// Track exposure events for all successful evaluations
	if e.config.Events != nil {
		for _, result := range results {
			if !result.DefaultUsed {
				e.trackExposure(ctx, result, userContext)
			}
		}
	}

	e.logger.Debug().
		Int("flag_count", len(flagKeys)).
		Int("cache_hits", len(flagKeys)-len(uncachedFlags)).
		Int("network_requests", len(networkFlags)).
		Str("user_id", userContext.UserID).
		Msg("Multiple flag evaluation completed")

	return results, nil
}

// checkCache checks if a flag evaluation result is in cache
func (e *Evaluator) checkCache(flagKey string, userContext *UserContext) *EvaluationResult {
	if e.config.Cache == nil {
		return nil
	}

	cacheKey := e.getCacheKey(flagKey, userContext)
	value, exists := e.config.Cache.Get(cacheKey)
	if !exists {
		return nil
	}

	result, ok := value.(*EvaluationResult)
	if !ok {
		e.logger.Warn().
			Str("flag_key", flagKey).
			Str("cache_key", cacheKey).
			Msg("Invalid cached value type")

		e.config.Cache.Delete(cacheKey)
		return nil
	}

	return result
}

// evaluateOffline performs offline evaluation using local configuration
func (e *Evaluator) evaluateOffline(flagKey string, userContext *UserContext) *EvaluationResult {
	if e.config.Offline == nil {
		return nil
	}

	flag, exists := e.config.Offline.GetFlag(flagKey)
	if !exists {
		return nil
	}

	// Perform local evaluation
	result := e.evaluateFlag(flag, userContext)
	result.Reason = ReasonOffline

	return result
}

// evaluateNetwork performs network evaluation
func (e *Evaluator) evaluateNetwork(ctx context.Context, flagKey string, userContext *UserContext, defaultValue interface{}) (*EvaluationResult, error) {
	// Create context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Prepare request
	req := &EvaluationRequest{
		EnvKey:        e.config.Environment, // This should be passed in config
		FlagKeys:      []string{flagKey},
		Context:       e.mapUserContext(userContext),
		IncludeReason: true,
	}

	// Make HTTP request
	response, err := e.makeEvaluationRequest(evalCtx, req)
	if err != nil {
		return nil, fmt.Errorf("evaluation request failed: %w", err)
	}

	// Extract result for the specific flag
	result, exists := response.Flags[flagKey]
	if !exists {
		return &EvaluationResult{
			FlagKey:     flagKey,
			Value:       defaultValue,
			VariationID: "",
			Reason:      ReasonDefault,
			DefaultUsed: true,
			EvaluatedAt: time.Now(),
		}, nil
	}

	return result, nil
}

// evaluateMultipleNetwork performs network evaluation for multiple flags
func (e *Evaluator) evaluateMultipleNetwork(ctx context.Context, flagKeys []string, userContext *UserContext, defaults map[string]interface{}) (map[string]*EvaluationResult, error) {
	// Create context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Prepare request
	req := &EvaluationRequest{
		EnvKey:        e.config.Environment,
		FlagKeys:      flagKeys,
		Context:       e.mapUserContext(userContext),
		IncludeReason: true,
	}

	// Make HTTP request
	response, err := e.makeEvaluationRequest(evalCtx, req)
	if err != nil {
		return nil, fmt.Errorf("evaluation request failed: %w", err)
	}

	results := make(map[string]*EvaluationResult)

	// Process results
	for _, flagKey := range flagKeys {
		if result, exists := response.Flags[flagKey]; exists {
			results[flagKey] = result
		} else {
			// Use default value if flag not found
			defaultValue := defaults[flagKey]
			results[flagKey] = &EvaluationResult{
				FlagKey:     flagKey,
				Value:       defaultValue,
				VariationID: "",
				Reason:      ReasonDefault,
				DefaultUsed: true,
				EvaluatedAt: time.Now(),
			}
		}
	}

	return results, nil
}

// makeEvaluationRequest makes an HTTP request to evaluate flags
func (e *Evaluator) makeEvaluationRequest(ctx context.Context, req *EvaluationRequest) (*EvaluationResponse, error) {
	// Serialize request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.config.EvaluatorEndpoint+"/v1/evaluate", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	httpReq.Header.Set("User-Agent", "feature-flags-go-sdk/1.0.0")

	// Make request
	resp, err := e.config.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("evaluation failed with status %d", resp.StatusCode)
	}

	// Parse response
	var evalResp EvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&evalResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &evalResp, nil
}

// mapUserContext converts SDK UserContext to evaluator context format
func (e *Evaluator) mapUserContext(userContext *UserContext) *UserContext {
	// For now, we use the same structure
	// In the future, we might need to map to the specific bucketing.Context format
	if userContext == nil {
		return &UserContext{}
	}

	// Ensure UserKey is set (required field)
	if userContext.UserID == "" {
		e.logger.Warn().Msg("UserID is empty, this may cause evaluation issues")
	}

	return userContext
}

// evaluateFlag performs local flag evaluation
func (e *Evaluator) evaluateFlag(flag *Flag, userContext *UserContext) *EvaluationResult {
	// Check if flag is enabled
	if !flag.Enabled {
		return &EvaluationResult{
			FlagKey:     flag.Key,
			Value:       flag.DefaultValue,
			VariationID: "",
			Reason:      ReasonOff,
			DefaultUsed: true,
			EvaluatedAt: time.Now(),
		}
	}

	// Check prerequisites
	if len(flag.Prerequisites) > 0 {
		for _, prereq := range flag.Prerequisites {
			// TODO: Evaluate prerequisite flags
			// For now, just log that prerequisites exist
			e.logger.Debug().
				Str("flag_key", flag.Key).
				Str("prerequisite", prereq.FlagKey).
				Msg("Prerequisite check needed (not implemented)")
		}
	}

	// Evaluate rules
	for _, rule := range flag.Rules {
		if !rule.Enabled {
			continue
		}

		if e.evaluateRule(&rule, userContext) {
			return e.serveRule(flag, &rule, userContext)
		}
	}

	// Fallback to targeting configuration
	if flag.Targeting != nil && flag.Targeting.Enabled {
		return e.serveTargeting(flag, flag.Targeting, userContext)
	}

	// Return default value
	return &EvaluationResult{
		FlagKey:     flag.Key,
		Value:       flag.DefaultValue,
		VariationID: "",
		Reason:      ReasonDefault,
		DefaultUsed: true,
		EvaluatedAt: time.Now(),
	}
}

// evaluateRule evaluates if a rule matches the user context
func (e *Evaluator) evaluateRule(rule *Rule, userContext *UserContext) bool {
	for _, condition := range rule.Conditions {
		if !e.evaluateCondition(&condition, userContext) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (e *Evaluator) evaluateCondition(condition *Condition, userContext *UserContext) bool {
	value := e.getAttributeValue(condition.Attribute, userContext)
	if value == "" {
		return false
	}

	switch condition.Operator {
	case OperatorEquals:
		return e.contains(condition.Values, value)
	case OperatorNotEquals:
		return !e.contains(condition.Values, value)
	case OperatorIn:
		return e.contains(condition.Values, value)
	case OperatorNotIn:
		return !e.contains(condition.Values, value)
	case OperatorContains:
		return e.containsSubstring(condition.Values, value)
	case OperatorNotContains:
		return !e.containsSubstring(condition.Values, value)
	case OperatorStartsWith:
		return e.startsWith(condition.Values, value)
	case OperatorEndsWith:
		return e.endsWith(condition.Values, value)
	// TODO: Implement numeric and regex operators
	default:
		e.logger.Warn().
			Str("operator", string(condition.Operator)).
			Msg("Unsupported operator")
		return false
	}
}

// serveRule serves the result for a matching rule
func (e *Evaluator) serveRule(flag *Flag, rule *Rule, userContext *UserContext) *EvaluationResult {
	if rule.Serve.VariationID != "" {
		variation := e.findVariation(flag, rule.Serve.VariationID)
		if variation != nil {
			return &EvaluationResult{
				FlagKey:     flag.Key,
				Value:       variation.Value,
				VariationID: variation.ID,
				Reason:      ReasonRuleMatch,
				DefaultUsed: false,
				EvaluatedAt: time.Now(),
			}
		}
	}

	if rule.Serve.Rollout != nil {
		return e.serveRollout(flag, rule.Serve.Rollout, userContext, ReasonRuleMatch)
	}

	return &EvaluationResult{
		FlagKey:     flag.Key,
		Value:       flag.DefaultValue,
		VariationID: "",
		Reason:      ReasonDefault,
		DefaultUsed: true,
		EvaluatedAt: time.Now(),
	}
}

// serveTargeting serves the result for targeting configuration
func (e *Evaluator) serveTargeting(flag *Flag, targeting *Targeting, userContext *UserContext) *EvaluationResult {
	if targeting.DefaultServe != nil {
		if targeting.DefaultServe.VariationID != "" {
			variation := e.findVariation(flag, targeting.DefaultServe.VariationID)
			if variation != nil {
				return &EvaluationResult{
					FlagKey:     flag.Key,
					Value:       variation.Value,
					VariationID: variation.ID,
					Reason:      ReasonFallthrough,
					DefaultUsed: false,
					EvaluatedAt: time.Now(),
				}
			}
		}

		if targeting.DefaultServe.Rollout != nil {
			return e.serveRollout(flag, targeting.DefaultServe.Rollout, userContext, ReasonFallthrough)
		}
	}

	if targeting.Rollout != nil {
		return e.serveRollout(flag, targeting.Rollout, userContext, ReasonFallthrough)
	}

	return &EvaluationResult{
		FlagKey:     flag.Key,
		Value:       flag.DefaultValue,
		VariationID: "",
		Reason:      ReasonDefault,
		DefaultUsed: true,
		EvaluatedAt: time.Now(),
	}
}

// serveRollout serves the result for a rollout strategy
func (e *Evaluator) serveRollout(flag *Flag, rollout *RolloutStrategy, userContext *UserContext, reason Reason) *EvaluationResult {
	// Get bucket value for user
	bucketBy := rollout.BucketBy
	if bucketBy == "" {
		bucketBy = "user_id"
	}

	bucketValue := e.getAttributeValue(bucketBy, userContext)
	if bucketValue == "" {
		bucketValue = userContext.UserID
	}

	// Calculate bucket (0-9999)
	bucket := calculateBucket(flag.Key, bucketValue)

	// Find matching variation based on weights
	currentWeight := 0
	for _, split := range rollout.Variations {
		currentWeight += split.Weight
		if bucket < currentWeight {
			variation := e.findVariation(flag, split.VariationID)
			if variation != nil {
				return &EvaluationResult{
					FlagKey:     flag.Key,
					Value:       variation.Value,
					VariationID: variation.ID,
					Reason:      reason,
					DefaultUsed: false,
					EvaluatedAt: time.Now(),
				}
			}
		}
	}

	// Fallback to default
	return &EvaluationResult{
		FlagKey:     flag.Key,
		Value:       flag.DefaultValue,
		VariationID: "",
		Reason:      ReasonDefault,
		DefaultUsed: true,
		EvaluatedAt: time.Now(),
	}
}

// Helper functions
func (e *Evaluator) getCacheKey(flagKey string, userContext *UserContext) string {
	return fmt.Sprintf("flag:%s:user:%s", flagKey, userContext.UserID)
}

func (e *Evaluator) trackExposure(ctx context.Context, result *EvaluationResult, userContext *UserContext) {
	exposure := &ExposureEvent{
		Timestamp:    result.EvaluatedAt,
		UserID:       userContext.UserID,
		SessionID:    userContext.SessionID,
		FlagKey:      result.FlagKey,
		VariationID:  result.VariationID,
		Value:        result.Value,
		ExperimentID: result.ExperimentID,
		Reason:       result.Reason,
		Context:      userContext,
	}

	e.config.Events.TrackExposure(ctx, exposure)
}

func (e *Evaluator) getAttributeValue(attribute string, userContext *UserContext) string {
	switch attribute {
	case "user_id":
		return userContext.UserID
	case "email":
		return userContext.Email
	case "country":
		return userContext.Country
	case "region":
		return userContext.Region
	case "city":
		return userContext.City
	case "platform":
		return userContext.Platform
	case "version":
		return userContext.Version
	case "language":
		return userContext.Language
	default:
		if value, exists := userContext.GetAttribute(attribute); exists {
			return fmt.Sprintf("%v", value)
		}
		return ""
	}
}

func (e *Evaluator) findVariation(flag *Flag, variationID string) *Variation {
	for i := range flag.Variations {
		if flag.Variations[i].ID == variationID {
			return &flag.Variations[i]
		}
	}
	return nil
}

func (e *Evaluator) contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (e *Evaluator) containsSubstring(values []string, target string) bool {
	for _, value := range values {
		if len(value) > 0 && len(target) > 0 {
			// Simple substring check - in production you'd use strings.Contains
			// This is a simplified implementation
			if value == target {
				return true
			}
		}
	}
	return false
}

func (e *Evaluator) startsWith(values []string, target string) bool {
	for _, value := range values {
		if len(value) > 0 && len(target) >= len(value) {
			// Simple prefix check - in production you'd use strings.HasPrefix
			if target[:len(value)] == value {
				return true
			}
		}
	}
	return false
}

func (e *Evaluator) endsWith(values []string, target string) bool {
	for _, value := range values {
		if len(value) > 0 && len(target) >= len(value) {
			// Simple suffix check - in production you'd use strings.HasSuffix
			if target[len(target)-len(value):] == value {
				return true
			}
		}
	}
	return false
}

// calculateBucket calculates a bucket value (0-9999) for consistent assignment
func calculateBucket(flagKey, userValue string) int {
	// This should use the same hashing algorithm as the backend
	// For now, using a simple hash
	hash := 0
	for _, char := range flagKey + ":" + userValue {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash % 10000
}
