package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/cmd/control-plane/internal/services"
)

// SegmentHandler handles segment HTTP requests
type SegmentHandler struct {
	segmentService *services.SegmentService
	logger         zerolog.Logger
}

// NewSegmentHandler creates a new segment handler
func NewSegmentHandler(segmentService *services.SegmentService, logger zerolog.Logger) *SegmentHandler {
	return &SegmentHandler{
		segmentService: segmentService,
		logger:         logger.With().Str("handler", "segment").Logger(),
	}
}

// List handles GET /segments
func (h *SegmentHandler) List(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid environment ID")
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	segments, total, err := h.segmentService.List(r.Context(), envID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list segments")
		h.sendError(w, http.StatusInternalServerError, "Failed to list segments")
		return
	}

	response := map[string]interface{}{
		"data":   segments,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Create handles POST /segments
func (h *SegmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid environment ID")
		return
	}

	var req repository.CreateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	segment, err := h.segmentService.Create(r.Context(), envID, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create segment")

		// Check for specific errors
		if err.Error() == "segment key already exists" ||
			err.Error() == "segment key is required" ||
			err.Error() == "segment name is required" ||
			err.Error() == "invalid rules" {
			h.sendError(w, http.StatusBadRequest, err.Error())
			return
		}

		h.sendError(w, http.StatusInternalServerError, "Failed to create segment")
		return
	}

	h.sendJSON(w, http.StatusCreated, segment)
}

// Get handles GET /segments/{segmentId}
func (h *SegmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	segmentIDStr := chi.URLParam(r, "segmentId")
	segmentID, err := uuid.Parse(segmentIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	segment, err := h.segmentService.GetByID(r.Context(), segmentID)
	if err != nil {
		if err.Error() == "segment not found" {
			h.sendError(w, http.StatusNotFound, "Segment not found")
			return
		}

		h.logger.Error().Err(err).Msg("Failed to get segment")
		h.sendError(w, http.StatusInternalServerError, "Failed to get segment")
		return
	}

	h.sendJSON(w, http.StatusOK, segment)
}

// Update handles PUT /segments/{segmentId}
func (h *SegmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	segmentIDStr := chi.URLParam(r, "segmentId")
	segmentID, err := uuid.Parse(segmentIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	var req repository.UpdateSegmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	segment, err := h.segmentService.Update(r.Context(), segmentID, &req)
	if err != nil {
		if err.Error() == "segment not found" {
			h.sendError(w, http.StatusNotFound, "Segment not found")
			return
		}

		if err.Error() == "invalid rules" {
			h.sendError(w, http.StatusBadRequest, err.Error())
			return
		}

		h.logger.Error().Err(err).Msg("Failed to update segment")
		h.sendError(w, http.StatusInternalServerError, "Failed to update segment")
		return
	}

	h.sendJSON(w, http.StatusOK, segment)
}

// Delete handles DELETE /segments/{segmentId}
func (h *SegmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	segmentIDStr := chi.URLParam(r, "segmentId")
	segmentID, err := uuid.Parse(segmentIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid segment ID")
		return
	}

	err = h.segmentService.Delete(r.Context(), segmentID)
	if err != nil {
		if err.Error() == "segment not found" {
			h.sendError(w, http.StatusNotFound, "Segment not found")
			return
		}

		h.logger.Error().Err(err).Msg("Failed to delete segment")
		h.sendError(w, http.StatusInternalServerError, "Failed to delete segment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sendJSON sends a JSON response
func (h *SegmentHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// sendError sends an error response
func (h *SegmentHandler) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode error response")
	}
}
