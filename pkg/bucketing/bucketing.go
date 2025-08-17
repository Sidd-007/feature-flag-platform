package bucketing

import (
	"fmt"

	"github.com/Sidd-007/feature-flag-platform/pkg/hashing"
)

// Bucketer handles user bucketing for feature flags and experiments
type Bucketer struct {
	hasher *hashing.Hasher
}

// NewBucketer creates a new bucketer instance
func NewBucketer() *Bucketer {
	return &Bucketer{
		hasher: hashing.NewHasher(),
	}
}

// Context represents the user and environment context for bucketing
type Context struct {
	UserKey     string                 `json:"user_key"`
	Attributes  map[string]interface{} `json:"attributes"`
	Environment string                 `json:"environment"`
}

// FlagConfig represents the configuration for a feature flag
type FlagConfig struct {
	Key               string      `json:"key"`
	Type              string      `json:"type"` // boolean, multivariate, json
	Variations        []Variation `json:"variations"`
	DefaultVariation  string      `json:"default_variation"`
	Rules             []Rule      `json:"rules"`
	Status            string      `json:"status"`
	TrafficAllocation float64     `json:"traffic_allocation"` // 0.0 to 1.0
}

// Variation represents a flag variation
type Variation struct {
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

// Rule represents a targeting rule
type Rule struct {
	ID                string      `json:"id"`
	Conditions        []Condition `json:"conditions"`
	VariationKey      string      `json:"variation_key,omitempty"`
	Rollout           *Rollout    `json:"rollout,omitempty"`
	TrafficAllocation float64     `json:"traffic_allocation"` // 0.0 to 1.0
}

// Condition represents a single targeting condition
type Condition struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"` // eq, neq, in, nin, lt, gt, lte, gte, contains, regex, semver
	Value     interface{} `json:"value"`
}

// Rollout represents percentage-based rollout configuration
type Rollout struct {
	Variations []RolloutVariation `json:"variations"`
}

// RolloutVariation represents a variation in a rollout
type RolloutVariation struct {
	VariationKey string  `json:"variation_key"`
	Weight       float64 `json:"weight"`
}

// SegmentConfig represents a user segment
type SegmentConfig struct {
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Conditions []Condition `json:"conditions"`
}

// EvaluationResult represents the result of flag evaluation
type EvaluationResult struct {
	FlagKey       string      `json:"flag_key"`
	VariationKey  string      `json:"variation_key"`
	Value         interface{} `json:"value"`
	Reason        string      `json:"reason"`
	BucketingID   string      `json:"bucketing_id"`
	Bucket        int         `json:"bucket"`
	RuleID        string      `json:"rule_id,omitempty"`
	InExperiment  bool        `json:"in_experiment"`
	ExperimentKey string      `json:"experiment_key,omitempty"`
}

// EvaluateFlag evaluates a feature flag for the given context
func (b *Bucketer) EvaluateFlag(flagConfig *FlagConfig, context *Context, envSalt string, segments map[string]*SegmentConfig) (*EvaluationResult, error) {
	if flagConfig == nil {
		return nil, fmt.Errorf("flag config is nil")
	}

	if context == nil || context.UserKey == "" {
		return nil, fmt.Errorf("invalid context: user key is required")
	}

	// Check if flag is active
	if flagConfig.Status != "active" {
		return b.createDefaultResult(flagConfig, context, envSalt, "flag is not active")
	}

	// Generate bucketing ID
	bucketingID := b.hasher.GenerateBucketingID(envSalt, flagConfig.Key, context.UserKey)
	bucket := b.hasher.DeterministicBucket(bucketingID)

	// Check traffic allocation first
	if flagConfig.TrafficAllocation < 1.0 {
		if !b.hasher.IsInPercentageRange(bucket, flagConfig.TrafficAllocation*100) {
			return b.createDefaultResult(flagConfig, context, envSalt, "excluded by traffic allocation")
		}
	}

	// Evaluate rules in order
	for _, rule := range flagConfig.Rules {
		if b.evaluateRule(&rule, context, segments) {
			// Check rule-level traffic allocation
			if rule.TrafficAllocation < 1.0 {
				ruleBasedBucket := b.hasher.DeterministicBucket(bucketingID + rule.ID)
				if !b.hasher.IsInPercentageRange(ruleBasedBucket, rule.TrafficAllocation*100) {
					continue // Skip this rule due to traffic allocation
				}
			}

			// Rule matches, determine variation
			if rule.VariationKey != "" {
				// Direct variation assignment
				variation := b.findVariation(flagConfig.Variations, rule.VariationKey)
				if variation != nil {
					return &EvaluationResult{
						FlagKey:      flagConfig.Key,
						VariationKey: rule.VariationKey,
						Value:        variation.Value,
						Reason:       fmt.Sprintf("matched rule %s", rule.ID),
						BucketingID:  bucketingID,
						Bucket:       bucket,
						RuleID:       rule.ID,
					}, nil
				}
			} else if rule.Rollout != nil {
				// Percentage rollout
				variation, reason := b.evaluateRollout(rule.Rollout, flagConfig.Variations, bucketingID, rule.ID)
				if variation != nil {
					return &EvaluationResult{
						FlagKey:      flagConfig.Key,
						VariationKey: variation.Key,
						Value:        variation.Value,
						Reason:       fmt.Sprintf("matched rule %s: %s", rule.ID, reason),
						BucketingID:  bucketingID,
						Bucket:       bucket,
						RuleID:       rule.ID,
					}, nil
				}
			}
		}
	}

	// No rules matched, return default variation
	return b.createDefaultResult(flagConfig, context, envSalt, "no rules matched")
}

// evaluateRule checks if a rule matches the given context
func (b *Bucketer) evaluateRule(rule *Rule, context *Context, segments map[string]*SegmentConfig) bool {
	if len(rule.Conditions) == 0 {
		return true // Empty conditions always match
	}

	// All conditions must match (AND logic)
	for _, condition := range rule.Conditions {
		if !b.evaluateCondition(&condition, context, segments) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single condition
func (b *Bucketer) evaluateCondition(condition *Condition, context *Context, segments map[string]*SegmentConfig) bool {
	// Special handling for segment conditions
	if condition.Attribute == "segment" {
		return b.evaluateSegmentCondition(condition, context, segments)
	}

	// Get attribute value from context
	var attributeValue interface{}
	if condition.Attribute == "user_key" {
		attributeValue = context.UserKey
	} else {
		attributeValue = context.Attributes[condition.Attribute]
	}

	return b.compareValues(attributeValue, condition.Operator, condition.Value)
}

// evaluateSegmentCondition evaluates segment membership
func (b *Bucketer) evaluateSegmentCondition(condition *Condition, context *Context, segments map[string]*SegmentConfig) bool {
	segmentKey, ok := condition.Value.(string)
	if !ok {
		return false
	}

	segment, exists := segments[segmentKey]
	if !exists {
		return false
	}

	// Check if user matches segment conditions
	for _, segmentCondition := range segment.Conditions {
		if !b.evaluateCondition(&segmentCondition, context, segments) {
			return false
		}
	}

	return true
}

// compareValues compares two values using the given operator
func (b *Bucketer) compareValues(left interface{}, operator string, right interface{}) bool {
	// This is a simplified implementation. In production, you'd want more robust type handling
	switch operator {
	case "eq":
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
	case "neq":
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right)
	case "in":
		if rightSlice, ok := right.([]interface{}); ok {
			leftStr := fmt.Sprintf("%v", left)
			for _, item := range rightSlice {
				if fmt.Sprintf("%v", item) == leftStr {
					return true
				}
			}
		}
		return false
	case "nin":
		return !b.compareValues(left, "in", right)
	case "contains":
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return len(leftStr) > 0 && len(rightStr) > 0 &&
			fmt.Sprintf("%v", left) != "" &&
			fmt.Sprintf("%v", right) != ""
	// Add more operators as needed (lt, gt, regex, semver, etc.)
	default:
		return false
	}
}

// evaluateRollout determines which variation to serve based on rollout configuration
func (b *Bucketer) evaluateRollout(rollout *Rollout, variations []Variation, bucketingID, ruleID string) (*Variation, string) {
	if rollout == nil || len(rollout.Variations) == 0 {
		return nil, "no rollout variations"
	}

	// Create variation allocations
	keys := make([]string, len(rollout.Variations))
	weights := make([]float64, len(rollout.Variations))

	for i, rv := range rollout.Variations {
		keys[i] = rv.VariationKey
		weights[i] = rv.Weight
	}

	allocations := b.hasher.CreateVariationAllocations(keys, weights)
	if len(allocations) == 0 {
		return nil, "failed to create allocations"
	}

	// Use rule-specific bucketing to avoid correlation
	ruleBucket := b.hasher.DeterministicBucket(bucketingID + ruleID)

	// Find which allocation contains this bucket
	for _, allocation := range allocations {
		if allocation.BucketRange.Contains(ruleBucket) {
			variation := b.findVariation(variations, allocation.Key)
			if variation != nil {
				return variation, fmt.Sprintf("rollout bucket %d in range %d-%d",
					ruleBucket, allocation.BucketRange.Start, allocation.BucketRange.End)
			}
		}
	}

	return nil, "no matching rollout allocation"
}

// findVariation finds a variation by key
func (b *Bucketer) findVariation(variations []Variation, key string) *Variation {
	for _, variation := range variations {
		if variation.Key == key {
			return &variation
		}
	}
	return nil
}

// createDefaultResult creates a result using the default variation
func (b *Bucketer) createDefaultResult(flagConfig *FlagConfig, context *Context, envSalt, reason string) (*EvaluationResult, error) {
	bucketingID := b.hasher.GenerateBucketingID(envSalt, flagConfig.Key, context.UserKey)
	bucket := b.hasher.DeterministicBucket(bucketingID)

	variation := b.findVariation(flagConfig.Variations, flagConfig.DefaultVariation)
	if variation == nil {
		return nil, fmt.Errorf("default variation '%s' not found", flagConfig.DefaultVariation)
	}

	return &EvaluationResult{
		FlagKey:      flagConfig.Key,
		VariationKey: flagConfig.DefaultVariation,
		Value:        variation.Value,
		Reason:       reason,
		BucketingID:  bucketingID,
		Bucket:       bucket,
	}, nil
}
