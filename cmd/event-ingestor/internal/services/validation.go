package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/event-ingestor/internal/storage"
)

// ValidationService handles event validation
type ValidationService struct {
	logger zerolog.Logger
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// NewValidationService creates a new validation service
func NewValidationService(logger zerolog.Logger) *ValidationService {
	return &ValidationService{
		logger: logger.With().Str("service", "validation").Logger(),
	}
}

// ValidateExposureEvent validates an exposure event
func (s *ValidationService) ValidateExposureEvent(event *storage.ExposureEvent) *ValidationResult {
	var errors []ValidationError

	// Required fields
	if event.EnvKey == "" {
		errors = append(errors, ValidationError{Field: "env_key", Message: "env_key is required"})
	}

	if event.FlagKey == "" {
		errors = append(errors, ValidationError{Field: "flag_key", Message: "flag_key is required"})
	}

	if event.VariationKey == "" {
		errors = append(errors, ValidationError{Field: "variation_key", Message: "variation_key is required"})
	}

	if event.UserKeyHash == "" {
		errors = append(errors, ValidationError{Field: "user_key_hash", Message: "user_key_hash is required"})
	}

	// Validate field formats
	if event.EnvKey != "" && !isValidKey(event.EnvKey) {
		errors = append(errors, ValidationError{Field: "env_key", Message: "env_key format is invalid"})
	}

	if event.FlagKey != "" && !isValidKey(event.FlagKey) {
		errors = append(errors, ValidationError{Field: "flag_key", Message: "flag_key format is invalid"})
	}

	// Validate timestamp
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now() // Set default timestamp
	} else if event.Timestamp.After(time.Now().Add(5 * time.Minute)) {
		errors = append(errors, ValidationError{Field: "timestamp", Message: "timestamp cannot be in the future"})
	} else if event.Timestamp.Before(time.Now().Add(-24 * time.Hour)) {
		errors = append(errors, ValidationError{Field: "timestamp", Message: "timestamp is too old (older than 24 hours)"})
	}

	// Generate event ID if missing
	if event.EventID == "" {
		event.EventID = storage.GenerateEventID()
	}

	return &ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateMetricEvent validates a metric event
func (s *ValidationService) ValidateMetricEvent(event *storage.MetricEvent) *ValidationResult {
	var errors []ValidationError

	// Required fields
	if event.EnvKey == "" {
		errors = append(errors, ValidationError{Field: "env_key", Message: "env_key is required"})
	}

	if event.MetricKey == "" {
		errors = append(errors, ValidationError{Field: "metric_key", Message: "metric_key is required"})
	}

	if event.UserKeyHash == "" {
		errors = append(errors, ValidationError{Field: "user_key_hash", Message: "user_key_hash is required"})
	}

	// Validate field formats
	if event.EnvKey != "" && !isValidKey(event.EnvKey) {
		errors = append(errors, ValidationError{Field: "env_key", Message: "env_key format is invalid"})
	}

	if event.MetricKey != "" && !isValidKey(event.MetricKey) {
		errors = append(errors, ValidationError{Field: "metric_key", Message: "metric_key format is invalid"})
	}

	// Validate value ranges
	if event.Value < -1e15 || event.Value > 1e15 {
		errors = append(errors, ValidationError{Field: "value", Message: "value is out of valid range"})
	}

	// Validate timestamp
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now() // Set default timestamp
	} else if event.Timestamp.After(time.Now().Add(5 * time.Minute)) {
		errors = append(errors, ValidationError{Field: "timestamp", Message: "timestamp cannot be in the future"})
	} else if event.Timestamp.Before(time.Now().Add(-24 * time.Hour)) {
		errors = append(errors, ValidationError{Field: "timestamp", Message: "timestamp is too old (older than 24 hours)"})
	}

	// Generate event ID if missing
	if event.EventID == "" {
		event.EventID = storage.GenerateEventID()
	}

	return &ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateExposureEventBatch validates a batch of exposure events
func (s *ValidationService) ValidateExposureEventBatch(events []storage.ExposureEvent) ([]storage.ExposureEvent, []ValidationError) {
	var validEvents []storage.ExposureEvent
	var allErrors []ValidationError

	for i, event := range events {
		result := s.ValidateExposureEvent(&event)
		if result.Valid {
			validEvents = append(validEvents, event)
		} else {
			// Add index to errors for batch processing
			for _, err := range result.Errors {
				allErrors = append(allErrors, ValidationError{
					Field:   fmt.Sprintf("events[%d].%s", i, err.Field),
					Message: err.Message,
				})
			}
		}
	}

	return validEvents, allErrors
}

// ValidateMetricEventBatch validates a batch of metric events
func (s *ValidationService) ValidateMetricEventBatch(events []storage.MetricEvent) ([]storage.MetricEvent, []ValidationError) {
	var validEvents []storage.MetricEvent
	var allErrors []ValidationError

	for i, event := range events {
		result := s.ValidateMetricEvent(&event)
		if result.Valid {
			validEvents = append(validEvents, event)
		} else {
			// Add index to errors for batch processing
			for _, err := range result.Errors {
				allErrors = append(allErrors, ValidationError{
					Field:   fmt.Sprintf("events[%d].%s", i, err.Field),
					Message: err.Message,
				})
			}
		}
	}

	return validEvents, allErrors
}

// Helper functions

// isValidKey validates that a key follows the expected format
func isValidKey(key string) bool {
	if len(key) == 0 || len(key) > 100 {
		return false
	}

	// Allow alphanumeric characters, underscores, hyphens, and dots
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' || r == '.') {
			return false
		}
	}

	// Must not start or end with special characters
	if strings.HasPrefix(key, "_") || strings.HasPrefix(key, "-") || strings.HasPrefix(key, ".") ||
		strings.HasSuffix(key, "_") || strings.HasSuffix(key, "-") || strings.HasSuffix(key, ".") {
		return false
	}

	return true
}
