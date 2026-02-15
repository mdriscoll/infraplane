package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// GraphService handles LLM-powered infrastructure topology graph generation.
type GraphService struct {
	graphs    repository.GraphRepo
	apps      repository.ApplicationRepo
	resources repository.ResourceRepo
	llm       llm.Client
}

// NewGraphService creates a new GraphService.
func NewGraphService(graphs repository.GraphRepo, apps repository.ApplicationRepo, resources repository.ResourceRepo, llmClient llm.Client) *GraphService {
	return &GraphService{
		graphs:    graphs,
		apps:      apps,
		resources: resources,
		llm:       llmClient,
	}
}

// GenerateGraph creates an LLM-powered infrastructure topology graph for an application.
func (s *GraphService) GenerateGraph(ctx context.Context, appID uuid.UUID) (domain.InfraGraph, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.InfraGraph{}, fmt.Errorf("get application: %w", err)
	}

	resources, err := s.resources.ListByApplicationID(ctx, appID)
	if err != nil {
		return domain.InfraGraph{}, fmt.Errorf("list resources: %w", err)
	}

	result, err := s.llm.GenerateGraph(ctx, app, resources)
	if err != nil {
		return domain.InfraGraph{}, fmt.Errorf("generate graph: %w", err)
	}

	graph := domain.NewInfraGraph(appID, result.Nodes, result.Edges)
	if err := s.graphs.Create(ctx, graph); err != nil {
		return domain.InfraGraph{}, fmt.Errorf("save graph: %w", err)
	}

	return graph, nil
}

// GetLatest returns the most recently generated graph for an application.
func (s *GraphService) GetLatest(ctx context.Context, appID uuid.UUID) (domain.InfraGraph, error) {
	return s.graphs.GetLatestByApplicationID(ctx, appID)
}
