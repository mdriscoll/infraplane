package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/provider"
	"github.com/matthewdriscoll/infraplane/internal/provider/terraform"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// InfraService orchestrates infrastructure deployment by generating Terraform
// from application resources and applying it via cloud provider adapters.
type InfraService struct {
	apps        repository.ApplicationRepo
	resources   repository.ResourceRepo
	deployments repository.DeploymentRepo
	providers   *provider.Registry
}

// NewInfraService creates a new InfraService.
func NewInfraService(
	apps repository.ApplicationRepo,
	resources repository.ResourceRepo,
	deployments repository.DeploymentRepo,
	providers *provider.Registry,
) *InfraService {
	return &InfraService{
		apps:        apps,
		resources:   resources,
		deployments: deployments,
		providers:   providers,
	}
}

// Apps returns the application repository used by this service.
func (s *InfraService) Apps() repository.ApplicationRepo { return s.apps }

// Providers returns the provider registry used by this service.
func (s *InfraService) Providers() *provider.Registry { return s.providers }

// GenerateTerraform generates a complete Terraform configuration for an application
// on its configured provider. It aggregates HCL from all resource provider mappings.
func (s *InfraService) GenerateTerraform(ctx context.Context, appID uuid.UUID) (string, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListByApplicationID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("list resources: %w", err)
	}

	config, err := terraform.GenerateConfig(app, resources, app.Provider)
	if err != nil {
		return "", fmt.Errorf("generate terraform: %w", err)
	}

	return config, nil
}

// DeployInfrastructure generates Terraform for the application, applies it via
// the appropriate provider adapter, and creates a deployment record.
func (s *InfraService) DeployInfrastructure(ctx context.Context, appID uuid.UUID, gitCommit, gitBranch string) (domain.Deployment, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("get application: %w", err)
	}

	// Generate Terraform config
	resources, err := s.resources.ListByApplicationID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("list resources: %w", err)
	}

	config, err := terraform.GenerateConfig(app, resources, app.Provider)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("generate terraform: %w", err)
	}

	// Create deployment record
	d := domain.NewDeployment(appID, app.Provider, gitCommit, gitBranch, nil)
	d.TerraformPlan = config
	if err := d.Validate(); err != nil {
		return domain.Deployment{}, err
	}

	if err := s.deployments.Create(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("create deployment: %w", err)
	}

	// Apply via provider adapter
	adapter, err := s.providers.Get(app.Provider)
	if err != nil {
		return s.markFailed(ctx, d, fmt.Sprintf("provider not available: %s", err))
	}

	planOutput, err := adapter.ApplyTerraform(ctx, config)
	if err != nil {
		return s.markFailed(ctx, d, fmt.Sprintf("terraform apply failed: %s", err))
	}

	// Mark succeeded
	d.Status = domain.DeploymentSucceeded
	d.TerraformPlan = planOutput
	now := d.StartedAt // use a relative time
	d.CompletedAt = &now
	if err := s.deployments.Update(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("update deployment: %w", err)
	}

	return d, nil
}

// ValidateProvider checks whether the provider adapter has valid credentials.
func (s *InfraService) ValidateProvider(ctx context.Context, providerName domain.CloudProvider) error {
	adapter, err := s.providers.Get(providerName)
	if err != nil {
		return err
	}
	return adapter.ValidateCredentials(ctx)
}

// DestroyInfrastructure destroys the infrastructure for a deployment.
func (s *InfraService) DestroyInfrastructure(ctx context.Context, deploymentID uuid.UUID) error {
	d, err := s.deployments.GetByID(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("get deployment: %w", err)
	}

	if d.TerraformPlan == "" {
		return fmt.Errorf("deployment has no terraform plan to destroy")
	}

	adapter, err := s.providers.Get(d.Provider)
	if err != nil {
		return fmt.Errorf("provider not available: %w", err)
	}

	return adapter.DestroyTerraform(ctx, d.TerraformPlan)
}

func (s *InfraService) markFailed(ctx context.Context, d domain.Deployment, reason string) (domain.Deployment, error) {
	d.Status = domain.DeploymentFailed
	now := d.StartedAt
	d.CompletedAt = &now
	d.TerraformPlan = reason
	_ = s.deployments.Update(ctx, d)
	return d, fmt.Errorf("%s", reason)
}
