package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/event-ingestor/internal/storage"
	"github.com/feature-flag-platform/pkg/config"
)

// IngestionService handles high-throughput event ingestion
type IngestionService struct {
	eventStorage      *storage.EventStorage
	nats              *nats.Conn
	validationService *ValidationService
	config            *config.Config
	logger            zerolog.Logger

	// Buffering and batching
	exposureBuffer []storage.ExposureEvent
	metricBuffer   []storage.MetricEvent
	bufferMutex    sync.Mutex

	// Statistics
	stats IngestionStats

	// Channels for shutdown
	shutdown chan struct{}
	done     chan struct{}
}

// IngestionStats holds ingestion statistics
type IngestionStats struct {
	EventsPerSecond        int64     `json:"events_per_second"`
	TotalEventsProcessed   int64     `json:"total_events_processed"`
	EventsInQueue          int64     `json:"events_in_queue"`
	AvgProcessingTimeMs    float64   `json:"avg_processing_time_ms"`
	ErrorsLastHour         int64     `json:"errors_last_hour"`
	LastProcessedAt        time.Time `json:"last_processed_at"`
	ExposureEventsBuffered int       `json:"exposure_events_buffered"`
	MetricEventsBuffered   int       `json:"metric_events_buffered"`
}

// EventBatch represents a mixed batch of events
type EventBatch struct {
	BatchID        string                  `json:"batch_id"`
	BatchType      string                  `json:"batch_type"`
	ExposureEvents []storage.ExposureEvent `json:"exposure_events,omitempty"`
	MetricEvents   []storage.MetricEvent   `json:"metric_events,omitempty"`
	Timestamp      time.Time               `json:"timestamp"`
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(
	eventStorage *storage.EventStorage,
	natsConn *nats.Conn,
	validationService *ValidationService,
	cfg *config.Config,
	logger zerolog.Logger,
) *IngestionService {
	return &IngestionService{
		eventStorage:      eventStorage,
		nats:              natsConn,
		validationService: validationService,
		config:            cfg,
		logger:            logger.With().Str("service", "ingestion").Logger(),
		exposureBuffer:    make([]storage.ExposureEvent, 0, 1000),
		metricBuffer:      make([]storage.MetricEvent, 0, 1000),
		shutdown:          make(chan struct{}),
		done:              make(chan struct{}),
	}
}

// Start starts the ingestion service background workers
func (s *IngestionService) Start() error {
	// Start buffer flush worker
	go s.bufferFlushWorker()

	// Start metrics worker
	go s.metricsWorker()

	s.logger.Info().Msg("Ingestion service started")
	return nil
}

// Close stops the ingestion service
func (s *IngestionService) Close() error {
	close(s.shutdown)

	// Wait for workers to finish with timeout
	select {
	case <-s.done:
		s.logger.Info().Msg("Ingestion service stopped gracefully")
	case <-time.After(10 * time.Second):
		s.logger.Warn().Msg("Ingestion service shutdown timeout")
	}

	// Flush remaining events
	s.flushBuffers()

	return nil
}

// IngestExposureEvents processes a batch of exposure events
func (s *IngestionService) IngestExposureEvents(ctx context.Context, events []storage.ExposureEvent) (int, []ValidationError, error) {
	start := time.Now()

	// Validate events
	validEvents, validationErrors := s.validationService.ValidateExposureEventBatch(events)

	// Add to buffer
	s.bufferMutex.Lock()
	s.exposureBuffer = append(s.exposureBuffer, validEvents...)
	bufferedCount := len(s.exposureBuffer)
	s.bufferMutex.Unlock()

	// Update statistics
	atomic.AddInt64(&s.stats.TotalEventsProcessed, int64(len(validEvents)))
	s.stats.LastProcessedAt = time.Now()

	// Log processing metrics
	duration := time.Since(start)
	s.logger.Info().
		Int("total_events", len(events)).
		Int("valid_events", len(validEvents)).
		Int("validation_errors", len(validationErrors)).
		Int("buffered_count", bufferedCount).
		Dur("duration", duration).
		Msg("Exposure events processed")

	// Trigger flush if buffer is large
	if bufferedCount >= 500 {
		go s.flushExposureEvents()
	}

	return len(validEvents), validationErrors, nil
}

// IngestMetricEvents processes a batch of metric events
func (s *IngestionService) IngestMetricEvents(ctx context.Context, events []storage.MetricEvent) (int, []ValidationError, error) {
	start := time.Now()

	// Validate events
	validEvents, validationErrors := s.validationService.ValidateMetricEventBatch(events)

	// Add to buffer
	s.bufferMutex.Lock()
	s.metricBuffer = append(s.metricBuffer, validEvents...)
	bufferedCount := len(s.metricBuffer)
	s.bufferMutex.Unlock()

	// Update statistics
	atomic.AddInt64(&s.stats.TotalEventsProcessed, int64(len(validEvents)))
	s.stats.LastProcessedAt = time.Now()

	// Log processing metrics
	duration := time.Since(start)
	s.logger.Info().
		Int("total_events", len(events)).
		Int("valid_events", len(validEvents)).
		Int("validation_errors", len(validationErrors)).
		Int("buffered_count", bufferedCount).
		Dur("duration", duration).
		Msg("Metric events processed")

	// Trigger flush if buffer is large
	if bufferedCount >= 500 {
		go s.flushMetricEvents()
	}

	return len(validEvents), validationErrors, nil
}

// IngestEventBatch processes a mixed batch of events
func (s *IngestionService) IngestEventBatch(ctx context.Context, batch EventBatch) (int, []ValidationError, error) {
	var totalAccepted int
	var allErrors []ValidationError

	// Process exposure events
	if len(batch.ExposureEvents) > 0 {
		accepted, errors, err := s.IngestExposureEvents(ctx, batch.ExposureEvents)
		if err != nil {
			return 0, nil, err
		}
		totalAccepted += accepted
		allErrors = append(allErrors, errors...)
	}

	// Process metric events
	if len(batch.MetricEvents) > 0 {
		accepted, errors, err := s.IngestMetricEvents(ctx, batch.MetricEvents)
		if err != nil {
			return totalAccepted, allErrors, err
		}
		totalAccepted += accepted
		allErrors = append(allErrors, errors...)
	}

	s.logger.Info().
		Str("batch_id", batch.BatchID).
		Str("batch_type", batch.BatchType).
		Int("total_accepted", totalAccepted).
		Int("total_errors", len(allErrors)).
		Msg("Event batch processed")

	return totalAccepted, allErrors, nil
}

// GetStats returns current ingestion statistics
func (s *IngestionService) GetStats() IngestionStats {
	s.bufferMutex.Lock()
	defer s.bufferMutex.Unlock()

	stats := s.stats
	stats.ExposureEventsBuffered = len(s.exposureBuffer)
	stats.MetricEventsBuffered = len(s.metricBuffer)
	stats.EventsInQueue = int64(stats.ExposureEventsBuffered + stats.MetricEventsBuffered)

	return stats
}

// Private methods

// bufferFlushWorker periodically flushes event buffers
func (s *IngestionService) bufferFlushWorker() {
	ticker := time.NewTicker(5 * time.Second) // Flush every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushBuffers()
		case <-s.shutdown:
			s.logger.Info().Msg("Buffer flush worker stopping")
			close(s.done)
			return
		}
	}
}

// metricsWorker updates performance metrics
func (s *IngestionService) metricsWorker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastProcessed int64

	for {
		select {
		case <-ticker.C:
			currentProcessed := atomic.LoadInt64(&s.stats.TotalEventsProcessed)
			s.stats.EventsPerSecond = currentProcessed - lastProcessed
			lastProcessed = currentProcessed
		case <-s.shutdown:
			s.logger.Info().Msg("Metrics worker stopping")
			return
		}
	}
}

// flushBuffers flushes both event buffers
func (s *IngestionService) flushBuffers() {
	s.flushExposureEvents()
	s.flushMetricEvents()
}

// flushExposureEvents flushes exposure events to storage
func (s *IngestionService) flushExposureEvents() {
	s.bufferMutex.Lock()
	if len(s.exposureBuffer) == 0 {
		s.bufferMutex.Unlock()
		return
	}

	events := make([]storage.ExposureEvent, len(s.exposureBuffer))
	copy(events, s.exposureBuffer)
	s.exposureBuffer = s.exposureBuffer[:0] // Clear buffer
	s.bufferMutex.Unlock()

	// Store events
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.eventStorage.StoreExposureEvents(ctx, events); err != nil {
		s.logger.Error().Err(err).Int("count", len(events)).Msg("Failed to store exposure events")
		atomic.AddInt64(&s.stats.ErrorsLastHour, 1)
	}
}

// flushMetricEvents flushes metric events to storage
func (s *IngestionService) flushMetricEvents() {
	s.bufferMutex.Lock()
	if len(s.metricBuffer) == 0 {
		s.bufferMutex.Unlock()
		return
	}

	events := make([]storage.MetricEvent, len(s.metricBuffer))
	copy(events, s.metricBuffer)
	s.metricBuffer = s.metricBuffer[:0] // Clear buffer
	s.bufferMutex.Unlock()

	// Store events
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.eventStorage.StoreMetricEvents(ctx, events); err != nil {
		s.logger.Error().Err(err).Int("count", len(events)).Msg("Failed to store metric events")
		atomic.AddInt64(&s.stats.ErrorsLastHour, 1)
	}
}
