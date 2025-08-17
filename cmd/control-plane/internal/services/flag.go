package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/pkg/rbac"
)

// FlagService handles flag operations
type FlagService struct {
	repos         *repository.Repositories
	redis         *redis.Client
	nats          *nats.Conn
	rbac          *rbac.RBAC
	configService *ConfigService
	logger        zerolog.Logger
}

// NewFlagService creates a new flag service
func NewFlagService(repos *repository.Repositories, redisClient *redis.Client, natsConn *nats.Conn, rbacManager *rbac.RBAC, configService *ConfigService, logger zerolog.Logger) *FlagService {
	return &FlagService{
		repos:         repos,
		redis:         redisClient,
		nats:          natsConn,
		rbac:          rbacManager,
		configService: configService,
		logger:        logger.With().Str("service", "flag").Logger(),
	}
}

func (s *FlagService) Create(ctx context.Context, envID uuid.UUID, req *repository.CreateFlagRequest) (*repository.Flag, error) {
	// ensure env exists
	if _, err := s.repos.Environment.GetByID(ctx, envID); err != nil {
		return nil, fmt.Errorf("environment not found")
	}
	req.EnvID = envID
	flag, err := s.repos.Flag.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create flag")
	}

	s.logger.Info().Str("env_id", envID.String()).Str("flag_key", flag.Key).Msg("Flag created successfully (unpublished)")
	return flag, nil
}

func (s *FlagService) List(ctx context.Context, envID uuid.UUID, limit, offset int) ([]*repository.Flag, int, error) {
	flags, total, err := s.repos.Flag.List(ctx, envID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve flags")
	}
	return flags, total, nil
}

func (s *FlagService) GetByKey(ctx context.Context, envID uuid.UUID, key string) (*repository.Flag, error) {
	f, err := s.repos.Flag.GetByKey(ctx, envID, key)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("flag not found")
		}
		return nil, fmt.Errorf("failed to retrieve flag")
	}
	return f, nil
}

func (s *FlagService) Update(ctx context.Context, id uuid.UUID, req *repository.UpdateFlagRequest) (*repository.Flag, error) {
	f, err := s.repos.Flag.Update(ctx, id, req)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("flag not found")
		}
		return nil, fmt.Errorf("failed to update flag")
	}

	s.logger.Info().Str("env_id", f.EnvID.String()).Str("flag_key", f.Key).Msg("Flag updated successfully")
	return f, nil
}

// UpdateByKey updates a flag identified by envID and key
func (s *FlagService) UpdateByKey(ctx context.Context, envID uuid.UUID, key string, req *repository.UpdateFlagRequest) (*repository.Flag, error) {
	f, err := s.repos.Flag.GetByKey(ctx, envID, key)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("flag not found")
		}
		return nil, fmt.Errorf("failed to retrieve flag")
	}
	return s.Update(ctx, f.ID, req)
}

func (s *FlagService) Delete(ctx context.Context, id uuid.UUID) error {
	// Get flag info before deletion for publishing
	flag, err := s.repos.Flag.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("flag not found")
		}
		return fmt.Errorf("failed to retrieve flag")
	}

	if err := s.repos.Flag.Delete(ctx, id); err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("flag not found")
		}
		return fmt.Errorf("failed to delete flag")
	}

	s.logger.Info().Str("env_id", flag.EnvID.String()).Str("flag_key", flag.Key).Msg("Flag deleted successfully")
	return nil
}

// PublishFlag publishes an individual flag
func (s *FlagService) PublishFlag(ctx context.Context, envID uuid.UUID, flagKey string) (*repository.Flag, error) {
	// Get the flag first
	flag, err := s.repos.Flag.GetByKey(ctx, envID, flagKey)
	if err != nil {
		return nil, fmt.Errorf("flag not found: %w", err)
	}

	// Set published to true
	publishedFlag, err := s.repos.Flag.SetPublished(ctx, flag.ID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to publish flag: %w", err)
	}

	s.logger.Info().Str("env_id", envID.String()).Str("flag_key", flagKey).Msg("Flag published successfully")
	return publishedFlag, nil
}

// UnpublishFlag unpublishes an individual flag
func (s *FlagService) UnpublishFlag(ctx context.Context, envID uuid.UUID, flagKey string) (*repository.Flag, error) {
	// Get the flag first
	flag, err := s.repos.Flag.GetByKey(ctx, envID, flagKey)
	if err != nil {
		return nil, fmt.Errorf("flag not found: %w", err)
	}

	// Set published to false
	unpublishedFlag, err := s.repos.Flag.SetPublished(ctx, flag.ID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to unpublish flag: %w", err)
	}

	s.logger.Info().Str("env_id", envID.String()).Str("flag_key", flagKey).Msg("Flag unpublished successfully")
	return unpublishedFlag, nil
}
