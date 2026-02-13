package llm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestBuildResourceAnalysisPrompt(t *testing.T) {
	prompt := buildResourceAnalysisPrompt("I need a PostgreSQL database for user data", domain.ProviderAWS)

	if !strings.Contains(prompt, "PostgreSQL database") {
		t.Error("prompt should contain the user's description")
	}
	if !strings.Contains(prompt, "aws") {
		t.Error("prompt should contain the preferred provider")
	}
}

func TestBuildHostingPlanPrompt(t *testing.T) {
	app := domain.Application{
		ID:          uuid.New(),
		Name:        "my-api",
		Description: "A REST API for widgets",
		GitRepoURL:  "https://github.com/test/repo",
		Provider:    domain.ProviderAWS,
	}

	resources := []domain.Resource{
		{
			ID:            uuid.New(),
			ApplicationID: app.ID,
			Kind:          domain.ResourceDatabase,
			Name:          "user-db",
			Spec:          json.RawMessage(`{"engine": "postgres"}`),
			ProviderMappings: map[domain.CloudProvider]domain.ProviderResource{
				domain.ProviderAWS: {ServiceName: "RDS", Config: map[string]any{"instance_class": "db.t3.micro"}},
			},
		},
	}

	prompt := buildHostingPlanPrompt(app, resources)

	if !strings.Contains(prompt, "my-api") {
		t.Error("prompt should contain app name")
	}
	if !strings.Contains(prompt, "user-db") {
		t.Error("prompt should contain resource name")
	}
	if !strings.Contains(prompt, "database") {
		t.Error("prompt should contain resource kind")
	}
	if !strings.Contains(prompt, "RDS") {
		t.Error("prompt should contain provider mapping")
	}
}

func TestBuildHostingPlanPrompt_NoResources(t *testing.T) {
	app := domain.Application{
		Name:     "empty-app",
		Provider: domain.ProviderGCP,
	}

	prompt := buildHostingPlanPrompt(app, nil)

	if !strings.Contains(prompt, "No resources defined") {
		t.Error("prompt should indicate no resources")
	}
}

func TestBuildMigrationPlanPrompt(t *testing.T) {
	app := domain.Application{
		ID:          uuid.New(),
		Name:        "migrate-me",
		Description: "App to migrate",
		Provider:    domain.ProviderAWS,
	}

	resources := []domain.Resource{
		{
			Kind: domain.ResourceDatabase,
			Name: "main-db",
			Spec: json.RawMessage(`{"engine": "postgres"}`),
			ProviderMappings: map[domain.CloudProvider]domain.ProviderResource{
				domain.ProviderAWS: {ServiceName: "RDS"},
				domain.ProviderGCP: {ServiceName: "Cloud SQL"},
			},
		},
	}

	prompt := buildMigrationPlanPrompt(app, resources, domain.ProviderAWS, domain.ProviderGCP)

	if !strings.Contains(prompt, "migrate-me") {
		t.Error("prompt should contain app name")
	}
	if !strings.Contains(prompt, "FROM: aws") {
		t.Error("prompt should contain source provider")
	}
	if !strings.Contains(prompt, "TO: gcp") {
		t.Error("prompt should contain target provider")
	}
	if !strings.Contains(prompt, "RDS") {
		t.Error("prompt should contain current provider mapping")
	}
	if !strings.Contains(prompt, "Cloud SQL") {
		t.Error("prompt should contain target provider mapping")
	}
}
