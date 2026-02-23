package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/compliance"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
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
	llm         llm.Client
	compliance  *compliance.Registry
}

// NewInfraService creates a new InfraService.
func NewInfraService(
	apps repository.ApplicationRepo,
	resources repository.ResourceRepo,
	deployments repository.DeploymentRepo,
	providers *provider.Registry,
	llmClient llm.Client,
	complianceRegistry *compliance.Registry,
) *InfraService {
	return &InfraService{
		apps:        apps,
		resources:   resources,
		deployments: deployments,
		providers:   providers,
		llm:         llmClient,
		compliance:  complianceRegistry,
	}
}

// Apps returns the application repository used by this service.
func (s *InfraService) Apps() repository.ApplicationRepo { return s.apps }

// Providers returns the provider registry used by this service.
func (s *InfraService) Providers() *provider.Registry { return s.providers }

// GenerateTerraform generates a complete Terraform configuration for an application
// on its configured provider. If target is provided, the provider block uses
// target-specific configuration (region, project, etc.).
// For any resource missing pre-generated TerraformHCL, it calls the LLM to generate it.
func (s *InfraService) GenerateTerraform(ctx context.Context, appID uuid.UUID, target *domain.DeployTarget) (string, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListCurrentByApplicationID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("list resources: %w", err)
	}

	// Generate HCL for any resource that's missing it
	resources, err = s.ensureHCL(ctx, app, resources)
	if err != nil {
		return "", fmt.Errorf("ensure terraform HCL: %w", err)
	}

	config, err := terraform.GenerateConfig(app, resources, app.Provider, target)
	if err != nil {
		return "", fmt.Errorf("generate terraform: %w", err)
	}

	return config, nil
}

// ResourceHCL holds per-resource Terraform HCL details for the review UI.
type ResourceHCL struct {
	ResourceID   uuid.UUID `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	ResourceKind string    `json:"resource_kind"`
	ServiceName  string    `json:"service_name"`
	HCL          string    `json:"hcl"`
}

// GenerateTerraformWithResources generates a complete Terraform configuration and
// also returns per-resource HCL details for the review UI.
func (s *InfraService) GenerateTerraformWithResources(ctx context.Context, appID uuid.UUID, target *domain.DeployTarget) (string, []ResourceHCL, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return "", nil, fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListCurrentByApplicationID(ctx, appID)
	if err != nil {
		return "", nil, fmt.Errorf("list resources: %w", err)
	}

	// Generate HCL for any resource that's missing it
	resources, err = s.ensureHCL(ctx, app, resources)
	if err != nil {
		return "", nil, fmt.Errorf("ensure terraform HCL: %w", err)
	}

	// Collect per-resource HCL for the review UI
	var resourceHCLs []ResourceHCL
	for _, r := range resources {
		mapping, ok := r.ProviderMappings[app.Provider]
		if !ok || mapping.TerraformHCL == "" {
			continue
		}
		resourceHCLs = append(resourceHCLs, ResourceHCL{
			ResourceID:   r.ID,
			ResourceName: r.Name,
			ResourceKind: string(r.Kind),
			ServiceName:  mapping.ServiceName,
			HCL:          mapping.TerraformHCL,
		})
	}

	config, err := terraform.GenerateConfig(app, resources, app.Provider, target)
	if err != nil {
		return "", nil, fmt.Errorf("generate terraform: %w", err)
	}

	return config, resourceHCLs, nil
}

// ensureHCL iterates over resources and generates Terraform HCL via the LLM
// for any resource that has a provider mapping but empty TerraformHCL.
// Generated HCL is persisted back to the resource so future deploys skip LLM calls.
func (s *InfraService) ensureHCL(ctx context.Context, app domain.Application, resources []domain.Resource) ([]domain.Resource, error) {
	provider := app.Provider

	for i, r := range resources {
		mapping, ok := r.ProviderMappings[provider]
		if !ok {
			continue
		}
		if mapping.TerraformHCL != "" {
			continue
		}

		// Build compliance context for this resource
		var complianceContext string
		if s.compliance != nil && len(app.ComplianceFrameworks) > 0 {
			rules := s.compliance.GetRulesForResource(
				app.ComplianceFrameworks,
				provider,
				r.Kind,
				mapping.ServiceName,
			)
			if len(rules) > 0 {
				complianceContext = s.compliance.FormatRulesForPrompt(rules)
			}
		}

		log.Printf("[infra] generating Terraform HCL for resource %s (%s) on %s", r.Name, r.Kind, provider)
		result, err := s.llm.GenerateTerraformHCL(ctx, r, provider, complianceContext)
		if err != nil {
			return nil, fmt.Errorf("generate HCL for %s: %w", r.Name, err)
		}

		// Update the mapping with generated HCL
		mapping.TerraformHCL = result.HCL
		r.ProviderMappings[provider] = mapping
		resources[i] = r

		// Persist back to DB so future deploys skip this
		if err := s.resources.Update(ctx, r); err != nil {
			log.Printf("[infra] warning: failed to persist HCL for resource %s: %v", r.Name, err)
		}
	}

	return resources, nil
}

// DeployInfrastructure generates Terraform for the application, applies it via
// the appropriate provider adapter, and creates a deployment record.
func (s *InfraService) DeployInfrastructure(ctx context.Context, appID uuid.UUID, gitCommit, gitBranch string, target *domain.DeployTarget) (domain.Deployment, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("get application: %w", err)
	}

	// Generate Terraform config
	resources, err := s.resources.ListCurrentByApplicationID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("list resources: %w", err)
	}

	config, err := terraform.GenerateConfig(app, resources, app.Provider, target)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("generate terraform: %w", err)
	}

	// Create deployment record
	d := domain.NewDeployment(appID, app.Provider, gitCommit, gitBranch, nil, target)
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

	planOutput, err := adapter.ApplyTerraform(ctx, config, target, nil)
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

// ValidateProvider checks whether the provider adapter has valid credentials for a target.
func (s *InfraService) ValidateProvider(ctx context.Context, providerName domain.CloudProvider, target *domain.DeployTarget) error {
	adapter, err := s.providers.Get(providerName)
	if err != nil {
		return err
	}
	return adapter.ValidateCredentials(ctx, target)
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

	return adapter.DestroyTerraform(ctx, d.TerraformPlan, d.DeployTarget)
}

func (s *InfraService) markFailed(ctx context.Context, d domain.Deployment, reason string) (domain.Deployment, error) {
	d.Status = domain.DeploymentFailed
	now := d.StartedAt
	d.CompletedAt = &now
	d.TerraformPlan = reason
	_ = s.deployments.Update(ctx, d)
	return d, fmt.Errorf("%s", reason)
}
