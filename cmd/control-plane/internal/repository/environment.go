package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Environment represents an environment record
type Environment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"`
	Key       string    `json:"key" db:"key"`
	Salt      string    `json:"salt" db:"salt"`
	IsProd    bool      `json:"is_prod" db:"is_prod"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Version   int       `json:"version" db:"version"`
}

// CreateEnvironmentRequest input for creating an environment
type CreateEnvironmentRequest struct {
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Description string    `json:"description"` // not persisted currently
	IsProd      bool      `json:"is_prod"`
}

// UpdateEnvironmentRequest input for updating an environment
type UpdateEnvironmentRequest struct {
	Name   string `json:"name"`
	IsProd bool   `json:"is_prod"`
}

// EnvironmentRepository handles environment data access
type EnvironmentRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewEnvironmentRepository creates a new environment repository
func NewEnvironmentRepository(db *pgxpool.Pool, logger zerolog.Logger) *EnvironmentRepository {
	return &EnvironmentRepository{
		db:     db,
		logger: logger.With().Str("repository", "environment").Logger(),
	}
}

// Create inserts a new environment
func (r *EnvironmentRepository) Create(ctx context.Context, req *CreateEnvironmentRequest) (*Environment, error) {
	env := &Environment{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Key:       req.Key,
		IsProd:    req.IsProd,
	}

	query := `INSERT INTO environments (id, project_id, name, key, is_prod) VALUES ($1, $2, $3, $4, $5) RETURNING salt, created_at, updated_at, version`
	if err := r.db.QueryRow(ctx, query, env.ID, env.ProjectID, env.Name, env.Key, env.IsProd).Scan(&env.Salt, &env.CreatedAt, &env.UpdatedAt, &env.Version); err != nil {
		r.logger.Error().Err(err).Msg("Failed to create environment")
		return nil, err
	}
	return env, nil
}

// GetByID fetches environment by id
func (r *EnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*Environment, error) {
	env := &Environment{}
	query := `SELECT id, project_id, name, key, salt, is_prod, created_at, updated_at, version FROM environments WHERE id = $1`
	if err := r.db.QueryRow(ctx, query, id).Scan(&env.ID, &env.ProjectID, &env.Name, &env.Key, &env.Salt, &env.IsProd, &env.CreatedAt, &env.UpdatedAt, &env.Version); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to get environment")
		return nil, err
	}
	return env, nil
}

// List returns environments for a project (paginated)
func (r *EnvironmentRepository) List(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Environment, int, error) {
	rows, err := r.db.Query(ctx, `SELECT id, project_id, name, key, salt, is_prod, created_at, updated_at, version FROM environments WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, projectID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list environments")
		return nil, 0, err
	}
	defer rows.Close()

	var envs []*Environment
	for rows.Next() {
		e := &Environment{}
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Key, &e.Salt, &e.IsProd, &e.CreatedAt, &e.UpdatedAt, &e.Version); err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan environment")
			return nil, 0, err
		}
		envs = append(envs, e)
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM environments WHERE project_id = $1`, projectID).Scan(&total); err != nil {
		r.logger.Error().Err(err).Msg("Failed to count environments")
		return nil, 0, err
	}
	return envs, total, nil
}

// Update modifies an environment
func (r *EnvironmentRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateEnvironmentRequest) (*Environment, error) {
	env := &Environment{}
	query := `UPDATE environments SET name = $2, is_prod = $3, updated_at = NOW(), version = version + 1 WHERE id = $1 RETURNING id, project_id, name, key, salt, is_prod, created_at, updated_at, version`
	if err := r.db.QueryRow(ctx, query, id, req.Name, req.IsProd).Scan(&env.ID, &env.ProjectID, &env.Name, &env.Key, &env.Salt, &env.IsProd, &env.CreatedAt, &env.UpdatedAt, &env.Version); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Msg("Failed to update environment")
		return nil, err
	}
	return env, nil
}

// Delete removes an environment
func (r *EnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM environments WHERE id = $1`, id)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to delete environment")
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// IncrementVersion increments the version of an environment
func (r *EnvironmentRepository) IncrementVersion(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE environments SET version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("env_id", id.String()).Msg("Failed to increment environment version")
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	r.logger.Debug().Str("env_id", id.String()).Msg("Environment version incremented")
	return nil
}

// GetByKey retrieves an environment by its key
func (r *EnvironmentRepository) GetByKey(ctx context.Context, key string) (*Environment, error) {
	env := &Environment{}
	query := `
		SELECT id, project_id, name, key, salt, is_prod, created_at, updated_at, version
		FROM environments 
		WHERE key = $1`

	err := r.db.QueryRow(ctx, query, key).Scan(
		&env.ID, &env.ProjectID, &env.Name, &env.Key, &env.Salt,
		&env.IsProd, &env.CreatedAt, &env.UpdatedAt, &env.Version,
	)

	if err != nil {
		r.logger.Debug().Err(err).Str("key", key).Msg("Failed to get environment by key")
		return nil, err
	}

	return env, nil
}
