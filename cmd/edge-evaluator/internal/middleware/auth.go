package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/pkg/auth"
)

// AuthMiddleware handles authentication for edge evaluator
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
	db           *pgxpool.Pool
	logger       zerolog.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(tokenManager *auth.TokenManager, db *pgxpool.Pool, logger zerolog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
		db:           db,
		logger:       logger.With().Str("middleware", "auth").Logger(),
	}
}

// AuthContextKey is the key for auth context
type AuthContextKey string

const (
	AuthContextKeyClaims = AuthContextKey("claims")
)

// AuthenticateAPIKey middleware validates API key tokens
func (m *AuthMiddleware) AuthenticateAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from Authorization header
		apiKey := extractAPIKeyFromHeader(r)
		if apiKey == "" {
			m.sendUnauthorized(w, "API key required")
			return
		}

		// Look up all active tokens and verify against them
		var tokenID, envID, scope, hashedToken string
		var isActive bool
		var expiresAt *time.Time

		// First, try to match by prefix for efficiency
		var prefix string
		if len(apiKey) > 11 { // "ff_" + 8 characters
			prefix = apiKey[3:11] // Skip "ff_" prefix, get first 8 chars
		} else {
			m.sendUnauthorized(w, "Invalid API key format")
			return
		}

		query := `
			SELECT id, env_id, scope, is_active, expires_at, hashed_token
			FROM api_tokens 
			WHERE prefix = $1 AND is_active = true`

		rows, err := m.db.Query(r.Context(), query, prefix)
		if err != nil {
			m.logger.Debug().Err(err).Msg("Failed to query API tokens")
			m.sendUnauthorized(w, "Invalid API key")
			return
		}
		defer rows.Close()

		// Check each token with matching prefix
		apiKeyMgr := auth.NewAPIKeyManager()
		var validToken bool

		for rows.Next() {
			err = rows.Scan(&tokenID, &envID, &scope, &isActive, &expiresAt, &hashedToken)
			if err != nil {
				continue
			}

			// Verify the API key against this hash
			if err := apiKeyMgr.VerifyAPIKey(apiKey, hashedToken); err == nil {
				validToken = true
				break
			}
		}

		if !validToken {
			m.logger.Debug().Str("prefix", prefix).Msg("API key verification failed")
			m.sendUnauthorized(w, "Invalid API key")
			return
		}

		// Check if token is expired
		if expiresAt != nil && time.Now().After(*expiresAt) {
			m.logger.Debug().Str("token_id", tokenID).Msg("API token is expired")
			m.sendUnauthorized(w, "API token has expired")
			return
		}

		// Create auth context for API token
		authCtx := &auth.Context{
			TokenType: auth.TokenTypeAPIKey,
			EnvID:     envID,
			Scope:     scope,
		}

		// Add to request context
		ctx := context.WithValue(r.Context(), AuthContextKeyClaims, authCtx)

		// Update last used timestamp (async)
		go func() {
			if tokenUUID, err := uuid.Parse(tokenID); err == nil {
				updateQuery := `UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
				if _, err := m.db.Exec(context.Background(), updateQuery, tokenUUID); err != nil {
					m.logger.Error().Err(err).Str("token_id", tokenID).Msg("Failed to update API token last used")
				}
			}
		}()

		m.logger.Debug().Str("token_id", tokenID).Str("env_id", envID).Str("scope", scope).Msg("API key authenticated")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims extracts JWT claims from request
func GetClaims(r *http.Request) *auth.Claims {
	claims, ok := r.Context().Value(AuthContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// Helper functions

func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Support both "Bearer <token>" and just "<token>" formats
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

func extractAPIKeyFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Support both "Bearer <token>" and just "<token>" formats
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

func (m *AuthMiddleware) sendUnauthorized(w http.ResponseWriter, message string) {
	m.sendError(w, http.StatusUnauthorized, "unauthorized", message)
}

func (m *AuthMiddleware) sendError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := map[string]interface{}{
		"error":   code,
		"message": message,
	}

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		m.logger.Error().Err(err).Msg("Failed to encode error response")
	}
}
