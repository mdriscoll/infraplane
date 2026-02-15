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

// GraphResult is the LLM's output for infrastructure topology graph generation.
type GraphResult struct {
	Nodes []domain.GraphNode `json:"nodes"`
	Edges []domain.GraphEdge `json:"edges"`
}

// TerraformHCLResult is the LLM's output for Terraform HCL generation.
type TerraformHCLResult struct {
	HCL string `json:"hcl"`
}

// DiscoveryCommand represents a single CLI command to run for resource discovery.
type DiscoveryCommand struct {
	Description  string `json:"description"`   // Human-readable: "List Cloud Run services"
	Command      string `json:"command"`        // The CLI command: "gcloud run services list ..."
	ResourceType string `json:"resource_type"`  // What this discovers: "Cloud Run Service"
}

// DiscoveryCommandResult is the LLM's output when generating CLI commands to discover live resources.
type DiscoveryCommandResult struct {
	Commands []DiscoveryCommand `json:"commands"`
}

// CommandOutput pairs a discovery command with its execution result.
type CommandOutput struct {
	Command DiscoveryCommand `json:"command"`
	Output  string           `json:"output"`
	Error   string           `json:"error,omitempty"`
}

// LiveResourceParseResult is the LLM's output when parsing CLI output into structured data.
type LiveResourceParseResult struct {
	Resources []domain.LiveResource `json:"resources"`
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

	// GenerateGraph analyzes an application's resources and produces a topology
	// graph showing how they connect to each other and the public internet.
	GenerateGraph(ctx context.Context, app domain.Application, resources []domain.Resource) (GraphResult, error)

	// GenerateTerraformHCL generates Terraform HCL for a single resource.
	GenerateTerraformHCL(ctx context.Context, resource domain.Resource, provider domain.CloudProvider) (TerraformHCLResult, error)

	// GenerateDiscoveryCommands analyzes deploy scripts and generates CLI commands
	// to discover live cloud resources for the given provider.
	GenerateDiscoveryCommands(ctx context.Context, app domain.Application, codeCtx analyzer.CodeContext) (DiscoveryCommandResult, error)

	// ParseDiscoveryOutput takes raw CLI output and parses it into structured LiveResource data.
	ParseDiscoveryOutput(ctx context.Context, app domain.Application, outputs []CommandOutput) (LiveResourceParseResult, error)
}
