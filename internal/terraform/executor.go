package terraform

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// Executor runs real Terraform CLI commands (init, plan, apply) in a temp workspace.
type Executor struct {
	binaryPath string
	timeout    time.Duration
}

// NewExecutor creates a new Terraform executor.
// binaryPath is the path to the terraform binary (default "terraform").
// timeout is the per-command timeout (default 10 minutes).
func NewExecutor(binaryPath string, timeout time.Duration) *Executor {
	if binaryPath == "" {
		binaryPath = "terraform"
	}
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	return &Executor{
		binaryPath: binaryPath,
		timeout:    timeout,
	}
}

// ExecuteOpts configures a terraform execution run.
type ExecuteOpts struct {
	HCL       string
	Target    *domain.DeployTarget
	Provider  domain.CloudProvider
	EventSink func(line string) // callback for each stdout/stderr line
}

// ExecuteResult holds the output of a terraform execution.
type ExecuteResult struct {
	PlanOutput  string
	ApplyOutput string
	Success     bool
}

// Execute runs terraform init → plan → apply in a temporary directory.
// It streams stdout/stderr line-by-line via opts.EventSink.
func (e *Executor) Execute(ctx context.Context, opts ExecuteOpts) (*ExecuteResult, error) {
	// Create temp workspace
	tmpDir, err := os.MkdirTemp("", "infraplane-deploy-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write main.tf
	tfFile := tmpDir + "/main.tf"
	if err := os.WriteFile(tfFile, []byte(opts.HCL), 0644); err != nil {
		return nil, fmt.Errorf("write main.tf: %w", err)
	}

	// Build environment variables for the provider
	envVars := e.buildEnvVars(opts.Provider, opts.Target)

	sink := opts.EventSink
	if sink == nil {
		sink = func(string) {} // no-op
	}

	// 1. terraform init
	sink("Running terraform init...")
	initOut, err := e.runCommand(ctx, tmpDir, envVars, sink, "init", "-no-color")
	if err != nil {
		return &ExecuteResult{Success: false}, fmt.Errorf("terraform init failed: %w\n%s", err, initOut)
	}
	sink("Terraform init completed.")

	// 2. terraform plan
	sink("Running terraform plan...")
	planOut, err := e.runCommand(ctx, tmpDir, envVars, sink, "plan", "-no-color", "-out=tfplan")
	if err != nil {
		return &ExecuteResult{PlanOutput: planOut, Success: false}, fmt.Errorf("terraform plan failed: %w\n%s", err, planOut)
	}
	sink("Terraform plan completed.")

	// 3. terraform apply
	sink("Running terraform apply...")
	applyOut, err := e.runCommand(ctx, tmpDir, envVars, sink, "apply", "-no-color", "-auto-approve", "tfplan")
	if err != nil {
		return &ExecuteResult{PlanOutput: planOut, ApplyOutput: applyOut, Success: false}, fmt.Errorf("terraform apply failed: %w\n%s", err, applyOut)
	}
	sink("Terraform apply completed successfully.")

	return &ExecuteResult{
		PlanOutput:  planOut,
		ApplyOutput: applyOut,
		Success:     true,
	}, nil
}

// Destroy runs terraform init → destroy in a temporary directory.
func (e *Executor) Destroy(ctx context.Context, opts ExecuteOpts) error {
	tmpDir, err := os.MkdirTemp("", "infraplane-destroy-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tfFile := tmpDir + "/main.tf"
	if err := os.WriteFile(tfFile, []byte(opts.HCL), 0644); err != nil {
		return fmt.Errorf("write main.tf: %w", err)
	}

	envVars := e.buildEnvVars(opts.Provider, opts.Target)
	sink := opts.EventSink
	if sink == nil {
		sink = func(string) {}
	}

	// init
	if _, err := e.runCommand(ctx, tmpDir, envVars, sink, "init", "-no-color"); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// destroy
	if _, err := e.runCommand(ctx, tmpDir, envVars, sink, "destroy", "-no-color", "-auto-approve"); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}

// runCommand runs a single terraform sub-command and streams its output.
func (e *Executor) runCommand(ctx context.Context, dir string, envVars []string, sink func(string), args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	fullArgs := append([]string{"-chdir=" + dir}, args...)
	cmd := exec.CommandContext(ctx, e.binaryPath, fullArgs...)
	cmd.Env = append(os.Environ(), envVars...)

	// Combine stdout and stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start command: %w", err)
	}

	var output strings.Builder
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
		sink(line)
	}

	if err := cmd.Wait(); err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

// buildEnvVars constructs provider-specific environment variables for the terraform process.
func (e *Executor) buildEnvVars(provider domain.CloudProvider, target *domain.DeployTarget) []string {
	var envs []string

	if target == nil {
		return envs
	}

	switch provider {
	case domain.ProviderAWS:
		if target.AWSRegion != "" {
			envs = append(envs, "AWS_DEFAULT_REGION="+target.AWSRegion)
		}
		// AWS credentials are expected to come from the environment or instance profile.
		// If a role ARN is specified, the Terraform provider's assume_role block handles it.

	case domain.ProviderGCP:
		if target.GCPProjectID != "" {
			envs = append(envs, "GOOGLE_PROJECT="+target.GCPProjectID)
		}
		if target.GCPRegion != "" {
			envs = append(envs, "GOOGLE_REGION="+target.GCPRegion)
		}
		// GCP credentials are expected via GOOGLE_APPLICATION_CREDENTIALS or gcloud auth.
	}

	return envs
}
