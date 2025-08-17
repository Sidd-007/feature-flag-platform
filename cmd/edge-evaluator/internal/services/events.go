package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/pkg/bucketing"
	"github.com/feature-flag-platform/pkg/config"
)

// EventService handles sending events to the event ingestor
type EventService struct {
	httpClient *http.Client
	config     *config.Config
	logger     zerolog.Logger
}

// ExposureEvent represents a flag exposure event
type ExposureEvent struct {
	Type           string                 `json:"type"`
	Timestamp      time.Time              `json:"timestamp"`
	UserKey        string                 `json:"user_key"`
	SessionID      string                 `json:"session_id,omitempty"`
	EnvKey         string                 `json:"env_key"`
	FlagKey        string                 `json:"flag_key"`
	Variation      string                 `json:"variation"`
	Value          interface{}            `json:"value"`
	DefaultUsed    bool                   `json:"default_used"`
	Reason         string                 `json:"reason,omitempty"`
	ConfigVersion  int                    `json:"config_version"`
	UserAttributes map[string]interface{} `json:"user_attributes,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
}

// CustomEvent represents a custom tracking event
type CustomEvent struct {
	Type       string                 `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	UserKey    string                 `json:"user_key"`
	SessionID  string                 `json:"session_id,omitempty"`
	EnvKey     string                 `json:"env_key"`
	EventName  string                 `json:"event_name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Value      float64                `json:"value,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// NewEventService creates a new event service
func NewEventService(cfg *config.Config, logger zerolog.Logger) *EventService {
	return &EventService{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		config: cfg,
		logger: logger.With().Str("service", "events").Logger(),
	}
}

// TrackExposure sends a flag exposure event to the event ingestor
func (s *EventService) TrackExposure(ctx context.Context, envKey, flagKey string, result *bucketing.EvaluationResult, userContext *bucketing.Context, configVersion int) error {
	event := &ExposureEvent{
		Type:          "flag_exposure",
		Timestamp:     time.Now(),
		UserKey:       userContext.UserKey,
		SessionID:     "", // SessionID is not available in bucketing.Context
		EnvKey:        envKey,
		FlagKey:       flagKey,
		Variation:     result.VariationKey,
		Value:         result.Value,
		DefaultUsed:   false, // We'll determine this based on evaluation success
		Reason:        result.Reason,
		ConfigVersion: configVersion,
		RequestID:     extractRequestID(ctx),
	}

	// Add user attributes if available
	if userContext.Attributes != nil {
		event.UserAttributes = userContext.Attributes
	}

	return s.sendEvent(ctx, event)
}

// TrackCustom sends a custom event to the event ingestor
func (s *EventService) TrackCustom(ctx context.Context, envKey string, event *CustomEvent) error {
	event.Type = "custom"
	event.Timestamp = time.Now()
	event.EnvKey = envKey
	event.RequestID = extractRequestID(ctx)

	return s.sendEvent(ctx, event)
}

// sendEvent sends an event to the event ingestor
func (s *EventService) sendEvent(ctx context.Context, event interface{}) error {
	// Check if event ingestor is configured
	if s.config.EventIngestor.URL == "" {
		s.logger.Debug().Msg("Event ingestor URL not configured, skipping event")
		return nil
	}

	// Serialize event
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create HTTP request
	url := s.config.EventIngestor.URL + "/v1/events"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(eventData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "edge-evaluator/1.0.0")

	// Add API key if configured
	if s.config.EventIngestor.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.EventIngestor.APIKey)
	}

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to send event to event ingestor")
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 400 {
		s.logger.Error().
			Int("status", resp.StatusCode).
			Str("url", url).
			Msg("Event ingestor returned error status")
		return fmt.Errorf("event ingestor returned status %d", resp.StatusCode)
	}

	s.logger.Debug().
		Int("status", resp.StatusCode).
		Str("url", url).
		Msg("Event sent successfully")

	return nil
}

// extractRequestID extracts request ID from context
func extractRequestID(ctx context.Context) string {
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
