package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func TestResourceService_AddFromDescription(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	mockLLM := &llm.MockClient{}
	svc := NewResourceService(resRepo, appRepo, mockLLM, nil)
	ctx := context.Background()

	// Create a parent app
	app := domain.NewApplication("test-app", "desc", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	t.Run("successful add from description", func(t *testing.T) {
		resource, err := svc.AddFromDescription(ctx, app.ID, "I need a PostgreSQL database")
		if err != nil {
			t.Fatalf("AddFromDescription() error = %v", err)
		}
		if resource.Kind != domain.ResourceDatabase {
			t.Errorf("Kind = %q, want %q", resource.Kind, domain.ResourceDatabase)
		}
		if resource.ApplicationID != app.ID {
			t.Errorf("ApplicationID = %v, want %v", resource.ApplicationID, app.ID)
		}
		if _, ok := resource.ProviderMappings[domain.ProviderAWS]; !ok {
			t.Error("missing AWS provider mapping")
		}

		// Verify it was persisted
		got, err := resRepo.GetByID(ctx, resource.ID)
		if err != nil {
			t.Fatalf("resource not found in repo: %v", err)
		}
		if got.Name != resource.Name {
			t.Errorf("persisted Name = %q, want %q", got.Name, resource.Name)
		}
	})

	t.Run("custom LLM response", func(t *testing.T) {
		mockLLM.AnalyzeResourceNeedFn = func(ctx context.Context, desc string, provider domain.CloudProvider) (llm.ResourceRecommendation, error) {
			return llm.ResourceRecommendation{
				Kind: domain.ResourceCache,
				Name: "session-cache",
				Spec: json.RawMessage(`{"engine": "redis"}`),
				Mappings: map[domain.CloudProvider]domain.ProviderResource{
					domain.ProviderAWS: {ServiceName: "ElastiCache"},
					domain.ProviderGCP: {ServiceName: "Memorystore"},
				},
			}, nil
		}
		defer func() { mockLLM.AnalyzeResourceNeedFn = nil }()

		resource, err := svc.AddFromDescription(ctx, app.ID, "I need a Redis cache for sessions")
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if resource.Kind != domain.ResourceCache {
			t.Errorf("Kind = %q, want %q", resource.Kind, domain.ResourceCache)
		}
		if resource.Name != "session-cache" {
			t.Errorf("Name = %q, want %q", resource.Name, "session-cache")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.AddFromDescription(ctx, uuid.New(), "some resource")
		if err == nil {
			t.Fatal("expected error for missing app")
		}
	})
}

func TestResourceService_ListByApplication(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	mockLLM := &llm.MockClient{}
	svc := NewResourceService(resRepo, appRepo, mockLLM, nil)
	ctx := context.Background()

	app := domain.NewApplication("list-test", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	svc.AddFromDescription(ctx, app.ID, "database")
	svc.AddFromDescription(ctx, app.ID, "cache")

	resources, err := svc.ListByApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if len(resources) != 2 {
		t.Errorf("len = %d, want 2", len(resources))
	}
}

func TestResourceService_Remove(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	mockLLM := &llm.MockClient{}
	svc := NewResourceService(resRepo, appRepo, mockLLM, nil)
	ctx := context.Background()

	app := domain.NewApplication("remove-test", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	resource, _ := svc.AddFromDescription(ctx, app.ID, "database")

	if err := svc.Remove(ctx, resource.ID); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	_, err := svc.Get(ctx, resource.ID)
	if err != domain.ErrNotFound {
		t.Errorf("after remove: got %v, want ErrNotFound", err)
	}
}
