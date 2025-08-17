package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/pkg/auth"
	"github.com/feature-flag-platform/pkg/config"
	"github.com/feature-flag-platform/pkg/rbac"
)

// AuthService handles authentication operations
type AuthService struct {
	repos        *repository.Repositories
	tokenManager *auth.TokenManager
	passwordMgr  *auth.PasswordManager
	rbac         *rbac.RBAC
	config       *config.Config
	logger       zerolog.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(repos *repository.Repositories, tokenManager *auth.TokenManager, rbacManager *rbac.RBAC, cfg *config.Config, logger zerolog.Logger) *AuthService {
	return &AuthService{
		repos:        repos,
		tokenManager: tokenManager,
		passwordMgr:  auth.NewPasswordManager(cfg.Auth.BCryptCost),
		rbac:         rbacManager,
		config:       cfg,
		logger:       logger.With().Str("service", "auth").Logger(),
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string           `json:"access_token"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int              `json:"expires_in"`
	RefreshToken string           `json:"refresh_token,omitempty"`
	User         *repository.User `json:"user"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Get user by email
	user, err := s.repos.User.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("invalid credentials")
		}
		s.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to get user")
		return nil, fmt.Errorf("authentication failed")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Verify password
	if err := s.passwordMgr.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		s.logger.Debug().Str("email", req.Email).Msg("Invalid password attempt")
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login
	if err := s.repos.User.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn().Err(err).Str("user_id", user.ID.String()).Msg("Failed to update last login")
	}

	// Generate access token
	accessToken, err := s.tokenManager.GenerateUserToken(
		user.ID.String(),
		user.Email,
		"", // Will be set when user selects an organization
		s.config.Auth.JWTExpiry,
	)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to generate access token")
		return nil, fmt.Errorf("token generation failed")
	}

	s.logger.Info().Str("user_id", user.ID.String()).Str("email", user.Email).Msg("Generated access token for user")
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to generate access token")
		return nil, fmt.Errorf("token generation failed")
	}

	// Clear sensitive data
	user.PasswordHash = ""

	response := &LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.config.Auth.JWTExpiry.Seconds()),
		User:        user,
	}

	s.logger.Info().Str("user_id", user.ID.String()).Str("email", user.Email).Msg("User logged in")
	return response, nil
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*repository.User, error) {
	// Check if email already exists
	exists, err := s.repos.User.CheckEmailExists(ctx, req.Email)
	if err != nil {
		s.logger.Error().Err(err).Str("email", req.Email).Msg("Failed to check email existence")
		return nil, fmt.Errorf("registration failed")
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	passwordHash, err := s.passwordMgr.HashPassword(req.Password)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to hash password")
		return nil, fmt.Errorf("registration failed")
	}

	// Create user
	createReq := &repository.CreateUserRequest{
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
	}

	user, err := s.repos.User.Create(ctx, createReq)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create user")
		return nil, fmt.Errorf("registration failed")
	}

	// Clear sensitive data
	user.PasswordHash = ""

	s.logger.Info().Str("user_id", user.ID.String()).Str("email", user.Email).Msg("User registered")
	return user, nil
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Validate refresh token
	claims, err := s.tokenManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get user
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// Generate new access token
	accessToken, err := s.tokenManager.GenerateUserToken(
		user.ID.String(),
		user.Email,
		claims.OrgID,
		s.config.Auth.JWTExpiry,
	)
	if err != nil {
		return nil, fmt.Errorf("token generation failed")
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.config.Auth.JWTExpiry.Seconds()),
		User:        user,
	}, nil
}

// ValidateToken validates a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*auth.Claims, error) {
	return s.tokenManager.ValidateToken(tokenString)
}
