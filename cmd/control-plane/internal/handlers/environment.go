package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/cmd/control-plane/internal/services"
)

// EnvironmentHandler handles environment endpoints
type EnvironmentHandler struct {
	envService *services.EnvironmentService
	logger     zerolog.Logger
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(envService *services.EnvironmentService, logger zerolog.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{
		envService: envService,
		logger:     logger.With().Str("handler", "environment").Logger(),
	}
}

func (h *EnvironmentHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *EnvironmentHandler) sendError(w http.ResponseWriter, status int, code, message string) {
	h.sendJSON(w, status, map[string]interface{}{"error": code, "message": message})
}

// List environments
func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_project_id", "Invalid project ID")
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

	envs, total, err := h.envService.List(r.Context(), projectID, limit, offset)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":       envs,
		"pagination": map[string]interface{}{"page": page, "limit": limit, "total": total, "pages": (total + limit - 1) / limit},
	})
}

// Create environment
func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_project_id", "Invalid project ID")
		return
	}

	var req repository.CreateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	env, err := h.envService.Create(r.Context(), projectID, &req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	h.sendJSON(w, http.StatusCreated, env)
}

// Get environment
func (h *EnvironmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "envId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	env, err := h.envService.GetByID(r.Context(), id)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	h.sendJSON(w, http.StatusOK, env)
}

// Update environment
func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "envId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	var req repository.UpdateEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload")
		return
	}

	env, err := h.envService.Update(r.Context(), id, &req)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	h.sendJSON(w, http.StatusOK, env)
}

// Delete environment
func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "envId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_env_id", "Invalid environment ID")
		return
	}

	if err := h.envService.Delete(r.Context(), id); err != nil {
		h.sendError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
