package provider

import (
	"context"
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// stubAdapter is a minimal CloudProviderAdapter for testing.
type stubAdapter struct {
	provider           domain.CloudProvider
	validateErr        error
	applyResult        string
	applyErr           error
	destroyErr         error
}

func (s *stubAdapter) Provider() domain.CloudProvider                              { return s.provider }
func (s *stubAdapter) ValidateCredentials(ctx context.Context) error               { return s.validateErr }
func (s *stubAdapter) ApplyTerraform(ctx context.Context, hcl string) (string, error) {
	return s.applyResult, s.applyErr
}
func (s *stubAdapter) DestroyTerraform(ctx context.Context, hcl string) error { return s.destroyErr }

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	aws := &stubAdapter{provider: domain.ProviderAWS, applyResult: "plan-ok"}
	gcp := &stubAdapter{provider: domain.ProviderGCP, applyResult: "plan-ok"}

	reg.Register(aws)
	reg.Register(gcp)

	t.Run("get registered adapter", func(t *testing.T) {
		adapter, err := reg.Get(domain.ProviderAWS)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if adapter.Provider() != domain.ProviderAWS {
			t.Errorf("provider = %v, want aws", adapter.Provider())
		}
	})

	t.Run("get unknown provider", func(t *testing.T) {
		_, err := reg.Get(domain.CloudProvider("azure"))
		if err == nil {
			t.Error("expected error for unknown provider")
		}
	})

	t.Run("providers list", func(t *testing.T) {
		providers := reg.Providers()
		if len(providers) != 2 {
			t.Errorf("len = %d, want 2", len(providers))
		}
	})
}
