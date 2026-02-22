package aws

import (
	"context"
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestAdapter_Provider(t *testing.T) {
	a := NewAdapter(nil)
	if a.Provider() != domain.ProviderAWS {
		t.Errorf("provider = %v, want aws", a.Provider())
	}
}

func TestAdapter_ValidateCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("nil target uses default config", func(t *testing.T) {
		a := NewAdapter(nil)
		// Will fail without real AWS creds
		err := a.ValidateCredentials(ctx, nil)
		if err == nil {
			t.Log("ValidateCredentials succeeded (AWS creds present)")
		}
	})

	t.Run("with target", func(t *testing.T) {
		a := NewAdapter(nil)
		target := &domain.DeployTarget{
			AWSRegion: "us-east-1",
		}
		// Will fail without real AWS creds
		err := a.ValidateCredentials(ctx, target)
		if err == nil {
			t.Log("ValidateCredentials with target succeeded (AWS creds present)")
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
