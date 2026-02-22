package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/compliance"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// PlannerService handles LLM-powered hosting and migration planning.
type PlannerService struct {
	plans      repository.PlanRepo
	apps       repository.ApplicationRepo
	resources  repository.ResourceRepo
	llm        llm.Client
	compliance *compliance.Registry
}

// NewPlannerService creates a new PlannerService.
func NewPlannerService(plans repository.PlanRepo, apps repository.ApplicationRepo, resources repository.ResourceRepo, llmClient llm.Client, complianceRegistry *compliance.Registry) *PlannerService {
	return &PlannerService{
		plans:      plans,
		apps:       apps,
		resources:  resources,
		llm:        llmClient,
		compliance: complianceRegistry,
	}
}

// GenerateHostingPlan creates an LLM-powered hosting recommendation.
func (s *PlannerService) GenerateHostingPlan(ctx context.Context, appID uuid.UUID) (domain.InfrastructurePlan, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListByApplicationID(ctx, appID)
	if err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("list resources: %w", err)
	}

	// Build compliance context filtered to only the resource kinds this app actually has.
	// This avoids bloating the prompt with irrelevant rules (e.g. Compute Engine VM rules
	// when the app only uses Cloud Run + Cloud SQL).
	var complianceContext string
	if len(app.ComplianceFrameworks) > 0 && s.compliance != nil {
		// Collect unique resource kinds present in this application
		kindSet := make(map[domain.ResourceKind]bool, len(resources))
		for _, r := range resources {
			kindSet[r.Kind] = true
		}

		var filteredRules []compliance.Rule
		allRules := s.compliance.GetRules(app.ComplianceFrameworks)
		for _, rule := range allRules {
			// Keep the rule if ANY of its target resource kinds matches an app resource
			for _, rk := range rule.ResourceKinds {
				if kindSet[rk] {
					filteredRules = append(filteredRules, rule)
					break
				}
			}
		}

		log.Printf("[planner] compliance: %d/%d rules relevant to %d resource kinds",
			len(filteredRules), len(allRules), len(kindSet))

		if len(filteredRules) > 0 {
			complianceContext = s.compliance.FormatRulesForPrompt(filteredRules)
		}
	}

	result, err := s.llm.GenerateHostingPlan(ctx, app, resources, complianceContext)
	if err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("generate hosting plan: %w", err)
	}

	plan := domain.NewHostingPlan(appID, result.Content, resources, result.EstimatedCost)
	if err := s.plans.Create(ctx, plan); err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("save hosting plan: %w", err)
	}

	return plan, nil
}

// GenerateMigrationPlan creates an LLM-powered migration plan between providers.
func (s *PlannerService) GenerateMigrationPlan(ctx context.Context, appID uuid.UUID, from, to domain.CloudProvider) (domain.InfrastructurePlan, error) {
	if !from.IsValid() {
		return domain.InfrastructurePlan{}, domain.ErrValidation("invalid source provider: " + from.String())
	}
	if !to.IsValid() {
		return domain.InfrastructurePlan{}, domain.ErrValidation("invalid target provider: " + to.String())
	}
	if from == to {
		return domain.InfrastructurePlan{}, domain.ErrValidation("source and target providers must be different")
	}

	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListByApplicationID(ctx, appID)
	if err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("list resources: %w", err)
	}

	result, err := s.llm.GenerateMigrationPlan(ctx, app, resources, from, to)
	if err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("generate migration plan: %w", err)
	}

	plan := domain.NewMigrationPlan(appID, from, to, result.Content, resources, result.EstimatedCost)
	if err := s.plans.Create(ctx, plan); err != nil {
		return domain.InfrastructurePlan{}, fmt.Errorf("save migration plan: %w", err)
	}

	return plan, nil
}

// GetPlan returns a plan by ID.
func (s *PlannerService) GetPlan(ctx context.Context, id uuid.UUID) (domain.InfrastructurePlan, error) {
	return s.plans.GetByID(ctx, id)
}

// ListPlansByApplication returns all plans for an application.
func (s *PlannerService) ListPlansByApplication(ctx context.Context, appID uuid.UUID) ([]domain.InfrastructurePlan, error) {
	return s.plans.ListByApplicationID(ctx, appID)
}
