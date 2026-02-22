package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/compliance"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// ApplicationService handles application lifecycle operations.
type ApplicationService struct {
	apps       repository.ApplicationRepo
	resources  repository.ResourceRepo
	llm        llm.Client
	compliance *compliance.Registry
}

// NewApplicationService creates a new ApplicationService.
// The resources and llmClient params are optional — pass nil to skip auto-detection.
// The compliance registry is optional — pass nil to skip compliance validation.
func NewApplicationService(apps repository.ApplicationRepo, resources repository.ResourceRepo, llmClient llm.Client, complianceRegistry *compliance.Registry) *ApplicationService {
	return &ApplicationService{
		apps:       apps,
		resources:  resources,
		llm:        llmClient,
		compliance: complianceRegistry,
	}
}

// RegisterOpts holds optional parameters for application registration.
type RegisterOpts struct {
	// UploadedFiles contains file contents uploaded from a browser (when
	// the source path can't be provided due to browser security restrictions).
	UploadedFiles *analyzer.CodeContext
}

// Register creates a new application. If a sourcePath is provided and the LLM
// client is configured, it analyzes the codebase and auto-detects resources.
// If opts.UploadedFiles is provided (browser upload flow), those files are
// analyzed instead of reading from the filesystem.
// complianceFrameworks is an optional list of framework IDs (e.g. "cis_gcp_v4").
func (s *ApplicationService) Register(ctx context.Context, name, description, gitRepoURL, sourcePath string, provider domain.CloudProvider, complianceFrameworks []string, opts *RegisterOpts) (domain.Application, error) {
	// Validate compliance frameworks if provided
	if len(complianceFrameworks) > 0 && s.compliance != nil {
		if err := s.compliance.ValidateFrameworks(complianceFrameworks); err != nil {
			return domain.Application{}, domain.ErrValidation(err.Error())
		}
	}

	app := domain.NewApplication(name, description, gitRepoURL, sourcePath, provider)
	app.ComplianceFrameworks = complianceFrameworks
	if err := app.Validate(); err != nil {
		return domain.Application{}, err
	}

	if err := s.apps.Create(ctx, app); err != nil {
		return domain.Application{}, fmt.Errorf("register application: %w", err)
	}

	// Auto-detect resources from source if configured
	if s.llm != nil && s.resources != nil {
		if opts != nil && opts.UploadedFiles != nil && len(opts.UploadedFiles.Files) > 0 {
			// Browser upload flow: analyze the uploaded file contents directly
			if err := s.AnalyzeUploadedFiles(ctx, app.ID, *opts.UploadedFiles); err != nil {
				log.Printf("auto-detect resources (uploaded) for %s: %v", app.Name, err)
			}
		} else if sourcePath != "" {
			// Server-side flow: read files from filesystem/git
			if err := s.autoDetectResources(ctx, app); err != nil {
				log.Printf("auto-detect resources for %s: %v", app.Name, err)
			}
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

// AnalyzeUploadedFiles analyzes file contents uploaded from the browser (since
// browsers can't expose filesystem paths). It runs LLM analysis on the provided
// CodeContext and creates resources for the application.
func (s *ApplicationService) AnalyzeUploadedFiles(ctx context.Context, appID uuid.UUID, codeCtx analyzer.CodeContext) error {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return err
	}

	if s.llm == nil || s.resources == nil {
		return domain.ErrValidation("auto-detection not configured")
	}

	if len(codeCtx.Files) == 0 {
		return nil
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

// OnboardResult holds the full onboarding result: app, detected resources, and hosting plan.
type OnboardResult struct {
	Application domain.Application       `json:"application"`
	Resources   []domain.Resource        `json:"resources"`
	Plan        domain.InfrastructurePlan `json:"plan"`
}

// Onboard performs the complete onboarding flow in a single operation:
// 1. Register the application (with auto-detect resources)
// 2. Load the detected resources
// 3. Generate a hosting plan
// The planner parameter avoids a circular dependency between services.
func (s *ApplicationService) Onboard(
	ctx context.Context,
	name, description, sourcePath string,
	provider domain.CloudProvider,
	complianceFrameworks []string,
	opts *RegisterOpts,
	planner *PlannerService,
) (OnboardResult, error) {
	// Step 1: Register the application (includes auto-detect resources)
	app, err := s.Register(ctx, name, description, "", sourcePath, provider, complianceFrameworks, opts)
	if err != nil {
		return OnboardResult{}, fmt.Errorf("register: %w", err)
	}

	// Step 2: Load the detected resources
	resources, err := s.resources.ListByApplicationID(ctx, app.ID)
	if err != nil {
		return OnboardResult{}, fmt.Errorf("list resources: %w", err)
	}

	// Step 3: Generate hosting plan
	// Use a detached context so plan generation isn't cancelled if the HTTP
	// request is interrupted — the plan is non-critical and we already have
	// the app and resources saved.
	planCtx, planCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer planCancel()
	plan, err := planner.GenerateHostingPlan(planCtx, app.ID)
	if err != nil {
		// Plan generation failed but app + resources are still valid
		log.Printf("onboard: hosting plan generation failed for %s: %v", name, err)
		return OnboardResult{
			Application: app,
			Resources:   resources,
		}, nil
	}

	return OnboardResult{
		Application: app,
		Resources:   resources,
		Plan:        plan,
	}, nil
}
