package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestIntegrationResourceRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := setupTestDB(t)
	appRepo := NewApplicationRepo(pool)
	repo := NewResourceRepo(pool)
	ctx := context.Background()

	// Create a parent application
	app := domain.NewApplication("resource-test-app", "desc", "", "", domain.ProviderAWS)
	if err := appRepo.Create(ctx, app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	t.Run("Create and GetByID", func(t *testing.T) {
		res := domain.NewResource(app.ID, domain.ResourceDatabase, "user-db", json.RawMessage(`{"engine":"postgres","version":"16"}`))
		res.ProviderMappings[domain.ProviderAWS] = domain.ProviderResource{
			ServiceName:  "RDS",
			Config:       map[string]any{"instance_class": "db.t3.micro"},
			TerraformHCL: `resource "aws_db_instance" "user_db" {}`,
		}

		if err := repo.Create(ctx, res); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.GetByID(ctx, res.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.Name != "user-db" {
			t.Errorf("Name = %q, want %q", got.Name, "user-db")
		}
		if got.Kind != domain.ResourceDatabase {
			t.Errorf("Kind = %q, want %q", got.Kind, domain.ResourceDatabase)
		}
		if mapping, ok := got.ProviderMappings[domain.ProviderAWS]; !ok {
			t.Error("missing AWS provider mapping")
		} else if mapping.ServiceName != "RDS" {
			t.Errorf("ServiceName = %q, want %q", mapping.ServiceName, "RDS")
		}
	})

	t.Run("GetByID not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		if err != domain.ErrNotFound {
			t.Errorf("got %v, want ErrNotFound", err)
		}
	})

	t.Run("ListByApplicationID", func(t *testing.T) {
		// Add a second resource
		res2 := domain.NewResource(app.ID, domain.ResourceCache, "session-cache", json.RawMessage(`{"engine":"redis"}`))
		if err := repo.Create(ctx, res2); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resources, err := repo.ListByApplicationID(ctx, app.ID)
		if err != nil {
			t.Fatalf("ListByApplicationID() error = %v", err)
		}
		if len(resources) != 2 {
			t.Errorf("len = %d, want 2", len(resources))
		}
	})

	t.Run("ListByApplicationID empty", func(t *testing.T) {
		resources, err := repo.ListByApplicationID(ctx, uuid.New())
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(resources) != 0 {
			t.Errorf("len = %d, want 0", len(resources))
		}
	})

	t.Run("Update", func(t *testing.T) {
		res := domain.NewResource(app.ID, domain.ResourceStorage, "uploads", json.RawMessage(`{}`))
		if err := repo.Create(ctx, res); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		res.Name = "media-uploads"
		res.ProviderMappings[domain.ProviderGCP] = domain.ProviderResource{
			ServiceName: "Cloud Storage",
		}
		if err := repo.Update(ctx, res); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		got, _ := repo.GetByID(ctx, res.ID)
		if got.Name != "media-uploads" {
			t.Errorf("Name = %q, want %q", got.Name, "media-uploads")
		}
		if _, ok := got.ProviderMappings[domain.ProviderGCP]; !ok {
			t.Error("missing GCP provider mapping after update")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		res := domain.NewResource(app.ID, domain.ResourceQueue, "job-queue", json.RawMessage(`{}`))
		if err := repo.Create(ctx, res); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := repo.Delete(ctx, res.ID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		_, err := repo.GetByID(ctx, res.ID)
		if err != domain.ErrNotFound {
			t.Errorf("after Delete: got %v, want ErrNotFound", err)
		}
	})
}
