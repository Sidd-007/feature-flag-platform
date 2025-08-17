package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/cache"
	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/handlers"
	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/middleware"
	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/services"
	"github.com/Sidd-007/feature-flag-platform/pkg/auth"
	"github.com/Sidd-007/feature-flag-platform/pkg/bucketing"
	"github.com/Sidd-007/feature-flag-platform/pkg/config"
)

// Server represents the edge evaluator server
type Server struct {
	config *config.Config
	logger zerolog.Logger

	// External connections
	redis *redis.Client
	nats  *nats.Conn
	db    *pgxpool.Pool

	// Core services
	configService     *services.ConfigService
	evaluationService *services.EvaluationService
	eventService      *services.EventService

	// Cache
	configCache *cache.ConfigCache

	// Handlers
	handlers *handlers.Handlers

	// Auth components
	tokenManager *auth.TokenManager
	bucketer     *bucketing.Bucketer
}

// New creates a new edge evaluator server instance
func New(cfg *config.Config, logger zerolog.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		logger: logger,
	}

	// Initialize database
	if err := s.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis
	if err := s.initRedis(); err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize NATS
	if err := s.initNATS(); err != nil {
		return nil, fmt.Errorf("failed to initialize NATS: %w", err)
	}

	// Initialize auth components
	if err := s.initAuth(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	// Initialize cache
	if err := s.initCache(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize services
	if err := s.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize handlers
	if err := s.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	logger.Info().Msg("Edge evaluator server initialized successfully")
	return s, nil
}

// SetupRoutes configures HTTP routes
func (s *Server) SetupRoutes(r *chi.Mux) {
	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(s.tokenManager, s.db, s.logger)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Evaluation endpoints (require API key authentication)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.AuthenticateAPIKey)

			r.Post("/evaluate", s.handlers.Evaluation.EvaluateFlags)
			r.Post("/evaluate/{envKey}", s.handlers.Evaluation.EvaluateAllFlags)
			r.Post("/evaluate/{envKey}/{flagKey}", s.handlers.Evaluation.EvaluateFlag)
		})

		// Configuration streaming (require API key authentication)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.AuthenticateAPIKey)
			r.Get("/stream/{envKey}", s.handlers.Config.StreamConfigUpdates)
		})

		// Health and readiness endpoints (no auth required)
		r.Get("/ready", s.handlers.Health.Ready)
		r.Get("/live", s.handlers.Health.Live)
	})

	// Metrics endpoint (no auth required in development)
	if s.config.IsDevelopment() {
		r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement Prometheus metrics endpoint
		})
	}
}

// Close gracefully closes all server resources
func (s *Server) Close() error {
	var errors []error

	if s.configService != nil {
		if err := s.configService.Close(); err != nil {
			errors = append(errors, fmt.Errorf("config service close error: %w", err))
		}
	}

	if s.nats != nil {
		s.nats.Close()
	}

	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			errors = append(errors, fmt.Errorf("redis close error: %w", err))
		}
	}

	if s.db != nil {
		s.db.Close()
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during close: %v", errors)
	}

	s.logger.Info().Msg("Edge evaluator server resources closed successfully")
	return nil
}

// Database initialization
func (s *Server) initDatabase() error {
	var err error
	s.db, err = pgxpool.New(context.Background(), s.config.GetDatabaseDSN())
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test connection
	if err := s.db.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	s.logger.Info().Msg("Database connection established")
	return nil
}

// Redis initialization
func (s *Server) initRedis() error {
	s.redis = redis.NewClient(&redis.Options{
		Addr:         s.config.GetRedisAddr(),
		Password:     s.config.Redis.Password,
		DB:           s.config.Redis.Database,
		PoolSize:     s.config.Redis.PoolSize,
		DialTimeout:  s.config.Redis.DialTimeout,
		ReadTimeout:  s.config.Redis.ReadTimeout,
		WriteTimeout: s.config.Redis.WriteTimeout,
	})

	// Test connection
	if err := s.redis.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	s.logger.Info().Msg("Redis connection established")
	return nil
}

// NATS initialization
func (s *Server) initNATS() error {
	opts := []nats.Option{
		nats.Name("edge-evaluator"),
		nats.MaxReconnects(s.config.NATS.MaxReconnect),
		nats.ReconnectWait(s.config.NATS.ReconnectWait),
		nats.Timeout(s.config.NATS.Timeout),
	}

	var err error
	s.nats, err = nats.Connect(s.config.NATS.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	s.logger.Info().Msg("NATS connection established")
	return nil
}

// Auth initialization
func (s *Server) initAuth() error {
	s.tokenManager = auth.NewTokenManager(s.config.Auth.JWTSecret)
	s.bucketer = bucketing.NewBucketer()

	s.logger.Info().Msg("Auth components initialized")
	return nil
}

// Cache initialization
func (s *Server) initCache() error {
	s.configCache = cache.NewConfigCache(s.redis, s.logger)
	s.logger.Info().Msg("Configuration cache initialized")
	return nil
}

// Service initialization
func (s *Server) initServices() error {
	s.eventService = services.NewEventService(s.config, s.logger)
	s.configService = services.NewConfigService(s.configCache, s.nats, s.config, s.logger)
	s.evaluationService = services.NewEvaluationService(s.configCache, s.bucketer, s.configService, s.eventService, s.logger)

	// Start config service (for receiving config updates)
	if err := s.configService.Start(); err != nil {
		return fmt.Errorf("failed to start config service: %w", err)
	}

	s.logger.Info().Msg("Services initialized")
	return nil
}

// Handler initialization
func (s *Server) initHandlers() error {
	s.handlers = handlers.New(
		s.evaluationService,
		s.configService,
		s.logger,
	)

	s.logger.Info().Msg("Handlers initialized")
	return nil
}
