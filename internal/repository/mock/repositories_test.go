package mock

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestApplicationRepo_CRUD(t *testing.T) {
	repo := NewApplicationRepo()
	ctx := context.Background()

	app := domain.NewApplication("test-app", "Test app", "https://github.com/test/repo", domain.ProviderAWS)

	// Create
	if err := repo.Create(ctx, app); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Create duplicate should fail
	dup := app
	dup.ID = uuid.New()
	if err := repo.Create(ctx, dup); err != domain.ErrConflict {
		t.Errorf("Create duplicate: got %v, want ErrConflict", err)
	}

	// GetByID
	got, err := repo.GetByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != app.Name {
		t.Errorf("GetByID().Name = %q, want %q", got.Name, app.Name)
	}

	// GetByID not found
	_, err = repo.GetByID(ctx, uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("GetByID(unknown): got %v, want ErrNotFound", err)
	}

	// GetByName
	got, err = repo.GetByName(ctx, "test-app")
	if err != nil {
		t.Fatalf("GetByName() error = %v", err)
	}
	if got.ID != app.ID {
		t.Errorf("GetByName().ID = %v, want %v", got.ID, app.ID)
	}

	// GetByName not found
	_, err = repo.GetByName(ctx, "nonexistent")
	if err != domain.ErrNotFound {
		t.Errorf("GetByName(unknown): got %v, want ErrNotFound", err)
	}

	// List
	apps, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("List() len = %d, want 1", len(apps))
	}

	// Update
	app.Description = "Updated description"
	if err := repo.Update(ctx, app); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	got, _ = repo.GetByID(ctx, app.ID)
	if got.Description != "Updated description" {
		t.Errorf("Update: Description = %q, want %q", got.Description, "Updated description")
	}

	// Update not found
	missing := app
	missing.ID = uuid.New()
	if err := repo.Update(ctx, missing); err != domain.ErrNotFound {
		t.Errorf("Update(unknown): got %v, want ErrNotFound", err)
	}

	// Delete
	if err := repo.Delete(ctx, app.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = repo.GetByID(ctx, app.ID)
	if err != domain.ErrNotFound {
		t.Errorf("after Delete: GetByID got %v, want ErrNotFound", err)
	}

	// Delete not found
	if err := repo.Delete(ctx, app.ID); err != domain.ErrNotFound {
		t.Errorf("Delete(unknown): got %v, want ErrNotFound", err)
	}
}

func TestResourceRepo_CRUD(t *testing.T) {
	repo := NewResourceRepo()
	ctx := context.Background()
	appID := uuid.New()

	res := domain.NewResource(appID, domain.ResourceDatabase, "user-db", json.RawMessage(`{"engine":"postgres"}`))

	// Create
	if err := repo.Create(ctx, res); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// GetByID
	got, err := repo.GetByID(ctx, res.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != res.Name {
		t.Errorf("GetByID().Name = %q, want %q", got.Name, res.Name)
	}

	// ListByApplicationID
	resources, err := repo.ListByApplicationID(ctx, appID)
	if err != nil {
		t.Fatalf("ListByApplicationID() error = %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("ListByApplicationID() len = %d, want 1", len(resources))
	}

	// ListByApplicationID empty
	resources, err = repo.ListByApplicationID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ListByApplicationID(empty) error = %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("ListByApplicationID(empty) len = %d, want 0", len(resources))
	}

	// Update
	res.Name = "updated-db"
	if err := repo.Update(ctx, res); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	got, _ = repo.GetByID(ctx, res.ID)
	if got.Name != "updated-db" {
		t.Errorf("Update: Name = %q, want %q", got.Name, "updated-db")
	}

	// Delete
	if err := repo.Delete(ctx, res.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = repo.GetByID(ctx, res.ID)
	if err != domain.ErrNotFound {
		t.Errorf("after Delete: got %v, want ErrNotFound", err)
	}
}

func TestDeploymentRepo_CRUD(t *testing.T) {
	repo := NewDeploymentRepo()
	ctx := context.Background()
	appID := uuid.New()

	d := domain.NewDeployment(appID, domain.ProviderAWS, "abc123", "main")

	// Create
	if err := repo.Create(ctx, d); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// GetByID
	got, err := repo.GetByID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.GitCommit != "abc123" {
		t.Errorf("GetByID().GitCommit = %q, want %q", got.GitCommit, "abc123")
	}

	// ListByApplicationID
	deps, err := repo.ListByApplicationID(ctx, appID)
	if err != nil {
		t.Fatalf("ListByApplicationID() error = %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("ListByApplicationID() len = %d, want 1", len(deps))
	}

	// GetLatestByApplicationID
	latest, err := repo.GetLatestByApplicationID(ctx, appID)
	if err != nil {
		t.Fatalf("GetLatestByApplicationID() error = %v", err)
	}
	if latest.ID != d.ID {
		t.Errorf("GetLatestByApplicationID().ID = %v, want %v", latest.ID, d.ID)
	}

	// GetLatestByApplicationID not found
	_, err = repo.GetLatestByApplicationID(ctx, uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("GetLatestByApplicationID(unknown): got %v, want ErrNotFound", err)
	}

	// Update
	d.Status = domain.DeploymentSucceeded
	if err := repo.Update(ctx, d); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	got, _ = repo.GetByID(ctx, d.ID)
	if got.Status != domain.DeploymentSucceeded {
		t.Errorf("Update: Status = %q, want %q", got.Status, domain.DeploymentSucceeded)
	}
}

func TestPlanRepo_CRUD(t *testing.T) {
	repo := NewPlanRepo()
	ctx := context.Background()
	appID := uuid.New()

	plan := domain.NewHostingPlan(appID, "Deploy on AWS ECS", nil, nil)

	// Create
	if err := repo.Create(ctx, plan); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// GetByID
	got, err := repo.GetByID(ctx, plan.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Content != plan.Content {
		t.Errorf("GetByID().Content = %q, want %q", got.Content, plan.Content)
	}

	// GetByID not found
	_, err = repo.GetByID(ctx, uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("GetByID(unknown): got %v, want ErrNotFound", err)
	}

	// ListByApplicationID
	plans, err := repo.ListByApplicationID(ctx, appID)
	if err != nil {
		t.Fatalf("ListByApplicationID() error = %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("ListByApplicationID() len = %d, want 1", len(plans))
	}
}
