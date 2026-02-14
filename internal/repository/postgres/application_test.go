package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestIntegrationApplicationRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	repo := NewApplicationRepo(pool)
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		app := domain.NewApplication("create-get-test", "desc", "https://github.com/test/repo", "", domain.ProviderAWS)
		if err := repo.Create(ctx, app); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByID(ctx, app.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.Name != app.Name {
			t.Errorf("Name = %q, want %q", got.Name, app.Name)
		}
		if got.Provider != domain.ProviderAWS {
			t.Errorf("Provider = %q, want %q", got.Provider, domain.ProviderAWS)
		}
		if got.Status != domain.AppStatusDraft {
			t.Errorf("Status = %q, want %q", got.Status, domain.AppStatusDraft)
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("GetByName", func(t *testing.T) {
		app := domain.NewApplication("name-lookup-test", "desc", "", "", domain.ProviderGCP)
		if err := repo.Create(ctx, app); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByName(ctx, "name-lookup-test")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if got.ID != app.ID {
			t.Errorf("ID = %v, want %v", got.ID, app.ID)
		}
	})

	t.Run("GetByName not found", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "nonexistent-app")
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		apps, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(apps) < 2 {
			t.Errorf("List() len = %d, want >= 2", len(apps))
		}
	})

	t.Run("Update", func(t *testing.T) {
		app := domain.NewApplication("update-test", "original", "", "", domain.ProviderAWS)
		if err := repo.Create(ctx, app); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		app.Description = "updated"
		app.Status = domain.AppStatusProvisioned
		if err := repo.Update(ctx, app); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		got, _ := repo.GetByID(ctx, app.ID)
		if got.Description != "updated" {
			t.Errorf("Description = %q, want %q", got.Description, "updated")
		}
		if got.Status != domain.AppStatusProvisioned {
			t.Errorf("Status = %q, want %q", got.Status, domain.AppStatusProvisioned)
		}
	})

	t.Run("Update not found", func(t *testing.T) {
		app := domain.NewApplication("ghost", "", "", "", domain.ProviderAWS)
		if err := repo.Update(ctx, app); err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		app := domain.NewApplication("delete-test", "", "", "", domain.ProviderAWS)
		if err := repo.Create(ctx, app); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := repo.Delete(ctx, app.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		_, err := repo.GetByID(ctx, app.ID)
		if err != domain.ErrNotFound {
			t.Errorf("after Delete: got %v, want ErrNotFound", err)
		}
	})

	t.Run("Delete not found", func(t *testing.T) {
		if err := repo.Delete(ctx, uuid.New()); err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}
