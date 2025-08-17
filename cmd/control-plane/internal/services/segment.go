package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/pkg/rbac"
)

// SegmentService handles segment business logic
type SegmentService struct {
	repos  *repository.Repositories
	rbac   *rbac.RBAC
	logger zerolog.Logger
}

// NewSegmentService creates a new segment service
func NewSegmentService(repos *repository.Repositories, rbacManager *rbac.RBAC, logger zerolog.Logger) *SegmentService {
	return &SegmentService{
		repos:  repos,
		rbac:   rbacManager,
		logger: logger.With().Str("service", "segment").Logger(),
	}
}

// Create creates a new segment
func (s *SegmentService) Create(ctx context.Context, envID uuid.UUID, req *repository.CreateSegmentRequest) (*repository.Segment, error) {
	// Validate request
	if req.Key == "" {
		return nil, fmt.Errorf("segment key is required")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("segment name is required")
	}

	// Check if key already exists
	exists, err := s.repos.Segment.CheckKeyExists(ctx, envID, req.Key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check key existence: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("segment key '%s' already exists", req.Key)
	}

	// Set environment ID
	req.EnvID = envID

	// Validate rules if provided
	if req.Rules != nil {
		if err := s.validateSegmentRules(req.Rules); err != nil {
			return nil, fmt.Errorf("invalid rules: %w", err)
		}
	} else {
		// Set default empty rules
		req.Rules = map[string]interface{}{
			"conditions": []interface{}{},
		}
	}

	// Create segment
	segment, err := s.repos.Segment.Create(ctx, req)
	if err != nil {
		s.logger.Error().Err(err).
			Str("segment_key", req.Key).
			Str("env_id", envID.String()).
			Msg("Failed to create segment")
		return nil, fmt.Errorf("failed to create segment: %w", err)
	}

	s.logger.Info().
		Str("segment_id", segment.ID.String()).
		Str("segment_key", segment.Key).
		Str("env_id", envID.String()).
		Msg("Segment created successfully")

	return segment, nil
}

// GetByID retrieves a segment by ID
func (s *SegmentService) GetByID(ctx context.Context, id uuid.UUID) (*repository.Segment, error) {
	segment, err := s.repos.Segment.GetByID(ctx, id)
	if err != nil {
		s.logger.Debug().Err(err).Str("segment_id", id.String()).Msg("Segment not found")
		return nil, fmt.Errorf("segment not found")
	}

	return segment, nil
}

// GetByKey retrieves a segment by key
func (s *SegmentService) GetByKey(ctx context.Context, envID uuid.UUID, key string) (*repository.Segment, error) {
	segment, err := s.repos.Segment.GetByKey(ctx, envID, key)
	if err != nil {
		s.logger.Debug().Err(err).
			Str("env_id", envID.String()).
			Str("segment_key", key).
			Msg("Segment not found")
		return nil, fmt.Errorf("segment not found")
	}

	return segment, nil
}

// List retrieves segments for an environment with pagination
func (s *SegmentService) List(ctx context.Context, envID uuid.UUID, limit, offset int) ([]*repository.Segment, int, error) {
	// Validate pagination parameters
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 200 {
		limit = 200 // Max limit
	}
	if offset < 0 {
		offset = 0
	}

	segments, total, err := s.repos.Segment.List(ctx, envID, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).
			Str("env_id", envID.String()).
			Msg("Failed to list segments")
		return nil, 0, fmt.Errorf("failed to list segments: %w", err)
	}

	s.logger.Debug().
		Str("env_id", envID.String()).
		Int("count", len(segments)).
		Int("total", total).
		Msg("Segments listed successfully")

	return segments, total, nil
}

// Update updates an existing segment
func (s *SegmentService) Update(ctx context.Context, id uuid.UUID, req *repository.UpdateSegmentRequest) (*repository.Segment, error) {
	// Get existing segment to validate environment access
	_, err := s.repos.Segment.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("segment not found")
	}

	// Validate rules if provided
	if req.Rules != nil {
		if err := s.validateSegmentRules(req.Rules); err != nil {
			return nil, fmt.Errorf("invalid rules: %w", err)
		}
	}

	// Update segment
	segment, err := s.repos.Segment.Update(ctx, id, req)
	if err != nil {
		s.logger.Error().Err(err).
			Str("segment_id", id.String()).
			Msg("Failed to update segment")
		return nil, fmt.Errorf("failed to update segment: %w", err)
	}

	s.logger.Info().
		Str("segment_id", segment.ID.String()).
		Str("segment_key", segment.Key).
		Msg("Segment updated successfully")

	return segment, nil
}

// Delete deletes a segment
func (s *SegmentService) Delete(ctx context.Context, id uuid.UUID) error {
	// Get existing segment to validate environment access
	existing, err := s.repos.Segment.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("segment not found")
	}

	// TODO: Check if segment is being used by any flags before deletion
	// For now, we'll allow deletion

	// Delete segment
	err = s.repos.Segment.Delete(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).
			Str("segment_id", id.String()).
			Msg("Failed to delete segment")
		return fmt.Errorf("failed to delete segment: %w", err)
	}

	s.logger.Info().
		Str("segment_id", id.String()).
		Str("segment_key", existing.Key).
		Msg("Segment deleted successfully")

	return nil
}

// validateSegmentRules validates segment targeting rules
func (s *SegmentService) validateSegmentRules(rules interface{}) error {
	// Convert to map for validation
	rulesMap, ok := rules.(map[string]interface{})
	if !ok {
		return fmt.Errorf("rules must be an object")
	}

	// Check for conditions array
	conditions, exists := rulesMap["conditions"]
	if !exists {
		return fmt.Errorf("rules must contain 'conditions' array")
	}

	conditionsArray, ok := conditions.([]interface{})
	if !ok {
		return fmt.Errorf("conditions must be an array")
	}

	// Validate each condition
	for i, cond := range conditionsArray {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			return fmt.Errorf("condition %d must be an object", i)
		}

		// Check required fields
		attribute, hasAttribute := condMap["attribute"]
		operator, hasOperator := condMap["operator"]
		value, hasValue := condMap["value"]

		if !hasAttribute || attribute == "" {
			return fmt.Errorf("condition %d must have 'attribute' field", i)
		}

		if !hasOperator || operator == "" {
			return fmt.Errorf("condition %d must have 'operator' field", i)
		}

		if !hasValue {
			return fmt.Errorf("condition %d must have 'value' field", i)
		}

		// Use the value to avoid unused variable error
		_ = value

		// Validate operator
		operatorStr, ok := operator.(string)
		if !ok {
			return fmt.Errorf("condition %d operator must be a string", i)
		}

		validOperators := []string{
			"equals", "not_equals", "contains", "not_contains",
			"starts_with", "ends_with", "regex", "in", "not_in",
			"greater_than", "greater_than_or_equal", "less_than", "less_than_or_equal",
			"exists", "not_exists", "semver_equals", "semver_greater", "semver_less",
		}

		isValidOperator := false
		for _, validOp := range validOperators {
			if operatorStr == validOp {
				isValidOperator = true
				break
			}
		}

		if !isValidOperator {
			return fmt.Errorf("condition %d has invalid operator '%s'", i, operatorStr)
		}
	}

	return nil
}
