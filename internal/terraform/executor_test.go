package terraform

import (
	"os"
	"testing"
	"time"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestNewExecutor_Defaults(t *testing.T) {
	e := NewExecutor("", 0)
	if e.binaryPath != "terraform" {
		t.Errorf("expected binary 'terraform', got %q", e.binaryPath)
	}
	if e.timeout != 10*time.Minute {
		t.Errorf("expected 10m timeout, got %v", e.timeout)
	}
}

func TestNewExecutor_Custom(t *testing.T) {
	e := NewExecutor("/usr/local/bin/terraform", 5*time.Minute)
	if e.binaryPath != "/usr/local/bin/terraform" {
		t.Errorf("expected custom binary path, got %q", e.binaryPath)
	}
	if e.timeout != 5*time.Minute {
		t.Errorf("expected 5m timeout, got %v", e.timeout)
	}
}

func TestBuildEnvVars_AWS(t *testing.T) {
	e := NewExecutor("", 0)

	target := &domain.DeployTarget{
		AWSRegion:    "eu-west-1",
		AWSAccountID: "123456789012",
		AWSRoleARN:   "arn:aws:iam::123456789012:role/deploy",
	}

	envs := e.buildEnvVars(domain.ProviderAWS, target)

	found := false
	for _, env := range envs {
		if env == "AWS_DEFAULT_REGION=eu-west-1" {
			found = true
		}
	}
	if !found {
		t.Error("expected AWS_DEFAULT_REGION in env vars")
	}
}

func TestBuildEnvVars_GCP(t *testing.T) {
	e := NewExecutor("", 0)

	target := &domain.DeployTarget{
		GCPProjectID: "my-project-123",
		GCPRegion:    "europe-west1",
	}

	envs := e.buildEnvVars(domain.ProviderGCP, target)

	hasProject := false
	hasRegion := false
	for _, env := range envs {
		if env == "GOOGLE_PROJECT=my-project-123" {
			hasProject = true
		}
		if env == "GOOGLE_REGION=europe-west1" {
			hasRegion = true
		}
	}
	if !hasProject {
		t.Error("expected GOOGLE_PROJECT in env vars")
	}
	if !hasRegion {
		t.Error("expected GOOGLE_REGION in env vars")
	}
}

func TestBuildEnvVars_NilTarget(t *testing.T) {
	e := NewExecutor("", 0)
	envs := e.buildEnvVars(domain.ProviderAWS, nil)
	if len(envs) != 0 {
		t.Errorf("expected empty env vars for nil target, got %v", envs)
	}
}

func TestExecute_WritesTempFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode — requires terraform binary")
	}

	// This test verifies the temp directory and file creation logic
	// without actually requiring terraform to be installed.
	// We use a fake binary that just echoes.
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a fake terraform script
	fakeTF := tmpDir + "/fake-terraform"
	err = os.WriteFile(fakeTF, []byte("#!/bin/sh\necho \"fake terraform $@\"\n"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	e := NewExecutor(fakeTF, 30*time.Second)

	var lines []string
	sink := func(line string) {
		lines = append(lines, line)
	}

	result, err := e.Execute(t.Context(), ExecuteOpts{
		HCL:       "resource \"null_resource\" \"test\" {}",
		Provider:  domain.ProviderAWS,
		Target:    &domain.DeployTarget{AWSRegion: "us-west-2"},
		EventSink: sink,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if len(lines) == 0 {
		t.Error("expected at least some output lines")
	}
}
