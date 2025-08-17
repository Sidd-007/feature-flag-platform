package handlers

import (
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/edge-evaluator/internal/services"
)

// Handlers holds all HTTP handlers for the edge evaluator
type Handlers struct {
	Evaluation *EvaluationHandler
	Config     *ConfigHandler
	Health     *HealthHandler
}

// New creates a new handlers collection
func New(
	evaluationService *services.EvaluationService,
	configService *services.ConfigService,
	logger zerolog.Logger,
) *Handlers {
	return &Handlers{
		Evaluation: NewEvaluationHandler(evaluationService, logger),
		Config:     NewConfigHandler(configService, logger),
		Health:     NewHealthHandler(evaluationService, configService, logger),
	}
}
