package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/pkg/auth"
	"github.com/feature-flag-platform/pkg/rbac"
)

// APITokenService handles business logic for API tokens
type APITokenService struct {
	repos        *repository.Repositories
	rbac         *rbac.RBAC
	tokenManager *auth.TokenManager
	apiKeyMgr    *auth.APIKeyManager
	logger       zerolog.Logger
}

// NewAPITokenService creates a new API token service
func NewAPITokenService(repos *repository.Repositories, tokenManager *auth.TokenManager, rbac *rbac.RBAC, logger zerolog.Logger) *APITokenService {
	return &APITokenService{
		repos:        repos,
		rbac:         rbac,
		tokenManager: tokenManager,
		apiKeyMgr:    auth.NewAPIKeyManager(),
		logger:       logger.With().Str("service", "api_token").Logger(),
	}
}

// CreateTokenRequest represents a request to create an API token
type CreateTokenRequest struct {
	EnvID       uuid.UUID  `json:"env_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Scope       string     `json:"scope"` // read, write
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateTokenResponse represents the response when creating an API token
type CreateTokenResponse struct {
	Token    *repository.APIToken `json:"token"`
	PlainKey string               `json:"plain_key"` // Only returned once
}

// Create creates a new API token
func (s *APITokenService) Create(ctx context.Context, req *CreateTokenRequest) (*CreateTokenResponse, error) {
	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("token name is required")
	}

	if req.Scope != "read" && req.Scope != "write" {
		return nil, fmt.Errorf("scope must be 'read' or 'write'")
	}

	// Verify environment exists
	_, err := s.repos.Environment.GetByID(ctx, req.EnvID)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %w", err)
	}

	// Generate API key
	plainKey, err := s.apiKeyMgr.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	hashedKey, err := s.apiKeyMgr.HashAPIKey(plainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	// Extract prefix (first 8 characters after ff_)
	prefix := plainKey[3:11] // Skip "ff_" prefix

	// Create repository request
	repoReq := &repository.CreateAPITokenRequest{
		EnvID:       req.EnvID,
		Name:        req.Name,
		Description: req.Description,
		Scope:       req.Scope,
		ExpiresAt:   req.ExpiresAt,
	}

	// Create token in database
	token, err := s.repos.APIToken.Create(ctx, repoReq, hashedKey, prefix)
	if err != nil {
		s.logger.Error().Err(err).Str("env_id", req.EnvID.String()).Msg("Failed to create API token")
		return nil, fmt.Errorf("failed to create API token: %w", err)
	}

	s.logger.Info().
		Str("token_id", token.ID.String()).
		Str("env_id", req.EnvID.String()).
		Str("scope", req.Scope).
		Msg("API token created successfully")

	return &CreateTokenResponse{
		Token:    token,
		PlainKey: plainKey,
	}, nil
}

// List retrieves API tokens for an environment
func (s *APITokenService) List(ctx context.Context, envID uuid.UUID, limit, offset int) ([]*repository.APIToken, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	tokens, err := s.repos.APIToken.List(ctx, envID, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Str("env_id", envID.String()).Msg("Failed to list API tokens")
		return nil, fmt.Errorf("failed to list API tokens: %w", err)
	}

	return tokens, nil
}

// Revoke deactivates an API token
func (s *APITokenService) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	err := s.repos.APIToken.Revoke(ctx, tokenID)
	if err != nil {
		s.logger.Error().Err(err).Str("token_id", tokenID.String()).Msg("Failed to revoke API token")
		return fmt.Errorf("failed to revoke API token: %w", err)
	}

	s.logger.Info().Str("token_id", tokenID.String()).Msg("API token revoked successfully")
	return nil
}

// ValidateToken validates an API token and returns the token details
func (s *APITokenService) ValidateToken(ctx context.Context, plainKey string) (*repository.APIToken, error) {
	// Hash the provided key
	hashedKey, err := s.apiKeyMgr.HashAPIKey(plainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	// Get token from database
	token, err := s.repos.APIToken.GetByToken(ctx, hashedKey)
	if err != nil {
		s.logger.Debug().Err(err).Msg("Token validation failed")
		return nil, fmt.Errorf("invalid token")
	}

	// Check if token is expired
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		s.logger.Debug().Str("token_id", token.ID.String()).Msg("Token is expired")
		return nil, fmt.Errorf("token has expired")
	}

	// Update last used timestamp (fire and forget)
	go func() {
		if err := s.repos.APIToken.UpdateLastUsed(context.Background(), token.ID); err != nil {
			s.logger.Error().Err(err).Str("token_id", token.ID.String()).Msg("Failed to update last used timestamp")
		}
	}()

	return token, nil
}
