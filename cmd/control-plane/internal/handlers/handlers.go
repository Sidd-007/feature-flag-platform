package handlers

import (
	"github.com/rs/zerolog"

	"github.com/Sidd-007/feature-flag-platform/cmd/control-plane/internal/services"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	Auth         *AuthHandler
	Organization *OrganizationHandler
	Project      *ProjectHandler
	Environment  *EnvironmentHandler
	Flag         *FlagHandler
	Segment      *SegmentHandler
	APIToken     *APITokenHandler
	Config       *ConfigHandler
}

// New creates a new handlers collection
func New(
	authService *services.AuthService,
	orgService *services.OrganizationService,
	projectService *services.ProjectService,
	envService *services.EnvironmentService,
	flagService *services.FlagService,
	segmentService *services.SegmentService,
	tokenService *services.APITokenService,
	configService *services.ConfigService,
	logger zerolog.Logger,
) *Handlers {
	return &Handlers{
		Auth:         NewAuthHandler(authService, logger),
		Organization: NewOrganizationHandler(orgService, logger),
		Project:      NewProjectHandler(projectService, logger),
		Environment:  NewEnvironmentHandler(envService, logger),
		Flag:         NewFlagHandler(flagService, logger),
		Segment:      NewSegmentHandler(segmentService, logger),
		APIToken:     NewAPITokenHandler(tokenService, logger),
		Config:       NewConfigHandler(configService, logger),
	}
}
