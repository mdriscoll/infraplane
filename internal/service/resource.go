package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// ResourceService handles resource management with LLM-powered analysis.
type ResourceService struct {
	resources repository.ResourceRepo
	apps      repository.ApplicationRepo
	llm       llm.Client
}

// NewResourceService creates a new ResourceService.
func NewResourceService(resources repository.ResourceRepo, apps repository.ApplicationRepo, llmClient llm.Client) *ResourceService {
	return &ResourceService{
		resources: resources,
		apps:      apps,
		llm:       llmClient,
	}
}

// AddFromDescription uses the LLM to analyze a natural language description
// and create a cloud-agnostic resource with provider mappings.
func (s *ResourceService) AddFromDescription(ctx context.Context, appID uuid.UUID, description string) (domain.Resource, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("get application: %w", err)
	}

	rec, err := s.llm.AnalyzeResourceNeed(ctx, description, app.Provider)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("analyze resource: %w", err)
	}

	resource := domain.NewResource(appID, rec.Kind, rec.Name, rec.Spec)
	resource.ProviderMappings = rec.Mappings

	if err := resource.Validate(); err != nil {
		return domain.Resource{}, err
	}

	if err := s.resources.Create(ctx, resource); err != nil {
		return domain.Resource{}, fmt.Errorf("create resource: %w", err)
	}

	return resource, nil
}

// Get returns a resource by ID.
func (s *ResourceService) Get(ctx context.Context, id uuid.UUID) (domain.Resource, error) {
	return s.resources.GetByID(ctx, id)
}

// ListByApplication returns all resources for an application.
func (s *ResourceService) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Resource, error) {
	return s.resources.ListByApplicationID(ctx, appID)
}

// Remove deletes a resource.
func (s *ResourceService) Remove(ctx context.Context, id uuid.UUID) error {
	return s.resources.Delete(ctx, id)
}

// GenerateTerraformHCL generates Terraform HCL for a single resource using the LLM.
func (s *ResourceService) GenerateTerraformHCL(ctx context.Context, resourceID uuid.UUID, provider domain.CloudProvider) (string, error) {
	resource, err := s.resources.GetByID(ctx, resourceID)
	if err != nil {
		return "", fmt.Errorf("get resource: %w", err)
	}

	result, err := s.llm.GenerateTerraformHCL(ctx, resource, provider)
	if err != nil {
		return "", fmt.Errorf("generate terraform HCL: %w", err)
	}

	return result.HCL, nil
}
