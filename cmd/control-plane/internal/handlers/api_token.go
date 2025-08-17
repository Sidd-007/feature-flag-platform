package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/services"
)

// APITokenHandler handles API token HTTP requests
type APITokenHandler struct {
	tokenService *services.APITokenService
	logger       zerolog.Logger
}

// NewAPITokenHandler creates a new API token handler
func NewAPITokenHandler(tokenService *services.APITokenService, logger zerolog.Logger) *APITokenHandler {
	return &APITokenHandler{
		tokenService: tokenService,
		logger:       logger.With().Str("handler", "api_token").Logger(),
	}
}

// List handles GET /orgs/{orgId}/projects/{projectId}/environments/{envId}/tokens
func (h *APITokenHandler) List(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	tokens, err := h.tokenService.List(r.Context(), envID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("env_id", envID.String()).Msg("Failed to list API tokens")
		h.sendError(w, http.StatusInternalServerError, "list_failed", "Failed to list API tokens")
		return
	}

	response := map[string]interface{}{
		"data":   tokens,
		"limit":  limit,
		"offset": offset,
		"total":  len(tokens), // TODO: Get actual count from repository
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Create handles POST /orgs/{orgId}/projects/{projectId}/environments/{envId}/tokens
func (h *APITokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	envIDStr := chi.URLParam(r, "envId")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Scope       string `json:"scope"`
		ExpiresAt   string `json:"expires_at,omitempty"` // ISO 8601 format
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// Validate required fields
	if body.Name == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Token name is required")
		return
	}

	if body.Scope != "read" && body.Scope != "write" {
		h.sendError(w, http.StatusBadRequest, "invalid_scope", "Scope must be 'read' or 'write'")
		return
	}

	// Parse expiration time if provided
	var expiresAt *time.Time
	if body.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, body.ExpiresAt); err != nil {
			h.sendError(w, http.StatusBadRequest, "invalid_expires_at", "Invalid expiration time format (use RFC3339)")
			return
		} else {
			expiresAt = &t
		}
	}

	req := &services.CreateTokenRequest{
		EnvID:       envID,
		Name:        body.Name,
		Description: body.Description,
		Scope:       body.Scope,
		ExpiresAt:   expiresAt,
	}

	result, err := h.tokenService.Create(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Str("env_id", envID.String()).Msg("Failed to create API token")
		h.sendError(w, http.StatusBadRequest, "creation_failed", err.Error())
		return
	}

	h.logger.Info().
		Str("token_id", result.Token.ID.String()).
		Str("env_id", envID.String()).
		Str("scope", body.Scope).
		Msg("API token created successfully")

	h.sendJSON(w, http.StatusCreated, result)
}

// Revoke handles DELETE /orgs/{orgId}/projects/{projectId}/environments/{envId}/tokens/{tokenId}
func (h *APITokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tokenIDStr := chi.URLParam(r, "tokenId")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_token_id", "Invalid token ID")
		return
	}

	err = h.tokenService.Revoke(r.Context(), tokenID)
	if err != nil {
		h.logger.Error().Err(err).Str("token_id", tokenID.String()).Msg("Failed to revoke API token")
		h.sendError(w, http.StatusInternalServerError, "revoke_failed", "Failed to revoke API token")
		return
	}

	h.logger.Info().Str("token_id", tokenID.String()).Msg("API token revoked successfully")

	response := map[string]interface{}{
		"message": "API token revoked successfully",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *APITokenHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *APITokenHandler) sendError(w http.ResponseWriter, status int, code, message string) {
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
