package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Organization represents an organization entity
type Organization struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	BillingTier string    `json:"billing_tier" db:"billing_tier"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Version     int       `json:"version" db:"version"`
}

// CreateOrganizationRequest represents request to create organization
type CreateOrganizationRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
	Slug string `json:"slug" validate:"required,min=1,max=100,alphanum"`
}

// UpdateOrganizationRequest represents request to update organization
type UpdateOrganizationRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// OrganizationRepository handles organization data access
type OrganizationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *pgxpool.Pool, logger zerolog.Logger) *OrganizationRepository {
	return &OrganizationRepository{
		db:     db,
		logger: logger.With().Str("repository", "organization").Logger(),
	}
}

// Create creates a new organization
func (r *OrganizationRepository) Create(ctx context.Context, req *CreateOrganizationRequest) (*Organization, error) {
	org := &Organization{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        req.Slug,
		BillingTier: "free", // Default billing tier
	}

	query := `
		INSERT INTO orgs (id, name, slug, billing_tier)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at, version`

	err := r.db.QueryRow(ctx, query, org.ID, org.Name, org.Slug, org.BillingTier).
		Scan(&org.CreatedAt, &org.UpdatedAt, &org.Version)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to create organization")
		return nil, err
	}

	r.logger.Info().Str("org_id", org.ID.String()).Str("slug", org.Slug).Msg("Organization created")
	return org, nil
}

// AddUserMembership inserts a membership row linking a user to an organization with a role.
// If the membership already exists, the operation is a no-op.
func (r *OrganizationRepository) AddUserMembership(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, role string) error {
	query := `
		INSERT INTO user_org_memberships (id, user_id, org_id, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, org_id) DO NOTHING`

	_, err := r.db.Exec(ctx, query, uuid.New(), userID, orgID, role)
	if err != nil {
		r.logger.Error().Err(err).
			Str("user_id", userID.String()).
			Str("org_id", orgID.String()).
			Msg("Failed to add user membership")
		return err
	}

	r.logger.Info().
		Str("user_id", userID.String()).
		Str("org_id", orgID.String()).
		Str("role", role).
		Msg("User membership added")
	return nil
}

// GetUserMemberships retrieves all organization memberships for a user
func (r *OrganizationRepository) GetUserMemberships(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	query := `
		SELECT uom.org_id, uom.role, o.name, o.slug
		FROM user_org_memberships uom
		JOIN orgs o ON uom.org_id = o.id
		WHERE uom.user_id = $1`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get user memberships")
		return nil, err
	}
	defer rows.Close()

	var memberships []map[string]interface{}
	for rows.Next() {
		var orgID, role, name, slug string
		err := rows.Scan(&orgID, &role, &name, &slug)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan membership")
			return nil, err
		}
		memberships = append(memberships, map[string]interface{}{
			"org_id": orgID,
			"role":   role,
			"name":   name,
			"slug":   slug,
		})
	}

	r.logger.Info().Str("user_id", userID.String()).Int("count", len(memberships)).Msg("Retrieved user memberships")
	return memberships, nil
}

// GetByID retrieves an organization by ID
func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	org := &Organization{}
	query := `
		SELECT id, name, slug, billing_tier, created_at, updated_at, version
		FROM orgs
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.BillingTier,
		&org.CreatedAt, &org.UpdatedAt, &org.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("org_id", id.String()).Msg("Failed to get organization")
		return nil, err
	}

	return org, nil
}

// GetBySlug retrieves an organization by slug
func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	org := &Organization{}
	query := `
		SELECT id, name, slug, billing_tier, created_at, updated_at, version
		FROM orgs
		WHERE slug = $1`

	err := r.db.QueryRow(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.BillingTier,
		&org.CreatedAt, &org.UpdatedAt, &org.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("slug", slug).Msg("Failed to get organization by slug")
		return nil, err
	}

	return org, nil
}

// List retrieves organizations with pagination
func (r *OrganizationRepository) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Organization, int, error) {
	// Get organizations where user has membership
	query := `
		SELECT o.id, o.name, o.slug, o.billing_tier, o.created_at, o.updated_at, o.version
		FROM orgs o
		INNER JOIN user_org_memberships uom ON o.id = uom.org_id
		WHERE uom.user_id = $1
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to list organizations")
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.BillingTier,
			&org.CreatedAt, &org.UpdatedAt, &org.Version,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan organization")
			return nil, 0, err
		}
		orgs = append(orgs, org)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM orgs o
		INNER JOIN user_org_memberships uom ON o.id = uom.org_id
		WHERE uom.user_id = $1`

	var total int
	err = r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to count organizations")
		return nil, 0, err
	}

	return orgs, total, nil
}

// Update updates an organization
func (r *OrganizationRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateOrganizationRequest) (*Organization, error) {
	query := `
		UPDATE orgs
		SET name = $2
		WHERE id = $1
		RETURNING id, name, slug, billing_tier, created_at, updated_at, version`

	org := &Organization{}
	err := r.db.QueryRow(ctx, query, id, req.Name).Scan(
		&org.ID, &org.Name, &org.Slug, &org.BillingTier,
		&org.CreatedAt, &org.UpdatedAt, &org.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("org_id", id.String()).Msg("Failed to update organization")
		return nil, err
	}

	r.logger.Info().Str("org_id", org.ID.String()).Msg("Organization updated")
	return org, nil
}

// Delete deletes an organization
func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orgs WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("org_id", id.String()).Msg("Failed to delete organization")
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	r.logger.Info().Str("org_id", id.String()).Msg("Organization deleted")
	return nil
}

// CheckSlugExists checks if a slug already exists
func (r *OrganizationRepository) CheckSlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM orgs WHERE slug = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		r.logger.Error().Err(err).Str("slug", slug).Msg("Failed to check slug existence")
		return false, err
	}

	return exists, nil
}
