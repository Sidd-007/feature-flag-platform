package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/services"
)

// FlagHandler handles flag endpoints
type FlagHandler struct {
	flagService *services.FlagService
	logger      zerolog.Logger
}

// NewFlagHandler creates a new flag handler
func NewFlagHandler(flagService *services.FlagService, logger zerolog.Logger) *FlagHandler {
	return &FlagHandler{
		flagService: flagService,
		logger:      logger.With().Str("handler", "flag").Logger(),
	}
}

func (h *FlagHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *FlagHandler) sendError(w http.ResponseWriter, status int, code, message string) {
	h.sendJSON(w, status, map[string]interface{}{"error": code, "message": message})
}

// List flags
func (h *FlagHandler) List(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	flags, total, err := h.flagService.List(r.Context(), envID, limit, offset)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	// Convert flags to response format with individual published status
	flagsWithStatus := make([]map[string]interface{}, len(flags))
	for i, flag := range flags {
		flagMap := map[string]interface{}{
			"id":            flag.ID,
			"key":           flag.Key,
			"name":          flag.Name,
			"description":   flag.Description,
			"type":          flag.Type,
			"status":        flag.Status,
			"enabled":       flag.Status == "active",
			"default_value": flag.DefaultVariation,
			"created_at":    flag.CreatedAt,
			"updated_at":    flag.UpdatedAt,
			"version":       flag.Version,
			"published":     flag.Published, // Individual flag published status
		}
		flagsWithStatus[i] = flagMap
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":       flagsWithStatus,
		"pagination": map[string]interface{}{"page": page, "limit": limit, "total": total, "pages": (total + limit - 1) / limit},
	})
}

// Create flag
func (h *FlagHandler) Create(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	var req repository.CreateFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	flag, err := h.flagService.Create(r.Context(), envID, &req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusCreated, flag)
}

// Get flag
func (h *FlagHandler) Get(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	key := chi.URLParam(r, "flagKey")
	flag, err := h.flagService.GetByKey(r.Context(), envID, key)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	h.sendJSON(w, http.StatusOK, flag)
}

// Update flag (by key)
func (h *FlagHandler) Update(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}
	key := chi.URLParam(r, "flagKey")

	// Load current flag so we can support partial updates safely
	current, err := h.flagService.GetByKey(r.Context(), envID, key)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	// Decode into a generic map to support both {enabled: bool} and explicit fields
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// Prepare request with defaults from current
	req := repository.UpdateFlagRequest{
		Name:        current.Name,
		Description: current.Description,
		Status:      current.Status,
	}

	if v, ok := raw["name"].(string); ok && v != "" {
		req.Name = v
	}
	if v, ok := raw["description"].(string); ok {
		req.Description = v
	}
	if v, ok := raw["status"].(string); ok && v != "" {
		req.Status = v
	}
	if v, ok := raw["enabled"].(bool); ok {
		if v {
			req.Status = "active"
		} else {
			req.Status = "archived"
		}
	}

	flag, err := h.flagService.Update(r.Context(), current.ID, &req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	h.sendJSON(w, http.StatusOK, flag)
}

// Delete flag (by key)
func (h *FlagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}
	key := chi.URLParam(r, "flagKey")

	flag, err := h.flagService.GetByKey(r.Context(), envID, key)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if err := h.flagService.Delete(r.Context(), flag.ID); err != nil {
		h.sendError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Publish handles POST /orgs/{orgId}/projects/{projectId}/environments/{envId}/flags/{flagKey}/publish
func (h *FlagHandler) Publish(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	flagKey := chi.URLParam(r, "flagKey")
	if flagKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_flag_key", "Flag key is required")
		return
	}

	// Publish the individual flag
	publishedFlag, err := h.flagService.PublishFlag(r.Context(), envID, flagKey)
	if err != nil {
		h.logger.Error().Err(err).Str("env_id", envID.String()).Str("flag_key", flagKey).Msg("Failed to publish flag")
		h.sendError(w, http.StatusInternalServerError, "publish_failed", err.Error())
		return
	}

	h.logger.Info().Str("env_id", envID.String()).Str("flag_key", flagKey).Msg("Flag published successfully")

	response := map[string]interface{}{
		"message": "Flag published successfully",
		"flag":    publishedFlag,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Unpublish handles POST /orgs/{orgId}/projects/{projectId}/environments/{envId}/flags/{flagKey}/unpublish
func (h *FlagHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	flagKey := chi.URLParam(r, "flagKey")
	if flagKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_flag_key", "Flag key is required")
		return
	}

	// Unpublish the individual flag
	unpublishedFlag, err := h.flagService.UnpublishFlag(r.Context(), envID, flagKey)
	if err != nil {
		h.logger.Error().Err(err).Str("env_id", envID.String()).Str("flag_key", flagKey).Msg("Failed to unpublish flag")
		h.sendError(w, http.StatusInternalServerError, "unpublish_failed", err.Error())
		return
	}

	h.logger.Info().Str("env_id", envID.String()).Str("flag_key", flagKey).Msg("Flag unpublished successfully")

	response := map[string]interface{}{
		"message": "Flag unpublished successfully",
		"flag":    unpublishedFlag,
	}

	h.sendJSON(w, http.StatusOK, response)
}
