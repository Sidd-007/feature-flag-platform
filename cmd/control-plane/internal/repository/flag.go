package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Flag represents a feature flag record
type Flag struct {
	ID               uuid.UUID `json:"id" db:"id"`
	EnvID            uuid.UUID `json:"env_id" db:"env_id"`
	Key              string    `json:"key" db:"key"`
	Name             string    `json:"name" db:"name"`
	Description      string    `json:"description" db:"description"`
	Type             string    `json:"type" db:"type"`
	Status           string    `json:"status" db:"status"`
	Published        bool      `json:"published" db:"published"`
	DefaultVariation string    `json:"default_variation" db:"default_variation"`
	Variations       any       `json:"variations" db:"variations"`
	RulesJSON        any       `json:"rules_json" db:"rules_json"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	Version          int       `json:"version" db:"version"`
}

// CreateFlagRequest input for creating a flag
type CreateFlagRequest struct {
	EnvID        uuid.UUID `json:"env_id"`
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Type         string    `json:"type"`
	Enabled      bool      `json:"enabled"`
	DefaultValue any       `json:"default_value"`
}

// UpdateFlagRequest input for updating a flag
type UpdateFlagRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// FlagRepository handles flag data access
type FlagRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewFlagRepository creates a new flag repository
func NewFlagRepository(db *pgxpool.Pool, logger zerolog.Logger) *FlagRepository {
	return &FlagRepository{db: db, logger: logger.With().Str("repository", "flag").Logger()}
}

// Create inserts a new flag
func (r *FlagRepository) Create(ctx context.Context, req *CreateFlagRequest) (*Flag, error) {
	flag := &Flag{ID: uuid.New(), EnvID: req.EnvID, Key: req.Key, Name: req.Name, Description: req.Description, Type: req.Type}
	status := "active"

	var defaultVariation string
	var variationsJSON string

	// Handle different flag types
	switch req.Type {
	case "boolean":
		defaultVariation = fmt.Sprintf("%v", req.DefaultValue)
		variationsJSON = `[{"key": "true", "value": true}, {"key": "false", "value": false}]`
	case "string":
		// For string flags, use the actual value as both key and value
		defaultValue := fmt.Sprintf("%v", req.DefaultValue)
		defaultVariation = defaultValue
		variationsJSON = fmt.Sprintf(`[{"key": "%s", "value": "%s"}]`, defaultValue, defaultValue)
	case "number":
		// For number flags, use the actual value
		defaultValue := fmt.Sprintf("%v", req.DefaultValue)
		defaultVariation = defaultValue
		// Try to parse as number, keep as number in JSON
		if num, err := json.Marshal(req.DefaultValue); err == nil {
			variationsJSON = fmt.Sprintf(`[{"key": "%s", "value": %s}]`, defaultValue, string(num))
		} else {
			variationsJSON = fmt.Sprintf(`[{"key": "%s", "value": "%s"}]`, defaultValue, defaultValue)
		}
	case "multivariate":
		// For multivariate flags, create a variation with the default value
		defaultValue := fmt.Sprintf("%v", req.DefaultValue)
		defaultVariation = defaultValue
		variationsJSON = fmt.Sprintf(`[{"key": "%s", "value": "%s"}]`, defaultValue, defaultValue)
	case "json":
		defaultVariation = "default"
		// Try to parse as JSON, fallback to string
		if jsonBytes, err := json.Marshal(req.DefaultValue); err == nil {
			variationsJSON = fmt.Sprintf(`[{"key": "default", "value": %s}]`, string(jsonBytes))
		} else {
			variationsJSON = fmt.Sprintf(`[{"key": "default", "value": "%v"}]`, req.DefaultValue)
		}
	default:
		defaultVariation = fmt.Sprintf("%v", req.DefaultValue)
		variationsJSON = `[]`
	}

	query := `INSERT INTO flags (id, env_id, key, name, description, type, status, published, default_variation, variations, rules_json)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, $10::jsonb, '{}'::jsonb)
		RETURNING created_at, updated_at, version`
	if err := r.db.QueryRow(ctx, query, flag.ID, flag.EnvID, flag.Key, flag.Name, flag.Description, flag.Type, status, false, defaultVariation, variationsJSON).Scan(&flag.CreatedAt, &flag.UpdatedAt, &flag.Version); err != nil {
		r.logger.Error().Err(err).Msg("Failed to create flag")
		return nil, err
	}
	flag.Status = status
	flag.DefaultVariation = defaultVariation
	flag.Published = false // Flags start unpublished
	return flag, nil
}

// GetByID returns flag by ID
func (r *FlagRepository) GetByID(ctx context.Context, id uuid.UUID) (*Flag, error) {
	f := &Flag{}
	q := `SELECT id, env_id, key, name, description, type, status, published, default_variation, variations, rules_json, created_at, updated_at, version FROM flags WHERE id=$1`
	if err := r.db.QueryRow(ctx, q, id).Scan(&f.ID, &f.EnvID, &f.Key, &f.Name, &f.Description, &f.Type, &f.Status, &f.Published, &f.DefaultVariation, &f.Variations, &f.RulesJSON, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to get flag by ID")
		return nil, err
	}
	return f, nil
}

// GetByKey returns flag by env and key
func (r *FlagRepository) GetByKey(ctx context.Context, envID uuid.UUID, key string) (*Flag, error) {
	f := &Flag{}
	q := `SELECT id, env_id, key, name, description, type, status, published, default_variation, variations, rules_json, created_at, updated_at, version FROM flags WHERE env_id=$1 AND key=$2`
	if err := r.db.QueryRow(ctx, q, envID, key).Scan(&f.ID, &f.EnvID, &f.Key, &f.Name, &f.Description, &f.Type, &f.Status, &f.Published, &f.DefaultVariation, &f.Variations, &f.RulesJSON, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to get flag")
		return nil, err
	}
	return f, nil
}

// List returns flags for an environment
func (r *FlagRepository) List(ctx context.Context, envID uuid.UUID, limit, offset int) ([]*Flag, int, error) {
	rows, err := r.db.Query(ctx, `SELECT id, env_id, key, name, description, type, status, published, default_variation, variations, rules_json, created_at, updated_at, version FROM flags WHERE env_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, envID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list flags")
		return nil, 0, err
	}
	defer rows.Close()
	var flags []*Flag
	for rows.Next() {
		f := &Flag{}
		if err := rows.Scan(&f.ID, &f.EnvID, &f.Key, &f.Name, &f.Description, &f.Type, &f.Status, &f.Published, &f.DefaultVariation, &f.Variations, &f.RulesJSON, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan flag")
			return nil, 0, err
		}
		flags = append(flags, f)
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM flags WHERE env_id=$1`, envID).Scan(&total); err != nil {
		r.logger.Error().Err(err).Msg("Failed to count flags")
		return nil, 0, err
	}
	return flags, total, nil
}

// Update updates a flag
func (r *FlagRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateFlagRequest) (*Flag, error) {
	f := &Flag{}
	q := `UPDATE flags SET name=$2, description=$3, status=$4, updated_at=NOW(), version = version + 1 WHERE id=$1 RETURNING id, env_id, key, name, description, type, status, published, default_variation, variations, rules_json, created_at, updated_at, version`
	if err := r.db.QueryRow(ctx, q, id, req.Name, req.Description, req.Status).Scan(&f.ID, &f.EnvID, &f.Key, &f.Name, &f.Description, &f.Type, &f.Status, &f.Published, &f.DefaultVariation, &f.Variations, &f.RulesJSON, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to update flag")
		return nil, err
	}
	return f, nil
}

// Delete deletes a flag
func (r *FlagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM flags WHERE id=$1`, id)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to delete flag")
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPublished sets the published status of a flag
func (r *FlagRepository) SetPublished(ctx context.Context, id uuid.UUID, published bool) (*Flag, error) {
	f := &Flag{}
	q := `UPDATE flags SET published=$2, updated_at=NOW(), version = version + 1 WHERE id=$1 RETURNING id, env_id, key, name, description, type, status, published, default_variation, variations, rules_json, created_at, updated_at, version`
	if err := r.db.QueryRow(ctx, q, id, published).Scan(&f.ID, &f.EnvID, &f.Key, &f.Name, &f.Description, &f.Type, &f.Status, &f.Published, &f.DefaultVariation, &f.Variations, &f.RulesJSON, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to set published status")
		return nil, err
	}
	return f, nil
}
