package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func TestApplicationService_Register(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo, nil, nil, nil)
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		app, err := svc.Register(ctx, "my-app", "A test app", "https://github.com/test/repo", "", domain.ProviderAWS, nil, nil)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if app.Name != "my-app" {
			t.Errorf("Name = %q, want %q", app.Name, "my-app")
		}
		if app.Status != domain.AppStatusDraft {
			t.Errorf("Status = %q, want %q", app.Status, domain.AppStatusDraft)
		}
		if app.ID == uuid.Nil {
			t.Error("ID should not be nil")
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		_, err := svc.Register(ctx, "", "desc", "", "", domain.ProviderAWS, nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
		if !domain.IsValidationError(err) {
			t.Errorf("expected ValidationError, got %T", err)
		}
	})

	t.Run("validation error - invalid provider", func(t *testing.T) {
		_, err := svc.Register(ctx, "valid-name", "desc", "", "", domain.CloudProvider("azure"), nil, nil)
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, err := svc.Register(ctx, "my-app", "duplicate", "", "", domain.ProviderGCP, nil, nil)
		if err == nil {
			t.Fatal("expected conflict error")
		}
	})
}

func TestApplicationService_Get(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo, nil, nil, nil)
	ctx := context.Background()

	app, _ := svc.Register(ctx, "get-test", "desc", "", "", domain.ProviderAWS, nil, nil)

	t.Run("found", func(t *testing.T) {
		got, err := svc.Get(ctx, app.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Name != "get-test" {
			t.Errorf("Name = %q, want %q", got.Name, "get-test")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Get(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestApplicationService_GetByName(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo, nil, nil, nil)
	ctx := context.Background()

	svc.Register(ctx, "name-test", "desc", "", "", domain.ProviderGCP, nil, nil)

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByName(ctx, "name-test")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if got.Provider != domain.ProviderGCP {
			t.Errorf("Provider = %q, want %q", got.Provider, domain.ProviderGCP)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByName(ctx, "nonexistent")
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestApplicationService_List(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo, nil, nil, nil)
	ctx := context.Background()

	svc.Register(ctx, "app-1", "", "", "", domain.ProviderAWS, nil, nil)
	svc.Register(ctx, "app-2", "", "", "", domain.ProviderGCP, nil, nil)

	apps, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(apps) != 2 {
		t.Errorf("len = %d, want 2", len(apps))
	}
}

func TestApplicationService_UpdateStatus(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo, nil, nil, nil)
	ctx := context.Background()

	app, _ := svc.Register(ctx, "status-test", "", "", "", domain.ProviderAWS, nil, nil)

	t.Run("update to provisioned", func(t *testing.T) {
		updated, err := svc.UpdateStatus(ctx, app.ID, domain.AppStatusProvisioned)
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
		if updated.Status != domain.AppStatusProvisioned {
			t.Errorf("Status = %q, want %q", updated.Status, domain.AppStatusProvisioned)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.UpdateStatus(ctx, uuid.New(), domain.AppStatusDeployed)
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestApplicationService_Delete(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo, nil, nil, nil)
	ctx := context.Background()

	app, _ := svc.Register(ctx, "delete-test", "", "", "", domain.ProviderAWS, nil, nil)

	if err := svc.Delete(ctx, app.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := svc.Get(ctx, app.ID)
	if err != domain.ErrNotFound {
		t.Errorf("after delete: got %v, want ErrNotFound", err)
	}
}

func TestApplicationService_AutoDetect(t *testing.T) {
	ctx := context.Background()

	t.Run("register with source path auto-detects resources", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		llmClient := &llm.MockClient{}
		svc := NewApplicationService(appRepo, resRepo, llmClient, nil)

		// Create a temp dir with infrastructure files
		tmpDir, err := os.MkdirTemp("", "infraplane-autodetect-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("services:\n  db:\n    image: postgres:16\n"), 0644)
		os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"pg": "^8.0"}}`), 0644)

		app, err := svc.Register(ctx, "auto-app", "test", "", tmpDir, domain.ProviderAWS, nil, nil)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		// The mock LLM returns 2 resources by default
		resources, err := resRepo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("list resources error = %v", err)
		}
		if len(resources) != 2 {
			t.Errorf("expected 2 auto-detected resources, got %d", len(resources))
		}
	})

	t.Run("register without source path creates no resources", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		llmClient := &llm.MockClient{}
		svc := NewApplicationService(appRepo, resRepo, llmClient, nil)

		app, err := svc.Register(ctx, "no-source", "test", "", "", domain.ProviderAWS, nil, nil)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		resources, err := resRepo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("list resources error = %v", err)
		}
		if len(resources) != 0 {
			t.Errorf("expected 0 resources, got %d", len(resources))
		}
	})

	t.Run("register with nil LLM skips auto-detection", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		svc := NewApplicationService(appRepo, nil, nil, nil)

		tmpDir, err := os.MkdirTemp("", "infraplane-no-llm-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/app"), 0644)

		app, err := svc.Register(ctx, "no-llm-app", "test", "", tmpDir, domain.ProviderAWS, nil, nil)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if app.Name != "no-llm-app" {
			t.Errorf("Name = %q, want %q", app.Name, "no-llm-app")
		}
	})

	t.Run("register with uploaded files uses them instead of source path", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		llmClient := &llm.MockClient{}
		svc := NewApplicationService(appRepo, resRepo, llmClient, nil)

		app, err := svc.Register(ctx, "upload-app", "test", "", "", domain.ProviderAWS, nil, &RegisterOpts{
			UploadedFiles: &analyzer.CodeContext{
				Files:   []analyzer.FileContent{{Path: "docker-compose.yml", Content: "services:\n  db:\n    image: postgres"}},
				Summary: "Uploaded from browser",
			},
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		resources, err := resRepo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("list resources error = %v", err)
		}
		if len(resources) != 2 {
			t.Errorf("expected 2 auto-detected resources from uploaded files, got %d", len(resources))
		}
	})

	t.Run("analysis error still creates app (graceful degradation)", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		llmClient := &llm.MockClient{
			AnalyzeCodebaseFn: func(ctx context.Context, codeCtx analyzer.CodeContext, provider domain.CloudProvider) ([]llm.ResourceRecommendation, error) {
				return nil, fmt.Errorf("LLM API error")
			},
		}
		svc := NewApplicationService(appRepo, resRepo, llmClient, nil)

		tmpDir, err := os.MkdirTemp("", "infraplane-llm-error-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)
		os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM node:20"), 0644)

		app, err := svc.Register(ctx, "error-app", "test", "", tmpDir, domain.ProviderAWS, nil, nil)
		if err != nil {
			t.Fatalf("Register() should succeed even if analysis fails, got error = %v", err)
		}
		if app.Name != "error-app" {
			t.Errorf("Name = %q, want %q", app.Name, "error-app")
		}

		// No resources should be created
		resources, err := resRepo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("list resources error = %v", err)
		}
		if len(resources) != 0 {
			t.Errorf("expected 0 resources after LLM error, got %d", len(resources))
		}
	})
}

func TestApplicationService_Onboard(t *testing.T) {
	ctx := context.Background()

	t.Run("successful onboard with resources and plan", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		planRepo := mock.NewPlanRepo()
		llmClient := &llm.MockClient{}
		appSvc := NewApplicationService(appRepo, resRepo, llmClient, nil)
		planSvc := NewPlannerService(planRepo, appRepo, resRepo, llmClient, nil)

		result, err := appSvc.Onboard(ctx, "onboard-app", "test app", "", domain.ProviderAWS, nil, &RegisterOpts{
			UploadedFiles: &analyzer.CodeContext{
				Files:   []analyzer.FileContent{{Path: "docker-compose.yml", Content: "services:\n  db:\n    image: postgres"}},
				Summary: "Uploaded from onboarding wizard",
			},
		}, planSvc)
		if err != nil {
			t.Fatalf("Onboard() error = %v", err)
		}

		if result.Application.Name != "onboard-app" {
			t.Errorf("Application.Name = %q, want %q", result.Application.Name, "onboard-app")
		}
		if result.Application.Provider != domain.ProviderAWS {
			t.Errorf("Application.Provider = %q, want %q", result.Application.Provider, domain.ProviderAWS)
		}
		// Mock LLM returns 2 default resources
		if len(result.Resources) != 2 {
			t.Errorf("len(Resources) = %d, want 2", len(result.Resources))
		}
		// Plan should be generated
		if result.Plan.ID == uuid.Nil {
			t.Error("Plan.ID should not be nil")
		}
		if result.Plan.PlanType != domain.PlanTypeHosting {
			t.Errorf("Plan.PlanType = %q, want %q", result.Plan.PlanType, domain.PlanTypeHosting)
		}
	})

	t.Run("onboard with plan generation failure still returns app and resources", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		planRepo := mock.NewPlanRepo()
		llmClient := &llm.MockClient{
			GenerateHostingPlanFn: func(ctx context.Context, app domain.Application, resources []domain.Resource, complianceContext string) (llm.HostingPlanResult, error) {
				return llm.HostingPlanResult{}, fmt.Errorf("LLM timeout")
			},
		}
		appSvc := NewApplicationService(appRepo, resRepo, llmClient, nil)
		planSvc := NewPlannerService(planRepo, appRepo, resRepo, llmClient, nil)

		result, err := appSvc.Onboard(ctx, "onboard-no-plan", "test", "", domain.ProviderGCP, nil, &RegisterOpts{
			UploadedFiles: &analyzer.CodeContext{
				Files:   []analyzer.FileContent{{Path: "Dockerfile", Content: "FROM node:20"}},
				Summary: "Uploaded",
			},
		}, planSvc)
		if err != nil {
			t.Fatalf("Onboard() should succeed even if plan generation fails, got error = %v", err)
		}

		if result.Application.Name != "onboard-no-plan" {
			t.Errorf("Application.Name = %q, want %q", result.Application.Name, "onboard-no-plan")
		}
		// Resources should still be present
		if len(result.Resources) != 2 {
			t.Errorf("len(Resources) = %d, want 2", len(result.Resources))
		}
		// Plan should be zero-value (no plan generated)
		if result.Plan.ID != uuid.Nil {
			t.Errorf("Plan.ID should be nil when generation fails, got %v", result.Plan.ID)
		}
	})

	t.Run("onboard with duplicate name returns error", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		planRepo := mock.NewPlanRepo()
		llmClient := &llm.MockClient{}
		appSvc := NewApplicationService(appRepo, resRepo, llmClient, nil)
		planSvc := NewPlannerService(planRepo, appRepo, resRepo, llmClient, nil)

		// Register first
		_, err := appSvc.Onboard(ctx, "dup-app", "first", "", domain.ProviderAWS, nil, nil, planSvc)
		if err != nil {
			t.Fatalf("first Onboard() error = %v", err)
		}

		// Try same name again
		_, err = appSvc.Onboard(ctx, "dup-app", "second", "", domain.ProviderAWS, nil, nil, planSvc)
		if err == nil {
			t.Fatal("expected error for duplicate name")
		}
	})

	t.Run("onboard with empty name returns error", func(t *testing.T) {
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		planRepo := mock.NewPlanRepo()
		llmClient := &llm.MockClient{}
		appSvc := NewApplicationService(appRepo, resRepo, llmClient, nil)
		planSvc := NewPlannerService(planRepo, appRepo, resRepo, llmClient, nil)

		_, err := appSvc.Onboard(ctx, "", "desc", "", domain.ProviderAWS, nil, nil, planSvc)
		if err == nil {
			t.Fatal("expected validation error for empty name")
		}
	})
}
