package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/middleware"
	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/cmd/control-plane/internal/services"
)

// OrganizationHandler handles organization endpoints
type OrganizationHandler struct {
	orgService *services.OrganizationService
	logger     zerolog.Logger
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgService *services.OrganizationService, logger zerolog.Logger) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
		logger:     logger.With().Str("handler", "organization").Logger(),
	}
}

// List handles GET /orgs
func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		h.sendError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	userID, err := uuid.Parse(authCtx.UserID)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_user", "Invalid user ID")
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Msg("Listing organizations for user")

	// Parse pagination parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	orgs, total, err := h.orgService.List(r.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to list organizations")
		h.sendError(w, http.StatusInternalServerError, "list_failed", "Failed to retrieve organizations")
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Int("orgs_count", len(orgs)).Int("total", total).Msg("Organizations retrieved")

	response := map[string]interface{}{
		"data": orgs,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Create handles POST /orgs
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	authCtx := middleware.GetAuthContext(r)
	if authCtx == nil {
		h.sendError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	userID, err := uuid.Parse(authCtx.UserID)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_user", "Invalid user ID")
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Msg("Creating organization for user")

	var req repository.CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// TODO: Add request validation

	org, err := h.orgService.Create(r.Context(), &req, userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to create organization")
		h.sendError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	h.logger.Info().Str("user_id", userID.String()).Str("org_id", org.ID.String()).Msg("Organization created successfully")

	h.sendJSON(w, http.StatusCreated, org)
}

// Get handles GET /orgs/{orgId}
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	org, err := h.orgService.GetByID(r.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("Failed to get organization")
		h.sendError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, org)
}

// Update handles PUT /orgs/{orgId}
func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	var req repository.UpdateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// TODO: Add request validation

	org, err := h.orgService.Update(r.Context(), orgID, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("Failed to update organization")
		h.sendError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, org)
}

// Delete handles DELETE /orgs/{orgId}
func (h *OrganizationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	err = h.orgService.Delete(r.Context(), orgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("Failed to delete organization")
		h.sendError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *OrganizationHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *OrganizationHandler) sendError(w http.ResponseWriter, status int, code, message string) {
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
