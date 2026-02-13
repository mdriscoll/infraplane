package llm

import (
	"context"
	"encoding/json"

	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// MockClient is a mock implementation of Client for testing.
// Set the return values before calling the methods.
type MockClient struct {
	AnalyzeResourceNeedFn    func(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error)
	GenerateHostingPlanFn    func(ctx context.Context, app domain.Application, resources []domain.Resource) (HostingPlanResult, error)
	GenerateMigrationPlanFn  func(ctx context.Context, app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) (MigrationPlanResult, error)
}

func (m *MockClient) AnalyzeResourceNeed(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error) {
	if m.AnalyzeResourceNeedFn != nil {
		return m.AnalyzeResourceNeedFn(ctx, description, provider)
	}
	return defaultResourceRecommendation(), nil
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
