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

	config, err := GenerateConfig(app, resources, domain.ProviderAWS, nil)
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

	config, err := GenerateConfig(app, resources, domain.ProviderGCP, nil)
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

	config, err := GenerateConfig(app, nil, domain.ProviderAWS, nil)
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

	config, err := GenerateConfig(app, resources, domain.ProviderAWS, nil)
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

	config, err := GenerateConfig(app, nil, domain.CloudProvider("azure"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(config, "Unsupported provider") {
		t.Error("expected unsupported provider comment in output")
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

	config, err := GenerateConfig(app, []domain.Resource{r}, domain.ProviderAWS, nil)
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

func TestDeduplicateHCL(t *testing.T) {
	t.Run("removes duplicate variables", func(t *testing.T) {
		input := `variable "project_id" {
  type = string
}

resource "google_sql_database_instance" "db" {
  project = var.project_id
}

variable "project_id" {
  type = string
}

resource "google_redis_instance" "cache" {
  project = var.project_id
}
`
		result := deduplicateHCL(input)

		// Should contain exactly one variable "project_id"
		count := strings.Count(result, `variable "project_id"`)
		if count != 1 {
			t.Errorf("expected 1 occurrence of variable project_id, got %d\n%s", count, result)
		}
		// Both resources should remain
		if !strings.Contains(result, "google_sql_database_instance") {
			t.Error("missing database resource")
		}
		if !strings.Contains(result, "google_redis_instance") {
			t.Error("missing cache resource")
		}
	})

	t.Run("removes duplicate resources", func(t *testing.T) {
		input := `resource "google_compute_network" "vpc" {
  name                    = "main-vpc"
  auto_create_subnetworks = false
}

resource "google_sql_database_instance" "db" {
  network = google_compute_network.vpc.id
}

resource "google_compute_network" "vpc" {
  name                    = "main-vpc"
  auto_create_subnetworks = false
}

resource "google_redis_instance" "cache" {
  network = google_compute_network.vpc.id
}
`
		result := deduplicateHCL(input)

		count := strings.Count(result, `resource "google_compute_network" "vpc"`)
		if count != 1 {
			t.Errorf("expected 1 VPC resource, got %d\n%s", count, result)
		}
		if !strings.Contains(result, "google_sql_database_instance") {
			t.Error("missing database resource")
		}
		if !strings.Contains(result, "google_redis_instance") {
			t.Error("missing cache resource")
		}
	})

	t.Run("removes duplicate outputs", func(t *testing.T) {
		input := `output "service_url" {
  value = google_cloud_run_service.app.url
}

output "service_account_email" {
  value = google_service_account.sa.email
}

output "service_url" {
  value = google_cloud_run_service.app.url
}
`
		result := deduplicateHCL(input)

		count := strings.Count(result, `output "service_url"`)
		if count != 1 {
			t.Errorf("expected 1 service_url output, got %d\n%s", count, result)
		}
		if !strings.Contains(result, `output "service_account_email"`) {
			t.Error("missing service_account_email output")
		}
	})

	t.Run("preserves unique blocks", func(t *testing.T) {
		input := `variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

resource "google_sql_database_instance" "db" {
  project = var.project_id
}

resource "google_redis_instance" "cache" {
  project = var.project_id
}
`
		result := deduplicateHCL(input)

		// All blocks are unique — nothing should be removed
		if !strings.Contains(result, `variable "project_id"`) {
			t.Error("missing variable project_id")
		}
		if !strings.Contains(result, `variable "region"`) {
			t.Error("missing variable region")
		}
		if !strings.Contains(result, "google_sql_database_instance") {
			t.Error("missing database resource")
		}
		if !strings.Contains(result, "google_redis_instance") {
			t.Error("missing cache resource")
		}
	})

	t.Run("handles nested braces", func(t *testing.T) {
		input := `resource "google_compute_network" "vpc" {
  name = "main-vpc"
  lifecycle {
    prevent_destroy = true
  }
}

resource "google_sql_database_instance" "db" {
  settings {
    tier = "db-f1-micro"
    ip_configuration {
      ipv4_enabled = true
    }
  }
}

resource "google_compute_network" "vpc" {
  name = "main-vpc"
  lifecycle {
    prevent_destroy = true
  }
}
`
		result := deduplicateHCL(input)

		count := strings.Count(result, `resource "google_compute_network" "vpc"`)
		if count != 1 {
			t.Errorf("expected 1 VPC resource, got %d\n%s", count, result)
		}
		// The database with nested braces should be preserved
		if !strings.Contains(result, "google_sql_database_instance") {
			t.Error("missing database resource with nested braces")
		}
		if !strings.Contains(result, "ip_configuration") {
			t.Error("nested block content should be preserved")
		}
	})

	t.Run("handles data sources", func(t *testing.T) {
		input := `data "google_project" "current" {
}

resource "google_sql_database_instance" "db" {
}

data "google_project" "current" {
}
`
		result := deduplicateHCL(input)

		count := strings.Count(result, `data "google_project" "current"`)
		if count != 1 {
			t.Errorf("expected 1 data source, got %d\n%s", count, result)
		}
	})

	t.Run("preserves comments and non-block lines", func(t *testing.T) {
		input := `# Resource 1
variable "project_id" {
  type = string
}

# Resource 2
variable "project_id" {
  type = string
}

# This comment should stay
resource "google_sql_database_instance" "db" {
}
`
		result := deduplicateHCL(input)

		count := strings.Count(result, `variable "project_id"`)
		if count != 1 {
			t.Errorf("expected 1 variable, got %d\n%s", count, result)
		}
		if !strings.Contains(result, "# Resource 1") {
			t.Error("first comment should be preserved")
		}
		if !strings.Contains(result, "# This comment should stay") {
			t.Error("non-block comment should be preserved")
		}
	})
}

func TestGenerateConfig_DeduplicatesAcrossResources(t *testing.T) {
	app := makeTestApp("dedup-app", domain.ProviderGCP)

	r1 := domain.Resource{
		ID:   uuid.New(),
		Kind: domain.ResourceDatabase,
		Name: "main-db",
		Spec: json.RawMessage(`{}`),
		ProviderMappings: map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderGCP: {
				ServiceName: "Cloud SQL",
				TerraformHCL: `variable "project_id" {
  type = string
}

resource "google_sql_database_instance" "main_db" {
  project = var.project_id
}`,
			},
		},
	}
	r2 := domain.Resource{
		ID:   uuid.New(),
		Kind: domain.ResourceCache,
		Name: "app-cache",
		Spec: json.RawMessage(`{}`),
		ProviderMappings: map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderGCP: {
				ServiceName: "Memorystore",
				TerraformHCL: `variable "project_id" {
  type = string
}

resource "google_redis_instance" "app_cache" {
  project = var.project_id
}`,
			},
		},
	}

	config, err := GenerateConfig(app, []domain.Resource{r1, r2}, domain.ProviderGCP, nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	// variable "project_id" should appear exactly once
	count := strings.Count(config, `variable "project_id"`)
	if count != 1 {
		t.Errorf("expected 1 variable project_id, got %d\n%s", count, config)
	}

	// Both resources should be present
	if !strings.Contains(config, "google_sql_database_instance") {
		t.Error("missing database resource")
	}
	if !strings.Contains(config, "google_redis_instance") {
		t.Error("missing cache resource")
	}
}
