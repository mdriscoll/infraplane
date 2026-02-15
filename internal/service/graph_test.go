package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func TestGraphService_GenerateGraph(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	graphRepo := mock.NewGraphRepo()
	mockLLM := &llm.MockClient{}
	svc := NewGraphService(graphRepo, appRepo, resRepo, mockLLM)
	ctx := context.Background()

	app := domain.NewApplication("graph-test", "A web API", "", "", domain.ProviderGCP)
	appRepo.Create(ctx, app)

	t.Run("successful graph generation", func(t *testing.T) {
		graph, err := svc.GenerateGraph(ctx, app.ID)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if graph.ApplicationID != app.ID {
			t.Errorf("ApplicationID = %v, want %v", graph.ApplicationID, app.ID)
		}
		if len(graph.Nodes) == 0 {
			t.Error("Nodes should not be empty")
		}
		if len(graph.Edges) == 0 {
			t.Error("Edges should not be empty")
		}

		// Verify internet node is present
		hasInternet := false
		for _, n := range graph.Nodes {
			if n.Kind == domain.GraphNodeInternet {
				hasInternet = true
				break
			}
		}
		if !hasInternet {
			t.Error("graph should contain an internet node")
		}

		// Verify persisted
		got, err := graphRepo.GetLatestByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("graph not found in repo: %v", err)
		}
		if got.ID != graph.ID {
			t.Errorf("persisted graph ID = %v, want %v", got.ID, graph.ID)
		}
		if len(got.Nodes) != len(graph.Nodes) {
			t.Errorf("persisted nodes count = %d, want %d", len(got.Nodes), len(graph.Nodes))
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.GenerateGraph(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestGraphService_GetLatest(t *testing.T) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	graphRepo := mock.NewGraphRepo()
	mockLLM := &llm.MockClient{}
	svc := NewGraphService(graphRepo, appRepo, resRepo, mockLLM)
	ctx := context.Background()

	app := domain.NewApplication("latest-graph-test", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	t.Run("no graph yet", func(t *testing.T) {
		_, err := svc.GetLatest(ctx, app.ID)
		if err == nil {
			t.Fatal("expected error for no graph")
		}
	})

	t.Run("returns latest after generation", func(t *testing.T) {
		generated, err := svc.GenerateGraph(ctx, app.ID)
		if err != nil {
			t.Fatalf("generate error = %v", err)
		}

		latest, err := svc.GetLatest(ctx, app.ID)
		if err != nil {
			t.Fatalf("get latest error = %v", err)
		}
		if latest.ID != generated.ID {
			t.Errorf("latest ID = %v, want %v", latest.ID, generated.ID)
		}
	})
}
