package server

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/feature-flag-platform/cmd/analytics-engine/internal/handlers"
	"github.com/feature-flag-platform/cmd/analytics-engine/internal/middleware"
	"github.com/feature-flag-platform/cmd/analytics-engine/internal/repository"
	"github.com/feature-flag-platform/cmd/analytics-engine/internal/services"
	"github.com/feature-flag-platform/pkg/config"
)

type Server struct {
	config     *config.Config
	clickhouse clickhouse.Conn

	// Repositories
	eventRepo repository.EventRepository

	// Services
	statsService      services.StatisticsService
	experimentService services.ExperimentService

	// Handlers
	handlers *handlers.Handlers
}

func NewServer(cfg *config.Config) (*Server, error) {
	log.Info().Msg("Initializing Analytics Engine server")

	// Initialize ClickHouse connection
	clickhouseConn, err := initClickHouse(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ClickHouse: %w", err)
	}

	// Initialize repositories
	eventRepo := repository.NewEventRepository(clickhouseConn)

	// Initialize services
	statsService := services.NewStatisticsService()
	experimentService := services.NewExperimentService(eventRepo, statsService)

	// Initialize handlers
	authMiddleware := middleware.NewAuthMiddleware(cfg)
	handlers := handlers.NewHandlers(experimentService, statsService, authMiddleware)

	server := &Server{
		config:            cfg,
		clickhouse:        clickhouseConn,
		eventRepo:         eventRepo,
		statsService:      statsService,
		experimentService: experimentService,
		handlers:          handlers,
	}

	log.Info().Msg("Analytics Engine server initialized successfully")
	return server, nil
}

func (s *Server) RegisterRoutes(r chi.Router) {
	log.Info().Msg("Registering Analytics Engine routes")

	// Health endpoints
	r.Get("/health", s.handlers.Health)
	r.Get("/ready", s.handlers.Ready)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication middleware for all API routes
		r.Use(s.handlers.AuthMiddleware.Authenticate)

		// Experiment analysis endpoints
		r.Route("/experiments", func(r chi.Router) {
			r.Get("/{experimentId}/results", s.handlers.GetExperimentResults)
			r.Get("/{experimentId}/summary", s.handlers.GetExperimentSummary)
			r.Get("/{experimentId}/timeline", s.handlers.GetExperimentTimeline)
			r.Post("/{experimentId}/analyze", s.handlers.AnalyzeExperiment)
		})

		// Statistical analysis endpoints
		r.Route("/analysis", func(r chi.Router) {
			r.Post("/ttest", s.handlers.RunTTest)
			r.Post("/chi-square", s.handlers.RunChiSquareTest)
			r.Post("/sequential", s.handlers.RunSequentialAnalysis)
		})

		// Funnel analysis
		r.Route("/funnels", func(r chi.Router) {
			r.Post("/analyze", s.handlers.AnalyzeFunnel)
		})

		// Cohort analysis
		r.Route("/cohorts", func(r chi.Router) {
			r.Post("/analyze", s.handlers.AnalyzeCohort)
		})

		// Metrics and insights
		r.Route("/metrics", func(r chi.Router) {
			r.Get("/exposure", s.handlers.GetExposureMetrics)
			r.Get("/conversion", s.handlers.GetConversionMetrics)
			r.Get("/retention", s.handlers.GetRetentionMetrics)
		})
	})

	log.Info().Msg("Analytics Engine routes registered")
}

func (s *Server) Cleanup() error {
	log.Info().Msg("Cleaning up Analytics Engine resources")

	if s.clickhouse != nil {
		if err := s.clickhouse.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing ClickHouse connection")
			return err
		}
	}

	log.Info().Msg("Analytics Engine cleanup completed")
	return nil
}

func initClickHouse(cfg *config.Config) (clickhouse.Conn, error) {
	log.Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("database", cfg.Database.Database).
		Msg("Connecting to ClickHouse")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database.Database,
			Username: cfg.Database.Username,
			Password: cfg.Database.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.Info().Msg("Successfully connected to ClickHouse")
	return conn, nil
}
