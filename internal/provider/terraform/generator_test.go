package terraform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func makeTestApp(name string, provider domain.CloudProvider) domain.Application {
	return domain.Application{
		ID:       uuid.New(),
		Name:     name,
		Provider: provider,
	}
}

func makeTestResource(name string, kind domain.ResourceKind, awsHCL, gcpHCL string) domain.Resource {
	return domain.Resource{
		ID:            uuid.New(),
		ApplicationID: uuid.New(),
		Kind:          kind,
		Name:          name,
		Spec:          json.RawMessage(`{}`),
		ProviderMappings: map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderAWS: {
				ServiceName:  "RDS",
				Config:       map[string]any{"instance_class": "db.t3.micro"},
				TerraformHCL: awsHCL,
			},
			domain.ProviderGCP: {
				ServiceName:  "Cloud SQL",
				Config:       map[string]any{"tier": "db-f1-micro"},
				TerraformHCL: gcpHCL,
			},
		},
	}
}

func TestGenerateConfig_AWS(t *testing.T) {
	app := makeTestApp("my-api", domain.ProviderAWS)
	resources := []domain.Resource{
		makeTestResource("user-db", domain.ResourceDatabase,
			`resource "aws_db_instance" "user_db" {
  engine         = "postgres"
  instance_class = "db.t3.micro"
}`,
			`resource "google_sql_database_instance" "user_db" {
  database_version = "POSTGRES_16"
}`,
		),
	}

	config, err := GenerateConfig(app, resources, domain.ProviderAWS)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// Should contain provider block
	if !strings.Contains(config, "hashicorp/aws") {
		t.Error("expected AWS provider block")
	}

	// Should contain resource
	if !strings.Contains(config, `resource "aws_db_instance" "user_db"`) {
		t.Error("expected AWS RDS resource block")
	}

	// Should NOT contain GCP resource
	if strings.Contains(config, "google_sql_database_instance") {
		t.Error("should not contain GCP resources")
	}

	// Should contain header
	if !strings.Contains(config, "my-api") {
		t.Error("expected app name in header")
	}
}

func TestGenerateConfig_GCP(t *testing.T) {
	app := makeTestApp("my-api", domain.ProviderGCP)
	resources := []domain.Resource{
		makeTestResource("user-db", domain.ResourceDatabase,
			`resource "aws_db_instance" "user_db" {}`,
			`resource "google_sql_database_instance" "user_db" {
  database_version = "POSTGRES_16"
}`,
		),
	}

	config, err := GenerateConfig(app, resources, domain.ProviderGCP)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(config, "hashicorp/google") {
		t.Error("expected GCP provider block")
	}

	if !strings.Contains(config, `resource "google_sql_database_instance" "user_db"`) {
		t.Error("expected Cloud SQL resource block")
	}
}

func TestGenerateConfig_NoResources(t *testing.T) {
	app := makeTestApp("empty-app", domain.ProviderAWS)

	config, err := GenerateConfig(app, nil, domain.ProviderAWS)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(config, "No resources") {
		t.Error("expected no-resources comment")
	}
}

func TestGenerateConfig_MultipleResources(t *testing.T) {
	app := makeTestApp("multi-app", domain.ProviderAWS)
	resources := []domain.Resource{
		makeTestResource("user-db", domain.ResourceDatabase,
			`resource "aws_db_instance" "user_db" {}`,
			`resource "google_sql_database_instance" "user_db" {}`,
		),
		makeTestResource("cache", domain.ResourceCache,
			`resource "aws_elasticache_cluster" "cache" {}`,
			`resource "google_redis_instance" "cache" {}`,
		),
	}

	config, err := GenerateConfig(app, resources, domain.ProviderAWS)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	if !strings.Contains(config, "aws_db_instance") {
		t.Error("expected DB resource")
	}
	if !strings.Contains(config, "aws_elasticache_cluster") {
		t.Error("expected cache resource")
	}
}

func TestGenerateConfig_UnsupportedProvider(t *testing.T) {
	app := makeTestApp("bad-app", domain.CloudProvider("azure"))

	_, err := GenerateConfig(app, nil, domain.CloudProvider("azure"))
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
}

func TestGenerateConfig_ResourceWithoutMapping(t *testing.T) {
	app := makeTestApp("partial-app", domain.ProviderAWS)

	// Resource with only GCP mapping
	r := domain.Resource{
		ID:   uuid.New(),
		Kind: domain.ResourceDatabase,
		Name: "gcp-only-db",
		Spec: json.RawMessage(`{}`),
		ProviderMappings: map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderGCP: {
				ServiceName:  "Cloud SQL",
				TerraformHCL: `resource "google_sql_database_instance" "db" {}`,
			},
		},
	}

	config, err := GenerateConfig(app, []domain.Resource{r}, domain.ProviderAWS)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// Should contain the no-resources comment since no AWS mapping exists
	if !strings.Contains(config, "No resources") {
		t.Error("expected no-resources comment when resource lacks AWS mapping")
	}
}

func TestGenerateResourceHCL(t *testing.T) {
	r := makeTestResource("my-db", domain.ResourceDatabase,
		`resource "aws_db_instance" "my_db" {}`,
		`resource "google_sql_database_instance" "my_db" {}`,
	)

	t.Run("AWS mapping", func(t *testing.T) {
		hcl := GenerateResourceHCL(r, domain.ProviderAWS)
		if !strings.Contains(hcl, "aws_db_instance") {
			t.Error("expected AWS HCL")
		}
	})

	t.Run("GCP mapping", func(t *testing.T) {
		hcl := GenerateResourceHCL(r, domain.ProviderGCP)
		if !strings.Contains(hcl, "google_sql_database_instance") {
			t.Error("expected GCP HCL")
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		hcl := GenerateResourceHCL(r, domain.CloudProvider("azure"))
		if hcl != "" {
			t.Error("expected empty string for unknown provider")
		}
	})
}
