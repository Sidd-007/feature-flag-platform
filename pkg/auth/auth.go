package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// TokenType represents the type of authentication token
type TokenType string

const (
	TokenTypeUser    TokenType = "user"
	TokenTypeAPIKey  TokenType = "api_key"
	TokenTypeService TokenType = "service"
)

// Claims represents JWT claims for the application
type Claims struct {
	UserID    string    `json:"user_id,omitempty"`
	Email     string    `json:"email,omitempty"`
	OrgID     string    `json:"org_id,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	EnvID     string    `json:"env_id,omitempty"`
	Scope     string    `json:"scope"`
	TokenType TokenType `json:"token_type"`
	TokenID   string    `json:"token_id,omitempty"`
	jwt.RegisteredClaims
}

// TokenManager handles JWT token operations
type TokenManager struct {
	secret []byte
}

// NewTokenManager creates a new token manager
func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
	}
}

// GenerateUserToken generates a JWT token for a user
func (tm *TokenManager) GenerateUserToken(userID, email, orgID string, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID:    userID,
		Email:     email,
		OrgID:     orgID,
		Scope:     "user",
		TokenType: TokenTypeUser,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "feature-flag-platform",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secret)
}

// GenerateAPIToken generates a JWT token for API access
func (tm *TokenManager) GenerateAPIToken(tokenID, orgID, projectID, envID, scope string, expiry time.Duration) (string, error) {
	claims := &Claims{
		TokenID:   tokenID,
		OrgID:     orgID,
		ProjectID: projectID,
		EnvID:     envID,
		Scope:     scope,
		TokenType: TokenTypeAPIKey,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "feature-flag-platform",
			Subject:   tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secret)
}

// GenerateServiceToken generates a JWT token for service-to-service communication
func (tm *TokenManager) GenerateServiceToken(serviceID string, scope string, expiry time.Duration) (string, error) {
	claims := &Claims{
		Scope:     scope,
		TokenType: TokenTypeService,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "feature-flag-platform",
			Subject:   serviceID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tm.secret)
}

// ValidateToken validates and parses a JWT token
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token has expired")
	}

	// Check if token is not yet valid
	if claims.NotBefore != nil && time.Now().Before(claims.NotBefore.Time) {
		return nil, fmt.Errorf("token not yet valid")
	}

	return claims, nil
}

// PasswordManager handles password hashing and verification
type PasswordManager struct {
	cost int
}

// NewPasswordManager creates a new password manager
func NewPasswordManager(cost int) *PasswordManager {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &PasswordManager{
		cost: cost,
	}
}

// HashPassword hashes a password using bcrypt
func (pm *PasswordManager) HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), pm.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (pm *PasswordManager) VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// APIKeyManager handles API key generation and hashing
type APIKeyManager struct{}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	return &APIKeyManager{}
}

// GenerateAPIKey generates a new random API key
func (akm *APIKeyManager) GenerateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as hex with a prefix
	return "ff_" + hex.EncodeToString(bytes), nil
}

// HashAPIKey hashes an API key for storage
func (akm *APIKeyManager) HashAPIKey(apiKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hash), nil
}

// VerifyAPIKey verifies an API key against its hash
func (akm *APIKeyManager) VerifyAPIKey(apiKey, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(apiKey))
}

// Scope represents authorization scope
type Scope string

const (
	ScopeRead  Scope = "read"
	ScopeWrite Scope = "write"
	ScopeAdmin Scope = "admin"
)

// Role represents user role in organization
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Permission represents a specific permission
type Permission string

const (
	// Organization permissions
	PermOrgCreate Permission = "org:create"
	PermOrgRead   Permission = "org:read"
	PermOrgUpdate Permission = "org:update"
	PermOrgDelete Permission = "org:delete"

	// Project permissions
	PermProjectCreate Permission = "project:create"
	PermProjectRead   Permission = "project:read"
	PermProjectUpdate Permission = "project:update"
	PermProjectDelete Permission = "project:delete"

	// Environment permissions
	PermEnvCreate Permission = "env:create"
	PermEnvRead   Permission = "env:read"
	PermEnvUpdate Permission = "env:update"
	PermEnvDelete Permission = "env:delete"

	// Flag permissions
	PermFlagCreate  Permission = "flag:create"
	PermFlagRead    Permission = "flag:read"
	PermFlagUpdate  Permission = "flag:update"
	PermFlagDelete  Permission = "flag:delete"
	PermFlagPublish Permission = "flag:publish"

	// Experiment permissions
	PermExperimentCreate Permission = "experiment:create"
	PermExperimentRead   Permission = "experiment:read"
	PermExperimentUpdate Permission = "experiment:update"
	PermExperimentDelete Permission = "experiment:delete"
	PermExperimentStart  Permission = "experiment:start"
	PermExperimentStop   Permission = "experiment:stop"

	// Analytics permissions
	PermAnalyticsRead Permission = "analytics:read"

	// Admin permissions
	PermUserManage  Permission = "user:manage"
	PermTokenManage Permission = "token:manage"
	PermAuditRead   Permission = "audit:read"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleOwner: {
		PermOrgCreate, PermOrgRead, PermOrgUpdate, PermOrgDelete,
		PermProjectCreate, PermProjectRead, PermProjectUpdate, PermProjectDelete,
		PermEnvCreate, PermEnvRead, PermEnvUpdate, PermEnvDelete,
		PermFlagCreate, PermFlagRead, PermFlagUpdate, PermFlagDelete, PermFlagPublish,
		PermExperimentCreate, PermExperimentRead, PermExperimentUpdate, PermExperimentDelete,
		PermExperimentStart, PermExperimentStop,
		PermAnalyticsRead,
		PermUserManage, PermTokenManage, PermAuditRead,
	},
	RoleAdmin: {
		PermOrgRead, PermOrgUpdate,
		PermProjectCreate, PermProjectRead, PermProjectUpdate, PermProjectDelete,
		PermEnvCreate, PermEnvRead, PermEnvUpdate, PermEnvDelete,
		PermFlagCreate, PermFlagRead, PermFlagUpdate, PermFlagDelete, PermFlagPublish,
		PermExperimentCreate, PermExperimentRead, PermExperimentUpdate, PermExperimentDelete,
		PermExperimentStart, PermExperimentStop,
		PermAnalyticsRead,
		PermUserManage, PermTokenManage, PermAuditRead,
	},
	RoleEditor: {
		PermOrgRead,
		PermProjectRead, PermProjectUpdate,
		PermEnvRead, PermEnvUpdate,
		PermFlagCreate, PermFlagRead, PermFlagUpdate, PermFlagPublish,
		PermExperimentCreate, PermExperimentRead, PermExperimentUpdate,
		PermExperimentStart, PermExperimentStop,
		PermAnalyticsRead,
	},
	RoleViewer: {
		PermOrgRead,
		PermProjectRead,
		PermEnvRead,
		PermFlagRead,
		PermExperimentRead,
		PermAnalyticsRead,
	},
}

// AuthorizationManager handles authorization checks
type AuthorizationManager struct{}

// NewAuthorizationManager creates a new authorization manager
func NewAuthorizationManager() *AuthorizationManager {
	return &AuthorizationManager{}
}

// HasPermission checks if a role has a specific permission
func (am *AuthorizationManager) HasPermission(role Role, permission Permission) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// CanAccessResource checks if a user can access a resource based on scope
func (am *AuthorizationManager) CanAccessResource(claims *Claims, resourceOrgID, resourceProjectID, resourceEnvID string) bool {
	if claims == nil {
		return false
	}

	// Service tokens have full access
	if claims.TokenType == TokenTypeService {
		return true
	}

	// Check organization access
	if claims.OrgID != "" && claims.OrgID != resourceOrgID {
		return false
	}

	// Check project access for API tokens
	if claims.TokenType == TokenTypeAPIKey {
		if claims.ProjectID != "" && claims.ProjectID != resourceProjectID {
			return false
		}

		if claims.EnvID != "" && claims.EnvID != resourceEnvID {
			return false
		}
	}

	return true
}

// GetScopePermissions returns permissions for a given scope
func (am *AuthorizationManager) GetScopePermissions(scope string) []Permission {
	switch Scope(scope) {
	case ScopeRead:
		return []Permission{
			PermOrgRead, PermProjectRead, PermEnvRead,
			PermFlagRead, PermExperimentRead, PermAnalyticsRead,
		}
	case ScopeWrite:
		return []Permission{
			PermOrgRead, PermProjectRead, PermEnvRead,
			PermFlagRead, PermFlagUpdate, PermFlagPublish,
			PermExperimentRead, PermExperimentUpdate,
			PermAnalyticsRead,
		}
	case ScopeAdmin:
		return RolePermissions[RoleAdmin]
	default:
		return []Permission{}
	}
}

// ValidateScope validates if a scope string is valid
func (am *AuthorizationManager) ValidateScope(scope string) bool {
	switch Scope(scope) {
	case ScopeRead, ScopeWrite, ScopeAdmin:
		return true
	default:
		return false
	}
}

// Context represents the authentication context
type Context struct {
	Claims      *Claims
	UserID      string
	Email       string
	OrgID       string
	ProjectID   string
	EnvID       string
	Scope       string
	TokenType   TokenType
	IsAnonymous bool
}

// NewContext creates a new authentication context from claims
func NewContext(claims *Claims) *Context {
	if claims == nil {
		return &Context{
			IsAnonymous: true,
		}
	}

	return &Context{
		Claims:      claims,
		UserID:      claims.UserID,
		Email:       claims.Email,
		OrgID:       claims.OrgID,
		ProjectID:   claims.ProjectID,
		EnvID:       claims.EnvID,
		Scope:       claims.Scope,
		TokenType:   claims.TokenType,
		IsAnonymous: false,
	}
}

// HasPermission checks if the context has a specific permission
func (ctx *Context) HasPermission(permission Permission, am *AuthorizationManager) bool {
	if ctx.IsAnonymous {
		return false
	}

	// For API tokens, check scope permissions
	if ctx.TokenType == TokenTypeAPIKey {
		scopePermissions := am.GetScopePermissions(ctx.Scope)
		for _, p := range scopePermissions {
			if p == permission {
				return true
			}
		}
		return false
	}

	// For user tokens, we'd typically look up the user's role in the organization
	// This is simplified - in a real implementation, you'd query the database
	// for the user's role in the specific organization
	return false
}

// CanAccessOrg checks if the context can access an organization
func (ctx *Context) CanAccessOrg(orgID string) bool {
	if ctx.IsAnonymous {
		return false
	}

	return ctx.OrgID == orgID || ctx.TokenType == TokenTypeService
}

// CanAccessProject checks if the context can access a project
func (ctx *Context) CanAccessProject(orgID, projectID string) bool {
	if !ctx.CanAccessOrg(orgID) {
		return false
	}

	if ctx.TokenType == TokenTypeAPIKey && ctx.ProjectID != "" {
		return ctx.ProjectID == projectID
	}

	return true
}

// CanAccessEnv checks if the context can access an environment
func (ctx *Context) CanAccessEnv(orgID, projectID, envID string) bool {
	if !ctx.CanAccessProject(orgID, projectID) {
		return false
	}

	if ctx.TokenType == TokenTypeAPIKey && ctx.EnvID != "" {
		return ctx.EnvID == envID
	}

	return true
}
