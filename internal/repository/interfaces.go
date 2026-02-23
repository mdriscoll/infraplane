package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// ApplicationRepo defines data access for applications.
type ApplicationRepo interface {
	Create(ctx context.Context, app domain.Application) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Application, error)
	GetByName(ctx context.Context, name string) (domain.Application, error)
	List(ctx context.Context) ([]domain.Application, error)
	Update(ctx context.Context, app domain.Application) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ResourceRepo defines data access for resources.
type ResourceRepo interface {
	Create(ctx context.Context, r domain.Resource) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Resource, error)
	ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.Resource, error)
	ListCurrentByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.Resource, error)
	ListByAnalysisRunID(ctx context.Context, runID uuid.UUID) ([]domain.Resource, error)
	Update(ctx context.Context, r domain.Resource) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AnalysisRunRepo defines data access for analysis runs.
type AnalysisRunRepo interface {
	Create(ctx context.Context, run domain.AnalysisRun) error
	ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.AnalysisRun, error)
	UpdateResourceCount(ctx context.Context, id uuid.UUID, count int) error
}

// DeploymentRepo defines data access for deployments.
type DeploymentRepo interface {
	Create(ctx context.Context, d domain.Deployment) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Deployment, error)
	ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error)
	Update(ctx context.Context, d domain.Deployment) error
	GetLatestByApplicationID(ctx context.Context, appID uuid.UUID) (domain.Deployment, error)
	// FailOrphaned marks any in_progress or pending deployments as failed.
	// Called on server startup to clean up deployments orphaned by a previous crash/restart.
	FailOrphaned(ctx context.Context) (int, error)
}

// PlanRepo defines data access for infrastructure plans.
type PlanRepo interface {
	Create(ctx context.Context, p domain.InfrastructurePlan) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.InfrastructurePlan, error)
	ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.InfrastructurePlan, error)
}

// GraphRepo defines data access for infrastructure topology graphs.
type GraphRepo interface {
	Create(ctx context.Context, g domain.InfraGraph) error
	GetLatestByApplicationID(ctx context.Context, appID uuid.UUID) (domain.InfraGraph, error)
	ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.InfraGraph, error)
}
