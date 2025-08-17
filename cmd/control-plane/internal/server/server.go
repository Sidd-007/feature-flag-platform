package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/feature-flag-platform/cmd/control-plane/internal/handlers"
	"github.com/feature-flag-platform/cmd/control-plane/internal/middleware"
	"github.com/feature-flag-platform/cmd/control-plane/internal/repository"
	"github.com/feature-flag-platform/cmd/control-plane/internal/services"
	"github.com/feature-flag-platform/pkg/auth"
	"github.com/feature-flag-platform/pkg/config"
	"github.com/feature-flag-platform/pkg/rbac"
)

// Server represents the control plane server
type Server struct {
	config *config.Config
	logger zerolog.Logger

	// Database connections
	db    *pgxpool.Pool
	redis *redis.Client
	nats  *nats.Conn

	// Core services
	authService    *services.AuthService
	orgService     *services.OrganizationService
	projectService *services.ProjectService
	envService     *services.EnvironmentService
	flagService    *services.FlagService
	segmentService *services.SegmentService
	tokenService   *services.APITokenService
	configService  *services.ConfigService

	// Repositories
	repos *repository.Repositories

	// Handlers
	handlers *handlers.Handlers

	// Auth components
	tokenManager *auth.TokenManager
	rbac         *rbac.RBAC
}

// New creates a new server instance
func New(cfg *config.Config, logger zerolog.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		logger: logger,
	}

	// Initialize database connections
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

	// Initialize repositories
	if err := s.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// Initialize services
	if err := s.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize handlers
	if err := s.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	logger.Info().Msg("Server initialized successfully")
	return s, nil
}

// SetupRoutes configures HTTP routes
func (s *Server) SetupRoutes(r *chi.Mux) {
	// Auth middleware
	authMiddleware := middleware.NewAuthMiddleware(s.tokenManager, s.rbac, s.db, s.logger)

	// Root/info
	r.Get("/", s.handleRoot)
	r.Get("/health", s.handleHealth)
	r.Get("/v1", s.handleAPIInfo)

	// API v1
	r.Route("/v1", func(r chi.Router) {
		// --- Public auth endpoints ---
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", s.handlers.Auth.Login)
			r.Post("/register", s.handlers.Auth.Register)
			r.Post("/refresh", s.handlers.Auth.RefreshToken)
		})

		// --- Public config (env API key auth) ---
		r.Route("/configs", func(r chi.Router) {
			r.Use(authMiddleware.AuthenticateAPIKey)
			r.Get("/{envKey}", s.handlers.Config.GetEnvironmentConfig)
		})

		// --- Protected (user auth) ---
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// Orgs
			r.Route("/orgs", func(r chi.Router) {
				r.Get("/", s.handlers.Organization.List)
				r.Post("/", s.handlers.Organization.Create)

				r.Route("/{orgId}", func(r chi.Router) {
					r.Get("/", s.handlers.Organization.Get)
					r.Put("/", s.handlers.Organization.Update)
					r.Delete("/", s.handlers.Organization.Delete)

					// Projects
					r.Route("/projects", func(r chi.Router) {
						r.Get("/", s.handlers.Project.List)
						r.Post("/", s.handlers.Project.Create)

						r.Route("/{projectId}", func(r chi.Router) {
							r.Get("/", s.handlers.Project.Get)
							r.Put("/", s.handlers.Project.Update)
							r.Delete("/", s.handlers.Project.Delete)

							// Environments
							r.Route("/environments", func(r chi.Router) {
								r.Get("/", s.handlers.Environment.List)
								r.Post("/", s.handlers.Environment.Create)

								r.Route("/{envId}", func(r chi.Router) {
									r.Get("/", s.handlers.Environment.Get)
									r.Put("/", s.handlers.Environment.Update)
									r.Delete("/", s.handlers.Environment.Delete)

									// Flags
									r.Route("/flags", func(r chi.Router) {
										r.Get("/", s.handlers.Flag.List)
										r.Post("/", s.handlers.Flag.Create)

										r.Route("/{flagKey}", func(r chi.Router) {
											r.Get("/", s.handlers.Flag.Get)
											r.Put("/", s.handlers.Flag.Update)
											r.Delete("/", s.handlers.Flag.Delete)
											r.Post("/publish", s.handlers.Flag.Publish)
											r.Post("/unpublish", s.handlers.Flag.Unpublish)
										})
									})

									// Segments
									r.Route("/segments", func(r chi.Router) {
										r.Get("/", s.handlers.Segment.List)
										r.Post("/", s.handlers.Segment.Create)

										r.Route("/{segmentId}", func(r chi.Router) {
											r.Get("/", s.handlers.Segment.Get)
											r.Put("/", s.handlers.Segment.Update)
											r.Delete("/", s.handlers.Segment.Delete)
										})
									})

									// API Tokens
									r.Route("/tokens", func(r chi.Router) {
										r.Get("/", s.handlers.APIToken.List)
										r.Post("/", s.handlers.APIToken.Create)
										r.Delete("/{tokenId}", s.handlers.APIToken.Revoke)
									})
								})
							})
						})
					})
				})
			})
		})
	})

	// Dev-only debug endpoints
	if s.config.IsDevelopment() {
		r.Get("/debug/*", func(w http.ResponseWriter, r *http.Request) {
			// hook pprof, etc.
		})
	}
}

// Close gracefully closes all server resources
func (s *Server) Close() error {
	var errors []error

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

	s.logger.Info().Msg("Server resources closed successfully")
	return nil
}

// Database initialization
func (s *Server) initDatabase() error {
	dbConfig, err := pgxpool.ParseConfig(s.config.GetDatabaseDSN())
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	dbConfig.MaxConns = int32(s.config.Database.MaxOpenConns)
	dbConfig.MinConns = int32(s.config.Database.MaxIdleConns)
	dbConfig.MaxConnLifetime = s.config.Database.MaxLifetime

	s.db, err = pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test connection
	if err := s.db.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
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
		nats.Name("control-plane"),
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

	var err error
	s.rbac, err = rbac.NewRBAC()
	if err != nil {
		return fmt.Errorf("failed to initialize RBAC: %w", err)
	}

	s.logger.Info().Msg("Auth components initialized")
	return nil
}

// Repository initialization
func (s *Server) initRepositories() error {
	s.repos = repository.New(s.db, s.logger)
	s.logger.Info().Msg("Repositories initialized")
	return nil
}

// Service initialization
func (s *Server) initServices() error {
	s.authService = services.NewAuthService(s.repos, s.tokenManager, s.rbac, s.config, s.logger)
	s.orgService = services.NewOrganizationService(s.repos, s.rbac, s.logger)
	s.projectService = services.NewProjectService(s.repos, s.rbac, s.logger)
	s.envService = services.NewEnvironmentService(s.repos, s.rbac, s.logger)
	s.segmentService = services.NewSegmentService(s.repos, s.rbac, s.logger)
	s.tokenService = services.NewAPITokenService(s.repos, s.tokenManager, s.rbac, s.logger)
	s.configService = services.NewConfigService(s.repos, s.redis, s.logger)
	s.flagService = services.NewFlagService(s.repos, s.redis, s.nats, s.rbac, s.configService, s.logger)

	s.logger.Info().Msg("Services initialized")
	return nil
}

// Handler initialization
func (s *Server) initHandlers() error {
	s.handlers = handlers.New(
		s.authService,
		s.orgService,
		s.projectService,
		s.envService,
		s.flagService,
		s.segmentService,
		s.tokenService,
		s.configService,
		s.logger,
	)

	s.logger.Info().Msg("Handlers initialized")
	return nil
}

// Basic HTTP handlers
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service": "Feature Flag Control Plane API",
		"version": "1.0.0",
		"status":  "running",
		"api":     "/v1",
		"docs":    "/v1",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services": map[string]string{
			"database": "connected",
			"redis":    "connected",
			"nats":     "connected",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAPIInfo(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"name":        "Feature Flag Control Plane API",
		"version":     "1.0.0",
		"description": "Control Plane API for Feature Flag & Experimentation Platform",
		"endpoints": map[string]interface{}{
			"authentication": "/v1/auth",
			"organizations":  "/v1/orgs",
			"health":         "/v1/health",
			"docs":           "https://github.com/feature-flag-platform/docs",
		},
		"status": "ready",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
