package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Repositories holds all repository instances
type Repositories struct {
	Organization *OrganizationRepository
	Project      *ProjectRepository
	Environment  *EnvironmentRepository
	Flag         *FlagRepository
	Segment      *SegmentRepository
	Experiment   *ExperimentRepository
	User         *UserRepository
	APIToken     *APITokenRepository
	AuditLog     *AuditLogRepository
}

// New creates a new repository collection
func New(db *pgxpool.Pool, logger zerolog.Logger) *Repositories {
	return &Repositories{
		Organization: NewOrganizationRepository(db, logger),
		Project:      NewProjectRepository(db, logger),
		Environment:  NewEnvironmentRepository(db, logger),
		Flag:         NewFlagRepository(db, logger),
		Segment:      NewSegmentRepository(db, logger),
		Experiment:   NewExperimentRepository(db, logger),
		User:         NewUserRepository(db, logger),
		APIToken:     NewAPITokenRepository(db, logger),
		AuditLog:     NewAuditLogRepository(db, logger),
	}
}
