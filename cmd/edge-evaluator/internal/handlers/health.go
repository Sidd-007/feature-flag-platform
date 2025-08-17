package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/edge-evaluator/internal/services"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	evaluationService *services.EvaluationService
	configService     *services.ConfigService
	logger            zerolog.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(evaluationService *services.EvaluationService, configService *services.ConfigService, logger zerolog.Logger) *HealthHandler {
	return &HealthHandler{
		evaluationService: evaluationService,
		configService:     configService,
		logger:            logger.With().Str("handler", "health").Logger(),
	}
}

// Ready handles GET /ready - readiness probe
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Check if all dependencies are ready
	status := "ready"
	dependencies := make(map[string]string)

	// Check cache statistics
	stats := h.configService.GetCacheStats()
	dependencies["config_cache"] = "ready"

	// TODO: Add more dependency checks (Redis, NATS, etc.)

	response := map[string]interface{}{
		"status":       status,
		"timestamp":    time.Now(),
		"service":      "edge-evaluator",
		"version":      "1.0.0",
		"dependencies": dependencies,
		"cache_stats":  stats,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Live handles GET /live - liveness probe
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"service":   "edge-evaluator",
		"version":   "1.0.0",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *HealthHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}
