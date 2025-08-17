package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
	logger      zerolog.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger.With().Str("handler", "auth").Logger(),
	}
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req services.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// TODO: Add request validation

	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("Login failed")
		h.sendError(w, http.StatusUnauthorized, "login_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req services.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// TODO: Add request validation

	user, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("Registration failed")
		h.sendError(w, http.StatusBadRequest, "registration_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusCreated, user)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	if req.RefreshToken == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Refresh token is required")
		return
	}

	response, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("Token refresh failed")
		h.sendError(w, http.StatusUnauthorized, "refresh_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *AuthHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *AuthHandler) sendError(w http.ResponseWriter, status int, code, message string) {
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
