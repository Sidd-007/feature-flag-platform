package handlers

import (
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/event-ingestor/internal/services"
)

// Handlers holds all HTTP handlers for the event ingestor
type Handlers struct {
	Events *EventsHandler
	Health *HealthHandler
}

// New creates a new handlers collection
func New(
	ingestionService *services.IngestionService,
	validationService *services.ValidationService,
	logger zerolog.Logger,
) *Handlers {
	return &Handlers{
		Events: NewEventsHandler(ingestionService, logger),
		Health: NewHealthHandler(ingestionService, logger),
	}
}
