package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestIntegrationDeploymentRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	appRepo := NewApplicationRepo(pool)
	repo := NewDeploymentRepo(pool)
	ctx := context.Background()

	app := domain.NewApplication("deploy-test-app", "desc", "", "", domain.ProviderAWS)
	if err := appRepo.Create(ctx, app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	t.Run("Create and GetByID", func(t *testing.T) {
		d := domain.NewDeployment(app.ID, domain.ProviderAWS, "abc123def", "main", nil)
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByID(ctx, d.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.GitCommit != "abc123def" {
			t.Errorf("GitCommit = %q, want %q", got.GitCommit, "abc123def")
		}
		if got.Status != domain.DeploymentPending {
			t.Errorf("Status = %q, want %q", got.Status, domain.DeploymentPending)
		}
		if got.CompletedAt != nil {
			t.Error("CompletedAt should be nil")
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("Update status", func(t *testing.T) {
		d := domain.NewDeployment(app.ID, domain.ProviderAWS, "def456", "feature-branch", nil)
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		now := time.Now().UTC()
		d.Status = domain.DeploymentSucceeded
		d.CompletedAt = &now
		d.TerraformPlan = "plan output"
		if err := repo.Update(ctx, d); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		got, _ := repo.GetByID(ctx, d.ID)
		if got.Status != domain.DeploymentSucceeded {
			t.Errorf("Status = %q, want %q", got.Status, domain.DeploymentSucceeded)
		}
		if got.CompletedAt == nil {
			t.Error("CompletedAt should not be nil")
		}
		if got.TerraformPlan != "plan output" {
			t.Errorf("TerraformPlan = %q, want %q", got.TerraformPlan, "plan output")
		}
	})

	t.Run("ListByApplicationID", func(t *testing.T) {
		deps, err := repo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(deps) < 2 {
			t.Errorf("len = %d, want >= 2", len(deps))
		}
	})

	t.Run("GetLatestByApplicationID", func(t *testing.T) {
		// Create a newer deployment
		d := domain.NewDeployment(app.ID, domain.ProviderAWS, "latest999", "main", nil)
		if err := repo.Create(ctx, d); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		latest, err := repo.GetLatestByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if latest.GitCommit != "latest999" {
			t.Errorf("GitCommit = %q, want %q", latest.GitCommit, "latest999")
		}
	})

	t.Run("GetLatestByApplicationID not found", func(t *testing.T) {
		_, err := repo.GetLatestByApplicationID(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}
