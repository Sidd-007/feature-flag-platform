package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/handlers"
	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/middleware"
	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/services"
	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/storage"
	"github.com/Sidd-007/feature-flag-platform/pkg/auth"
	"github.com/Sidd-007/feature-flag-platform/pkg/config"
)

// Server represents the event ingestor server
type Server struct {
	config *config.Config
	logger zerolog.Logger

	// External connections
	clickhouse clickhouse.Conn
	nats       *nats.Conn

	// Core services
	ingestionService  *services.IngestionService
	validationService *services.ValidationService

	// Storage
	eventStorage *storage.EventStorage

	// Handlers
	handlers *handlers.Handlers

	// Auth components
	tokenManager *auth.TokenManager
}

// New creates a new event ingestor server instance
func New(cfg *config.Config, logger zerolog.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		logger: logger,
	}

	// Initialize ClickHouse
	if err := s.initClickHouse(); err != nil {
		return nil, fmt.Errorf("failed to initialize ClickHouse: %w", err)
	}

	// Initialize NATS
	if err := s.initNATS(); err != nil {
		return nil, fmt.Errorf("failed to initialize NATS: %w", err)
	}

	// Initialize auth components
	if err := s.initAuth(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	// Initialize storage
	if err := s.initStorage(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize services
	if err := s.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize handlers
	if err := s.initHandlers(); err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	logger.Info().Msg("Event ingestor server initialized successfully")
	return s, nil
}

// SetupRoutes configures HTTP routes
func (s *Server) SetupRoutes(r *chi.Mux) {
	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(s.tokenManager, s.logger)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Event ingestion endpoints (require API key authentication)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.AuthenticateAPIKey)

			r.Post("/events/exposure", s.handlers.Events.IngestExposureEvents)
			r.Post("/events/metrics", s.handlers.Events.IngestMetricEvents)
			r.Post("/events/batch", s.handlers.Events.IngestEventBatch)
		})

		// Health and readiness endpoints (no auth required)
		r.Get("/health", s.handlers.Health.Health)
		r.Get("/ready", s.handlers.Health.Ready)
		r.Get("/live", s.handlers.Health.Live)
		r.Get("/stats", s.handlers.Health.Stats)
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

	if s.ingestionService != nil {
		if err := s.ingestionService.Close(); err != nil {
			errors = append(errors, fmt.Errorf("ingestion service close error: %w", err))
		}
	}

	if s.nats != nil {
		s.nats.Close()
	}

	if s.clickhouse != nil {
		if err := s.clickhouse.Close(); err != nil {
			errors = append(errors, fmt.Errorf("clickhouse close error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during close: %v", errors)
	}

	s.logger.Info().Msg("Event ingestor server resources closed successfully")
	return nil
}

// ClickHouse initialization
func (s *Server) initClickHouse() error {
	// TODO: Build ClickHouse connection string from config
	dsn := fmt.Sprintf("tcp://%s:%d/analytics",
		s.config.Database.Host, // Reusing database config for simplicity
		9000)                   // ClickHouse default port

	var err error
	s.clickhouse, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{dsn},
		Auth: clickhouse.Auth{
			Database: "analytics",
			Username: "default",
			Password: "",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: s.config.Database.MaxLifetime,
	})

	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := s.clickhouse.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	s.logger.Info().Msg("ClickHouse connection established")
	return nil
}

// NATS initialization
func (s *Server) initNATS() error {
	opts := []nats.Option{
		nats.Name("event-ingestor"),
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
	s.logger.Info().Msg("Auth components initialized")
	return nil
}

// Storage initialization
func (s *Server) initStorage() error {
	s.eventStorage = storage.NewEventStorage(s.clickhouse, s.logger)
	s.logger.Info().Msg("Event storage initialized")
	return nil
}

// Service initialization
func (s *Server) initServices() error {
	s.validationService = services.NewValidationService(s.logger)
	s.ingestionService = services.NewIngestionService(
		s.eventStorage,
		s.nats,
		s.validationService,
		s.config,
		s.logger,
	)

	// Start ingestion service
	if err := s.ingestionService.Start(); err != nil {
		return fmt.Errorf("failed to start ingestion service: %w", err)
	}

	s.logger.Info().Msg("Services initialized")
	return nil
}

// Handler initialization
func (s *Server) initHandlers() error {
	s.handlers = handlers.New(
		s.ingestionService,
		s.validationService,
		s.logger,
	)

	s.logger.Info().Msg("Handlers initialized")
	return nil
}
