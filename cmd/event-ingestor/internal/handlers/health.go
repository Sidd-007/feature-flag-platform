package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/event-ingestor/internal/services"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	ingestionService *services.IngestionService
	logger           zerolog.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(ingestionService *services.IngestionService, logger zerolog.Logger) *HealthHandler {
	return &HealthHandler{
		ingestionService: ingestionService,
		logger:           logger.With().Str("handler", "health").Logger(),
	}
}

// Health handles GET /health - basic health check
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "event-ingestor",
		"version":   "1.0.0",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Ready handles GET /ready - readiness probe
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Check if all dependencies are ready
	status := "ready"
	dependencies := make(map[string]string)

	// Check ingestion service
	stats := h.ingestionService.GetStats()
	dependencies["ingestion_service"] = "ready"

	// TODO: Add checks for ClickHouse, NATS, etc.

	response := map[string]interface{}{
		"status":       status,
		"timestamp":    time.Now(),
		"service":      "event-ingestor",
		"version":      "1.0.0",
		"dependencies": dependencies,
		"stats":        stats,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Live handles GET /live - liveness probe
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"service":   "event-ingestor",
		"version":   "1.0.0",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Stats handles GET /stats - detailed ingestion statistics
func (h *HealthHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats := h.ingestionService.GetStats()

	response := map[string]interface{}{
		"stats":     stats,
		"timestamp": time.Now(),
		"service":   "event-ingestor",
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
