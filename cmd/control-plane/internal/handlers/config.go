package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/services"
)

// ConfigHandler handles configuration endpoints for edge evaluators
type ConfigHandler struct {
	configService *services.ConfigService
	logger        zerolog.Logger
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configService *services.ConfigService, logger zerolog.Logger) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
		logger:        logger.With().Str("handler", "config").Logger(),
	}
}

// GetEnvironmentConfig handles GET /configs/{envKey}
func (h *ConfigHandler) GetEnvironmentConfig(w http.ResponseWriter, r *http.Request) {
	envKey := chi.URLParam(r, "envKey")
	if envKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_env_key", "Environment key is required")
		return
	}

	// Check if envKey is actually a UUID (environment ID) and convert to key
	if len(envKey) == 36 && strings.Count(envKey, "-") == 4 {
		// This looks like a UUID, try to get environment by ID first
		if envID, err := uuid.Parse(envKey); err == nil {
			// Get environment by ID and use its key instead
			if env, err := h.configService.GetEnvironmentByID(r.Context(), envID); err == nil {
				envKey = env.Key // Use the actual environment key
				h.logger.Debug().Str("env_id", envID.String()).Str("env_key", envKey).Msg("Converted environment ID to key")
			}
		}
	}

	// Get environment config
	config, err := h.configService.GetEnvironmentConfig(r.Context(), envKey)
	if err != nil {
		h.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to get environment config")
		h.sendError(w, http.StatusNotFound, "not_found", "Environment configuration not found")
		return
	}

	// Set cache headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", config.ETag)
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 minutes

	// Check If-None-Match header for 304 Not Modified
	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == config.ETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Return config
	h.sendJSON(w, http.StatusOK, config)
}

// Helper methods

func (h *ConfigHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *ConfigHandler) sendError(w http.ResponseWriter, status int, code, message string) {
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
