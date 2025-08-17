package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// User represents a user entity
type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	FirstName     string     `json:"first_name" db:"first_name"`
	LastName      string     `json:"last_name" db:"last_name"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	Version       int        `json:"version" db:"version"`
}

// CreateUserRequest represents request to create user
type CreateUserRequest struct {
	Email        string `json:"email" validate:"required,email"`
	PasswordHash string `json:"-" validate:"required"`
	FirstName    string `json:"first_name" validate:"required,min=1,max=255"`
	LastName     string `json:"last_name" validate:"required,min=1,max=255"`
}

// UserRepository handles user data access
type UserRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool, logger zerolog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger.With().Str("repository", "user").Logger(),
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	user := &User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
	}

	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at, version`

	err := r.db.QueryRow(ctx, query,
		user.ID, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.IsActive).
		Scan(&user.CreatedAt, &user.UpdatedAt, &user.Version)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to create user")
		return nil, err
	}

	r.logger.Info().Str("user_id", user.ID.String()).Str("email", user.Email).Msg("User created")
	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, is_active, 
		       email_verified, last_login_at, created_at, updated_at, version
		FROM users
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.EmailVerified, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("user_id", id.String()).Msg("Failed to get user")
		return nil, err
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, is_active, 
		       email_verified, last_login_at, created_at, updated_at, version
		FROM users
		WHERE email = $1`

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.EmailVerified, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		r.logger.Error().Err(err).Str("email", email).Msg("Failed to get user by email")
		return nil, err
	}

	return user, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error().Err(err).Str("user_id", id.String()).Msg("Failed to update last login")
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// CheckEmailExists checks if an email already exists
func (r *UserRepository) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		r.logger.Error().Err(err).Str("email", email).Msg("Failed to check email existence")
		return false, err
	}

	return exists, nil
}
