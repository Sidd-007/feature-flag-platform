package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/feature-flag-platform/pkg/auth"
	"github.com/feature-flag-platform/pkg/config"
)

type contextKey string

const (
	UserContextKey        contextKey = "user"
	EnvironmentContextKey contextKey = "environment"
)

type AuthMiddleware struct {
	tokenManager *auth.TokenManager
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: auth.NewTokenManager(cfg.Auth.JWTSecret),
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Warn().
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("Missing authorization header")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			log.Warn().
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("Invalid authorization header format")
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := tokenParts[1]

		// Check if it's an API key or JWT token
		var userID string
		var environmentID string
		var err error

		if strings.HasPrefix(token, "ak_") {
			// API Key authentication
			userID, environmentID, err = m.validateAPIKey(token)
			if err != nil {
				log.Warn().
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Err(err).
					Msg("Invalid API key")
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
		} else {
			// JWT token authentication
			claims, err := m.tokenManager.ValidateToken(token)
			if err != nil {
				log.Warn().
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Err(err).
					Msg("Invalid JWT token")
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			userID = claims.UserID
			// For JWT tokens, environment access is checked per request
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserContextKey, userID)
		if environmentID != "" {
			ctx = context.WithValue(ctx, EnvironmentContextKey, environmentID)
		}

		log.Debug().
			Str("user_id", userID).
			Str("environment_id", environmentID).
			Str("path", r.URL.Path).
			Msg("Request authenticated")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) validateAPIKey(token string) (userID, environmentID string, err error) {
	// TODO: Implement API key validation
	// This would typically:
	// 1. Hash the token
	// 2. Query the database for matching API key
	// 3. Check if the key is active and not expired
	// 4. Return the associated user and environment IDs

	// For now, return placeholder values
	// In production, this would query the control plane database
	return "analytics-service", "env-1", nil
}

// Helper functions to extract context values
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserContextKey).(string); ok {
		return userID
	}
	return ""
}

func GetEnvironmentID(ctx context.Context) string {
	if envID, ok := ctx.Value(EnvironmentContextKey).(string); ok {
		return envID
	}
	return ""
}
