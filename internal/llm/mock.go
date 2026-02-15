package llm

import (
	"context"
	"encoding/json"

	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// MockClient is a mock implementation of Client for testing.
// Set the return values before calling the methods.
type MockClient struct {
	AnalyzeResourceNeedFn        func(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error)
	AnalyzeCodebaseFn            func(ctx context.Context, codeCtx analyzer.CodeContext, provider domain.CloudProvider) ([]ResourceRecommendation, error)
	GenerateHostingPlanFn        func(ctx context.Context, app domain.Application, resources []domain.Resource) (HostingPlanResult, error)
	GenerateMigrationPlanFn      func(ctx context.Context, app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) (MigrationPlanResult, error)
	GenerateGraphFn              func(ctx context.Context, app domain.Application, resources []domain.Resource) (GraphResult, error)
	GenerateTerraformHCLFn       func(ctx context.Context, resource domain.Resource, provider domain.CloudProvider) (TerraformHCLResult, error)
	GenerateDiscoveryCommandsFn  func(ctx context.Context, app domain.Application, codeCtx analyzer.CodeContext) (DiscoveryCommandResult, error)
	ParseDiscoveryOutputFn       func(ctx context.Context, app domain.Application, outputs []CommandOutput) (LiveResourceParseResult, error)
}

func (m *MockClient) AnalyzeResourceNeed(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error) {
	if m.AnalyzeResourceNeedFn != nil {
		return m.AnalyzeResourceNeedFn(ctx, description, provider)
	}
	return defaultResourceRecommendation(), nil
}

func (m *MockClient) AnalyzeCodebase(ctx context.Context, codeCtx analyzer.CodeContext, provider domain.CloudProvider) ([]ResourceRecommendation, error) {
	if m.AnalyzeCodebaseFn != nil {
		return m.AnalyzeCodebaseFn(ctx, codeCtx, provider)
	}
	return defaultCodebaseRecommendations(), nil
}

func (m *MockClient) GenerateHostingPlan(ctx context.Context, app domain.Application, resources []domain.Resource) (HostingPlanResult, error) {
	if m.GenerateHostingPlanFn != nil {
		return m.GenerateHostingPlanFn(ctx, app, resources)
	}
	return defaultHostingPlan(), nil
}

func (m *MockClient) GenerateMigrationPlan(ctx context.Context, app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) (MigrationPlanResult, error) {
	if m.GenerateMigrationPlanFn != nil {
		return m.GenerateMigrationPlanFn(ctx, app, resources, from, to)
	}
	return defaultMigrationPlan(), nil
}

func (m *MockClient) GenerateGraph(ctx context.Context, app domain.Application, resources []domain.Resource) (GraphResult, error) {
	if m.GenerateGraphFn != nil {
		return m.GenerateGraphFn(ctx, app, resources)
	}
	return defaultGraph(), nil
}

func (m *MockClient) GenerateTerraformHCL(ctx context.Context, resource domain.Resource, provider domain.CloudProvider) (TerraformHCLResult, error) {
	if m.GenerateTerraformHCLFn != nil {
		return m.GenerateTerraformHCLFn(ctx, resource, provider)
	}
	return TerraformHCLResult{HCL: `resource "example" "mock" { name = "` + resource.Name + `" }`}, nil
}

func defaultCodebaseRecommendations() []ResourceRecommendation {
	return []ResourceRecommendation{
		{
			Kind: domain.ResourceDatabase,
			Name: "app-database",
			Spec: json.RawMessage(`{"engine": "postgres", "version": "16"}`),
			Mappings: map[domain.CloudProvider]domain.ProviderResource{
				domain.ProviderAWS: {
					ServiceName:  "RDS",
					Config:       map[string]any{"instance_class": "db.t3.micro"},
					TerraformHCL: `resource "aws_db_instance" "app_database" {}`,
				},
				domain.ProviderGCP: {
					ServiceName:  "Cloud SQL",
					Config:       map[string]any{"tier": "db-f1-micro"},
					TerraformHCL: `resource "google_sql_database_instance" "app_database" {}`,
				},
			},
		},
		{
			Kind: domain.ResourceCache,
			Name: "app-cache",
			Spec: json.RawMessage(`{"engine": "redis", "version": "7"}`),
			Mappings: map[domain.CloudProvider]domain.ProviderResource{
				domain.ProviderAWS: {
					ServiceName:  "ElastiCache",
					Config:       map[string]any{"node_type": "cache.t3.micro"},
					TerraformHCL: `resource "aws_elasticache_cluster" "app_cache" {}`,
				},
				domain.ProviderGCP: {
					ServiceName:  "Memorystore",
					Config:       map[string]any{"tier": "BASIC"},
					TerraformHCL: `resource "google_redis_instance" "app_cache" {}`,
				},
			},
		},
	}
}

func defaultResourceRecommendation() ResourceRecommendation {
	return ResourceRecommendation{
		Kind: domain.ResourceDatabase,
		Name: "mock-database",
		Spec: json.RawMessage(`{"engine": "postgres", "version": "16"}`),
		Mappings: map[domain.CloudProvider]domain.ProviderResource{
			domain.ProviderAWS: {
				ServiceName:  "RDS",
				Config:       map[string]any{"instance_class": "db.t3.micro"},
				TerraformHCL: `resource "aws_db_instance" "mock_database" {}`,
			},
			domain.ProviderGCP: {
				ServiceName:  "Cloud SQL",
				Config:       map[string]any{"tier": "db-f1-micro"},
				TerraformHCL: `resource "google_sql_database_instance" "mock_database" {}`,
			},
		},
	}
}

func defaultHostingPlan() HostingPlanResult {
	return HostingPlanResult{
		Content: "# Mock Hosting Plan\n\nDeploy using containerized services.",
		EstimatedCost: &domain.CostEstimate{
			MonthlyCostUSD: 100.00,
			Breakdown:      map[string]float64{"compute": 60, "database": 30, "storage": 10},
		},
	}
}

func defaultMigrationPlan() MigrationPlanResult {
	return MigrationPlanResult{
		Content: "# Mock Migration Plan\n\nMigrate services one at a time.",
		EstimatedCost: &domain.CostEstimate{
			MonthlyCostUSD: 120.00,
			Breakdown:      map[string]float64{"compute": 70, "database": 35, "storage": 15},
		},
	}
}

func (m *MockClient) GenerateDiscoveryCommands(ctx context.Context, app domain.Application, codeCtx analyzer.CodeContext) (DiscoveryCommandResult, error) {
	if m.GenerateDiscoveryCommandsFn != nil {
		return m.GenerateDiscoveryCommandsFn(ctx, app, codeCtx)
	}
	return defaultDiscoveryCommands(), nil
}

func (m *MockClient) ParseDiscoveryOutput(ctx context.Context, app domain.Application, outputs []CommandOutput) (LiveResourceParseResult, error) {
	if m.ParseDiscoveryOutputFn != nil {
		return m.ParseDiscoveryOutputFn(ctx, app, outputs)
	}
	return defaultLiveResources(), nil
}

func defaultDiscoveryCommands() DiscoveryCommandResult {
	return DiscoveryCommandResult{
		Commands: []DiscoveryCommand{
			{
				Description:  "List Cloud Run services",
				Command:      "gcloud run services list --project=my-project --region=us-central1 --format=json",
				ResourceType: "Cloud Run Service",
			},
			{
				Description:  "Describe Cloud SQL instance",
				Command:      "gcloud sql instances describe my-db --project=my-project --format=json",
				ResourceType: "Cloud SQL Instance",
			},
		},
	}
}

func defaultLiveResources() LiveResourceParseResult {
	return LiveResourceParseResult{
		Resources: []domain.LiveResource{
			{
				ResourceType: "Cloud Run Service",
				Name:         "api-server",
				Provider:     domain.ProviderGCP,
				Region:       "us-central1",
				Status:       domain.LiveResourceActive,
				Details:      map[string]string{"url": "https://api-server-xxx.run.app", "memory": "512Mi"},
			},
			{
				ResourceType: "Cloud SQL Instance",
				Name:         "app-database",
				Provider:     domain.ProviderGCP,
				Region:       "us-central1",
				Status:       domain.LiveResourceActive,
				Details:      map[string]string{"tier": "db-f1-micro", "database_version": "POSTGRES_15"},
			},
		},
	}
}

func defaultGraph() GraphResult {
	return GraphResult{
		Nodes: []domain.GraphNode{
			{ID: "internet", Label: "Internet", Kind: domain.GraphNodeInternet, Service: "Public Internet"},
			{ID: "api-server", Label: "API Server", Kind: domain.GraphNodeCompute, Service: "Cloud Run"},
			{ID: "app-database", Label: "App Database", Kind: domain.GraphNodeDatabase, Service: "Cloud SQL"},
			{ID: "app-cache", Label: "App Cache", Kind: domain.GraphNodeCache, Service: "Memorystore"},
		},
		Edges: []domain.GraphEdge{
			{ID: "internet-to-api", Source: "internet", Target: "api-server", Label: "HTTPS"},
			{ID: "api-to-db", Source: "api-server", Target: "app-database", Label: "TCP/5432"},
			{ID: "api-to-cache", Source: "api-server", Target: "app-cache", Label: "Redis/6379"},
		},
	}
}
