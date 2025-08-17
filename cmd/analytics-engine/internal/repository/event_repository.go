package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/rs/zerolog/log"
)

type EventRepository interface {
	GetExposureEvents(ctx context.Context, req *ExposureEventsRequest) ([]ExposureEvent, error)
	GetMetricEvents(ctx context.Context, req *MetricEventsRequest) ([]MetricEvent, error)
	GetExperimentSummary(ctx context.Context, experimentID string, timeRange TimeRange) (*ExperimentSummary, error)
	GetConversionFunnel(ctx context.Context, req *FunnelRequest) ([]FunnelStep, error)
	GetCohortData(ctx context.Context, req *CohortRequest) ([]CohortData, error)
	GetRetentionData(ctx context.Context, req *RetentionRequest) ([]RetentionData, error)
}

type eventRepository struct {
	db clickhouse.Conn
}

func NewEventRepository(db clickhouse.Conn) EventRepository {
	return &eventRepository{db: db}
}

// Data types
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ExposureEvent struct {
	Timestamp     time.Time         `json:"timestamp"`
	UserID        string            `json:"user_id"`
	ExperimentID  string            `json:"experiment_id"`
	VariationID   string            `json:"variation_id"`
	EnvironmentID string            `json:"environment_id"`
	Properties    map[string]string `json:"properties"`
}

type MetricEvent struct {
	Timestamp     time.Time         `json:"timestamp"`
	UserID        string            `json:"user_id"`
	MetricName    string            `json:"metric_name"`
	Value         float64           `json:"value"`
	ExperimentID  string            `json:"experiment_id,omitempty"`
	VariationID   string            `json:"variation_id,omitempty"`
	EnvironmentID string            `json:"environment_id"`
	Properties    map[string]string `json:"properties"`
}

type ExperimentSummary struct {
	ExperimentID    string                   `json:"experiment_id"`
	TotalExposures  int64                    `json:"total_exposures"`
	UniqueUsers     int64                    `json:"unique_users"`
	Variations      []VariationSummary       `json:"variations"`
	ConversionRates map[string]float64       `json:"conversion_rates"`
	MetricSummaries map[string]MetricSummary `json:"metric_summaries"`
	TimeRange       TimeRange                `json:"time_range"`
}

type VariationSummary struct {
	VariationID     string  `json:"variation_id"`
	Exposures       int64   `json:"exposures"`
	UniqueUsers     int64   `json:"unique_users"`
	ConversionRate  float64 `json:"conversion_rate"`
	AllocationRatio float64 `json:"allocation_ratio"`
}

type MetricSummary struct {
	MetricName   string  `json:"metric_name"`
	Mean         float64 `json:"mean"`
	StdDev       float64 `json:"std_dev"`
	Count        int64   `json:"count"`
	Sum          float64 `json:"sum"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Percentile50 float64 `json:"percentile_50"`
	Percentile95 float64 `json:"percentile_95"`
	Percentile99 float64 `json:"percentile_99"`
}

// Request types
type ExposureEventsRequest struct {
	ExperimentID  string            `json:"experiment_id"`
	EnvironmentID string            `json:"environment_id"`
	TimeRange     TimeRange         `json:"time_range"`
	VariationIDs  []string          `json:"variation_ids,omitempty"`
	Filters       map[string]string `json:"filters,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
}

type MetricEventsRequest struct {
	MetricNames   []string          `json:"metric_names"`
	ExperimentID  string            `json:"experiment_id,omitempty"`
	EnvironmentID string            `json:"environment_id"`
	TimeRange     TimeRange         `json:"time_range"`
	VariationIDs  []string          `json:"variation_ids,omitempty"`
	Filters       map[string]string `json:"filters,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
}

type FunnelRequest struct {
	Steps         []FunnelStepDefinition `json:"steps"`
	EnvironmentID string                 `json:"environment_id"`
	TimeRange     TimeRange              `json:"time_range"`
	ExperimentID  string                 `json:"experiment_id,omitempty"`
	Filters       map[string]string      `json:"filters,omitempty"`
}

type FunnelStepDefinition struct {
	Name       string            `json:"name"`
	MetricName string            `json:"metric_name"`
	Filters    map[string]string `json:"filters,omitempty"`
}

type FunnelStep struct {
	StepNumber  int                            `json:"step_number"`
	Name        string                         `json:"name"`
	Users       int64                          `json:"users"`
	Conversion  float64                        `json:"conversion"`
	DropOff     float64                        `json:"drop_off"`
	ByVariation map[string]FunnelStepVariation `json:"by_variation,omitempty"`
}

type FunnelStepVariation struct {
	Users      int64   `json:"users"`
	Conversion float64 `json:"conversion"`
}

type CohortRequest struct {
	CohortBy      string            `json:"cohort_by"` // day, week, month
	EnvironmentID string            `json:"environment_id"`
	TimeRange     TimeRange         `json:"time_range"`
	MetricName    string            `json:"metric_name"`
	ExperimentID  string            `json:"experiment_id,omitempty"`
	Filters       map[string]string `json:"filters,omitempty"`
}

type CohortData struct {
	CohortDate  time.Time `json:"cohort_date"`
	Period      int       `json:"period"`
	Users       int64     `json:"users"`
	Value       float64   `json:"value"`
	VariationID string    `json:"variation_id,omitempty"`
}

type RetentionRequest struct {
	RetentionBy   string            `json:"retention_by"` // day, week, month
	EnvironmentID string            `json:"environment_id"`
	TimeRange     TimeRange         `json:"time_range"`
	ExperimentID  string            `json:"experiment_id,omitempty"`
	Filters       map[string]string `json:"filters,omitempty"`
}

type RetentionData struct {
	CohortDate    time.Time `json:"cohort_date"`
	Period        int       `json:"period"`
	CohortSize    int64     `json:"cohort_size"`
	ReturnedUsers int64     `json:"returned_users"`
	RetentionRate float64   `json:"retention_rate"`
	VariationID   string    `json:"variation_id,omitempty"`
}

// Implementation
func (r *eventRepository) GetExposureEvents(ctx context.Context, req *ExposureEventsRequest) ([]ExposureEvent, error) {
	query := `
		SELECT 
			timestamp,
			user_id,
			experiment_id,
			variation_id,
			environment_id,
			properties
		FROM events_exposure 
		WHERE environment_id = ? 
			AND timestamp >= ? 
			AND timestamp <= ?`

	args := []interface{}{req.EnvironmentID, req.TimeRange.Start, req.TimeRange.End}

	if req.ExperimentID != "" {
		query += " AND experiment_id = ?"
		args = append(args, req.ExperimentID)
	}

	if len(req.VariationIDs) > 0 {
		query += " AND variation_id IN (?)"
		args = append(args, req.VariationIDs)
	}

	query += " ORDER BY timestamp DESC"

	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	}

	if req.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, req.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query exposure events: %w", err)
	}
	defer rows.Close()

	var events []ExposureEvent
	for rows.Next() {
		var event ExposureEvent
		if err := rows.Scan(
			&event.Timestamp,
			&event.UserID,
			&event.ExperimentID,
			&event.VariationID,
			&event.EnvironmentID,
			&event.Properties,
		); err != nil {
			return nil, fmt.Errorf("failed to scan exposure event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating exposure events: %w", err)
	}

	log.Debug().
		Str("experiment_id", req.ExperimentID).
		Str("environment_id", req.EnvironmentID).
		Int("count", len(events)).
		Msg("Retrieved exposure events")

	return events, nil
}

func (r *eventRepository) GetMetricEvents(ctx context.Context, req *MetricEventsRequest) ([]MetricEvent, error) {
	query := `
		SELECT 
			timestamp,
			user_id,
			metric_name,
			value,
			experiment_id,
			variation_id,
			environment_id,
			properties
		FROM events_metric 
		WHERE environment_id = ? 
			AND timestamp >= ? 
			AND timestamp <= ?`

	args := []interface{}{req.EnvironmentID, req.TimeRange.Start, req.TimeRange.End}

	if len(req.MetricNames) > 0 {
		query += " AND metric_name IN (?)"
		args = append(args, req.MetricNames)
	}

	if req.ExperimentID != "" {
		query += " AND experiment_id = ?"
		args = append(args, req.ExperimentID)
	}

	if len(req.VariationIDs) > 0 {
		query += " AND variation_id IN (?)"
		args = append(args, req.VariationIDs)
	}

	query += " ORDER BY timestamp DESC"

	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	}

	if req.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, req.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metric events: %w", err)
	}
	defer rows.Close()

	var events []MetricEvent
	for rows.Next() {
		var event MetricEvent
		if err := rows.Scan(
			&event.Timestamp,
			&event.UserID,
			&event.MetricName,
			&event.Value,
			&event.ExperimentID,
			&event.VariationID,
			&event.EnvironmentID,
			&event.Properties,
		); err != nil {
			return nil, fmt.Errorf("failed to scan metric event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metric events: %w", err)
	}

	log.Debug().
		Strs("metric_names", req.MetricNames).
		Str("environment_id", req.EnvironmentID).
		Int("count", len(events)).
		Msg("Retrieved metric events")

	return events, nil
}

func (r *eventRepository) GetExperimentSummary(ctx context.Context, experimentID string, timeRange TimeRange) (*ExperimentSummary, error) {
	// Query exposure summary
	exposureQuery := `
		SELECT 
			variation_id,
			count(*) as exposures,
			uniq(user_id) as unique_users
		FROM events_exposure 
		WHERE experiment_id = ? 
			AND timestamp >= ? 
			AND timestamp <= ?
		GROUP BY variation_id
		ORDER BY variation_id`

	rows, err := r.db.Query(ctx, exposureQuery, experimentID, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to query exposure summary: %w", err)
	}
	defer rows.Close()

	var variations []VariationSummary
	var totalExposures, totalUniqueUsers int64

	for rows.Next() {
		var variation VariationSummary
		if err := rows.Scan(&variation.VariationID, &variation.Exposures, &variation.UniqueUsers); err != nil {
			return nil, fmt.Errorf("failed to scan variation summary: %w", err)
		}
		totalExposures += variation.Exposures
		totalUniqueUsers += variation.UniqueUsers
		variations = append(variations, variation)
	}

	// Calculate allocation ratios
	for i := range variations {
		if totalExposures > 0 {
			variations[i].AllocationRatio = float64(variations[i].Exposures) / float64(totalExposures)
		}
	}

	summary := &ExperimentSummary{
		ExperimentID:    experimentID,
		TotalExposures:  totalExposures,
		UniqueUsers:     totalUniqueUsers,
		Variations:      variations,
		ConversionRates: make(map[string]float64),
		MetricSummaries: make(map[string]MetricSummary),
		TimeRange:       timeRange,
	}

	log.Debug().
		Str("experiment_id", experimentID).
		Int64("total_exposures", totalExposures).
		Int64("unique_users", totalUniqueUsers).
		Int("variations", len(variations)).
		Msg("Retrieved experiment summary")

	return summary, nil
}

func (r *eventRepository) GetConversionFunnel(ctx context.Context, req *FunnelRequest) ([]FunnelStep, error) {
	// TODO: Implement funnel analysis
	// This would be a complex query involving:
	// 1. CTEs to define each funnel step
	// 2. User progression tracking between steps
	// 3. Conversion rate calculations
	// 4. Variation-specific breakdowns if experiment_id is provided

	log.Info().
		Str("environment_id", req.EnvironmentID).
		Int("steps", len(req.Steps)).
		Msg("Funnel analysis requested - placeholder implementation")

	return []FunnelStep{}, nil
}

func (r *eventRepository) GetCohortData(ctx context.Context, req *CohortRequest) ([]CohortData, error) {
	// TODO: Implement cohort analysis
	// This would involve:
	// 1. Grouping users by cohort period (day/week/month)
	// 2. Tracking their behavior over subsequent periods
	// 3. Calculating retention/engagement metrics
	// 4. Variation-specific analysis if experiment_id is provided

	log.Info().
		Str("environment_id", req.EnvironmentID).
		Str("cohort_by", req.CohortBy).
		Str("metric_name", req.MetricName).
		Msg("Cohort analysis requested - placeholder implementation")

	return []CohortData{}, nil
}

func (r *eventRepository) GetRetentionData(ctx context.Context, req *RetentionRequest) ([]RetentionData, error) {
	// TODO: Implement retention analysis
	// This would involve:
	// 1. Defining cohorts by first activity date
	// 2. Tracking return visits in subsequent periods
	// 3. Calculating retention rates by period
	// 4. Variation-specific analysis if experiment_id is provided

	log.Info().
		Str("environment_id", req.EnvironmentID).
		Str("retention_by", req.RetentionBy).
		Msg("Retention analysis requested - placeholder implementation")

	return []RetentionData{}, nil
}
