package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// ApplicationService handles application lifecycle operations.
type ApplicationService struct {
	apps      repository.ApplicationRepo
	resources repository.ResourceRepo
	llm       llm.Client
}

// NewApplicationService creates a new ApplicationService.
// The resources and llmClient params are optional — pass nil to skip auto-detection.
func NewApplicationService(apps repository.ApplicationRepo, resources repository.ResourceRepo, llmClient llm.Client) *ApplicationService {
	return &ApplicationService{
		apps:      apps,
		resources: resources,
		llm:       llmClient,
	}
}

// Register creates a new application. If a sourcePath is provided and the LLM
// client is configured, it analyzes the codebase and auto-detects resources.
func (s *ApplicationService) Register(ctx context.Context, name, description, gitRepoURL, sourcePath string, provider domain.CloudProvider) (domain.Application, error) {
	app := domain.NewApplication(name, description, gitRepoURL, sourcePath, provider)
	if err := app.Validate(); err != nil {
		return domain.Application{}, err
	}

	if err := s.apps.Create(ctx, app); err != nil {
		return domain.Application{}, fmt.Errorf("register application: %w", err)
	}

	// Auto-detect resources from source if configured
	if sourcePath != "" && s.llm != nil && s.resources != nil {
		if err := s.autoDetectResources(ctx, app); err != nil {
			// Graceful degradation: log the error but still return the app
			log.Printf("auto-detect resources for %s: %v", app.Name, err)
		}
	}

	return app, nil
}

// autoDetectResources analyzes the application's source code and creates
// resources based on the LLM's recommendations.
func (s *ApplicationService) autoDetectResources(ctx context.Context, app domain.Application) error {
	codeCtx, err := analyzer.Analyze(app.SourcePath)
	if err != nil {
		return fmt.Errorf("analyze source: %w", err)
	}

	if len(codeCtx.Files) == 0 {
		return nil // Nothing to analyze
	}

	recommendations, err := s.llm.AnalyzeCodebase(ctx, codeCtx, app.Provider)
	if err != nil {
		return fmt.Errorf("LLM codebase analysis: %w", err)
	}

	for _, rec := range recommendations {
		resource := domain.NewResource(app.ID, rec.Kind, rec.Name, rec.Spec)
		resource.ProviderMappings = rec.Mappings

		if err := resource.Validate(); err != nil {
			log.Printf("skip invalid resource %s: %v", rec.Name, err)
			continue
		}

		if err := s.resources.Create(ctx, resource); err != nil {
			log.Printf("create resource %s: %v", rec.Name, err)
			continue
		}
	}

	return nil
}

// ReanalyzeSource re-runs code analysis on an existing application's source.
func (s *ApplicationService) ReanalyzeSource(ctx context.Context, appID uuid.UUID) error {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return err
	}

	if app.SourcePath == "" {
		return domain.ErrValidation("application has no source path configured")
	}

	if s.llm == nil || s.resources == nil {
		return domain.ErrValidation("auto-detection not configured")
	}

	return s.autoDetectResources(ctx, app)
}

// Get returns an application by ID.
func (s *ApplicationService) Get(ctx context.Context, id uuid.UUID) (domain.Application, error) {
	return s.apps.GetByID(ctx, id)
}

// GetByName returns an application by name.
func (s *ApplicationService) GetByName(ctx context.Context, name string) (domain.Application, error) {
	return s.apps.GetByName(ctx, name)
}

// List returns all applications.
func (s *ApplicationService) List(ctx context.Context) ([]domain.Application, error) {
	return s.apps.List(ctx)
}

// UpdateStatus changes the application status.
func (s *ApplicationService) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AppStatus) (domain.Application, error) {
	app, err := s.apps.GetByID(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	app.Status = status
	app.UpdatedAt = time.Now().UTC()
	if err := s.apps.Update(ctx, app); err != nil {
		return domain.Application{}, fmt.Errorf("update application status: %w", err)
	}
	return app, nil
}

// Delete removes an application.
func (s *ApplicationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.apps.Delete(ctx, id)
}
