package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func TestPlannerService_GenerateHostingPlan(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	planRepo := mock.NewPlanRepo()
	mockLLM := &llm.MockClient{}
	svc := NewPlannerService(planRepo, appRepo, resRepo, mockLLM)
	ctx := context.Background()

	app := domain.NewApplication("hosting-test", "A web API", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	t.Run("successful hosting plan", func(t *testing.T) {
		plan, err := svc.GenerateHostingPlan(ctx, app.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if plan.PlanType != domain.PlanTypeHosting {
			t.Errorf("PlanType = %q, want %q", plan.PlanType, domain.PlanTypeHosting)
		}
		if plan.Content == "" {
			t.Error("Content should not be empty")
		}
		if plan.EstimatedCost == nil {
			t.Error("EstimatedCost should not be nil")
		}
		if plan.ApplicationID != app.ID {
			t.Errorf("ApplicationID = %v, want %v", plan.ApplicationID, app.ID)
		}

		// Verify persisted
		got, err := planRepo.GetByID(ctx, plan.ID)
		if err != nil {
			t.Fatalf("plan not found in repo: %v", err)
		}
		if got.Content != plan.Content {
			t.Error("persisted content mismatch")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.GenerateHostingPlan(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestPlannerService_GenerateMigrationPlan(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	planRepo := mock.NewPlanRepo()
	mockLLM := &llm.MockClient{}
	svc := NewPlannerService(planRepo, appRepo, resRepo, mockLLM)
	ctx := context.Background()

	app := domain.NewApplication("migration-test", "A web API", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	t.Run("successful migration plan", func(t *testing.T) {
		plan, err := svc.GenerateMigrationPlan(ctx, app.ID, domain.ProviderAWS, domain.ProviderGCP)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if plan.PlanType != domain.PlanTypeMigration {
			t.Errorf("PlanType = %q, want %q", plan.PlanType, domain.PlanTypeMigration)
		}
		if *plan.FromProvider != domain.ProviderAWS {
			t.Errorf("FromProvider = %q, want %q", *plan.FromProvider, domain.ProviderAWS)
		}
		if *plan.ToProvider != domain.ProviderGCP {
			t.Errorf("ToProvider = %q, want %q", *plan.ToProvider, domain.ProviderGCP)
		}
	})

	t.Run("same provider error", func(t *testing.T) {
		_, err := svc.GenerateMigrationPlan(ctx, app.ID, domain.ProviderAWS, domain.ProviderAWS)
		if err == nil {
			t.Fatal("expected error for same provider")
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		_, err := svc.GenerateMigrationPlan(ctx, app.ID, domain.CloudProvider("azure"), domain.ProviderGCP)
		if err == nil {
			t.Fatal("expected error for invalid provider")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.GenerateMigrationPlan(ctx, uuid.New(), domain.ProviderAWS, domain.ProviderGCP)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestPlannerService_ListPlansByApplication(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	planRepo := mock.NewPlanRepo()
	mockLLM := &llm.MockClient{}
	svc := NewPlannerService(planRepo, appRepo, resRepo, mockLLM)
	ctx := context.Background()

	app := domain.NewApplication("list-plans-test", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	svc.GenerateHostingPlan(ctx, app.ID)
	svc.GenerateMigrationPlan(ctx, app.ID, domain.ProviderAWS, domain.ProviderGCP)

	plans, err := svc.ListPlansByApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("len = %d, want 2", len(plans))
	}
}
