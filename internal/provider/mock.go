package provider

import (
	"context"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// MockAdapter is a mock CloudProviderAdapter for testing.
type MockAdapter struct {
	ProviderVal        domain.CloudProvider
	ValidateErr        error
	ApplyResult        string
	ApplyErr           error
	DestroyErr         error
	ApplyTerraformFn   func(ctx context.Context, hcl string) (string, error)
}

// Provider returns the mock provider.
func (m *MockAdapter) Provider() domain.CloudProvider {
	return m.ProviderVal
}

// ValidateCredentials returns the configured error.
func (m *MockAdapter) ValidateCredentials(ctx context.Context) error {
	return m.ValidateErr
}

// ApplyTerraform returns the configured result or calls the custom function.
func (m *MockAdapter) ApplyTerraform(ctx context.Context, hcl string) (string, error) {
	if m.ApplyTerraformFn != nil {
		return m.ApplyTerraformFn(ctx, hcl)
	}
	return m.ApplyResult, m.ApplyErr
}

// DestroyTerraform returns the configured error.
func (m *MockAdapter) DestroyTerraform(ctx context.Context, hcl string) error {
	return m.DestroyErr
}
