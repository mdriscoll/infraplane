package mcp

import (
	"context"
	"encoding/json"
	"testing"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// setupTestHandlers creates ToolHandlers backed by mock repos and mock LLM.
func setupTestHandlers() *ToolHandlers {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	depRepo := mock.NewDeploymentRepo()
	planRepo := mock.NewPlanRepo()
	mockLLM := &llm.MockClient{}

	graphRepo := mock.NewGraphRepo()

	appSvc := service.NewApplicationService(appRepo, resRepo, mockLLM, nil)
	resSvc := service.NewResourceService(resRepo, appRepo, mockLLM, nil)
	planSvc := service.NewPlannerService(planRepo, appRepo, resRepo, mockLLM, nil)
	depSvc := service.NewDeploymentService(depRepo, appRepo)
	graphSvc := service.NewGraphService(graphRepo, appRepo, resRepo, mockLLM)
	discSvc := service.NewDiscoveryService(appRepo, mockLLM, nil)

	return NewToolHandlers(appSvc, resSvc, planSvc, depSvc, graphSvc, discSvc, nil)
}

func makeRequest(args map[string]any) gomcp.CallToolRequest {
	return gomcp.CallToolRequest{
		Params: gomcp.CallToolParams{
			Arguments: args,
		},
	}
}

func TestHandleRegisterApplication(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		result, err := h.handleRegisterApplication(ctx, makeRequest(map[string]any{
			"name":     "my-api",
			"description": "A REST API",
			"provider": "aws",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected tool error: %s", result.Content[0].(gomcp.TextContent).Text)
		}

		// Parse the JSON response
		text := result.Content[0].(gomcp.TextContent).Text
		var resp map[string]any
		if err := json.Unmarshal([]byte(text), &resp); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		if resp["name"] != "my-api" {
			t.Errorf("name = %v, want my-api", resp["name"])
		}
	})

	t.Run("missing name", func(t *testing.T) {
		result, err := h.handleRegisterApplication(ctx, makeRequest(map[string]any{
			"provider": "aws",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !result.IsError {
			t.Error("expected tool error for missing name")
		}
	})
}

func TestHandleListApplications(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	// Register two apps
	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "app-1", "provider": "aws",
	}))
	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "app-2", "provider": "gcp",
	}))

	result, err := h.handleListApplications(ctx, makeRequest(nil))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error")
	}

	text := result.Content[0].(gomcp.TextContent).Text
	var resp map[string]any
	json.Unmarshal([]byte(text), &resp)

	count := resp["count"].(float64)
	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestHandleGetApplication(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "get-test", "provider": "aws", "description": "Test app",
	}))

	t.Run("found", func(t *testing.T) {
		result, err := h.handleGetApplication(ctx, makeRequest(map[string]any{
			"name": "get-test",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
		}

		text := result.Content[0].(gomcp.TextContent).Text
		var resp map[string]any
		json.Unmarshal([]byte(text), &resp)
		app := resp["application"].(map[string]any)
		if app["name"] != "get-test" {
			t.Errorf("name = %v, want get-test", app["name"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		result, _ := h.handleGetApplication(ctx, makeRequest(map[string]any{
			"name": "nonexistent",
		}))
		if !result.IsError {
			t.Error("expected tool error")
		}
	})
}

func TestHandleAddResource(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "resource-app", "provider": "aws",
	}))

	t.Run("successful add", func(t *testing.T) {
		result, err := h.handleAddResource(ctx, makeRequest(map[string]any{
			"app_name":    "resource-app",
			"description": "I need a PostgreSQL database for user data",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
		}

		text := result.Content[0].(gomcp.TextContent).Text
		var resp map[string]any
		json.Unmarshal([]byte(text), &resp)
		if resp["message"] == nil {
			t.Error("expected message in response")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		result, _ := h.handleAddResource(ctx, makeRequest(map[string]any{
			"app_name":    "nonexistent",
			"description": "some resource",
		}))
		if !result.IsError {
			t.Error("expected tool error")
		}
	})
}

func TestHandleRemoveResource(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "remove-app", "provider": "aws",
	}))

	// Add a resource
	addResult, _ := h.handleAddResource(ctx, makeRequest(map[string]any{
		"app_name":    "remove-app",
		"description": "a database",
	}))
	text := addResult.Content[0].(gomcp.TextContent).Text
	var addResp map[string]any
	json.Unmarshal([]byte(text), &addResp)
	resource := addResp["resource"].(map[string]any)
	resourceID := resource["id"].(string)

	t.Run("successful remove", func(t *testing.T) {
		result, err := h.handleRemoveResource(ctx, makeRequest(map[string]any{
			"resource_id": resourceID,
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
		}
	})

	t.Run("not found", func(t *testing.T) {
		result, _ := h.handleRemoveResource(ctx, makeRequest(map[string]any{
			"resource_id": "00000000-0000-0000-0000-000000000000",
		}))
		if !result.IsError {
			t.Error("expected tool error")
		}
	})
}

func TestHandleDeploy(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "deploy-app", "provider": "gcp",
	}))

	t.Run("successful deploy", func(t *testing.T) {
		result, err := h.handleDeploy(ctx, makeRequest(map[string]any{
			"app_name":   "deploy-app",
			"git_branch": "main",
			"git_commit": "abc123",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
		}

		text := result.Content[0].(gomcp.TextContent).Text
		var resp map[string]any
		json.Unmarshal([]byte(text), &resp)
		if resp["status"] != string(domain.DeploymentPending) {
			t.Errorf("status = %v, want pending", resp["status"])
		}
		if resp["provider"] != "gcp" {
			t.Errorf("provider = %v, want gcp", resp["provider"])
		}
	})
}

func TestHandleGetHostingPlan(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "plan-app", "provider": "aws",
	}))

	result, err := h.handleGetHostingPlan(ctx, makeRequest(map[string]any{
		"app_name": "plan-app",
	}))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
	}

	text := result.Content[0].(gomcp.TextContent).Text
	var resp map[string]any
	json.Unmarshal([]byte(text), &resp)
	if resp["content"] == nil || resp["content"] == "" {
		t.Error("expected content in hosting plan")
	}
	if resp["estimated_cost"] == nil {
		t.Error("expected estimated_cost in hosting plan")
	}
}

func TestHandlePlanMigration(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "migrate-app", "provider": "aws",
	}))

	t.Run("successful migration plan", func(t *testing.T) {
		result, err := h.handlePlanMigration(ctx, makeRequest(map[string]any{
			"app_name":      "migrate-app",
			"from_provider": "aws",
			"to_provider":   "gcp",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
		}
	})

	t.Run("same provider error", func(t *testing.T) {
		result, _ := h.handlePlanMigration(ctx, makeRequest(map[string]any{
			"app_name":      "migrate-app",
			"from_provider": "aws",
			"to_provider":   "aws",
		}))
		if !result.IsError {
			t.Error("expected tool error for same provider")
		}
	})
}

func TestHandleGetDeploymentStatus(t *testing.T) {
	h := setupTestHandlers()
	ctx := context.Background()

	h.handleRegisterApplication(ctx, makeRequest(map[string]any{
		"name": "status-app", "provider": "aws",
	}))
	h.handleDeploy(ctx, makeRequest(map[string]any{
		"app_name": "status-app", "git_branch": "main", "git_commit": "xyz789",
	}))

	t.Run("get latest deployment", func(t *testing.T) {
		result, err := h.handleGetDeploymentStatus(ctx, makeRequest(map[string]any{
			"app_name": "status-app",
		}))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.IsError {
			t.Fatalf("unexpected error: %s", result.Content[0].(gomcp.TextContent).Text)
		}
	})

	t.Run("no deployments", func(t *testing.T) {
		h.handleRegisterApplication(ctx, makeRequest(map[string]any{
			"name": "no-deploys", "provider": "aws",
		}))
		result, _ := h.handleGetDeploymentStatus(ctx, makeRequest(map[string]any{
			"app_name": "no-deploys",
		}))
		if !result.IsError {
			t.Error("expected error for no deployments")
		}
	})
}
