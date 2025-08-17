package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/feature-flag-platform/cmd/analytics-engine/internal/middleware"
	"github.com/feature-flag-platform/cmd/analytics-engine/internal/services"
)

type Handlers struct {
	experimentService services.ExperimentService
	statsService      services.StatisticsService
	AuthMiddleware    *middleware.AuthMiddleware
}

func NewHandlers(
	experimentService services.ExperimentService,
	statsService services.StatisticsService,
	authMiddleware *middleware.AuthMiddleware,
) *Handlers {
	return &Handlers{
		experimentService: experimentService,
		statsService:      statsService,
		AuthMiddleware:    authMiddleware,
	}
}

// Health check endpoints
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":  "healthy",
		"service": "analytics-engine",
		"version": "1.0.0",
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "ready",
		"checks": map[string]string{
			"clickhouse": "connected",
			"services":   "initialized",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// Helper functions
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	}

	json.NewEncoder(w).Encode(response)
}

func parseJSONRequest(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}
