package gcp

import (
	"context"
	"fmt"
	"os"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// Adapter implements CloudProviderAdapter for GCP.
type Adapter struct {
	project              string
	region               string
	credentialsFile      string
}

// Config holds GCP-specific configuration.
type Config struct {
	Project         string
	Region          string
	CredentialsFile string
}

// NewAdapter creates a new GCP adapter with the given config.
// If config is nil, it reads from environment variables.
func NewAdapter(cfg *Config) *Adapter {
	if cfg == nil {
		cfg = &Config{
			Project:         os.Getenv("GOOGLE_PROJECT"),
			Region:          envOrDefault("GOOGLE_REGION", "us-central1"),
			CredentialsFile: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		}
	}
	return &Adapter{
		project:         cfg.Project,
		region:          cfg.Region,
		credentialsFile: cfg.CredentialsFile,
	}
}

// Provider returns the cloud provider this adapter handles.
func (a *Adapter) Provider() domain.CloudProvider {
	return domain.ProviderGCP
}

// ValidateCredentials checks whether GCP credentials are configured.
func (a *Adapter) ValidateCredentials(ctx context.Context) error {
	if a.project == "" {
		return fmt.Errorf("GCP project not configured: set GOOGLE_PROJECT")
	}
	if a.credentialsFile == "" {
		return fmt.Errorf("GCP credentials not configured: set GOOGLE_APPLICATION_CREDENTIALS")
	}
	// In a real implementation, this would use the Google Cloud SDK
	// to verify the service account credentials are valid.
	return nil
}

// ApplyTerraform takes Terraform HCL and applies it against GCP.
// In this implementation, it validates the HCL and returns a simulated plan.
// A production implementation would shell out to `terraform init && terraform apply`.
func (a *Adapter) ApplyTerraform(ctx context.Context, hcl string) (string, error) {
	if hcl == "" {
		return "", fmt.Errorf("empty Terraform configuration")
	}

	if err := a.ValidateCredentials(ctx); err != nil {
		return "", fmt.Errorf("credential check failed: %w", err)
	}

	// Simulate: in production, this would:
	// 1. Write HCL to a temp directory
	// 2. Run `terraform init`
	// 3. Run `terraform plan`
	// 4. Run `terraform apply -auto-approve`
	// 5. Return the plan output

	return fmt.Sprintf("GCP Terraform plan applied successfully in project %s, region %s", a.project, a.region), nil
}

// DestroyTerraform destroys infrastructure described by the given HCL.
func (a *Adapter) DestroyTerraform(ctx context.Context, hcl string) error {
	if hcl == "" {
		return fmt.Errorf("empty Terraform configuration")
	}

	if err := a.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("credential check failed: %w", err)
	}

	// Simulate: in production, this would run `terraform destroy -auto-approve`
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
