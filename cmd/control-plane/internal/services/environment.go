package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/pkg/rbac"
)

// EnvironmentService handles environment operations
type EnvironmentService struct {
	repos  *repository.Repositories
	rbac   *rbac.RBAC
	logger zerolog.Logger
}

// NewEnvironmentService creates a new environment service
func NewEnvironmentService(repos *repository.Repositories, rbacManager *rbac.RBAC, logger zerolog.Logger) *EnvironmentService {
	return &EnvironmentService{
		repos:  repos,
		rbac:   rbacManager,
		logger: logger.With().Str("service", "environment").Logger(),
	}
}

func (s *EnvironmentService) Create(ctx context.Context, projectID uuid.UUID, req *repository.CreateEnvironmentRequest) (*repository.Environment, error) {
	// ensure project exists
	if _, err := s.repos.Project.GetByID(ctx, projectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	req.ProjectID = projectID
	env, err := s.repos.Environment.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment")
	}
	return env, nil
}

func (s *EnvironmentService) GetByID(ctx context.Context, id uuid.UUID) (*repository.Environment, error) {
	env, err := s.repos.Environment.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("environment not found")
		}
		return nil, fmt.Errorf("failed to retrieve environment")
	}
	return env, nil
}

func (s *EnvironmentService) List(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*repository.Environment, int, error) {
	envs, total, err := s.repos.Environment.List(ctx, projectID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve environments")
	}
	return envs, total, nil
}

func (s *EnvironmentService) Update(ctx context.Context, id uuid.UUID, req *repository.UpdateEnvironmentRequest) (*repository.Environment, error) {
	env, err := s.repos.Environment.Update(ctx, id, req)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("environment not found")
		}
		return nil, fmt.Errorf("failed to update environment")
	}
	return env, nil
}

func (s *EnvironmentService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repos.Environment.Delete(ctx, id); err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("environment not found")
		}
		return fmt.Errorf("failed to delete environment")
	}
	return nil
}
