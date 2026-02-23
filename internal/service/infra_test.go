package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/provider"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
)

func setupInfraService(providerName domain.CloudProvider, applyErr error) (*InfraService, *mock.ApplicationRepo, *mock.ResourceRepo, *llm.MockClient) {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	depRepo := mock.NewDeploymentRepo()
	mockLLM := &llm.MockClient{}

	reg := provider.NewRegistry()
	reg.Register(&provider.MockAdapter{
		ProviderVal: providerName,
		ApplyResult: "Plan applied successfully",
		ApplyErr:    applyErr,
	})

	svc := NewInfraService(appRepo, resRepo, depRepo, reg, mockLLM, nil)
	return svc, appRepo, resRepo, mockLLM
}

func TestInfraService_GenerateTerraform(t *testing.T) {
	svc, appRepo, resRepo, mockLLM := setupInfraService(domain.ProviderAWS, nil)
	ctx := context.Background()

	app := domain.NewApplication("tf-app", "", "", "", domain.ProviderAWS)
	appRepo.Create(ctx, app)

	resource := domain.NewResource(app.ID, domain.ResourceDatabase, "user-db", json.RawMessage(`{"engine": "postgres"}`))
	resource.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
		domain.ProviderAWS: {
			ServiceName:  "RDS",
			Config:       map[string]any{"instance_class": "db.t3.micro"},
			TerraformHCL: `resource "aws_db_instance" "user_db" { engine = "postgres" }`,
		},
	}
	resRepo.Create(ctx, resource)

	// Set up mock to return a realistic AWS config
	mockLLM.GenerateFullTerraformFn = func(ctx context.Context, app domain.Application, resources []domain.Resource, provider domain.CloudProvider, target *domain.DeployTarget, complianceContext string) (llm.FullTerraformResult, error) {
		return llm.FullTerraformResult{
			HCL: `terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_db_instance" "user_db" {
  engine         = "postgres"
  instance_class = "db.t3.micro"
}`,
			Resources: []llm.FullTerraformResource{
				{
					ResourceName: "user-db",
					ResourceKind: "database",
					ServiceName:  "RDS",
					HCL:          `resource "aws_db_instance" "user_db" { engine = "postgres" instance_class = "db.t3.micro" }`,
				},
			},
		}, nil
	}

	t.Run("successful generation", func(t *testing.T) {
		config, err := svc.GenerateTerraform(ctx, app.ID, nil)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !strings.Contains(config, "aws_db_instance") {
			t.Error("expected AWS DB resource in config")
		}
		if !strings.Contains(config, "hashicorp/aws") {
			t.Error("expected AWS provider block")
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, err := svc.GenerateTerraform(ctx, uuid.New(), nil)
		if err == nil {
			t.Error("expected error for nonexistent app")
		}
	})
}

func TestInfraService_GenerateTerraformWithResources(t *testing.T) {
	svc, appRepo, resRepo, mockLLM := setupInfraService(domain.ProviderGCP, nil)
	ctx := context.Background()

	app := domain.NewApplication("multi-resource-app", "App with DB and cache", "", "", domain.ProviderGCP)
	appRepo.Create(ctx, app)

	db := domain.NewResource(app.ID, domain.ResourceDatabase, "main-db", json.RawMessage(`{"engine": "postgres"}`))
	db.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
		domain.ProviderGCP: {ServiceName: "Cloud SQL", Config: map[string]any{"tier": "db-f1-micro"}},
	}
	resRepo.Create(ctx, db)

	cache := domain.NewResource(app.ID, domain.ResourceCache, "app-cache", json.RawMessage(`{"engine": "redis"}`))
	cache.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
		domain.ProviderGCP: {ServiceName: "Memorystore", Config: map[string]any{"tier": "BASIC"}},
	}
	resRepo.Create(ctx, cache)

	// Set up mock to return a cohesive GCP config with both resources on the same VPC
	mockLLM.GenerateFullTerraformFn = func(ctx context.Context, app domain.Application, resources []domain.Resource, provider domain.CloudProvider, target *domain.DeployTarget, complianceContext string) (llm.FullTerraformResult, error) {
		if len(resources) != 2 {
			return llm.FullTerraformResult{}, fmt.Errorf("expected 2 resources, got %d", len(resources))
		}
		return llm.FullTerraformResult{
			HCL: `terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "us-central1"
}

resource "google_compute_network" "vpc" {
  name                    = "multi-resource-app-vpc"
  auto_create_subnetworks = false
}

resource "google_sql_database_instance" "main_db" {
  name             = "main-db"
  database_version = "POSTGRES_16"
  region           = var.region
  project          = var.project_id

  settings {
    tier = "db-f1-micro"
    ip_configuration {
      private_network = google_compute_network.vpc.id
    }
  }
}

resource "google_redis_instance" "app_cache" {
  name           = "app-cache"
  tier           = "BASIC"
  memory_size_gb = 1
  region         = var.region
  project        = var.project_id

  authorized_network = google_compute_network.vpc.id
}`,
			Resources: []llm.FullTerraformResource{
				{
					ResourceName: "main-db",
					ResourceKind: "database",
					ServiceName:  "Cloud SQL",
					HCL: `resource "google_sql_database_instance" "main_db" {
  name             = "main-db"
  database_version = "POSTGRES_16"
}`,
				},
				{
					ResourceName: "app-cache",
					ResourceKind: "cache",
					ServiceName:  "Memorystore",
					HCL: `resource "google_redis_instance" "app_cache" {
  name           = "app-cache"
  tier           = "BASIC"
  memory_size_gb = 1
}`,
				},
			},
		}, nil
	}

	t.Run("returns full config and per-resource breakdown", func(t *testing.T) {
		config, resourceHCLs, err := svc.GenerateTerraformWithResources(ctx, app.ID, nil)
		if err != nil {
			t.Fatalf("error = %v", err)
		}

		// Full config should contain shared VPC declared once
		if !strings.Contains(config, "google_compute_network") {
			t.Error("expected VPC in full config")
		}
		// Full config should contain both resources
		if !strings.Contains(config, "google_sql_database_instance") {
			t.Error("expected Cloud SQL in full config")
		}
		if !strings.Contains(config, "google_redis_instance") {
			t.Error("expected Redis in full config")
		}
		// Both resources should reference the same VPC
		if !strings.Contains(config, "google_compute_network.vpc.id") {
			t.Error("expected resources to reference shared VPC")
		}

		// Per-resource breakdown should have 2 entries
		if len(resourceHCLs) != 2 {
			t.Fatalf("expected 2 resource HCLs, got %d", len(resourceHCLs))
		}

		// Check resource IDs are populated
		if resourceHCLs[0].ResourceID != db.ID {
			t.Errorf("expected resource ID %s for main-db, got %s", db.ID, resourceHCLs[0].ResourceID)
		}
		if resourceHCLs[1].ResourceID != cache.ID {
			t.Errorf("expected resource ID %s for app-cache, got %s", cache.ID, resourceHCLs[1].ResourceID)
		}

		// Check per-resource metadata
		if resourceHCLs[0].ServiceName != "Cloud SQL" {
			t.Errorf("expected Cloud SQL, got %s", resourceHCLs[0].ServiceName)
		}
		if resourceHCLs[1].ServiceName != "Memorystore" {
			t.Errorf("expected Memorystore, got %s", resourceHCLs[1].ServiceName)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		_, _, err := svc.GenerateTerraformWithResources(ctx, uuid.New(), nil)
		if err == nil {
			t.Error("expected error for nonexistent app")
		}
	})
}

func TestInfraService_DeployInfrastructure(t *testing.T) {
	ctx := context.Background()

	t.Run("successful deployment", func(t *testing.T) {
		svc, appRepo, resRepo, _ := setupInfraService(domain.ProviderAWS, nil)

		app := domain.NewApplication("deploy-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		resource := domain.NewResource(app.ID, domain.ResourceDatabase, "db", json.RawMessage(`{}`))
		resource.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderAWS: {
				ServiceName:  "RDS",
				TerraformHCL: `resource "aws_db_instance" "db" {}`,
			},
		}
		resRepo.Create(ctx, resource)

		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc123", "main", nil)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if d.Status != domain.DeploymentSucceeded {
			t.Errorf("status = %v, want succeeded", d.Status)
		}
		if d.Provider != domain.ProviderAWS {
			t.Errorf("provider = %v, want aws", d.Provider)
		}
		if d.GitBranch != "main" {
			t.Errorf("git_branch = %v, want main", d.GitBranch)
		}
		if d.CompletedAt == nil {
			t.Error("expected CompletedAt to be set")
		}
	})

	t.Run("apply failure marks deployment as failed", func(t *testing.T) {
		svc, appRepo, _, _ := setupInfraService(domain.ProviderAWS, fmt.Errorf("terraform error"))

		app := domain.NewApplication("fail-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc123", "main", nil)
		if err == nil {
			t.Fatal("expected error from failed apply")
		}
		if d.Status != domain.DeploymentFailed {
			t.Errorf("status = %v, want failed", d.Status)
		}
	})

	t.Run("no provider adapter", func(t *testing.T) {
		// Register only GCP adapter but app uses AWS
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		depRepo := mock.NewDeploymentRepo()

		reg := provider.NewRegistry()
		reg.Register(&provider.MockAdapter{
			ProviderVal: domain.ProviderGCP,
			ApplyResult: "ok",
		})

		svc := NewInfraService(appRepo, resRepo, depRepo, reg, &llm.MockClient{}, nil)

		app := domain.NewApplication("no-adapter-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc", "main", nil)
		if err == nil {
			t.Fatal("expected error for missing adapter")
		}
		if d.Status != domain.DeploymentFailed {
			t.Errorf("status = %v, want failed", d.Status)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		svc, _, _, _ := setupInfraService(domain.ProviderAWS, nil)
		_, err := svc.DeployInfrastructure(ctx, uuid.New(), "abc", "main", nil)
		if err == nil {
			t.Error("expected error for nonexistent app")
		}
	})

	t.Run("missing branch", func(t *testing.T) {
		svc, appRepo, _, _ := setupInfraService(domain.ProviderAWS, nil)
		app := domain.NewApplication("no-branch", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		_, err := svc.DeployInfrastructure(ctx, app.ID, "abc", "", nil)
		if err == nil {
			t.Error("expected validation error for missing branch")
		}
	})
}

func TestInfraService_ValidateProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("valid provider", func(t *testing.T) {
		svc, _, _, _ := setupInfraService(domain.ProviderAWS, nil)
		err := svc.ValidateProvider(ctx, domain.ProviderAWS, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		svc, _, _, _ := setupInfraService(domain.ProviderAWS, nil)
		err := svc.ValidateProvider(ctx, domain.CloudProvider("azure"), nil)
		if err == nil {
			t.Error("expected error for unknown provider")
		}
	})
}

func TestInfraService_DestroyInfrastructure(t *testing.T) {
	ctx := context.Background()

	t.Run("successful destroy", func(t *testing.T) {
		svc, appRepo, resRepo, _ := setupInfraService(domain.ProviderAWS, nil)

		app := domain.NewApplication("destroy-app", "", "", "", domain.ProviderAWS)
		appRepo.Create(ctx, app)

		resource := domain.NewResource(app.ID, domain.ResourceDatabase, "db", json.RawMessage(`{}`))
		resource.ProviderMappings = map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderAWS: {
				ServiceName:  "RDS",
				TerraformHCL: `resource "aws_db_instance" "db" {}`,
			},
		}
		resRepo.Create(ctx, resource)

		// Deploy first
		d, err := svc.DeployInfrastructure(ctx, app.ID, "abc", "main", nil)
		if err != nil {
			t.Fatalf("deploy error = %v", err)
		}

		// Destroy
		err = svc.DestroyInfrastructure(ctx, d.ID)
		if err != nil {
			t.Errorf("destroy error = %v", err)
		}
	})

	t.Run("deployment not found", func(t *testing.T) {
		svc, _, _, _ := setupInfraService(domain.ProviderAWS, nil)
		err := svc.DestroyInfrastructure(ctx, uuid.New())
		if err == nil {
			t.Error("expected error for nonexistent deployment")
		}
	})
}
