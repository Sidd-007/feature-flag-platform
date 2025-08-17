package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// APIToken represents an API token for environment access
type APIToken struct {
	ID          uuid.UUID  `json:"id"`
	EnvID       uuid.UUID  `json:"env_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Scope       string     `json:"scope"`  // read, write
	HashedToken string     `json:"-"`      // Never expose
	Prefix      string     `json:"prefix"` // First 8 chars for identification
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// CreateAPITokenRequest represents request to create an API token
type CreateAPITokenRequest struct {
	EnvID       uuid.UUID  `json:"env_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Scope       string     `json:"scope"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APITokenRepository handles API token data access
type APITokenRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAPITokenRepository creates a new API token repository
func NewAPITokenRepository(db *pgxpool.Pool, logger zerolog.Logger) *APITokenRepository {
	return &APITokenRepository{
		db:     db,
		logger: logger.With().Str("repository", "api_token").Logger(),
	}
}

// Create creates a new API token
func (r *APITokenRepository) Create(ctx context.Context, req *CreateAPITokenRequest, hashedToken, prefix string) (*APIToken, error) {
	token := &APIToken{
		ID:          uuid.New(),
		EnvID:       req.EnvID,
		Name:        req.Name,
		Description: req.Description,
		Scope:       req.Scope,
		HashedToken: hashedToken,
		Prefix:      prefix,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
	}

	query := `
		INSERT INTO api_tokens (id, env_id, name, description, scope, hashed_token, prefix, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		token.ID, token.EnvID, token.Name, token.Description, token.Scope,
		token.HashedToken, token.Prefix, token.ExpiresAt, token.IsActive,
	).Scan(&token.CreatedAt, &token.UpdatedAt)

	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to create API token")
		return nil, err
	}

	r.logger.Info().Str("token_id", token.ID.String()).Str("env_id", token.EnvID.String()).Msg("API token created")
	return token, nil
}

// GetByToken retrieves an API token by its hashed value
func (r *APITokenRepository) GetByToken(ctx context.Context, hashedToken string) (*APIToken, error) {
	token := &APIToken{}
	query := `
		SELECT id, env_id, name, description, scope, hashed_token, prefix, expires_at, 
		       created_at, updated_at, last_used_at, is_active
		FROM api_tokens 
		WHERE hashed_token = $1 AND is_active = true`

	err := r.db.QueryRow(ctx, query, hashedToken).Scan(
		&token.ID, &token.EnvID, &token.Name, &token.Description, &token.Scope,
		&token.HashedToken, &token.Prefix, &token.ExpiresAt,
		&token.CreatedAt, &token.UpdatedAt, &token.LastUsedAt, &token.IsActive,
	)

	if err != nil {
		r.logger.Debug().Err(err).Msg("Failed to get API token by hash")
		return nil, err
	}

	return token, nil
}

// List retrieves API tokens for an environment
func (r *APITokenRepository) List(ctx context.Context, envID uuid.UUID, limit, offset int) ([]*APIToken, error) {
	query := `
		SELECT id, env_id, name, description, scope, prefix, expires_at,
		       created_at, updated_at, last_used_at, is_active
		FROM api_tokens 
		WHERE env_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, envID, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).Str("env_id", envID.String()).Msg("Failed to list API tokens")
		return nil, err
	}
	defer rows.Close()

	var tokens []*APIToken
	for rows.Next() {
		token := &APIToken{}
		err := rows.Scan(
			&token.ID, &token.EnvID, &token.Name, &token.Description, &token.Scope,
			&token.Prefix, &token.ExpiresAt,
			&token.CreatedAt, &token.UpdatedAt, &token.LastUsedAt, &token.IsActive,
		)
		if err != nil {
			r.logger.Error().Err(err).Msg("Failed to scan API token")
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// UpdateLastUsed updates the last used timestamp for a token
func (r *APITokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("token_id", id.String()).Msg("Failed to update last used")
	}
	return err
}

// Revoke deactivates an API token
func (r *APITokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_tokens SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("token_id", id.String()).Msg("Failed to revoke API token")
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	r.logger.Info().Str("token_id", id.String()).Msg("API token revoked")
	return nil
}
