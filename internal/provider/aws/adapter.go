package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// Adapter implements CloudProviderAdapter for AWS.
type Adapter struct {
	region          string
	accessKeyID     string
	secretAccessKey string
}

// Config holds AWS-specific configuration.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

// NewAdapter creates a new AWS adapter with the given config.
// If config is nil, it reads from environment variables.
func NewAdapter(cfg *Config) *Adapter {
	if cfg == nil {
		cfg = &Config{
			Region:          envOrDefault("AWS_REGION", "us-east-1"),
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}
	}
	return &Adapter{
		region:          cfg.Region,
		accessKeyID:     cfg.AccessKeyID,
		secretAccessKey: cfg.SecretAccessKey,
	}
}

// Provider returns the cloud provider this adapter handles.
func (a *Adapter) Provider() domain.CloudProvider {
	return domain.ProviderAWS
}

// ValidateCredentials checks whether AWS credentials are configured.
func (a *Adapter) ValidateCredentials(ctx context.Context) error {
	if a.accessKeyID == "" || a.secretAccessKey == "" {
		return fmt.Errorf("AWS credentials not configured: set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
	}
	// In a real implementation, this would call AWS STS GetCallerIdentity
	// to verify the credentials are valid and not expired.
	return nil
}

// ApplyTerraform takes Terraform HCL and applies it against AWS.
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

	return fmt.Sprintf("AWS Terraform plan applied successfully in region %s", a.region), nil
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
