package featureflags

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// EventProcessor handles event tracking and batching
type EventProcessor struct {
	config *EventProcessorConfig
	logger zerolog.Logger

	// Event queues
	exposureQueue []interface{}
	metricQueue   []interface{}
	customQueue   []interface{}

	// Synchronization
	mutex       sync.Mutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	doneChan    chan struct{}

	// Stats
	eventsQueued  int64
	eventsSent    int64
	eventsFailed  int64
	batchesSent   int64
	batchesFailed int64
	lastFlushTime time.Time
}

// EventProcessorConfig holds configuration for the event processor
type EventProcessorConfig struct {
	EventsEndpoint string
	APIKey         string
	BatchSize      int
	FlushInterval  time.Duration
	HTTPClient     *http.Client
	RetryAttempts  int
	RetryBackoff   time.Duration
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(config *EventProcessorConfig, logger zerolog.Logger) (*EventProcessor, error) {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 10 * time.Second
	}
	if config.RetryAttempts <= 0 {
		config.RetryAttempts = 3
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = time.Second
	}

	processor := &EventProcessor{
		config:        config,
		logger:        logger.With().Str("component", "events").Logger(),
		exposureQueue: make([]interface{}, 0, config.BatchSize),
		metricQueue:   make([]interface{}, 0, config.BatchSize),
		customQueue:   make([]interface{}, 0, config.BatchSize),
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}

	processor.logger.Info().
		Str("endpoint", config.EventsEndpoint).
		Int("batch_size", config.BatchSize).
		Dur("flush_interval", config.FlushInterval).
		Msg("Event processor created")

	return processor, nil
}

// Start starts the event processor
func (ep *EventProcessor) Start(ctx context.Context) error {
	ep.logger.Info().Msg("Starting event processor")

	// Start flush ticker
	ep.flushTicker = time.NewTicker(ep.config.FlushInterval)

	// Start background goroutine
	go ep.run()

	ep.logger.Info().Msg("Event processor started")
	return nil
}

// run is the main event processing loop
func (ep *EventProcessor) run() {
	defer close(ep.doneChan)
	defer ep.flushTicker.Stop()

	for {
		select {
		case <-ep.flushTicker.C:
			ep.flushAll()
		case <-ep.stopChan:
			ep.logger.Info().Msg("Event processor stopping")
			ep.flushAll() // Final flush
			return
		}
	}
}

// TrackExposure tracks an exposure event
func (ep *EventProcessor) TrackExposure(ctx context.Context, event *ExposureEvent) error {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	ep.exposureQueue = append(ep.exposureQueue, event)
	ep.eventsQueued++

	ep.logger.Debug().
		Str("flag_key", event.FlagKey).
		Str("variation_id", event.VariationID).
		Str("user_id", event.UserID).
		Msg("Exposure event queued")

	// Flush if batch is full
	if len(ep.exposureQueue) >= ep.config.BatchSize {
		go ep.flushExposures()
	}

	return nil
}

// TrackMetric tracks a metric event
func (ep *EventProcessor) TrackMetric(ctx context.Context, event *MetricEvent) error {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	ep.metricQueue = append(ep.metricQueue, event)
	ep.eventsQueued++

	ep.logger.Debug().
		Str("metric_name", event.MetricName).
		Float64("value", event.Value).
		Str("user_id", event.UserID).
		Msg("Metric event queued")

	// Flush if batch is full
	if len(ep.metricQueue) >= ep.config.BatchSize {
		go ep.flushMetrics()
	}

	return nil
}

// TrackEvent tracks a custom event
func (ep *EventProcessor) TrackEvent(ctx context.Context, event *CustomEvent) error {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	ep.customQueue = append(ep.customQueue, event)
	ep.eventsQueued++

	ep.logger.Debug().
		Str("event_name", event.EventName).
		Str("user_id", event.UserID).
		Msg("Custom event queued")

	// Flush if batch is full
	if len(ep.customQueue) >= ep.config.BatchSize {
		go ep.flushCustom()
	}

	return nil
}

// Flush flushes all pending events
func (ep *EventProcessor) Flush(ctx context.Context) error {
	ep.flushAll()
	return nil
}

// flushAll flushes all event types
func (ep *EventProcessor) flushAll() {
	ep.flushExposures()
	ep.flushMetrics()
	ep.flushCustom()
}

// flushExposures flushes pending exposure events
func (ep *EventProcessor) flushExposures() {
	ep.mutex.Lock()
	if len(ep.exposureQueue) == 0 {
		ep.mutex.Unlock()
		return
	}

	// Take current batch
	batch := make([]interface{}, len(ep.exposureQueue))
	copy(batch, ep.exposureQueue)
	ep.exposureQueue = ep.exposureQueue[:0] // Clear queue
	ep.mutex.Unlock()

	ep.sendBatch("exposure", batch)
}

// flushMetrics flushes pending metric events
func (ep *EventProcessor) flushMetrics() {
	ep.mutex.Lock()
	if len(ep.metricQueue) == 0 {
		ep.mutex.Unlock()
		return
	}

	// Take current batch
	batch := make([]interface{}, len(ep.metricQueue))
	copy(batch, ep.metricQueue)
	ep.metricQueue = ep.metricQueue[:0] // Clear queue
	ep.mutex.Unlock()

	ep.sendBatch("metric", batch)
}

// flushCustom flushes pending custom events
func (ep *EventProcessor) flushCustom() {
	ep.mutex.Lock()
	if len(ep.customQueue) == 0 {
		ep.mutex.Unlock()
		return
	}

	// Take current batch
	batch := make([]interface{}, len(ep.customQueue))
	copy(batch, ep.customQueue)
	ep.customQueue = ep.customQueue[:0] // Clear queue
	ep.mutex.Unlock()

	ep.sendBatch("custom", batch)
}

// sendBatch sends a batch of events to the server
func (ep *EventProcessor) sendBatch(eventType string, events []interface{}) {
	if len(events) == 0 {
		return
	}

	ep.logger.Debug().
		Str("event_type", eventType).
		Int("batch_size", len(events)).
		Msg("Sending event batch")

	// Prepare batch
	batch := &EventBatch{
		Events:    events,
		Timestamp: time.Now(),
		BatchID:   fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano()),
	}

	// Try to send with retries
	var lastErr error
	for attempt := 0; attempt < ep.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := ep.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)

			ep.logger.Debug().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Retrying event batch send")
		}

		err := ep.sendBatchHTTP(eventType, batch)
		if err == nil {
			ep.batchesSent++
			ep.eventsSent += int64(len(events))
			ep.lastFlushTime = time.Now()

			ep.logger.Debug().
				Str("event_type", eventType).
				Int("batch_size", len(events)).
				Str("batch_id", batch.BatchID).
				Msg("Event batch sent successfully")
			return
		}

		lastErr = err
		ep.logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Str("event_type", eventType).
			Msg("Failed to send event batch")
	}

	// All retries failed
	ep.batchesFailed++
	ep.eventsFailed += int64(len(events))

	ep.logger.Error().
		Err(lastErr).
		Str("event_type", eventType).
		Int("batch_size", len(events)).
		Str("batch_id", batch.BatchID).
		Msg("Failed to send event batch after all retries")
}

// sendBatchHTTP sends a batch via HTTP
func (ep *EventProcessor) sendBatchHTTP(eventType string, batch *EventBatch) error {
	// Serialize batch
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	// Determine endpoint
	endpoint := fmt.Sprintf("%s/api/v1/events/%s", ep.config.EventsEndpoint, eventType)

	// Create request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ep.config.APIKey)
	req.Header.Set("User-Agent", "feature-flags-go-sdk/1.0.0")
	req.Header.Set("X-Batch-ID", batch.BatchID)

	// Send request
	resp, err := ep.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// Stats returns event processing statistics
func (ep *EventProcessor) Stats() *EventStats {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	queueSize := len(ep.exposureQueue) + len(ep.metricQueue) + len(ep.customQueue)
	successRate := 0.0

	if ep.eventsSent+ep.eventsFailed > 0 {
		successRate = float64(ep.eventsSent) / float64(ep.eventsSent+ep.eventsFailed)
	}

	return &EventStats{
		EventsQueued:  ep.eventsQueued,
		EventsSent:    ep.eventsSent,
		EventsFailed:  ep.eventsFailed,
		BatchesSent:   ep.batchesSent,
		BatchesFailed: ep.batchesFailed,
		QueueSize:     queueSize,
		LastFlushTime: ep.lastFlushTime,
		SuccessRate:   successRate,
	}
}

// GetQueueSizes returns the current queue sizes
func (ep *EventProcessor) GetQueueSizes() map[string]int {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	return map[string]int{
		"exposure": len(ep.exposureQueue),
		"metric":   len(ep.metricQueue),
		"custom":   len(ep.customQueue),
	}
}

// SetBatchSize updates the batch size
func (ep *EventProcessor) SetBatchSize(size int) {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	if size > 0 {
		ep.config.BatchSize = size

		ep.logger.Info().
			Int("new_batch_size", size).
			Msg("Event processor batch size updated")
	}
}

// SetFlushInterval updates the flush interval
func (ep *EventProcessor) SetFlushInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}

	ep.config.FlushInterval = interval

	// Restart ticker with new interval
	if ep.flushTicker != nil {
		ep.flushTicker.Stop()
		ep.flushTicker = time.NewTicker(interval)
	}

	ep.logger.Info().
		Dur("new_flush_interval", interval).
		Msg("Event processor flush interval updated")
}

// Close stops the event processor and flushes remaining events
func (ep *EventProcessor) Close() {
	ep.logger.Info().Msg("Closing event processor")

	// Signal stop
	close(ep.stopChan)

	// Wait for completion
	<-ep.doneChan

	stats := ep.Stats()
	ep.logger.Info().
		Int64("events_sent", stats.EventsSent).
		Int64("events_failed", stats.EventsFailed).
		Int64("batches_sent", stats.BatchesSent).
		Int64("batches_failed", stats.BatchesFailed).
		Float64("success_rate", stats.SuccessRate).
		Msg("Event processor closed")
}

// IsHealthy returns true if the event processor is healthy
func (ep *EventProcessor) IsHealthy() bool {
	stats := ep.Stats()

	// Consider healthy if:
	// 1. Queue size is not too large
	// 2. Success rate is reasonable
	// 3. Recent flush activity

	maxQueueSize := ep.config.BatchSize * 5 // 5 batches worth
	if stats.QueueSize > maxQueueSize {
		return false
	}

	if stats.EventsSent+stats.EventsFailed > 100 && stats.SuccessRate < 0.5 {
		return false
	}

	if time.Since(stats.LastFlushTime) > ep.config.FlushInterval*3 {
		return false
	}

	return true
}

// DrainQueues empties all queues without sending events (for testing)
func (ep *EventProcessor) DrainQueues() {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	drainedCount := len(ep.exposureQueue) + len(ep.metricQueue) + len(ep.customQueue)

	ep.exposureQueue = ep.exposureQueue[:0]
	ep.metricQueue = ep.metricQueue[:0]
	ep.customQueue = ep.customQueue[:0]

	ep.logger.Debug().
		Int("drained_count", drainedCount).
		Msg("Event queues drained")
}

// GetPendingEvents returns copies of pending events (for debugging)
func (ep *EventProcessor) GetPendingEvents() map[string][]interface{} {
	ep.mutex.Lock()
	defer ep.mutex.Unlock()

	result := make(map[string][]interface{})

	if len(ep.exposureQueue) > 0 {
		exposure := make([]interface{}, len(ep.exposureQueue))
		copy(exposure, ep.exposureQueue)
		result["exposure"] = exposure
	}

	if len(ep.metricQueue) > 0 {
		metric := make([]interface{}, len(ep.metricQueue))
		copy(metric, ep.metricQueue)
		result["metric"] = metric
	}

	if len(ep.customQueue) > 0 {
		custom := make([]interface{}, len(ep.customQueue))
		copy(custom, ep.customQueue)
		result["custom"] = custom
	}

	return result
}

// UpdateEndpoint updates the events endpoint
func (ep *EventProcessor) UpdateEndpoint(endpoint string) {
	ep.config.EventsEndpoint = endpoint

	ep.logger.Info().
		Str("new_endpoint", endpoint).
		Msg("Event processor endpoint updated")
}

// UpdateAPIKey updates the API key
func (ep *EventProcessor) UpdateAPIKey(apiKey string) {
	ep.config.APIKey = apiKey

	ep.logger.Info().Msg("Event processor API key updated")
}
