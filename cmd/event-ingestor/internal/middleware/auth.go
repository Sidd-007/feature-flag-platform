package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/pkg/auth"
)

// AuthMiddleware handles authentication for event ingestor
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
	logger       zerolog.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(tokenManager *auth.TokenManager, logger zerolog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
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
		token := extractTokenFromHeader(r)
		if token == "" {
			m.sendUnauthorized(w, "API key required")
			return
		}

		claims, err := m.tokenManager.ValidateToken(token)
		if err != nil {
			m.logger.Debug().Err(err).Msg("API key validation failed")
			m.sendUnauthorized(w, "Invalid API key")
			return
		}

		// Ensure this is an API key token
		if claims.TokenType != auth.TokenTypeAPIKey {
			m.sendUnauthorized(w, "Invalid token type")
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), AuthContextKeyClaims, claims)

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
