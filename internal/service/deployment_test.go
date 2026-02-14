package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func TestDeploymentService_Deploy(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	depRepo := mock.NewDeploymentRepo()
	svc := NewDeploymentService(depRepo, appRepo)
	ctx := context.Background()

	app := domain.NewApplication("deploy-app", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	t.Run("successful deploy", func(t *testing.T) {
		d, err := svc.Deploy(ctx, app.ID, "abc123", "main")
		if err != nil {
			t.Fatalf("Deploy() error = %v", err)
		}
		if d.Status != domain.DeploymentPending {
			t.Errorf("Status = %q, want %q", d.Status, domain.DeploymentPending)
		}
		if d.GitCommit != "abc123" {
			t.Errorf("GitCommit = %q, want %q", d.GitCommit, "abc123")
		}
		if d.Provider != domain.ProviderAWS {
			t.Errorf("Provider = %q, want %q", d.Provider, domain.ProviderAWS)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.Deploy(ctx, uuid.New(), "abc", "main")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing branch", func(t *testing.T) {
		_, err := svc.Deploy(ctx, app.ID, "abc", "")
		if err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestDeploymentService_GetStatus(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	depRepo := mock.NewDeploymentRepo()
	svc := NewDeploymentService(depRepo, appRepo)
	ctx := context.Background()

	app := domain.NewApplication("status-app", "", "", "", domain.ProviderGCP)
	appRepo.Create(ctx, app)

	d, _ := svc.Deploy(ctx, app.ID, "abc", "main")

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetStatus(ctx, d.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if got.ID != d.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetStatus(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestDeploymentService_MarkSucceeded(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	depRepo := mock.NewDeploymentRepo()
	svc := NewDeploymentService(depRepo, appRepo)
	ctx := context.Background()

	app := domain.NewApplication("succeed-app", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	d, _ := svc.Deploy(ctx, app.ID, "abc", "main")

	updated, err := svc.MarkSucceeded(ctx, d.ID, "terraform plan output")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if updated.Status != domain.DeploymentSucceeded {
		t.Errorf("Status = %q, want %q", updated.Status, domain.DeploymentSucceeded)
	}
	if updated.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
	if updated.TerraformPlan != "terraform plan output" {
		t.Errorf("TerraformPlan = %q, want %q", updated.TerraformPlan, "terraform plan output")
	}
}

func TestDeploymentService_MarkFailed(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	depRepo := mock.NewDeploymentRepo()
	svc := NewDeploymentService(depRepo, appRepo)
	ctx := context.Background()

	app := domain.NewApplication("fail-app", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	d, _ := svc.Deploy(ctx, app.ID, "abc", "main")

	updated, err := svc.MarkFailed(ctx, d.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if updated.Status != domain.DeploymentFailed {
		t.Errorf("Status = %q, want %q", updated.Status, domain.DeploymentFailed)
	}
	if updated.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

func TestDeploymentService_GetLatest(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	depRepo := mock.NewDeploymentRepo()
	svc := NewDeploymentService(depRepo, appRepo)
	ctx := context.Background()

	app := domain.NewApplication("latest-app", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	first, _ := svc.Deploy(ctx, app.ID, "first", "main")
	// Push the first deployment's timestamp back so "second" is clearly newer
	first.StartedAt = first.StartedAt.Add(-time.Minute)
	depRepo.Update(ctx, first)

	second, _ := svc.Deploy(ctx, app.ID, "second", "main")

	latest, err := svc.GetLatest(ctx, app.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if latest.GitCommit != second.GitCommit {
		t.Errorf("GitCommit = %q, want %q", latest.GitCommit, second.GitCommit)
	}
}
