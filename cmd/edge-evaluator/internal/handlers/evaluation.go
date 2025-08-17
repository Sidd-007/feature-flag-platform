package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/edge-evaluator/internal/services"
	"github.com/feature-flag-platform/pkg/bucketing"
)

// EvaluationHandler handles flag evaluation endpoints
type EvaluationHandler struct {
	evaluationService *services.EvaluationService
	logger            zerolog.Logger
}

// NewEvaluationHandler creates a new evaluation handler
func NewEvaluationHandler(evaluationService *services.EvaluationService, logger zerolog.Logger) *EvaluationHandler {
	return &EvaluationHandler{
		evaluationService: evaluationService,
		logger:            logger.With().Str("handler", "evaluation").Logger(),
	}
}

// EvaluateFlags handles POST /evaluate
func (h *EvaluationHandler) EvaluateFlags(w http.ResponseWriter, r *http.Request) {
	var req services.EvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// Validate request
	if req.EnvKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Environment key is required")
		return
	}

	if req.Context == nil || req.Context.UserKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "User context with user_key is required")
		return
	}

	// Add request ID to response
	requestID := middleware.GetReqID(r.Context())

	response, err := h.evaluationService.EvaluateFlags(r.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Str("env_key", req.EnvKey).Msg("Failed to evaluate flags")
		h.sendError(w, http.StatusInternalServerError, "evaluation_failed", err.Error())
		return
	}

	response.RequestID = requestID
	h.sendJSON(w, http.StatusOK, response)
}

// EvaluateAllFlags handles POST /evaluate/{envKey}
func (h *EvaluationHandler) EvaluateAllFlags(w http.ResponseWriter, r *http.Request) {
	envKey := chi.URLParam(r, "envKey")
	if envKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_env_key", "Environment key is required")
		return
	}

	var body struct {
		Context       *bucketing.Context `json:"context"`
		IncludeReason bool               `json:"include_reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	if body.Context == nil || body.Context.UserKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "User context with user_key is required")
		return
	}

	req := &services.EvaluationRequest{
		EnvKey:        envKey,
		Context:       body.Context,
		IncludeReason: body.IncludeReason,
	}

	requestID := middleware.GetReqID(r.Context())

	response, err := h.evaluationService.EvaluateFlags(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to evaluate all flags")
		h.sendError(w, http.StatusInternalServerError, "evaluation_failed", err.Error())
		return
	}

	response.RequestID = requestID
	h.sendJSON(w, http.StatusOK, response)
}

// EvaluateFlag handles POST /evaluate/{envKey}/{flagKey}
func (h *EvaluationHandler) EvaluateFlag(w http.ResponseWriter, r *http.Request) {
	envKey := chi.URLParam(r, "envKey")
	flagKey := chi.URLParam(r, "flagKey")

	if envKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_env_key", "Environment key is required")
		return
	}

	if flagKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_flag_key", "Flag key is required")
		return
	}

	var body struct {
		Context *bucketing.Context `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	if body.Context == nil || body.Context.UserKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "User context with user_key is required")
		return
	}

	result, err := h.evaluationService.EvaluateFlag(r.Context(), envKey, flagKey, body.Context)
	if err != nil {
		h.logger.Error().Err(err).Str("env_key", envKey).Str("flag_key", flagKey).Msg("Failed to evaluate flag")
		h.sendError(w, http.StatusInternalServerError, "evaluation_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, result)
}

// Helper methods

func (h *EvaluationHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *EvaluationHandler) sendError(w http.ResponseWriter, status int, code, message string) {
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
