package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/services"
	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/storage"
)

// EventsHandler handles event ingestion endpoints
type EventsHandler struct {
	ingestionService *services.IngestionService
	logger           zerolog.Logger
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(ingestionService *services.IngestionService, logger zerolog.Logger) *EventsHandler {
	return &EventsHandler{
		ingestionService: ingestionService,
		logger:           logger.With().Str("handler", "events").Logger(),
	}
}

// IngestExposureEvents handles POST /events/exposure
func (h *EventsHandler) IngestExposureEvents(w http.ResponseWriter, r *http.Request) {
	var batch struct {
		Events []storage.ExposureEvent `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	if len(batch.Events) == 0 {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "No events provided")
		return
	}

	if len(batch.Events) > 1000 {
		h.sendError(w, http.StatusBadRequest, "batch_too_large", "Batch size cannot exceed 1000 events")
		return
	}

	accepted, validationErrors, err := h.ingestionService.IngestExposureEvents(r.Context(), batch.Events)
	if err != nil {
		h.logger.Error().Err(err).Int("events_count", len(batch.Events)).Msg("Failed to ingest exposure events")
		h.sendError(w, http.StatusInternalServerError, "ingestion_failed", "Failed to process events")
		return
	}

	response := map[string]interface{}{
		"accepted_count": accepted,
		"rejected_count": len(batch.Events) - accepted,
		"processed_at":   time.Now(),
		"request_id":     middleware.GetReqID(r.Context()),
	}

	if len(validationErrors) > 0 {
		response["errors"] = validationErrors
	}

	h.sendJSON(w, http.StatusAccepted, response)
}

// IngestMetricEvents handles POST /events/metrics
func (h *EventsHandler) IngestMetricEvents(w http.ResponseWriter, r *http.Request) {
	var batch struct {
		Events []storage.MetricEvent `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	if len(batch.Events) == 0 {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "No events provided")
		return
	}

	if len(batch.Events) > 1000 {
		h.sendError(w, http.StatusBadRequest, "batch_too_large", "Batch size cannot exceed 1000 events")
		return
	}

	accepted, validationErrors, err := h.ingestionService.IngestMetricEvents(r.Context(), batch.Events)
	if err != nil {
		h.logger.Error().Err(err).Int("events_count", len(batch.Events)).Msg("Failed to ingest metric events")
		h.sendError(w, http.StatusInternalServerError, "ingestion_failed", "Failed to process events")
		return
	}

	response := map[string]interface{}{
		"accepted_count": accepted,
		"rejected_count": len(batch.Events) - accepted,
		"processed_at":   time.Now(),
		"request_id":     middleware.GetReqID(r.Context()),
	}

	if len(validationErrors) > 0 {
		response["errors"] = validationErrors
	}

	h.sendJSON(w, http.StatusAccepted, response)
}

// IngestEventBatch handles POST /events/batch
func (h *EventsHandler) IngestEventBatch(w http.ResponseWriter, r *http.Request) {
	var batch services.EventBatch

	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	totalEvents := len(batch.ExposureEvents) + len(batch.MetricEvents)
	if totalEvents == 0 {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "No events provided")
		return
	}

	if totalEvents > 1000 {
		h.sendError(w, http.StatusBadRequest, "batch_too_large", "Batch size cannot exceed 1000 events")
		return
	}

	accepted, validationErrors, err := h.ingestionService.IngestEventBatch(r.Context(), batch)
	if err != nil {
		h.logger.Error().Err(err).
			Int("exposure_events", len(batch.ExposureEvents)).
			Int("metric_events", len(batch.MetricEvents)).
			Msg("Failed to ingest event batch")
		h.sendError(w, http.StatusInternalServerError, "ingestion_failed", "Failed to process events")
		return
	}

	response := map[string]interface{}{
		"accepted_count": accepted,
		"rejected_count": totalEvents - accepted,
		"batch_id":       batch.BatchID,
		"processed_at":   time.Now(),
		"request_id":     middleware.GetReqID(r.Context()),
	}

	if len(validationErrors) > 0 {
		response["errors"] = validationErrors
	}

	h.sendJSON(w, http.StatusAccepted, response)
}

// Helper methods

func (h *EventsHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *EventsHandler) sendError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := map[string]interface{}{
		"error":   code,
		"message": message,
	}

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode error response")
	}
}
