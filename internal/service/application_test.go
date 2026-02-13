package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func TestApplicationService_Register(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo)
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		app, err := svc.Register(ctx, "my-app", "A test app", "https://github.com/test/repo", domain.ProviderAWS)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if app.Name != "my-app" {
			t.Errorf("Name = %q, want %q", app.Name, "my-app")
		}
		if app.Status != domain.AppStatusDraft {
			t.Errorf("Status = %q, want %q", app.Status, domain.AppStatusDraft)
		}
		if app.ID == uuid.Nil {
			t.Error("ID should not be nil")
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		_, err := svc.Register(ctx, "", "desc", "", domain.ProviderAWS)
		if err == nil {
			t.Fatal("expected validation error")
		}
		if !domain.IsValidationError(err) {
			t.Errorf("expected ValidationError, got %T", err)
		}
	})

	t.Run("validation error - invalid provider", func(t *testing.T) {
		_, err := svc.Register(ctx, "valid-name", "desc", "", domain.CloudProvider("azure"))
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, err := svc.Register(ctx, "my-app", "duplicate", "", domain.ProviderGCP)
		if err == nil {
			t.Fatal("expected conflict error")
		}
	})
}

func TestApplicationService_Get(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo)
	ctx := context.Background()

	app, _ := svc.Register(ctx, "get-test", "desc", "", domain.ProviderAWS)

	t.Run("found", func(t *testing.T) {
		got, err := svc.Get(ctx, app.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Name != "get-test" {
			t.Errorf("Name = %q, want %q", got.Name, "get-test")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Get(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestApplicationService_GetByName(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo)
	ctx := context.Background()

	svc.Register(ctx, "name-test", "desc", "", domain.ProviderGCP)

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByName(ctx, "name-test")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if got.Provider != domain.ProviderGCP {
			t.Errorf("Provider = %q, want %q", got.Provider, domain.ProviderGCP)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetByName(ctx, "nonexistent")
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestApplicationService_List(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo)
	ctx := context.Background()

	svc.Register(ctx, "app-1", "", "", domain.ProviderAWS)
	svc.Register(ctx, "app-2", "", "", domain.ProviderGCP)

	apps, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(apps) != 2 {
		t.Errorf("len = %d, want 2", len(apps))
	}
}

func TestApplicationService_UpdateStatus(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo)
	ctx := context.Background()

	app, _ := svc.Register(ctx, "status-test", "", "", domain.ProviderAWS)

	t.Run("update to provisioned", func(t *testing.T) {
		updated, err := svc.UpdateStatus(ctx, app.ID, domain.AppStatusProvisioned)
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
		if updated.Status != domain.AppStatusProvisioned {
			t.Errorf("Status = %q, want %q", updated.Status, domain.AppStatusProvisioned)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.UpdateStatus(ctx, uuid.New(), domain.AppStatusDeployed)
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})
}

func TestApplicationService_Delete(t *testing.T) {
	repo := mock.NewApplicationRepo()
	svc := NewApplicationService(repo)
	ctx := context.Background()

	app, _ := svc.Register(ctx, "delete-test", "", "", domain.ProviderAWS)

	if err := svc.Delete(ctx, app.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := svc.Get(ctx, app.ID)
	if err != domain.ErrNotFound {
		t.Errorf("after delete: got %v, want ErrNotFound", err)
	}
}
