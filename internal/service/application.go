package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// ApplicationService handles application lifecycle operations.
type ApplicationService struct {
	apps repository.ApplicationRepo
}

// NewApplicationService creates a new ApplicationService.
func NewApplicationService(apps repository.ApplicationRepo) *ApplicationService {
	return &ApplicationService{apps: apps}
}

// Register creates a new application.
func (s *ApplicationService) Register(ctx context.Context, name, description, gitRepoURL string, provider domain.CloudProvider) (domain.Application, error) {
	app := domain.NewApplication(name, description, gitRepoURL, provider)
	if err := app.Validate(); err != nil {
		return domain.Application{}, err
	}

	if err := s.apps.Create(ctx, app); err != nil {
		return domain.Application{}, fmt.Errorf("register application: %w", err)
	}
	return app, nil
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
