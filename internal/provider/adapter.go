package provider

import (
	"context"
	"fmt"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// CloudProviderAdapter defines operations that each cloud provider must implement.
type CloudProviderAdapter interface {
	// Provider returns the cloud provider this adapter handles.
	Provider() domain.CloudProvider

	// ValidateCredentials checks whether the provided credentials are valid.
	ValidateCredentials(ctx context.Context) error

	// ApplyTerraform takes generated Terraform HCL and applies it.
	// Returns the Terraform plan output or an error.
	ApplyTerraform(ctx context.Context, hcl string) (string, error)

	// DestroyTerraform destroys infrastructure described by the given HCL.
	DestroyTerraform(ctx context.Context, hcl string) error
}

// Registry holds adapters for each cloud provider.
type Registry struct {
	adapters map[domain.CloudProvider]CloudProviderAdapter
}

// NewRegistry creates a new empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[domain.CloudProvider]CloudProviderAdapter),
	}
}

// Register adds a provider adapter to the registry.
func (r *Registry) Register(adapter CloudProviderAdapter) {
	r.adapters[adapter.Provider()] = adapter
}

// Get returns the adapter for a given provider.
func (r *Registry) Get(provider domain.CloudProvider) (CloudProviderAdapter, error) {
	adapter, ok := r.adapters[provider]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for provider: %s", provider)
	}
	return adapter, nil
}

// Providers returns all registered provider names.
func (r *Registry) Providers() []domain.CloudProvider {
	providers := make([]domain.CloudProvider, 0, len(r.adapters))
	for p := range r.adapters {
		providers = append(providers, p)
	}
	return providers
}
