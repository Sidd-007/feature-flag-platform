package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ExposureEvent represents a flag exposure event
type ExposureEvent struct {
	EventID       string                 `json:"event_id"`
	EnvKey        string                 `json:"env_key"`
	FlagKey       string                 `json:"flag_key"`
	VariationKey  string                 `json:"variation_key"`
	UserKeyHash   string                 `json:"user_key_hash"`
	BucketingID   string                 `json:"bucketing_id"`
	ExperimentKey string                 `json:"experiment_key,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Reason        string                 `json:"reason,omitempty"`
	Bucket        int                    `json:"bucket,omitempty"`
	RuleID        string                 `json:"rule_id,omitempty"`
}

// MetricEvent represents a custom metric event
type MetricEvent struct {
	EventID     string                 `json:"event_id"`
	EnvKey      string                 `json:"env_key"`
	MetricKey   string                 `json:"metric_key"`
	UserKeyHash string                 `json:"user_key_hash"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	SessionID   string                 `json:"session_id,omitempty"`
}

// EventStorage handles storing events in ClickHouse
type EventStorage struct {
	clickhouse clickhouse.Conn
	logger     zerolog.Logger
}

// NewEventStorage creates a new event storage instance
func NewEventStorage(clickhouseConn clickhouse.Conn, logger zerolog.Logger) *EventStorage {
	return &EventStorage{
		clickhouse: clickhouseConn,
		logger:     logger.With().Str("component", "event_storage").Logger(),
	}
}

// StoreExposureEvents stores exposure events in ClickHouse
func (s *EventStorage) StoreExposureEvents(ctx context.Context, events []ExposureEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Prepare batch insert
	batch, err := s.clickhouse.PrepareBatch(ctx, `
		INSERT INTO events_exposure 
		(date, timestamp, env_key, flag_key, variation_key, user_key_hash, bucketing_id, 
		 experiment_key, session_id, context_json, meta_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare exposure events batch: %w", err)
	}

	// Add events to batch
	for _, event := range events {
		contextJSON := "{}"
		if event.Context != nil {
			if jsonStr, err := jsonMarshal(event.Context); err == nil {
				contextJSON = jsonStr
			}
		}

		metaJSON := "{}"
		if event.Meta != nil {
			if jsonStr, err := jsonMarshal(event.Meta); err == nil {
				metaJSON = jsonStr
			}
		}

		err = batch.Append(
			event.Timestamp.Truncate(24*time.Hour), // date
			event.Timestamp,                        // timestamp
			event.EnvKey,
			event.FlagKey,
			event.VariationKey,
			event.UserKeyHash,
			event.BucketingID,
			event.ExperimentKey,
			event.SessionID,
			contextJSON,
			metaJSON,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to append exposure event to batch: %w", err)
		}
	}

	// Send batch
	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send exposure events batch: %w", err)
	}

	s.logger.Info().Int("count", len(events)).Msg("Stored exposure events")
	return nil
}

// StoreMetricEvents stores metric events in ClickHouse
func (s *EventStorage) StoreMetricEvents(ctx context.Context, events []MetricEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Prepare batch insert
	batch, err := s.clickhouse.PrepareBatch(ctx, `
		INSERT INTO events_metric 
		(date, timestamp, env_key, metric_key, user_key_hash, value, unit, 
		 context_json, meta_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare metric events batch: %w", err)
	}

	// Add events to batch
	for _, event := range events {
		contextJSON := "{}"
		if event.Context != nil {
			if jsonStr, err := jsonMarshal(event.Context); err == nil {
				contextJSON = jsonStr
			}
		}

		metaJSON := "{}"
		if event.Meta != nil {
			if jsonStr, err := jsonMarshal(event.Meta); err == nil {
				metaJSON = jsonStr
			}
		}

		err = batch.Append(
			event.Timestamp.Truncate(24*time.Hour), // date
			event.Timestamp,                        // timestamp
			event.EnvKey,
			event.MetricKey,
			event.UserKeyHash,
			event.Value,
			event.Unit,
			contextJSON,
			metaJSON,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to append metric event to batch: %w", err)
		}
	}

	// Send batch
	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send metric events batch: %w", err)
	}

	s.logger.Info().Int("count", len(events)).Msg("Stored metric events")
	return nil
}

// GetStorageStats returns storage statistics
func (s *EventStorage) GetStorageStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get exposure events count
	var exposureCount uint64
	err := s.clickhouse.QueryRow(ctx, "SELECT count() FROM events_exposure WHERE date >= today() - 1").Scan(&exposureCount)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get exposure events count")
	} else {
		stats["exposure_events_24h"] = exposureCount
	}

	// Get metric events count
	var metricCount uint64
	err = s.clickhouse.QueryRow(ctx, "SELECT count() FROM events_metric WHERE date >= today() - 1").Scan(&metricCount)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get metric events count")
	} else {
		stats["metric_events_24h"] = metricCount
	}

	return stats, nil
}

// Helper functions

func jsonMarshal(v interface{}) (string, error) {
	// Simple JSON marshaling - in production, you might want more sophisticated handling
	return fmt.Sprintf("%v", v), nil
}

// GenerateEventID generates a unique event ID
func GenerateEventID() string {
	return uuid.New().String()
}
