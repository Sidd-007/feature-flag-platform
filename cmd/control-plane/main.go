package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feature-flag-platform/cmd/control-plane/internal/server"
	"github.com/feature-flag-platform/pkg/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger := setupLogger(cfg)
	logger.Info().Msg("Starting Feature Flag Control Plane API")

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create server")
	}

	// Setup HTTP server
	r := chi.NewRouter()
	setupMiddleware(r, cfg, logger)
	srv.SetupRoutes(r)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info().
			Str("host", cfg.Server.Host).
			Int("port", cfg.Server.Port).
			Msg("Control Plane API server starting")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	// Close server resources
	if err := srv.Close(); err != nil {
		logger.Error().Err(err).Msg("Error closing server resources")
	}

	logger.Info().Msg("Server exited")
}

func setupLogger(cfg *config.Config) zerolog.Logger {
	// Set log level
	level, err := zerolog.ParseLevel(cfg.Observability.Logging.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure logger output
	var logger zerolog.Logger
	if cfg.Observability.Logging.Structured {
		logger = zerolog.New(os.Stdout).With().
			Timestamp().
			Str("service", "control-plane").
			Str("version", "1.0.0").
			Logger()
	} else {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().
			Timestamp().
			Str("service", "control-plane").
			Logger()
	}

	return logger
}

func setupMiddleware(r *chi.Mux, cfg *config.Config, logger zerolog.Logger) {
	// Basic middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Logging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Str("user_agent", r.UserAgent()).
					Int("status", ww.Status()).
					Int("bytes", ww.BytesWritten()).
					Dur("duration", time.Since(start)).
					Str("request_id", middleware.GetReqID(r.Context())).
					Msg("HTTP request")
			}()

			next.ServeHTTP(ww, r)
		})
	})

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure properly for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check endpoint (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","service":"control-plane"}`, time.Now().Format(time.RFC3339))
	})
}
