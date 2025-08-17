package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/pkg/auth"
	"github.com/feature-flag-platform/pkg/rbac"
)

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
	rbac         *rbac.RBAC
	db           *pgxpool.Pool
	logger       zerolog.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(tokenManager *auth.TokenManager, rbacManager *rbac.RBAC, db *pgxpool.Pool, logger zerolog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
		rbac:         rbacManager,
		db:           db,
		logger:       logger,
	}
}

// AuthContextKey is the key for auth context
type AuthContextKey string

const (
	AuthContextKeyUser   = AuthContextKey("user")
	AuthContextKeyClaims = AuthContextKey("claims")
)

// Authenticate middleware validates JWT tokens
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractTokenFromHeader(r)
		if token == "" {
			m.sendUnauthorized(w, "Missing or invalid authorization header")
			return
		}

		claims, err := m.tokenManager.ValidateToken(token)
		if err != nil {
			m.logger.Debug().Err(err).Msg("Token validation failed")
			m.sendUnauthorized(w, "Invalid token")
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), AuthContextKeyClaims, claims)

		// Create auth context
		authCtx := auth.NewContext(claims)
		ctx = context.WithValue(ctx, AuthContextKeyUser, authCtx)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateAPIKey middleware for API key authentication (used by edge evaluators)
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
		ctx := context.WithValue(r.Context(), AuthContextKeyUser, authCtx)

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

// RequireOrgAccess checks if user has access to the organization
func (m *AuthMiddleware) RequireOrgAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			m.sendUnauthorized(w, "Authentication required")
			return
		}

		orgID := chi.URLParam(r, "orgId")
		if orgID == "" {
			m.sendBadRequest(w, "Organization ID required")
			return
		}

		// For user tokens, check if user has membership in the organization
		if authCtx.TokenType == auth.TokenTypeUser {
			m.logger.Info().Str("user_id", authCtx.UserID).Str("org_id", orgID).Msg("Checking organization access for user")

			// Parse user ID from context
			userID, err := uuid.Parse(authCtx.UserID)
			if err != nil {
				m.logger.Error().Err(err).Str("user_id", authCtx.UserID).Msg("Invalid user ID in token")
				m.sendForbidden(w, "Invalid user token")
				return
			}

			// Parse org ID from URL
			orgUUID, err := uuid.Parse(orgID)
			if err != nil {
				m.logger.Error().Err(err).Str("org_id", orgID).Msg("Invalid organization ID")
				m.sendBadRequest(w, "Invalid organization ID")
				return
			}

			// Check if user has membership in this organization
			// We'll use a simple query to check membership
			query := `SELECT EXISTS(SELECT 1 FROM user_org_memberships WHERE user_id = $1 AND org_id = $2)`
			var exists bool
			err = m.db.QueryRow(r.Context(), query, userID, orgUUID).Scan(&exists)
			if err != nil {
				m.logger.Error().Err(err).Str("user_id", userID.String()).Str("org_id", orgID).Msg("Failed to check organization membership")
				m.sendForbidden(w, "Failed to verify organization access")
				return
			}

			m.logger.Info().Str("user_id", userID.String()).Str("org_id", orgID).Bool("has_membership", exists).Msg("Organization membership check")

			if !exists {
				m.logger.Debug().Str("user_id", userID.String()).Str("org_id", orgID).Msg("User does not have access to organization")
				m.sendForbidden(w, "Access denied to organization")
				return
			}
		} else {
			// For API tokens and service tokens, use the existing logic
			if !authCtx.CanAccessOrg(orgID) {
				m.sendForbidden(w, "Access denied to organization")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireProjectAccess checks if user has access to the project
func (m *AuthMiddleware) RequireProjectAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			m.sendUnauthorized(w, "Authentication required")
			return
		}

		orgID := chi.URLParam(r, "orgId")
		projectID := chi.URLParam(r, "projectId")

		if orgID == "" || projectID == "" {
			m.sendBadRequest(w, "Organization and Project ID required")
			return
		}

		// For user tokens, verify org membership and that project belongs to org
		if authCtx.TokenType == auth.TokenTypeUser {
			userUUID, err := uuid.Parse(authCtx.UserID)
			if err != nil {
				m.logger.Error().Err(err).Str("user_id", authCtx.UserID).Msg("Invalid user ID in token")
				m.sendForbidden(w, "Invalid user token")
				return
			}

			orgUUID, err := uuid.Parse(orgID)
			if err != nil {
				m.logger.Error().Err(err).Str("org_id", orgID).Msg("Invalid organization ID")
				m.sendBadRequest(w, "Invalid organization ID")
				return
			}

			projectUUID, err := uuid.Parse(projectID)
			if err != nil {
				m.logger.Error().Err(err).Str("project_id", projectID).Msg("Invalid project ID")
				m.sendBadRequest(w, "Invalid project ID")
				return
			}

			// Check membership in organization
			var hasMembership bool
			if err := m.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM user_org_memberships WHERE user_id = $1 AND org_id = $2)`, userUUID, orgUUID).Scan(&hasMembership); err != nil {
				m.logger.Error().Err(err).Str("user_id", userUUID.String()).Str("org_id", orgID).Msg("Failed to verify org membership")
				m.sendForbidden(w, "Failed to verify organization access")
				return
			}

			// Ensure project belongs to organization
			var projectInOrg bool
			if err := m.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND org_id = $2)`, projectUUID, orgUUID).Scan(&projectInOrg); err != nil {
				m.logger.Error().Err(err).Str("project_id", projectID).Str("org_id", orgID).Msg("Failed to verify project access")
				m.sendForbidden(w, "Failed to verify project access")
				return
			}

			m.logger.Info().Str("user_id", userUUID.String()).Str("org_id", orgID).Str("project_id", projectID).Bool("has_membership", hasMembership).Bool("project_in_org", projectInOrg).Msg("Project access check")

			if !hasMembership || !projectInOrg {
				m.sendForbidden(w, "Access denied to project")
				return
			}
		} else {
			// For API/service tokens, fall back to existing checks
			if !authCtx.CanAccessProject(orgID, projectID) {
				m.sendForbidden(w, "Access denied to project")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireEnvAccess checks if user has access to the environment
func (m *AuthMiddleware) RequireEnvAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r)
		if authCtx == nil {
			m.sendUnauthorized(w, "Authentication required")
			return
		}

		orgID := chi.URLParam(r, "orgId")
		projectID := chi.URLParam(r, "projectId")
		envID := chi.URLParam(r, "envId")

		if orgID == "" || projectID == "" || envID == "" {
			m.sendBadRequest(w, "Organization, Project, and Environment ID required")
			return
		}

		// For user tokens, verify membership and resource relationships via DB
		if authCtx.TokenType == auth.TokenTypeUser {
			userUUID, err := uuid.Parse(authCtx.UserID)
			if err != nil {
				m.logger.Error().Err(err).Str("user_id", authCtx.UserID).Msg("Invalid user ID in token")
				m.sendForbidden(w, "Invalid user token")
				return
			}

			orgUUID, err := uuid.Parse(orgID)
			if err != nil {
				m.logger.Error().Err(err).Str("org_id", orgID).Msg("Invalid organization ID")
				m.sendBadRequest(w, "Invalid organization ID")
				return
			}

			projectUUID, err := uuid.Parse(projectID)
			if err != nil {
				m.logger.Error().Err(err).Str("project_id", projectID).Msg("Invalid project ID")
				m.sendBadRequest(w, "Invalid project ID")
				return
			}

			envUUID, err := uuid.Parse(envID)
			if err != nil {
				m.logger.Error().Err(err).Str("env_id", envID).Msg("Invalid environment ID")
				m.sendBadRequest(w, "Invalid environment ID")
				return
			}

			// Check membership in org
			var hasMembership bool
			if err := m.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM user_org_memberships WHERE user_id = $1 AND org_id = $2)`, userUUID, orgUUID).Scan(&hasMembership); err != nil {
				m.logger.Error().Err(err).Str("user_id", userUUID.String()).Str("org_id", orgID).Msg("Failed to verify org membership")
				m.sendForbidden(w, "Failed to verify organization access")
				return
			}

			// Ensure project belongs to org
			var projectInOrg bool
			if err := m.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND org_id = $2)`, projectUUID, orgUUID).Scan(&projectInOrg); err != nil {
				m.logger.Error().Err(err).Str("project_id", projectID).Str("org_id", orgID).Msg("Failed to verify project access")
				m.sendForbidden(w, "Failed to verify project access")
				return
			}

			// Ensure environment belongs to project
			var envInProject bool
			if err := m.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM environments WHERE id = $1 AND project_id = $2)`, envUUID, projectUUID).Scan(&envInProject); err != nil {
				m.logger.Error().Err(err).Str("env_id", envID).Str("project_id", projectID).Msg("Failed to verify environment access")
				m.sendForbidden(w, "Failed to verify environment access")
				return
			}

			m.logger.Info().Str("user_id", userUUID.String()).Str("org_id", orgID).Str("project_id", projectID).Str("env_id", envID).Bool("has_membership", hasMembership).Bool("project_in_org", projectInOrg).Bool("env_in_project", envInProject).Msg("Environment access check")

			if !hasMembership || !projectInOrg || !envInProject {
				m.sendForbidden(w, "Access denied to environment")
				return
			}
		} else {
			// API/service tokens fallback
			if !authCtx.CanAccessEnv(orgID, projectID, envID) {
				m.sendForbidden(w, "Access denied to environment")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequirePermission checks if user has a specific permission
func (m *AuthMiddleware) RequirePermission(permission auth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r)
			if authCtx == nil {
				m.sendUnauthorized(w, "Authentication required")
				return
			}

			authManager := auth.NewAuthorizationManager()
			if !authCtx.HasPermission(permission, authManager) {
				m.sendForbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthContext extracts auth context from request
func GetAuthContext(r *http.Request) *auth.Context {
	authCtx, ok := r.Context().Value(AuthContextKeyUser).(*auth.Context)
	if !ok {
		return nil
	}
	return authCtx
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

	// API keys can be sent as "Bearer <api_key>" or just "<api_key>"
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

func (m *AuthMiddleware) sendUnauthorized(w http.ResponseWriter, message string) {
	m.sendError(w, http.StatusUnauthorized, "unauthorized", message)
}

func (m *AuthMiddleware) sendForbidden(w http.ResponseWriter, message string) {
	m.sendError(w, http.StatusForbidden, "forbidden", message)
}

func (m *AuthMiddleware) sendBadRequest(w http.ResponseWriter, message string) {
	m.sendError(w, http.StatusBadRequest, "bad_request", message)
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
