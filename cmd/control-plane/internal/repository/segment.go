package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Segment represents a user segment for targeting
type Segment struct {
	ID          uuid.UUID `json:"id"`
	EnvID       uuid.UUID `json:"env_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Rules       []byte    `json:"rules"` // JSON rules for segment matching
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int       `json:"version"`
}

// CreateSegmentRequest represents a request to create a segment
type CreateSegmentRequest struct {
	EnvID       uuid.UUID   `json:"env_id"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Rules       interface{} `json:"rules"` // Will be marshaled to JSON
}

// UpdateSegmentRequest represents a request to update a segment
type UpdateSegmentRequest struct {
	Name        *string     `json:"name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Rules       interface{} `json:"rules,omitempty"`
	IsActive    *bool       `json:"is_active,omitempty"`
}

// SegmentRepository handles segment persistence
type SegmentRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewSegmentRepository creates a new segment repository
func NewSegmentRepository(db *pgxpool.Pool, logger zerolog.Logger) *SegmentRepository {
	return &SegmentRepository{
		db:     db,
		logger: logger.With().Str("repository", "segment").Logger(),
	}
}

// Create creates a new segment
func (r *SegmentRepository) Create(ctx context.Context, req *CreateSegmentRequest) (*Segment, error) {
	// Marshal rules to JSON
	rulesJSON, err := json.Marshal(req.Rules)
	if err != nil {
		return nil, err
	}

	segment := &Segment{
		ID:          uuid.New(),
		EnvID:       req.EnvID,
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Rules:       rulesJSON,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}

	query := `
		INSERT INTO segments (id, env_id, key, name, description, rules_json, is_active, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, env_id, key, name, description, rules_json, is_active, created_at, updated_at, version`

	err = r.db.QueryRow(ctx, query,
		segment.ID, segment.EnvID, segment.Key, segment.Name, segment.Description,
		segment.Rules, segment.IsActive, segment.CreatedAt, segment.UpdatedAt, segment.Version,
	).Scan(
		&segment.ID, &segment.EnvID, &segment.Key, &segment.Name, &segment.Description,
		&segment.Rules, &segment.IsActive, &segment.CreatedAt, &segment.UpdatedAt, &segment.Version,
	)

	if err != nil {
		r.logger.Error().Err(err).
			Str("segment_key", req.Key).
			Str("env_id", req.EnvID.String()).
			Msg("Failed to create segment")
		return nil, err
	}

	r.logger.Info().
		Str("segment_id", segment.ID.String()).
		Str("segment_key", segment.Key).
		Str("env_id", segment.EnvID.String()).
		Msg("Segment created")

	return segment, nil
}

// GetByID retrieves a segment by ID
func (r *SegmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*Segment, error) {
	segment := &Segment{}

	query := `
		SELECT id, env_id, key, name, description, rules_json, is_active, created_at, updated_at, version
		FROM segments
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&segment.ID, &segment.EnvID, &segment.Key, &segment.Name, &segment.Description,
		&segment.Rules, &segment.IsActive, &segment.CreatedAt, &segment.UpdatedAt, &segment.Version,
	)

	if err != nil {
		r.logger.Debug().Err(err).Str("segment_id", id.String()).Msg("Segment not found")
		return nil, err
	}

	return segment, nil
}

// GetByKey retrieves a segment by key within an environment
func (r *SegmentRepository) GetByKey(ctx context.Context, envID uuid.UUID, key string) (*Segment, error) {
	segment := &Segment{}

	query := `
		SELECT id, env_id, key, name, description, rules_json, is_active, created_at, updated_at, version
		FROM segments
		WHERE env_id = $1 AND key = $2`

	err := r.db.QueryRow(ctx, query, envID, key).Scan(
		&segment.ID, &segment.EnvID, &segment.Key, &segment.Name, &segment.Description,
		&segment.Rules, &segment.IsActive, &segment.CreatedAt, &segment.UpdatedAt, &segment.Version,
	)

	if err != nil {
		r.logger.Debug().Err(err).
			Str("env_id", envID.String()).
			Str("segment_key", key).
			Msg("Segment not found")
		return nil, err
	}

	return segment, nil
}

// List retrieves segments for an environment with pagination
func (r *SegmentRepository) List(ctx context.Context, envID uuid.UUID, limit, offset int) ([]*Segment, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM segments WHERE env_id = $1`
	err := r.db.QueryRow(ctx, countQuery, envID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get segments
	query := `
		SELECT id, env_id, key, name, description, rules_json, is_active, created_at, updated_at, version
		FROM segments
		WHERE env_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, envID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var segments []*Segment
	for rows.Next() {
		segment := &Segment{}
		err := rows.Scan(
			&segment.ID, &segment.EnvID, &segment.Key, &segment.Name, &segment.Description,
			&segment.Rules, &segment.IsActive, &segment.CreatedAt, &segment.UpdatedAt, &segment.Version,
		)
		if err != nil {
			return nil, 0, err
		}
		segments = append(segments, segment)
	}

	return segments, total, rows.Err()
}

// Update updates an existing segment
func (r *SegmentRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateSegmentRequest) (*Segment, error) {
	// Start building the query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, "name = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Description != nil {
		setParts = append(setParts, "description = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.Rules != nil {
		rulesJSON, err := json.Marshal(req.Rules)
		if err != nil {
			return nil, err
		}
		setParts = append(setParts, "rules_json = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, rulesJSON)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, "is_active = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id) // No updates, return current state
	}

	// Always update version and updated_at
	setParts = append(setParts, "version = version + 1")
	setParts = append(setParts, "updated_at = CURRENT_TIMESTAMP")

	// Add ID to args
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE segments 
		SET %s
		WHERE id = $%d
		RETURNING id, env_id, key, name, description, rules_json, is_active, created_at, updated_at, version`,
		strings.Join(setParts, ", "), argIndex)

	segment := &Segment{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&segment.ID, &segment.EnvID, &segment.Key, &segment.Name, &segment.Description,
		&segment.Rules, &segment.IsActive, &segment.CreatedAt, &segment.UpdatedAt, &segment.Version,
	)

	if err != nil {
		r.logger.Error().Err(err).Str("segment_id", id.String()).Msg("Failed to update segment")
		return nil, err
	}

	r.logger.Info().
		Str("segment_id", segment.ID.String()).
		Str("segment_key", segment.Key).
		Msg("Segment updated")

	return segment, nil
}

// Delete deletes a segment
func (r *SegmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM segments WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("segment_id", id.String()).Msg("Failed to delete segment")
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("segment not found")
	}

	r.logger.Info().Str("segment_id", id.String()).Msg("Segment deleted")
	return nil
}

// CheckKeyExists checks if a segment key already exists in the environment
func (r *SegmentRepository) CheckKeyExists(ctx context.Context, envID uuid.UUID, key string, excludeID *uuid.UUID) (bool, error) {
	query := `SELECT COUNT(*) FROM segments WHERE env_id = $1 AND key = $2`
	args := []interface{}{envID, key}

	if excludeID != nil {
		query += ` AND id != $3`
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
