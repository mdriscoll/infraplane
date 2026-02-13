package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// DeploymentService handles deployment orchestration.
type DeploymentService struct {
	deployments repository.DeploymentRepo
	apps        repository.ApplicationRepo
}

// NewDeploymentService creates a new DeploymentService.
func NewDeploymentService(deployments repository.DeploymentRepo, apps repository.ApplicationRepo) *DeploymentService {
	return &DeploymentService{
		deployments: deployments,
		apps:        apps,
	}
}

// Deploy creates a new deployment for an application.
func (s *DeploymentService) Deploy(ctx context.Context, appID uuid.UUID, gitCommit, gitBranch string) (domain.Deployment, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("get application: %w", err)
	}

	d := domain.NewDeployment(appID, app.Provider, gitCommit, gitBranch)
	if err := d.Validate(); err != nil {
		return domain.Deployment{}, err
	}

	if err := s.deployments.Create(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("create deployment: %w", err)
	}

	return d, nil
}

// GetStatus returns a deployment by ID.
func (s *DeploymentService) GetStatus(ctx context.Context, id uuid.UUID) (domain.Deployment, error) {
	return s.deployments.GetByID(ctx, id)
}

// ListByApplication returns all deployments for an application.
func (s *DeploymentService) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	return s.deployments.ListByApplicationID(ctx, appID)
}

// GetLatest returns the most recent deployment for an application.
func (s *DeploymentService) GetLatest(ctx context.Context, appID uuid.UUID) (domain.Deployment, error) {
	return s.deployments.GetLatestByApplicationID(ctx, appID)
}

// MarkSucceeded marks a deployment as succeeded.
func (s *DeploymentService) MarkSucceeded(ctx context.Context, id uuid.UUID, terraformPlan string) (domain.Deployment, error) {
	d, err := s.deployments.GetByID(ctx, id)
	if err != nil {
		return domain.Deployment{}, err
	}
	now := time.Now().UTC()
	d.Status = domain.DeploymentSucceeded
	d.CompletedAt = &now
	d.TerraformPlan = terraformPlan
	if err := s.deployments.Update(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("update deployment: %w", err)
	}
	return d, nil
}

// MarkFailed marks a deployment as failed.
func (s *DeploymentService) MarkFailed(ctx context.Context, id uuid.UUID) (domain.Deployment, error) {
	d, err := s.deployments.GetByID(ctx, id)
	if err != nil {
		return domain.Deployment{}, err
	}
	now := time.Now().UTC()
	d.Status = domain.DeploymentFailed
	d.CompletedAt = &now
	if err := s.deployments.Update(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("update deployment: %w", err)
	}
	return d, nil
}
