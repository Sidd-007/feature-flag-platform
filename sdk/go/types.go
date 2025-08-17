package featureflags

import (
	"fmt"
	"time"
)

// UserContext contains information about the user for flag evaluation
type UserContext struct {
	UserID     string                 `json:"user_id"`
	SessionID  string                 `json:"session_id,omitempty"`
	Email      string                 `json:"email,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Country    string                 `json:"country,omitempty"`
	Region     string                 `json:"region,omitempty"`
	City       string                 `json:"city,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Platform   string                 `json:"platform,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Language   string                 `json:"language,omitempty"`
	Timezone   string                 `json:"timezone,omitempty"`
	Groups     []string               `json:"groups,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// EvaluationResult represents the result of a flag evaluation
type EvaluationResult struct {
	FlagKey      string      `json:"flag_key"`
	Value        interface{} `json:"value"`
	VariationID  string      `json:"variation_id"`
	ExperimentID string      `json:"experiment_id,omitempty"`
	Reason       Reason      `json:"reason"`
	Error        error       `json:"error,omitempty"`
	DefaultUsed  bool        `json:"default_used"`
	CacheHit     bool        `json:"cache_hit"`
	EvaluatedAt  time.Time   `json:"evaluated_at"`
}

// Reason represents why a particular evaluation result was returned
type Reason string

const (
	ReasonTargetMatch  Reason = "TARGET_MATCH" // User matched targeting rules
	ReasonFallthrough  Reason = "FALLTHROUGH"  // User fell through to default serve
	ReasonRuleMatch    Reason = "RULE_MATCH"   // User matched a specific rule
	ReasonPrerequisite Reason = "PREREQUISITE" // Flag prerequisite evaluation
	ReasonOff          Reason = "OFF"          // Flag is turned off
	ReasonExperiment   Reason = "EXPERIMENT"   // User is in an experiment
	ReasonDefault      Reason = "DEFAULT"      // Default value returned
	ReasonError        Reason = "ERROR"        // Error occurred during evaluation
	ReasonOffline      Reason = "OFFLINE"      // Client is in offline mode
)

// Flag represents a feature flag configuration
type Flag struct {
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Type          FlagType       `json:"type"`
	Enabled       bool           `json:"enabled"`
	DefaultValue  interface{}    `json:"default_value"`
	Variations    []Variation    `json:"variations"`
	Rules         []Rule         `json:"rules"`
	Targeting     *Targeting     `json:"targeting,omitempty"`
	Prerequisites []Prerequisite `json:"prerequisites,omitempty"`
	ExperimentID  string         `json:"experiment_id,omitempty"`
	Version       int64          `json:"version"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// FlagType represents the type of a feature flag
type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeString  FlagType = "string"
	FlagTypeNumber  FlagType = "number"
	FlagTypeJSON    FlagType = "json"
)

// Variation represents a variation of a feature flag
type Variation struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Value       interface{} `json:"value"`
	Weight      int         `json:"weight,omitempty"`
}

// Rule represents a targeting rule for a feature flag
type Rule struct {
	ID          string      `json:"id"`
	Description string      `json:"description,omitempty"`
	Conditions  []Condition `json:"conditions"`
	Serve       *Serve      `json:"serve"`
	Enabled     bool        `json:"enabled"`
}

// Condition represents a condition in a targeting rule
type Condition struct {
	Attribute string   `json:"attribute"`
	Operator  Operator `json:"operator"`
	Values    []string `json:"values"`
}

// Operator represents a comparison operator
type Operator string

const (
	OperatorEquals        Operator = "equals"
	OperatorNotEquals     Operator = "not_equals"
	OperatorIn            Operator = "in"
	OperatorNotIn         Operator = "not_in"
	OperatorContains      Operator = "contains"
	OperatorNotContains   Operator = "not_contains"
	OperatorStartsWith    Operator = "starts_with"
	OperatorEndsWith      Operator = "ends_with"
	OperatorGreaterThan   Operator = "greater_than"
	OperatorGreaterThanEq Operator = "greater_than_eq"
	OperatorLessThan      Operator = "less_than"
	OperatorLessThanEq    Operator = "less_than_eq"
	OperatorRegex         Operator = "regex"
	OperatorSemverEq      Operator = "semver_eq"
	OperatorSemverGt      Operator = "semver_gt"
	OperatorSemverLt      Operator = "semver_lt"
	OperatorSemverGte     Operator = "semver_gte"
	OperatorSemverLte     Operator = "semver_lte"
)

// Serve represents what to serve when a rule matches
type Serve struct {
	VariationID string           `json:"variation_id,omitempty"`
	Rollout     *RolloutStrategy `json:"rollout,omitempty"`
}

// RolloutStrategy represents a rollout strategy for serving variations
type RolloutStrategy struct {
	Type          RolloutType    `json:"type"`
	Variations    []RolloutSplit `json:"variations"`
	BucketBy      string         `json:"bucket_by,omitempty"`
	StickyBuckets bool           `json:"sticky_buckets"`
}

// RolloutType represents the type of rollout strategy
type RolloutType string

const (
	RolloutTypePercentage RolloutType = "percentage"
	RolloutTypeExperiment RolloutType = "experiment"
)

// RolloutSplit represents a split in a rollout strategy
type RolloutSplit struct {
	VariationID string `json:"variation_id"`
	Weight      int    `json:"weight"`
}

// Targeting represents default targeting configuration
type Targeting struct {
	Enabled      bool             `json:"enabled"`
	DefaultServe *Serve           `json:"default_serve"`
	OffVariation string           `json:"off_variation,omitempty"`
	Rollout      *RolloutStrategy `json:"rollout,omitempty"`
}

// Prerequisite represents a prerequisite flag dependency
type Prerequisite struct {
	FlagKey     string `json:"flag_key"`
	VariationID string `json:"variation_id"`
}

// Environment represents an environment configuration
type Environment struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Flags     map[string]*Flag    `json:"flags"`
	Segments  map[string]*Segment `json:"segments"`
	Version   int64               `json:"version"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// Segment represents a user segment
type Segment struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Conditions  []Condition `json:"conditions"`
	Included    []string    `json:"included,omitempty"`
	Excluded    []string    `json:"excluded,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ExposureEvent represents an exposure event for analytics
type ExposureEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id,omitempty"`
	FlagKey      string                 `json:"flag_key"`
	VariationID  string                 `json:"variation_id"`
	Value        interface{}            `json:"value"`
	ExperimentID string                 `json:"experiment_id,omitempty"`
	Reason       Reason                 `json:"reason"`
	Context      *UserContext           `json:"context,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// MetricEvent represents a metric event for analytics
type MetricEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id,omitempty"`
	MetricName   string                 `json:"metric_name"`
	Value        float64                `json:"value"`
	ExperimentID string                 `json:"experiment_id,omitempty"`
	VariationID  string                 `json:"variation_id,omitempty"`
	FlagKey      string                 `json:"flag_key,omitempty"`
	Context      *UserContext           `json:"context,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// CustomEvent represents a custom event for analytics
type CustomEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id,omitempty"`
	EventName    string                 `json:"event_name"`
	ExperimentID string                 `json:"experiment_id,omitempty"`
	VariationID  string                 `json:"variation_id,omitempty"`
	FlagKey      string                 `json:"flag_key,omitempty"`
	Context      *UserContext           `json:"context,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// Legacy types - these will be replaced by the new API-compatible versions at the end of the file

// ConfigUpdate represents a configuration update from streaming
type ConfigUpdate struct {
	Type        UpdateType `json:"type"`
	Environment string     `json:"environment"`
	FlagKey     string     `json:"flag_key,omitempty"`
	Flag        *Flag      `json:"flag,omitempty"`
	Version     int64      `json:"version"`
	Timestamp   time.Time  `json:"timestamp"`
}

// UpdateType represents the type of configuration update
type UpdateType string

const (
	UpdateTypeFlag        UpdateType = "flag"
	UpdateTypeSegment     UpdateType = "segment"
	UpdateTypeEnvironment UpdateType = "environment"
	UpdateTypeHeartbeat   UpdateType = "heartbeat"
	UpdateTypeReconnect   UpdateType = "reconnect"
	UpdateTypeError       UpdateType = "error"
)

// StreamingStatus represents the status of the streaming connection
type StreamingStatus string

const (
	StatusDisconnected StreamingStatus = "disconnected"
	StatusConnecting   StreamingStatus = "connecting"
	StatusConnected    StreamingStatus = "connected"
	StatusReconnecting StreamingStatus = "reconnecting"
	StatusError        StreamingStatus = "error"
)

// CacheEntry represents an entry in the cache
type CacheEntry struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	ExpiresAt   time.Time   `json:"expires_at"`
	CreatedAt   time.Time   `json:"created_at"`
	AccessedAt  time.Time   `json:"accessed_at"`
	AccessCount int64       `json:"access_count"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Size      int     `json:"size"`
	MaxSize   int     `json:"max_size"`
	HitRate   float64 `json:"hit_rate"`
	MissRate  float64 `json:"miss_rate"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Evictions int64   `json:"evictions"`
	Expiries  int64   `json:"expiries"`
}

// EventStats represents event processing statistics
type EventStats struct {
	EventsQueued  int64     `json:"events_queued"`
	EventsSent    int64     `json:"events_sent"`
	EventsFailed  int64     `json:"events_failed"`
	BatchesSent   int64     `json:"batches_sent"`
	BatchesFailed int64     `json:"batches_failed"`
	QueueSize     int       `json:"queue_size"`
	LastFlushTime time.Time `json:"last_flush_time"`
	SuccessRate   float64   `json:"success_rate"`
}

// ClientStats represents overall client statistics
type ClientStats struct {
	Evaluations     int64                  `json:"evaluations"`
	CacheHits       int64                  `json:"cache_hits"`
	CacheMisses     int64                  `json:"cache_misses"`
	NetworkRequests int64                  `json:"network_requests"`
	Errors          int64                  `json:"errors"`
	Cache           *CacheStats            `json:"cache,omitempty"`
	Events          *EventStats            `json:"events,omitempty"`
	Streaming       map[string]interface{} `json:"streaming,omitempty"`
	Uptime          time.Duration          `json:"uptime"`
	StartTime       time.Time              `json:"start_time"`
}

// ErrorType represents different types of SDK errors
type ErrorType string

const (
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeTimeout       ErrorType = "timeout"
	ErrorTypeInvalidConfig ErrorType = "invalid_config"
	ErrorTypeInvalidFlag   ErrorType = "invalid_flag"
	ErrorTypeEvaluation    ErrorType = "evaluation"
	ErrorTypeCache         ErrorType = "cache"
	ErrorTypeStreaming     ErrorType = "streaming"
	ErrorTypeEvents        ErrorType = "events"
	ErrorTypeOffline       ErrorType = "offline"
	ErrorTypeAuth          ErrorType = "auth"
	ErrorTypeUnknown       ErrorType = "unknown"
)

// SDKError represents an SDK-specific error
type SDKError struct {
	Type      ErrorType              `json:"type"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Cause     error                  `json:"cause,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *SDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *SDKError) Unwrap() error {
	return e.Cause
}

// NewSDKError creates a new SDK error
func NewSDKError(errType ErrorType, message string, cause error) *SDKError {
	return &SDKError{
		Type:      errType,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// EventBatch represents a batch of events
type EventBatch struct {
	Events    []interface{} `json:"events"`
	Timestamp time.Time     `json:"timestamp"`
	BatchID   string        `json:"batch_id"`
}

// HealthCheck represents the health status of the client
type HealthCheck struct {
	Status     string                 `json:"status"`
	Components map[string]interface{} `json:"components"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    string                 `json:"version"`
}

// Feature evaluation context helpers
func (uc *UserContext) GetAttribute(key string) (interface{}, bool) {
	if uc.Attributes == nil {
		return nil, false
	}
	value, exists := uc.Attributes[key]
	return value, exists
}

func (uc *UserContext) SetAttribute(key string, value interface{}) {
	if uc.Attributes == nil {
		uc.Attributes = make(map[string]interface{})
	}
	uc.Attributes[key] = value
}

func (uc *UserContext) HasGroup(group string) bool {
	for _, g := range uc.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// Validation helpers
func (f *Flag) IsValid() bool {
	return f.Key != "" && len(f.Variations) > 0
}

func (r *Rule) IsValid() bool {
	return len(r.Conditions) > 0 && r.Serve != nil
}

func (c *Condition) IsValid() bool {
	return c.Attribute != "" && c.Operator != "" && len(c.Values) > 0
}

// EvaluationRequest represents a request to evaluate flags
type EvaluationRequest struct {
	EnvKey        string       `json:"env_key"`
	FlagKeys      []string     `json:"flag_keys,omitempty"` // If empty, evaluate all flags
	Context       *UserContext `json:"context"`
	IncludeReason bool         `json:"include_reason,omitempty"`
}

// EvaluationResponse represents the response containing evaluated flags
type EvaluationResponse struct {
	Flags         map[string]*EvaluationResult `json:"flags"`
	ConfigVersion int                          `json:"config_version"`
	EvaluatedAt   time.Time                    `json:"evaluated_at"`
	RequestID     string                       `json:"request_id,omitempty"`
}
