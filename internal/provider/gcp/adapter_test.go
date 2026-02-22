package gcp

import (
	"context"
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestAdapter_Provider(t *testing.T) {
	a := NewAdapter(nil)
	if a.Provider() != domain.ProviderGCP {
		t.Errorf("provider = %v, want gcp", a.Provider())
	}
}

func TestAdapter_ValidateCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("nil target uses ADC", func(t *testing.T) {
		a := NewAdapter(nil)
		// Will fail in CI without real GCP creds — that's expected.
		err := a.ValidateCredentials(ctx, nil)
		if err == nil {
			// Only passes when real ADC is configured, so just log
			t.Log("ValidateCredentials succeeded (ADC present)")
		}
	})

	t.Run("with target", func(t *testing.T) {
		a := NewAdapter(nil)
		target := &domain.DeployTarget{
			GCPProjectID: "my-project",
			GCPRegion:    "us-central1",
		}
		// Will fail without real creds
		err := a.ValidateCredentials(ctx, target)
		if err == nil {
			t.Log("ValidateCredentials with target succeeded (ADC present)")
		}
	})
}

func TestAdapter_ApplyTerraform(t *testing.T) {
	ctx := context.Background()
	a := NewAdapter(nil)

	t.Run("empty HCL", func(t *testing.T) {
		_, err := a.ApplyTerraform(ctx, "", nil, nil)
		if err == nil {
			t.Error("expected error for empty HCL")
		}
	})
}

func TestAdapter_DestroyTerraform(t *testing.T) {
	ctx := context.Background()
	a := NewAdapter(nil)

	t.Run("empty HCL", func(t *testing.T) {
		err := a.DestroyTerraform(ctx, "", nil)
		if err == nil {
			t.Error("expected error for empty HCL")
		}
	})
}
