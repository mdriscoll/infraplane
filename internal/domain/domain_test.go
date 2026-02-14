package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestCloudProvider_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		provider CloudProvider
		want     bool
	}{
		{"aws is valid", ProviderAWS, true},
		{"gcp is valid", ProviderGCP, true},
		{"empty is invalid", CloudProvider(""), false},
		{"azure is invalid", CloudProvider("azure"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewApplication(t *testing.T) {
	app := NewApplication("myapp", "A test app", "https://github.com/test/repo", "", ProviderAWS)

	if app.Name != "myapp" {
		t.Errorf("Name = %q, want %q", app.Name, "myapp")
	}
	if app.Status != AppStatusDraft {
		t.Errorf("Status = %q, want %q", app.Status, AppStatusDraft)
	}
	if app.ID == uuid.Nil {
		t.Error("ID should not be nil")
	}
	if app.Provider != ProviderAWS {
		t.Errorf("Provider = %q, want %q", app.Provider, ProviderAWS)
	}
}

func TestApplication_Validate(t *testing.T) {
	tests := []struct {
		name    string
		app     Application
		wantErr bool
	}{
		{
			name:    "valid application",
			app:     NewApplication("myapp", "desc", "https://github.com/test/repo", "", ProviderAWS),
			wantErr: false,
		},
		{
			name: "missing name",
			app: Application{
				ID:       uuid.New(),
				Provider: ProviderAWS,
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			app: Application{
				ID:       uuid.New(),
				Name:     "myapp",
				Provider: CloudProvider("azure"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewResource(t *testing.T) {
	appID := uuid.New()
	spec := json.RawMessage(`{"engine": "postgres"}`)
	r := NewResource(appID, ResourceDatabase, "user-db", spec)

	if r.ApplicationID != appID {
		t.Errorf("ApplicationID = %v, want %v", r.ApplicationID, appID)
	}
	if r.Kind != ResourceDatabase {
		t.Errorf("Kind = %q, want %q", r.Kind, ResourceDatabase)
	}
	if r.Name != "user-db" {
		t.Errorf("Name = %q, want %q", r.Name, "user-db")
	}
	if r.ProviderMappings == nil {
		t.Error("ProviderMappings should be initialized")
	}
}

func TestResource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		r       Resource
		wantErr bool
	}{
		{
			name:    "valid resource",
			r:       NewResource(uuid.New(), ResourceDatabase, "user-db", json.RawMessage(`{}`)),
			wantErr: false,
		},
		{
			name: "missing name",
			r: Resource{
				ID:            uuid.New(),
				ApplicationID: uuid.New(),
				Kind:          ResourceDatabase,
			},
			wantErr: true,
		},
		{
			name: "invalid kind",
			r: Resource{
				ID:            uuid.New(),
				ApplicationID: uuid.New(),
				Kind:          ResourceKind("serverless-magic"),
				Name:          "test",
			},
			wantErr: true,
		},
		{
			name: "missing application ID",
			r: Resource{
				ID:   uuid.New(),
				Kind: ResourceDatabase,
				Name: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewDeployment(t *testing.T) {
	appID := uuid.New()
	d := NewDeployment(appID, ProviderAWS, "abc123", "main")

	if d.ApplicationID != appID {
		t.Errorf("ApplicationID = %v, want %v", d.ApplicationID, appID)
	}
	if d.Status != DeploymentPending {
		t.Errorf("Status = %q, want %q", d.Status, DeploymentPending)
	}
	if d.CompletedAt != nil {
		t.Error("CompletedAt should be nil for new deployment")
	}
}

func TestDeployment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		d       Deployment
		wantErr bool
	}{
		{
			name:    "valid deployment",
			d:       NewDeployment(uuid.New(), ProviderGCP, "abc123", "main"),
			wantErr: false,
		},
		{
			name: "missing app ID",
			d: Deployment{
				Provider:  ProviderAWS,
				GitBranch: "main",
			},
			wantErr: true,
		},
		{
			name: "missing branch",
			d: Deployment{
				ApplicationID: uuid.New(),
				Provider:      ProviderAWS,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.d.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewHostingPlan(t *testing.T) {
	appID := uuid.New()
	p := NewHostingPlan(appID, "Deploy on AWS ECS", nil, nil)

	if p.PlanType != PlanTypeHosting {
		t.Errorf("PlanType = %q, want %q", p.PlanType, PlanTypeHosting)
	}
	if p.FromProvider != nil {
		t.Error("FromProvider should be nil for hosting plan")
	}
}

func TestNewMigrationPlan(t *testing.T) {
	appID := uuid.New()
	p := NewMigrationPlan(appID, ProviderAWS, ProviderGCP, "Migrate from AWS to GCP", nil, nil)

	if p.PlanType != PlanTypeMigration {
		t.Errorf("PlanType = %q, want %q", p.PlanType, PlanTypeMigration)
	}
	if *p.FromProvider != ProviderAWS {
		t.Errorf("FromProvider = %q, want %q", *p.FromProvider, ProviderAWS)
	}
	if *p.ToProvider != ProviderGCP {
		t.Errorf("ToProvider = %q, want %q", *p.ToProvider, ProviderGCP)
	}
}

func TestInfrastructurePlan_Validate(t *testing.T) {
	aws := ProviderAWS
	gcp := ProviderGCP
	tests := []struct {
		name    string
		p       InfrastructurePlan
		wantErr bool
	}{
		{
			name:    "valid hosting plan",
			p:       NewHostingPlan(uuid.New(), "content", nil, nil),
			wantErr: false,
		},
		{
			name:    "valid migration plan",
			p:       NewMigrationPlan(uuid.New(), ProviderAWS, ProviderGCP, "content", nil, nil),
			wantErr: false,
		},
		{
			name: "migration missing from provider",
			p: InfrastructurePlan{
				ID:            uuid.New(),
				ApplicationID: uuid.New(),
				PlanType:      PlanTypeMigration,
				ToProvider:    &gcp,
				Content:       "content",
			},
			wantErr: true,
		},
		{
			name: "migration missing to provider",
			p: InfrastructurePlan{
				ID:            uuid.New(),
				ApplicationID: uuid.New(),
				PlanType:      PlanTypeMigration,
				FromProvider:  &aws,
				Content:       "content",
			},
			wantErr: true,
		},
		{
			name: "missing content",
			p: InfrastructurePlan{
				ID:            uuid.New(),
				ApplicationID: uuid.New(),
				PlanType:      PlanTypeHosting,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	err := ErrValidation("test error")
	if !IsValidationError(err) {
		t.Error("expected IsValidationError to return true")
	}
	if IsValidationError(ErrNotFound) {
		t.Error("expected IsValidationError to return false for ErrNotFound")
	}
}

func TestResourceKind_IsValid(t *testing.T) {
	tests := []struct {
		kind ResourceKind
		want bool
	}{
		{ResourceDatabase, true},
		{ResourceCompute, true},
		{ResourceStorage, true},
		{ResourceCache, true},
		{ResourceQueue, true},
		{ResourceCDN, true},
		{ResourceNetwork, true},
		{ResourceKind("lambda"), false},
		{ResourceKind(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
