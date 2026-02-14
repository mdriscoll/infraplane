package llm

import (
	"context"
	"encoding/json"

	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// ResourceRecommendation is the LLM's output when analyzing a resource need.
type ResourceRecommendation struct {
	Kind     domain.ResourceKind                          `json:"kind"`
	Name     string                                      `json:"name"`
	Spec     json.RawMessage                              `json:"spec"`
	Mappings map[domain.CloudProvider]domain.ProviderResource `json:"mappings"`
}

// HostingPlanResult is the LLM's output for a hosting plan.
type HostingPlanResult struct {
	Content       string              `json:"content"`
	EstimatedCost *domain.CostEstimate `json:"estimated_cost,omitempty"`
}

// MigrationPlanResult is the LLM's output for a migration plan.
type MigrationPlanResult struct {
	Content       string              `json:"content"`
	EstimatedCost *domain.CostEstimate `json:"estimated_cost,omitempty"`
}

// Client defines the interface for LLM-powered reasoning operations.
// All methods accept context for cancellation and return structured results.
type Client interface {
	// AnalyzeResourceNeed interprets a natural language resource description
	// and returns a structured resource recommendation with provider mappings.
	AnalyzeResourceNeed(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error)

	// AnalyzeCodebase examines extracted code files from a project and recommends
	// infrastructure resources based on dependencies, configs, and deploy scripts.
	AnalyzeCodebase(ctx context.Context, codeCtx analyzer.CodeContext, provider domain.CloudProvider) ([]ResourceRecommendation, error)

	// GenerateHostingPlan analyzes an application's resources and generates
	// a hosting strategy with cost estimates.
	GenerateHostingPlan(ctx context.Context, app domain.Application, resources []domain.Resource) (HostingPlanResult, error)

	// GenerateMigrationPlan creates a migration plan to move an application
	// from one cloud provider to another.
	GenerateMigrationPlan(ctx context.Context, app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) (MigrationPlanResult, error)
}
