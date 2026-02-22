package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	tfexec "github.com/matthewdriscoll/infraplane/internal/terraform"
)

// Adapter implements CloudProviderAdapter for AWS.
type Adapter struct {
	executor *tfexec.Executor
}

// NewAdapter creates a new AWS adapter backed by a real terraform executor.
func NewAdapter(executor *tfexec.Executor) *Adapter {
	return &Adapter{executor: executor}
}

// Provider returns the cloud provider this adapter handles.
func (a *Adapter) Provider() domain.CloudProvider {
	return domain.ProviderAWS
}

// ValidateCredentials uses AWS STS GetCallerIdentity to verify credentials.
// If a role ARN is specified in the target, it attempts to assume the role.
func (a *Adapter) ValidateCredentials(ctx context.Context, target *domain.DeployTarget) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	if target != nil && target.AWSRegion != "" {
		cfg.Region = target.AWSRegion
	}

	stsClient := sts.NewFromConfig(cfg)

	if target != nil && target.AWSRoleARN != "" {
		// Try to assume the role
		assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, target.AWSRoleARN)
		cfg.Credentials = assumeRoleProvider
		stsClient = sts.NewFromConfig(cfg)
	}

	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("AWS credential validation failed: %w", err)
	}

	// Optionally verify account ID matches
	if target != nil && target.AWSAccountID != "" && result.Account != nil {
		if *result.Account != target.AWSAccountID {
			return fmt.Errorf("AWS account mismatch: expected %s, got %s", target.AWSAccountID, *result.Account)
		}
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
		Provider:  domain.ProviderAWS,
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
		Provider: domain.ProviderAWS,
	})
}
