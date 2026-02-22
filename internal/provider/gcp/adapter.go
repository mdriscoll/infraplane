package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	tfexec "github.com/matthewdriscoll/infraplane/internal/terraform"
)

// Adapter implements CloudProviderAdapter for GCP.
type Adapter struct {
	executor *tfexec.Executor
}

// NewAdapter creates a new GCP adapter backed by a real terraform executor.
func NewAdapter(executor *tfexec.Executor) *Adapter {
	return &Adapter{executor: executor}
}

// Provider returns the cloud provider this adapter handles.
func (a *Adapter) Provider() domain.CloudProvider {
	return domain.ProviderGCP
}

// ValidateCredentials verifies GCP credentials using Application Default Credentials.
func (a *Adapter) ValidateCredentials(ctx context.Context, target *domain.DeployTarget) error {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("GCP credential validation failed: %w", err)
	}

	if target != nil && target.GCPProjectID != "" {
		// If credentials have a project, verify it matches
		if creds.ProjectID != "" && creds.ProjectID != target.GCPProjectID {
			// This is a warning — the terraform provider block will use the target's project.
			// Credential discovery might return a different project, which is usually fine.
		}
	}

	// Verify we can get a token
	_, err = creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("GCP token validation failed: %w", err)
	}

	return nil
}

// ApplyTerraform runs real terraform init/plan/apply via the executor.
func (a *Adapter) ApplyTerraform(ctx context.Context, hcl string, target *domain.DeployTarget, eventSink func(string)) (string, error) {
	if hcl == "" {
		return "", fmt.Errorf("empty Terraform configuration")
	}

	result, err := a.executor.Execute(ctx, tfexec.ExecuteOpts{
		HCL:       hcl,
		Target:    target,
		Provider:  domain.ProviderGCP,
		EventSink: eventSink,
	})
	if err != nil {
		return "", fmt.Errorf("terraform execution failed: %w", err)
	}

	return result.PlanOutput, nil
}

// DestroyTerraform destroys infrastructure described by the given HCL.
func (a *Adapter) DestroyTerraform(ctx context.Context, hcl string, target *domain.DeployTarget) error {
	if hcl == "" {
		return fmt.Errorf("empty Terraform configuration")
	}

	return a.executor.Destroy(ctx, tfexec.ExecuteOpts{
		HCL:      hcl,
		Target:   target,
		Provider: domain.ProviderGCP,
	})
}
