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

// ProjectHandler handles project endpoints
type ProjectHandler struct {
	projectService *services.ProjectService
	logger         zerolog.Logger
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService *services.ProjectService, logger zerolog.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		logger:         logger.With().Str("handler", "project").Logger(),
	}
}

// List handles GET /orgs/{orgId}/projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

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

	projects, total, err := h.projectService.List(r.Context(), orgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("Failed to list projects")
		h.sendError(w, http.StatusInternalServerError, "list_failed", "Failed to retrieve projects")
		return
	}

	response := map[string]interface{}{
		"data": projects,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + limit - 1) / limit,
		},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Create handles POST /orgs/{orgId}/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	var req repository.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// TODO: Add request validation

	project, err := h.projectService.Create(r.Context(), orgID, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("Failed to create project")
		h.sendError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusCreated, project)
}

// Get handles GET /orgs/{orgId}/projects/{projectId}
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_project_id", "Invalid project ID")
		return
	}

	project, err := h.projectService.GetByID(r.Context(), projectID)
	if err != nil {
		h.logger.Error().Err(err).Str("project_id", projectID.String()).Msg("Failed to get project")
		h.sendError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, project)
}

// Update handles PUT /orgs/{orgId}/projects/{projectId}
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_project_id", "Invalid project ID")
		return
	}

	var req repository.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	// TODO: Add request validation

	project, err := h.projectService.Update(r.Context(), projectID, &req)
	if err != nil {
		h.logger.Error().Err(err).Str("project_id", projectID.String()).Msg("Failed to update project")
		h.sendError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, project)
}

// Delete handles DELETE /orgs/{orgId}/projects/{projectId}
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_project_id", "Invalid project ID")
		return
	}

	err = h.projectService.Delete(r.Context(), projectID)
	if err != nil {
		h.logger.Error().Err(err).Str("project_id", projectID.String()).Msg("Failed to delete project")
		h.sendError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *ProjectHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (h *ProjectHandler) sendError(w http.ResponseWriter, status int, code, message string) {
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
