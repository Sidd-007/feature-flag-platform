package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Project represents a project entity
type Project struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"org_id" db:"org_id"`
	Name        string    `json:"name" db:"name"`
	Key         string    `json:"key" db:"key"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Version     int       `json:"version" db:"version"`
}

// CreateProjectRequest represents request to create project
type CreateProjectRequest struct {
	OrgID       uuid.UUID `json:"org_id" validate:"required"`
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Key         string    `json:"key" validate:"required,min=1,max=100,alphanum"`
	Description string    `json:"description" validate:"max=1000"`
}

// UpdateProjectRequest represents request to update project
type UpdateProjectRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

// ProjectRepository handles project data access
type ProjectRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *pgxpool.Pool, logger zerolog.Logger) *ProjectRepository {
	return &ProjectRepository{
		db:     db,
		logger: logger.With().Str("repository", "project").Logger(),
	}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	project := &Project{
		ID:          uuid.New(),
		OrgID:       req.OrgID,
		Name:        req.Name,
		Key:         req.Key,
		Description: req.Description,
	}

	query := `
		INSERT INTO projects (id, org_id, name, key, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at, version`

	err := r.db.QueryRow(ctx, query, project.ID, project.OrgID, project.Name, project.Key, project.Description).
		Scan(&project.CreatedAt, &project.UpdatedAt, &project.Version)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to create project")
		return nil, err
	}

	r.logger.Info().Str("project_id", project.ID.String()).Str("org_id", project.OrgID.String()).Msg("Project created")
	return project, nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	project := &Project{}
	query := `
		SELECT id, org_id, name, key, description, created_at, updated_at, version
		FROM projects
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&project.ID, &project.OrgID, &project.Name, &project.Key, &project.Description,
		&project.CreatedAt, &project.UpdatedAt, &project.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("project_id", id.String()).Msg("Failed to get project")
		return nil, err
	}

	return project, nil
}

// GetByKey retrieves a project by key within an organization
func (r *ProjectRepository) GetByKey(ctx context.Context, orgID uuid.UUID, key string) (*Project, error) {
	project := &Project{}
	query := `
		SELECT id, org_id, name, key, description, created_at, updated_at, version
		FROM projects
		WHERE org_id = $1 AND key = $2`

	err := r.db.QueryRow(ctx, query, orgID, key).Scan(
		&project.ID, &project.OrgID, &project.Name, &project.Key, &project.Description,
		&project.CreatedAt, &project.UpdatedAt, &project.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("org_id", orgID.String()).Str("key", key).Msg("Failed to get project by key")
		return nil, err
	}

	return project, nil
}

// List retrieves projects for an organization with pagination
func (r *ProjectRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*Project, int, error) {
	query := `
		SELECT id, org_id, name, key, description, created_at, updated_at, version
		FROM projects
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list projects")
		return nil, 0, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		project := &Project{}
		err := rows.Scan(
			&project.ID, &project.OrgID, &project.Name, &project.Key, &project.Description,
			&project.CreatedAt, &project.UpdatedAt, &project.Version,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan project")
			return nil, 0, err
		}
		projects = append(projects, project)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM projects WHERE org_id = $1`
	var total int
	err = r.db.QueryRow(ctx, countQuery, orgID).Scan(&total)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count projects")
		return nil, 0, err
	}

	return projects, total, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateProjectRequest) (*Project, error) {
	query := `
		UPDATE projects
		SET name = $2, description = $3, updated_at = NOW(), version = version + 1
		WHERE id = $1
		RETURNING id, org_id, name, key, description, created_at, updated_at, version`

	project := &Project{}
	err := r.db.QueryRow(ctx, query, id, req.Name, req.Description).Scan(
		&project.ID, &project.OrgID, &project.Name, &project.Key, &project.Description,
		&project.CreatedAt, &project.UpdatedAt, &project.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("project_id", id.String()).Msg("Failed to update project")
		return nil, err
	}

	r.logger.Info().Str("project_id", project.ID.String()).Msg("Project updated")
	return project, nil
}

// Delete deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("project_id", id.String()).Msg("Failed to delete project")
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	r.logger.Info().Str("project_id", id.String()).Msg("Project deleted")
	return nil
}

// CheckKeyExists checks if a project key already exists within an organization
func (r *ProjectRepository) CheckKeyExists(ctx context.Context, orgID uuid.UUID, key string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id = $1 AND key = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, orgID, key).Scan(&exists)
	if err != nil {
		r.logger.Error().Err(err).Str("org_id", orgID.String()).Str("key", key).Msg("Failed to check key existence")
		return false, err
	}

	return exists, nil
}
