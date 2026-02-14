package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// ToolHandlers holds references to all services and provides MCP tool handlers.
type ToolHandlers struct {
	apps        *service.ApplicationService
	resources   *service.ResourceService
	planner     *service.PlannerService
	deployments *service.DeploymentService
}

// NewToolHandlers creates a new ToolHandlers.
func NewToolHandlers(
	apps *service.ApplicationService,
	resources *service.ResourceService,
	planner *service.PlannerService,
	deployments *service.DeploymentService,
) *ToolHandlers {
	return &ToolHandlers{
		apps:        apps,
		resources:   resources,
		planner:     planner,
		deployments: deployments,
	}
}

// RegisterAll registers all tools on the MCP server.
func (h *ToolHandlers) RegisterAll(s *server.MCPServer) {
	s.AddTool(registerApplicationTool(), h.handleRegisterApplication)
	s.AddTool(listApplicationsTool(), h.handleListApplications)
	s.AddTool(getApplicationTool(), h.handleGetApplication)
	s.AddTool(addResourceTool(), h.handleAddResource)
	s.AddTool(removeResourceTool(), h.handleRemoveResource)
	s.AddTool(getHostingPlanTool(), h.handleGetHostingPlan)
	s.AddTool(planMigrationTool(), h.handlePlanMigration)
	s.AddTool(deployTool(), h.handleDeploy)
	s.AddTool(getDeploymentStatusTool(), h.handleGetDeploymentStatus)
}

// --- Tool Definitions ---

func registerApplicationTool() gomcp.Tool {
	return gomcp.NewTool("register_application",
		gomcp.WithDescription("Register a new application in Infraplane. Provide a source path (local directory or git URL) to auto-detect infrastructure resources from the codebase."),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Application name (must be unique, kebab-case recommended)")),
		gomcp.WithString("description", gomcp.Description("Brief description of what the application does")),
		gomcp.WithString("git_repo_url", gomcp.Description("Git repository URL (e.g. https://github.com/org/repo)")),
		gomcp.WithString("source_path", gomcp.Description("Local filesystem path or git URL to analyze for auto-detecting infrastructure resources (e.g. '/path/to/project' or 'https://github.com/org/repo')")),
		gomcp.WithString("provider", gomcp.Required(), gomcp.Description("Preferred cloud provider"), gomcp.Enum("aws", "gcp")),
	)
}

func listApplicationsTool() gomcp.Tool {
	return gomcp.NewTool("list_applications",
		gomcp.WithDescription("List all applications registered in Infraplane with their current status and provider."),
	)
}

func getApplicationTool() gomcp.Tool {
	return gomcp.NewTool("get_application",
		gomcp.WithDescription("Get detailed information about an application, including all its resources and latest deployment."),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Application name")),
	)
}

func addResourceTool() gomcp.Tool {
	return gomcp.NewTool("add_resource",
		gomcp.WithDescription("Add a cloud resource to an application by describing what you need in natural language. The LLM will analyze your description and create a cloud-agnostic resource with provider-specific mappings and Terraform."),
		gomcp.WithString("app_name", gomcp.Required(), gomcp.Description("Application name to add the resource to")),
		gomcp.WithString("description", gomcp.Required(), gomcp.Description("Natural language description of what you need (e.g. 'a PostgreSQL database for user data', 'a Redis cache for sessions', 'object storage for file uploads')")),
	)
}

func removeResourceTool() gomcp.Tool {
	return gomcp.NewTool("remove_resource",
		gomcp.WithDescription("Remove a resource from an application."),
		gomcp.WithString("resource_id", gomcp.Required(), gomcp.Description("UUID of the resource to remove")),
	)
}

func getHostingPlanTool() gomcp.Tool {
	return gomcp.NewTool("get_hosting_plan",
		gomcp.WithDescription("Generate an LLM-powered hosting plan for an application. Analyzes all resources and recommends optimal deployment architecture with cost estimates."),
		gomcp.WithString("app_name", gomcp.Required(), gomcp.Description("Application name")),
	)
}

func planMigrationTool() gomcp.Tool {
	return gomcp.NewTool("plan_migration",
		gomcp.WithDescription("Generate an LLM-powered migration plan to move an application from one cloud provider to another. Includes service mapping, data migration strategy, and new Terraform configurations."),
		gomcp.WithString("app_name", gomcp.Required(), gomcp.Description("Application name")),
		gomcp.WithString("from_provider", gomcp.Required(), gomcp.Description("Source cloud provider"), gomcp.Enum("aws", "gcp")),
		gomcp.WithString("to_provider", gomcp.Required(), gomcp.Description("Target cloud provider"), gomcp.Enum("aws", "gcp")),
	)
}

func deployTool() gomcp.Tool {
	return gomcp.NewTool("deploy",
		gomcp.WithDescription("Trigger a deployment for an application from a git branch and commit."),
		gomcp.WithString("app_name", gomcp.Required(), gomcp.Description("Application name")),
		gomcp.WithString("git_branch", gomcp.Required(), gomcp.Description("Git branch to deploy from (e.g. 'main')")),
		gomcp.WithString("git_commit", gomcp.Description("Git commit SHA (optional, defaults to latest)")),
	)
}

func getDeploymentStatusTool() gomcp.Tool {
	return gomcp.NewTool("get_deployment_status",
		gomcp.WithDescription("Check the status of a deployment or get the latest deployment for an application."),
		gomcp.WithString("app_name", gomcp.Required(), gomcp.Description("Application name")),
		gomcp.WithString("deployment_id", gomcp.Description("Specific deployment UUID (if omitted, returns the latest deployment)")),
	)
}

// --- Tool Handlers ---

func (h *ToolHandlers) handleRegisterApplication(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	name, _ := req.RequireString("name")
	description := req.GetString("description", "")
	gitRepoURL := req.GetString("git_repo_url", "")
	provider, _ := req.RequireString("provider")

	sourcePath := req.GetString("source_path", "")
	app, err := h.apps.Register(ctx, name, description, gitRepoURL, sourcePath, domain.CloudProvider(provider))
	if err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]any{
		"id":       app.ID,
		"name":     app.Name,
		"provider": app.Provider,
		"status":   app.Status,
		"message":  fmt.Sprintf("Application '%s' registered successfully on %s.", app.Name, app.Provider),
	})
}

func (h *ToolHandlers) handleListApplications(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	apps, err := h.apps.List(ctx)
	if err != nil {
		return toolError(err), nil
	}

	type appSummary struct {
		ID       uuid.UUID           `json:"id"`
		Name     string              `json:"name"`
		Provider domain.CloudProvider `json:"provider"`
		Status   domain.AppStatus    `json:"status"`
	}

	summaries := make([]appSummary, len(apps))
	for i, app := range apps {
		summaries[i] = appSummary{
			ID:       app.ID,
			Name:     app.Name,
			Provider: app.Provider,
			Status:   app.Status,
		}
	}

	return toolJSON(map[string]any{
		"applications": summaries,
		"count":        len(summaries),
	})
}

func (h *ToolHandlers) handleGetApplication(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	name, _ := req.RequireString("name")

	app, err := h.apps.GetByName(ctx, name)
	if err != nil {
		return toolError(err), nil
	}

	resources, err := h.resources.ListByApplication(ctx, app.ID)
	if err != nil {
		return toolError(err), nil
	}

	latest, latestErr := h.deployments.GetLatest(ctx, app.ID)

	result := map[string]any{
		"application": app,
		"resources":   resources,
	}
	if latestErr == nil {
		result["latest_deployment"] = latest
	}

	return toolJSON(result)
}

func (h *ToolHandlers) handleAddResource(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	appName, _ := req.RequireString("app_name")
	description, _ := req.RequireString("description")

	app, err := h.apps.GetByName(ctx, appName)
	if err != nil {
		return toolError(fmt.Errorf("application '%s' not found", appName)), nil
	}

	resource, err := h.resources.AddFromDescription(ctx, app.ID, description)
	if err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]any{
		"resource": resource,
		"message":  fmt.Sprintf("Added %s resource '%s' to application '%s'.", resource.Kind, resource.Name, appName),
	})
}

func (h *ToolHandlers) handleRemoveResource(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	resourceIDStr, _ := req.RequireString("resource_id")
	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		return toolError(fmt.Errorf("invalid resource ID: %s", resourceIDStr)), nil
	}

	if err := h.resources.Remove(ctx, resourceID); err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]any{
		"message": "Resource removed successfully.",
	})
}

func (h *ToolHandlers) handleGetHostingPlan(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	appName, _ := req.RequireString("app_name")

	app, err := h.apps.GetByName(ctx, appName)
	if err != nil {
		return toolError(fmt.Errorf("application '%s' not found", appName)), nil
	}

	plan, err := h.planner.GenerateHostingPlan(ctx, app.ID)
	if err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]any{
		"plan_id":        plan.ID,
		"content":        plan.Content,
		"estimated_cost": plan.EstimatedCost,
	})
}

func (h *ToolHandlers) handlePlanMigration(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	appName, _ := req.RequireString("app_name")
	fromProvider, _ := req.RequireString("from_provider")
	toProvider, _ := req.RequireString("to_provider")

	app, err := h.apps.GetByName(ctx, appName)
	if err != nil {
		return toolError(fmt.Errorf("application '%s' not found", appName)), nil
	}

	plan, err := h.planner.GenerateMigrationPlan(ctx, app.ID, domain.CloudProvider(fromProvider), domain.CloudProvider(toProvider))
	if err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]any{
		"plan_id":        plan.ID,
		"from_provider":  plan.FromProvider,
		"to_provider":    plan.ToProvider,
		"content":        plan.Content,
		"estimated_cost": plan.EstimatedCost,
	})
}

func (h *ToolHandlers) handleDeploy(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	appName, _ := req.RequireString("app_name")
	gitBranch, _ := req.RequireString("git_branch")
	gitCommit := req.GetString("git_commit", "")

	app, err := h.apps.GetByName(ctx, appName)
	if err != nil {
		return toolError(fmt.Errorf("application '%s' not found", appName)), nil
	}

	d, err := h.deployments.Deploy(ctx, app.ID, gitCommit, gitBranch)
	if err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]any{
		"deployment_id": d.ID,
		"status":        d.Status,
		"provider":      d.Provider,
		"git_branch":    d.GitBranch,
		"git_commit":    d.GitCommit,
		"message":       fmt.Sprintf("Deployment started for '%s' from branch '%s'.", appName, gitBranch),
	})
}

func (h *ToolHandlers) handleGetDeploymentStatus(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	appName, _ := req.RequireString("app_name")
	deploymentIDStr := req.GetString("deployment_id", "")

	if deploymentIDStr != "" {
		deploymentID, err := uuid.Parse(deploymentIDStr)
		if err != nil {
			return toolError(fmt.Errorf("invalid deployment ID: %s", deploymentIDStr)), nil
		}
		d, err := h.deployments.GetStatus(ctx, deploymentID)
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(d)
	}

	// Get latest deployment
	app, err := h.apps.GetByName(ctx, appName)
	if err != nil {
		return toolError(fmt.Errorf("application '%s' not found", appName)), nil
	}

	d, err := h.deployments.GetLatest(ctx, app.ID)
	if err != nil {
		return toolError(fmt.Errorf("no deployments found for '%s'", appName)), nil
	}

	return toolJSON(d)
}

// --- Helper Functions ---

func toolJSON(data any) (*gomcp.CallToolResult, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return toolError(fmt.Errorf("marshal response: %w", err)), nil
	}
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			gomcp.TextContent{
				Type: "text",
				Text: string(jsonBytes),
			},
		},
	}, nil
}

func toolError(err error) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			gomcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error: %s", err.Error()),
			},
		},
		IsError: true,
	}
}
