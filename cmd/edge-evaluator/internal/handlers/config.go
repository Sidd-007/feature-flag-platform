package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/edge-evaluator/internal/services"
)

// ConfigHandler handles configuration endpoints
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

// StreamConfigUpdates handles GET /stream/{envKey} - Server-Sent Events for config updates
func (h *ConfigHandler) StreamConfigUpdates(w http.ResponseWriter, r *http.Request) {
	envKey := chi.URLParam(r, "envKey")
	if envKey == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_env_key", "Environment key is required")
		return
	}

	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get initial configuration
	config, err := h.configService.GetConfig(r.Context(), envKey)
	if err != nil {
		h.logger.Error().Err(err).Str("env_key", envKey).Msg("Failed to get initial config")
		h.sendError(w, http.StatusInternalServerError, "config_failed", "Failed to retrieve configuration")
		return
	}

	if config == nil {
		h.sendError(w, http.StatusNotFound, "env_not_found", "Environment not found")
		return
	}

	// Send initial configuration
	h.sendSSEEvent(w, "config", map[string]interface{}{
		"env_key": envKey,
		"version": config.Version,
		"etag":    config.ETag,
	})

	// TODO: Implement real-time updates via NATS subscription
	// For now, just send a heartbeat and close
	h.sendSSEEvent(w, "heartbeat", map[string]interface{}{
		"timestamp": config.UpdatedAt.Unix(),
	})

	h.logger.Info().Str("env_key", envKey).Msg("Config streaming session ended")
}

// Helper methods

func (h *ConfigHandler) sendSSEEvent(w http.ResponseWriter, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal SSE data")
		return
	}

	if _, err := w.Write([]byte("event: " + event + "\n")); err != nil {
		return
	}
	if _, err := w.Write([]byte("data: " + string(jsonData) + "\n\n")); err != nil {
		return
	}

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
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
