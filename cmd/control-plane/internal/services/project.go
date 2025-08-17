package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/pkg/rbac"
)

// ProjectService handles project operations
type ProjectService struct {
	repos  *repository.Repositories
	rbac   *rbac.RBAC
	logger zerolog.Logger
}

// NewProjectService creates a new project service
func NewProjectService(repos *repository.Repositories, rbacManager *rbac.RBAC, logger zerolog.Logger) *ProjectService {
	return &ProjectService{
		repos:  repos,
		rbac:   rbacManager,
		logger: logger.With().Str("service", "project").Logger(),
	}
}

// Create creates a new project
func (s *ProjectService) Create(ctx context.Context, orgID uuid.UUID, req *repository.CreateProjectRequest) (*repository.Project, error) {
	// Check if project key already exists in the organization
	exists, err := s.repos.Project.CheckKeyExists(ctx, orgID, req.Key)
	if err != nil {
		s.logger.Error().Err(err).Str("org_id", orgID.String()).Str("key", req.Key).Msg("Failed to check key existence")
		return nil, fmt.Errorf("failed to validate project key")
	}
	if exists {
		return nil, fmt.Errorf("project key already exists in organization")
	}

	// Set the org ID from the context
	req.OrgID = orgID

	// Create project
	project, err := s.repos.Project.Create(ctx, req)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create project")
		return nil, fmt.Errorf("failed to create project")
	}

	s.logger.Info().Str("project_id", project.ID.String()).Str("org_id", orgID.String()).Msg("Project created")
	return project, nil
}

// GetByID retrieves a project by ID
func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*repository.Project, error) {
	project, err := s.repos.Project.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("project not found")
		}
		s.logger.Error().Err(err).Str("project_id", id.String()).Msg("Failed to get project")
		return nil, fmt.Errorf("failed to retrieve project")
	}

	return project, nil
}

// List retrieves projects for an organization
func (s *ProjectService) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*repository.Project, int, error) {
	projects, total, err := s.repos.Project.List(ctx, orgID, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("Failed to list projects")
		return nil, 0, fmt.Errorf("failed to retrieve projects")
	}

	s.logger.Info().Str("org_id", orgID.String()).Int("projects_count", len(projects)).Int("total", total).Msg("Projects listed")
	return projects, total, nil
}

// Update updates a project
func (s *ProjectService) Update(ctx context.Context, id uuid.UUID, req *repository.UpdateProjectRequest) (*repository.Project, error) {
	project, err := s.repos.Project.Update(ctx, id, req)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("project not found")
		}
		s.logger.Error().Err(err).Str("project_id", id.String()).Msg("Failed to update project")
		return nil, fmt.Errorf("failed to update project")
	}

	s.logger.Info().Str("project_id", project.ID.String()).Msg("Project updated")
	return project, nil
}

// Delete deletes a project
func (s *ProjectService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repos.Project.Delete(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("project not found")
		}
		s.logger.Error().Err(err).Str("project_id", id.String()).Msg("Failed to delete project")
		return fmt.Errorf("failed to delete project")
	}

	s.logger.Info().Str("project_id", id.String()).Msg("Project deleted")
	return nil
}
