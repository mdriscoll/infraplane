package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/provider"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func setupInfraService(providerName domain.CloudProvider, applyErr error) (*InfraService, *mock.ApplicationRepo, *mock.ResourceRepo) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	depRepo := mock.NewDeploymentRepo()

	reg := provider.NewRegistry()
	reg.Register(&provider.MockAdapter{
		ProviderVal: providerName,
		ApplyResult: "Plan applied successfully",
		ApplyErr:    applyErr,
	})

	svc := NewInfraService(appRepo, resRepo, depRepo, reg)
	return svc, appRepo, resRepo
}

func TestInfraService_GenerateTerraform(t *testing.T) {
	svc, appRepo, resRepo := setupInfraService(domain.ProviderAWS, nil)
	ctx := context.Background()

	app := domain.NewApplication("tf-app", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	resource := domain.NewResource(app.ID, domain.ResourceDatabase, "user-db", json.RawMessage(`{"engine": "postgres"}`))
	resource.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
		domain.ProviderAWS: {
			ServiceName:  "RDS",
			Config:       map[string]any{"instance_class": "db.t3.micro"},
			TerraformHCL: `resource "aws_db_instance" "user_db" { engine = "postgres" }`,
		},
	}
	resRepo.Create(ctx, resource)

	t.Run("successful generation", func(t *testing.T) {
		config, err := svc.GenerateTerraform(ctx, app.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(config, "aws_db_instance") {
			t.Error("expected AWS DB resource in config")
		}
		if !strings.Contains(config, "hashicorp/aws") {
			t.Error("expected AWS provider block")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.GenerateTerraform(ctx, uuid.New())
		if err == nil {
			t.Error("expected error for nonexistent app")
		}
	})
}

func TestInfraService_DeployInfrastructure(t *testing.T) {
	ctx := context.Background()

	t.Run("successful deployment", func(t *testing.T) {
		svc, appRepo, resRepo := setupInfraService(domain.ProviderAWS, nil)

		app := domain.NewApplication("deploy-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		resource := domain.NewResource(app.ID, domain.ResourceDatabase, "db", json.RawMessage(`{}`))
		resource.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderAWS: {
				ServiceName:  "RDS",
				TerraformHCL: `resource "aws_db_instance" "db" {}`,
			},
		}
		resRepo.Create(ctx, resource)

		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc123", "main")
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if d.Status != domain.DeploymentSucceeded {
			t.Errorf("status = %v, want succeeded", d.Status)
		}
		if d.Provider != domain.ProviderAWS {
			t.Errorf("provider = %v, want aws", d.Provider)
		}
		if d.GitBranch != "main" {
			t.Errorf("git_branch = %v, want main", d.GitBranch)
		}
		if d.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
	})

	t.Run("apply failure marks deployment as failed", func(t *testing.T) {
		svc, appRepo, _ := setupInfraService(domain.ProviderAWS, fmt.Errorf("terraform error"))

		app := domain.NewApplication("fail-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc123", "main")
		if err == nil {
			t.Fatal("expected error from failed apply")
		}
		if d.Status != domain.DeploymentFailed {
			t.Errorf("status = %v, want failed", d.Status)
		}
	})

	t.Run("no provider adapter", func(t *testing.T) {
		// Register only GCP adapter but app uses AWS
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		depRepo := mock.NewDeploymentRepo()

		reg := provider.NewRegistry()
		reg.Register(&provider.MockAdapter{
			ProviderVal: domain.ProviderGCP,
			ApplyResult: "ok",
		})

		svc := NewInfraService(appRepo, resRepo, depRepo, reg)

		app := domain.NewApplication("no-adapter-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc", "main")
		if err == nil {
			t.Fatal("expected error for missing adapter")
		}
		if d.Status != domain.DeploymentFailed {
			t.Errorf("status = %v, want failed", d.Status)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		svc, _, _ := setupInfraService(domain.ProviderAWS, nil)
		_, err := svc.DeployInfrastructure(ctx, uuid.New(), "abc", "main")
		if err == nil {
			t.Error("expected error for nonexistent app")
		}
	})

	t.Run("missing branch", func(t *testing.T) {
		svc, appRepo, _ := setupInfraService(domain.ProviderAWS, nil)
		app := domain.NewApplication("no-branch", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		_, err := svc.DeployInfrastructure(ctx, app.ID, "abc", "")
		if err == nil {
			t.Error("expected validation error for missing branch")
		}
	})
}

func TestInfraService_ValidateProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("valid provider", func(t *testing.T) {
		svc, _, _ := setupInfraService(domain.ProviderAWS, nil)
		err := svc.ValidateProvider(ctx, domain.ProviderAWS)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		svc, _, _ := setupInfraService(domain.ProviderAWS, nil)
		err := svc.ValidateProvider(ctx, domain.CloudProvider("azure"))
		if err == nil {
			t.Error("expected error for unknown provider")
		}
	})
}

func TestInfraService_DestroyInfrastructure(t *testing.T) {
	ctx := context.Background()

	t.Run("successful destroy", func(t *testing.T) {
		svc, appRepo, resRepo := setupInfraService(domain.ProviderAWS, nil)

		app := domain.NewApplication("destroy-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		resource := domain.NewResource(app.ID, domain.ResourceDatabase, "db", json.RawMessage(`{}`))
		resource.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderAWS: {
				ServiceName:  "RDS",
				TerraformHCL: `resource "aws_db_instance" "db" {}`,
			},
		}
		resRepo.Create(ctx, resource)

		// Deploy first
		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc", "main")
		if err != nil {
			t.Fatalf("deploy error = %v", err)
		}

		// Destroy
		err = svc.DestroyInfrastructure(ctx, d.ID)
		if err != nil {
			t.Errorf("destroy error = %v", err)
		}
	})

	t.Run("deployment not found", func(t *testing.T) {
		svc, _, _ := setupInfraService(domain.ProviderAWS, nil)
		err := svc.DestroyInfrastructure(ctx, uuid.New())
		if err == nil {
			t.Error("expected error for nonexistent deployment")
		}
	})
}
