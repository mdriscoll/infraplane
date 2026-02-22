package domain

import (
	"time"

	"github.com/google/uuid"
)

// AppStatus represents the lifecycle state of an application.
type AppStatus string

const (
	AppStatusDraft       AppStatus = "draft"
	AppStatusProvisioned AppStatus = "provisioned"
	AppStatusDeployed    AppStatus = "deployed"
)

// Application represents a registered application in Infraplane.
type Application struct {
	ID                   uuid.UUID     `json:"id"`
	Name                 string        `json:"name"`
	Description          string        `json:"description"`
	GitRepoURL           string        `json:"git_repo_url"`
	SourcePath           string        `json:"source_path"`
	Provider             CloudProvider `json:"provider"`
	Status               AppStatus     `json:"status"`
	ComplianceFrameworks []string      `json:"compliance_frameworks"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

// NewApplication creates a new Application in draft status.
func NewApplication(name, description, gitRepoURL, sourcePath string, provider CloudProvider) Application {
	now := time.Now().UTC()
	return Application{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		GitRepoURL:  gitRepoURL,
		SourcePath:  sourcePath,
		Provider:    provider,
		Status:      AppStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate checks that the application has valid required fields.
func (a Application) Validate() error {
	if a.Name == "" {
		return ErrValidation("application name is required")
	}
	if !a.Provider.IsValid() {
		return ErrValidation("invalid cloud provider: " + a.Provider.String())
	}
	return nil
}
