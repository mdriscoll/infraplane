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
	StartedAt     time.Time        `json:"started_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
}

// NewDeployment creates a new pending deployment.
func NewDeployment(appID uuid.UUID, provider CloudProvider, gitCommit, gitBranch string, planID *uuid.UUID) Deployment {
	return Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		PlanID:        planID,
		Provider:      provider,
		GitCommit:     gitCommit,
		GitBranch:     gitBranch,
		Status:        DeploymentPending,
		StartedAt:     time.Now().UTC(),
	}
}

// DeploymentStep represents a named stage in the deployment pipeline.
type DeploymentStep string

const (
	StepInitializing        DeploymentStep = "initializing"
	StepGeneratingTerraform DeploymentStep = "generating_terraform"
	StepValidating          DeploymentStep = "validating"
	StepApplying            DeploymentStep = "applying"
	StepComplete            DeploymentStep = "complete"
	StepFailed              DeploymentStep = "failed"
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
	return nil
}
