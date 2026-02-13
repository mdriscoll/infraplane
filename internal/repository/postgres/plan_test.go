package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestIntegrationPlanRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	appRepo := NewApplicationRepo(pool)
	repo := NewPlanRepo(pool)
	ctx := context.Background()

	app := domain.NewApplication("plan-test-app", "desc", "", domain.ProviderAWS)
	if err := appRepo.Create(ctx, app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	t.Run("Create and GetByID hosting plan", func(t *testing.T) {
		cost := &domain.CostEstimate{
			MonthlyCostUSD: 150.00,
			Breakdown:      map[string]float64{"compute": 100, "database": 50},
		}
		plan := domain.NewHostingPlan(app.ID, "# Hosting Plan\nDeploy on AWS ECS with RDS.", nil, cost)

		if err := repo.Create(ctx, plan); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByID(ctx, plan.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.PlanType != domain.PlanTypeHosting {
			t.Errorf("PlanType = %q, want %q", got.PlanType, domain.PlanTypeHosting)
		}
		if got.Content != plan.Content {
			t.Errorf("Content mismatch")
		}
		if got.EstimatedCost == nil {
			t.Fatal("EstimatedCost should not be nil")
		}
		if got.EstimatedCost.MonthlyCostUSD != 150.00 {
			t.Errorf("MonthlyCostUSD = %f, want 150.00", got.EstimatedCost.MonthlyCostUSD)
		}
	})

	t.Run("Create and GetByID migration plan", func(t *testing.T) {
		plan := domain.NewMigrationPlan(app.ID, domain.ProviderAWS, domain.ProviderGCP, "Migrate to GCP", nil, nil)

		if err := repo.Create(ctx, plan); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByID(ctx, plan.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.PlanType != domain.PlanTypeMigration {
			t.Errorf("PlanType = %q, want %q", got.PlanType, domain.PlanTypeMigration)
		}
		if got.FromProvider == nil || *got.FromProvider != domain.ProviderAWS {
			t.Errorf("FromProvider = %v, want aws", got.FromProvider)
		}
		if got.ToProvider == nil || *got.ToProvider != domain.ProviderGCP {
			t.Errorf("ToProvider = %v, want gcp", got.ToProvider)
		}
		if got.EstimatedCost != nil {
			t.Error("EstimatedCost should be nil")
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("ListByApplicationID", func(t *testing.T) {
		plans, err := repo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(plans) != 2 {
			t.Errorf("len = %d, want 2", len(plans))
		}
	})

	t.Run("ListByApplicationID empty", func(t *testing.T) {
		plans, err := repo.ListByApplicationID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(plans) != 0 {
			t.Errorf("len = %d, want 0", len(plans))
		}
	})
}
