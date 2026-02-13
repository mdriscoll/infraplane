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
	Provider      CloudProvider    `json:"provider"`
	GitCommit     string           `json:"git_commit"`
	GitBranch     string           `json:"git_branch"`
	Status        DeploymentStatus `json:"status"`
	TerraformPlan string           `json:"terraform_plan,omitempty"`
	StartedAt     time.Time        `json:"started_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
}

// NewDeployment creates a new pending deployment.
func NewDeployment(appID uuid.UUID, provider CloudProvider, gitCommit, gitBranch string) Deployment {
	return Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		Provider:      provider,
		GitCommit:     gitCommit,
		GitBranch:     gitBranch,
		Status:        DeploymentPending,
		StartedAt:     time.Now().UTC(),
	}
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
