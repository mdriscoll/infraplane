package gcp

import (
	"context"
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestAdapter_Provider(t *testing.T) {
	a := NewAdapter(&Config{Project: "test", Region: "us-central1"})
	if a.Provider() != domain.ProviderGCP {
		t.Errorf("provider = %v, want gcp", a.Provider())
	}
}

func TestAdapter_ValidateCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("missing project", func(t *testing.T) {
		a := NewAdapter(&Config{Region: "us-central1", CredentialsFile: "/path/to/creds.json"})
		err := a.ValidateCredentials(ctx)
		if err == nil {
			t.Error("expected error for missing project")
		}
	})

	t.Run("missing credentials file", func(t *testing.T) {
		a := NewAdapter(&Config{Project: "my-project", Region: "us-central1"})
		err := a.ValidateCredentials(ctx)
		if err == nil {
			t.Error("expected error for missing credentials")
		}
	})

	t.Run("valid credentials", func(t *testing.T) {
		a := NewAdapter(&Config{
			Project:         "my-project",
			Region:          "us-central1",
			CredentialsFile: "/path/to/credentials.json",
		})
		err := a.ValidateCredentials(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestAdapter_ApplyTerraform(t *testing.T) {
	ctx := context.Background()
	a := NewAdapter(&Config{
		Project:         "my-project",
		Region:          "us-central1",
		CredentialsFile: "/path/to/credentials.json",
	})

	t.Run("successful apply", func(t *testing.T) {
		result, err := a.ApplyTerraform(ctx, `resource "google_sql_database_instance" "db" {}`)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result == "" {
			t.Error("expected non-empty plan output")
		}
	})

	t.Run("empty HCL", func(t *testing.T) {
		_, err := a.ApplyTerraform(ctx, "")
		if err == nil {
			t.Error("expected error for empty HCL")
		}
	})

	t.Run("no credentials", func(t *testing.T) {
		noCreds := NewAdapter(&Config{Region: "us-central1"})
		_, err := noCreds.ApplyTerraform(ctx, `resource "google_sql_database_instance" "db" {}`)
		if err == nil {
			t.Error("expected error for missing credentials")
		}
	})
}

func TestAdapter_DestroyTerraform(t *testing.T) {
	ctx := context.Background()
	a := NewAdapter(&Config{
		Project:         "my-project",
		Region:          "us-central1",
		CredentialsFile: "/path/to/credentials.json",
	})

	t.Run("successful destroy", func(t *testing.T) {
		err := a.DestroyTerraform(ctx, `resource "google_sql_database_instance" "db" {}`)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty HCL", func(t *testing.T) {
		err := a.DestroyTerraform(ctx, "")
		if err == nil {
			t.Error("expected error for empty HCL")
		}
	})
}
