package domain

import (
	"time"

	"github.com/google/uuid"
)

// DeploymentStatus represents the state of a deployment.
type DeploymentStatus string

const (
	DeploymentPending    DeploymentStatus = "pending"
	DeploymentInProgress DeploymentStatus = "in_progress"
	DeploymentSucceeded  DeploymentStatus = "succeeded"
	DeploymentFailed     DeploymentStatus = "failed"
)

// DeployTarget holds provider-specific deployment target configuration.
// It is stored as JSONB on the deployment record.
type DeployTarget struct {
	// AWS fields
	AWSRegion    string `json:"aws_region,omitempty"`
	AWSAccountID string `json:"aws_account_id,omitempty"`
	AWSRoleARN   string `json:"aws_role_arn,omitempty"`

	// GCP fields
	GCPProjectID string `json:"gcp_project_id,omitempty"`
	GCPRegion    string `json:"gcp_region,omitempty"`
	GCPFolder    string `json:"gcp_folder,omitempty"`
}

// Validate checks that provider-specific required fields are present.
func (t DeployTarget) Validate(provider CloudProvider) error {
	switch provider {
	case ProviderAWS:
		if t.AWSRegion == "" {
			return ErrValidation("AWS region is required")
		}
	case ProviderGCP:
		if t.GCPProjectID == "" {
			return ErrValidation("GCP project ID is required")
		}
		if t.GCPRegion == "" {
			return ErrValidation("GCP region is required")
		}
	}
	return nil
}

// Deployment represents a deployment event for an application.
type Deployment struct {
	ID            uuid.UUID        `json:"id"`
	ApplicationID uuid.UUID        `json:"application_id"`
	PlanID        *uuid.UUID       `json:"plan_id,omitempty"`
	Provider      CloudProvider    `json:"provider"`
	GitCommit     string           `json:"git_commit"`
	GitBranch     string           `json:"git_branch"`
	Status        DeploymentStatus `json:"status"`
	TerraformPlan string           `json:"terraform_plan,omitempty"`
	DeployTarget  *DeployTarget    `json:"deploy_target,omitempty"`
	StartedAt     time.Time        `json:"started_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
}

// NewDeployment creates a new pending deployment.
func NewDeployment(appID uuid.UUID, provider CloudProvider, gitCommit, gitBranch string, planID *uuid.UUID, target *DeployTarget) Deployment {
	return Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		PlanID:        planID,
		Provider:      provider,
		GitCommit:     gitCommit,
		GitBranch:     gitBranch,
		Status:        DeploymentPending,
		DeployTarget:  target,
		StartedAt:     time.Now().UTC(),
	}
}

// DeploymentStep represents a named stage in the deployment pipeline.
type DeploymentStep string

const (
	StepInitializing            DeploymentStep = "initializing"
	StepGeneratingTerraform     DeploymentStep = "generating_terraform"
	StepValidatingCredentials   DeploymentStep = "validating_credentials"
	StepValidating              DeploymentStep = "validating"
	StepApplying                DeploymentStep = "applying"
	StepComplete                DeploymentStep = "complete"
	StepFailed                  DeploymentStep = "failed"
)

// DeploymentEvent is a single log entry streamed to the client during deployment execution.
type DeploymentEvent struct {
	Step      DeploymentStep   `json:"step"`
	Message   string           `json:"message"`
	Timestamp time.Time        `json:"timestamp"`
	Status    DeploymentStatus `json:"status"`
	Detail    string           `json:"detail,omitempty"`
}

// Validate checks that the deployment has valid required fields.
func (d Deployment) Validate() error {
	if d.ApplicationID == uuid.Nil {
		return ErrValidation("application ID is required")
	}
	if !d.Provider.IsValid() {
		return ErrValidation("invalid cloud provider: " + d.Provider.String())
	}
	if d.GitBranch == "" {
		return ErrValidation("git branch is required")
	}
	if d.DeployTarget != nil {
		if err := d.DeployTarget.Validate(d.Provider); err != nil {
			return err
		}
	}
	return nil
}
