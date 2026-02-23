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
// on its configured provider using a single LLM call that produces a cohesive config
// with all resources properly wired together.
func (s *InfraService) GenerateTerraform(ctx context.Context, appID uuid.UUID, target *domain.DeployTarget) (string, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListCurrentByApplicationID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("list resources: %w", err)
	}

	// Build compliance context across all resources
	complianceContext := s.buildComplianceContext(app, resources)

	log.Printf("[infra] generating full Terraform config for %s (%d resources) on %s via single LLM call",
		app.Name, len(resources), app.Provider)

	result, err := s.llm.GenerateFullTerraform(ctx, app, resources, app.Provider, target, complianceContext)
	if err != nil {
		return "", fmt.Errorf("generate full terraform: %w", err)
	}

	// Safety net: deduplicate any duplicate blocks the LLM may have produced
	return terraform.DeduplicateHCL(result.HCL), nil
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
// also returns per-resource HCL details for the review UI. Uses a single LLM call
// that produces a cohesive config with all resources properly wired together.
func (s *InfraService) GenerateTerraformWithResources(ctx context.Context, appID uuid.UUID, target *domain.DeployTarget) (string, []ResourceHCL, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return "", nil, fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListCurrentByApplicationID(ctx, appID)
	if err != nil {
		return "", nil, fmt.Errorf("list resources: %w", err)
	}

	// Build compliance context across all resources
	complianceContext := s.buildComplianceContext(app, resources)

	log.Printf("[infra] generating full Terraform config with resource breakdown for %s (%d resources) on %s",
		app.Name, len(resources), app.Provider)

	result, err := s.llm.GenerateFullTerraform(ctx, app, resources, app.Provider, target, complianceContext)
	if err != nil {
		return "", nil, fmt.Errorf("generate full terraform: %w", err)
	}

	// Build a lookup from resource name to resource ID for the review UI
	resourceIDByName := make(map[string]uuid.UUID)
	for _, r := range resources {
		resourceIDByName[r.Name] = r.ID
	}

	// Map the LLM's per-resource breakdown to our ResourceHCL format
	var resourceHCLs []ResourceHCL
	for _, lr := range result.Resources {
		resourceHCLs = append(resourceHCLs, ResourceHCL{
			ResourceID:   resourceIDByName[lr.ResourceName],
			ResourceName: lr.ResourceName,
			ResourceKind: lr.ResourceKind,
			ServiceName:  lr.ServiceName,
			HCL:          lr.HCL,
		})
	}

	// Safety net: deduplicate any duplicate blocks the LLM may have produced
	return terraform.DeduplicateHCL(result.HCL), resourceHCLs, nil
}

// buildComplianceContext collects compliance rules for all resources in an application
// and returns a formatted string for the LLM prompt.
func (s *InfraService) buildComplianceContext(app domain.Application, resources []domain.Resource) string {
	if s.compliance == nil || len(app.ComplianceFrameworks) == 0 {
		return ""
	}

	var allRules []compliance.Rule
	seen := make(map[string]bool)

	for _, r := range resources {
		mapping, ok := r.ProviderMappings[app.Provider]
		if !ok {
			continue
		}
		rules := s.compliance.GetRulesForResource(
			app.ComplianceFrameworks,
			app.Provider,
			r.Kind,
			mapping.ServiceName,
		)
		for _, rule := range rules {
			key := rule.ID
			if !seen[key] {
				seen[key] = true
				allRules = append(allRules, rule)
			}
		}
	}

	if len(allRules) == 0 {
		return ""
	}

	return s.compliance.FormatRulesForPrompt(allRules)
}

// DeployInfrastructure generates Terraform for the application using a single LLM call,
// applies it via the appropriate provider adapter, and creates a deployment record.
func (s *InfraService) DeployInfrastructure(ctx context.Context, appID uuid.UUID, gitCommit, gitBranch string, target *domain.DeployTarget) (domain.Deployment, error) {
	// Use GenerateTerraform (single LLM call) to produce the full config
	config, err := s.GenerateTerraform(ctx, appID, target)
	if err != nil {
		return domain.Deployment{}, err
	}

	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("get application: %w", err)
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
