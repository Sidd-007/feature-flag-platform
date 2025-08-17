package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/pkg/rbac"
)

// OrganizationService handles organization operations
type OrganizationService struct {
	repos  *repository.Repositories
	rbac   *rbac.RBAC
	logger zerolog.Logger
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(repos *repository.Repositories, rbacManager *rbac.RBAC, logger zerolog.Logger) *OrganizationService {
	return &OrganizationService{
		repos:  repos,
		rbac:   rbacManager,
		logger: logger.With().Str("service", "organization").Logger(),
	}
}

// Create creates a new organization
func (s *OrganizationService) Create(ctx context.Context, req *repository.CreateOrganizationRequest, userID uuid.UUID) (*repository.Organization, error) {
	// Check if slug already exists
	exists, err := s.repos.Organization.CheckSlugExists(ctx, req.Slug)
	if err != nil {
		s.logger.Error().Err(err).Str("slug", req.Slug).Msg("Failed to check slug existence")
		return nil, fmt.Errorf("failed to validate organization slug")
	}
	if exists {
		return nil, fmt.Errorf("organization slug already exists")
	}

	// Create organization
	org, err := s.repos.Organization.Create(ctx, req)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create organization")
		return nil, fmt.Errorf("failed to create organization")
	}

	// Assign owner role to the creator
	subject := rbac.Subject{ID: userID.String(), Type: "user"}
	if err := s.rbac.AssignRole(subject, rbac.RoleOwner, org.ID.String()); err != nil {
		s.logger.Error().Err(err).Str("org_id", org.ID.String()).Str("user_id", userID.String()).
			Msg("Failed to assign owner role")
		// Note: In a production system, you might want to rollback the organization creation
	}

	// Ensure the user is a member of the organization so that list queries return it
	if err := s.repos.Organization.AddUserMembership(ctx, userID, org.ID, "owner"); err != nil {
		s.logger.Error().Err(err).Str("org_id", org.ID.String()).Str("user_id", userID.String()).
			Msg("Failed to add creator as organization member")
		// Do not fail the creation if membership insert fails; RBAC assignment above should still grant access
	}

	s.logger.Info().Str("org_id", org.ID.String()).Str("user_id", userID.String()).
		Msg("Organization created and owner role assigned")

	return org, nil
}

// GetByID retrieves an organization by ID
func (s *OrganizationService) GetByID(ctx context.Context, id uuid.UUID) (*repository.Organization, error) {
	org, err := s.repos.Organization.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("organization not found")
		}
		s.logger.Error().Err(err).Str("org_id", id.String()).Msg("Failed to get organization")
		return nil, fmt.Errorf("failed to retrieve organization")
	}

	return org, nil
}

// List retrieves organizations for a user
func (s *OrganizationService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*repository.Organization, int, error) {
	// First, let's debug by getting user memberships
	memberships, err := s.repos.Organization.GetUserMemberships(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get user memberships")
	} else {
		s.logger.Info().Str("user_id", userID.String()).Interface("memberships", memberships).Msg("User memberships found")
	}

	orgs, total, err := s.repos.Organization.List(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to list organizations")
		return nil, 0, fmt.Errorf("failed to retrieve organizations")
	}

	s.logger.Info().Str("user_id", userID.String()).Int("orgs_count", len(orgs)).Int("total", total).Msg("Organizations listed")
	return orgs, total, nil
}

// Update updates an organization
func (s *OrganizationService) Update(ctx context.Context, id uuid.UUID, req *repository.UpdateOrganizationRequest) (*repository.Organization, error) {
	org, err := s.repos.Organization.Update(ctx, id, req)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("organization not found")
		}
		s.logger.Error().Err(err).Str("org_id", id.String()).Msg("Failed to update organization")
		return nil, fmt.Errorf("failed to update organization")
	}

	s.logger.Info().Str("org_id", org.ID.String()).Msg("Organization updated")
	return org, nil
}

// Delete deletes an organization
func (s *OrganizationService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repos.Organization.Delete(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("organization not found")
		}
		s.logger.Error().Err(err).Str("org_id", id.String()).Msg("Failed to delete organization")
		return fmt.Errorf("failed to delete organization")
	}

	s.logger.Info().Str("org_id", id.String()).Msg("Organization deleted")
	return nil
}
