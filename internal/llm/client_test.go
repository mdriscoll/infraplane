package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "raw JSON",
			input: `{"kind": "database"}`,
			want:  `{"kind": "database"}`,
		},
		{
			name:  "JSON in markdown fence",
			input: "Here is the result:\n```json\n{\"kind\": \"database\"}\n```\nDone.",
			want:  `{"kind": "database"}`,
		},
		{
			name:  "JSON with surrounding text",
			input: "I recommend: {\"kind\": \"database\", \"name\": \"test\"} as the resource.",
			want:  `{"kind": "database", "name": "test"}`,
		},
		{
			name:  "nested JSON",
			input: `{"kind": "database", "spec": {"engine": "postgres"}}`,
			want:  `{"kind": "database", "spec": {"engine": "postgres"}}`,
		},
		{
			name:  "JSON array",
			input: `[{"kind": "database"}, {"kind": "cache"}]`,
			want:  `[{"kind": "database"}, {"kind": "cache"}]`,
		},
		{
			name:  "JSON array with surrounding text",
			input: "Here are the resources: [{\"kind\": \"database\"}, {\"kind\": \"cache\"}] done.",
			want:  `[{"kind": "database"}, {"kind": "cache"}]`,
		},
		{
			name:  "JSON array in markdown fence",
			input: "```json\n[{\"kind\": \"database\"}]\n```",
			want:  `[{"kind": "database"}]`,
		},
		{
			name:  "no JSON at all",
			input: "Just some plain text",
			want:  "Just some plain text",
		},
		{
			name:  "JSON with braces inside string values",
			input: `{"content": "Use resource \"foo\" {\n  name = \"bar\"\n}\nfor config", "cost": 100}`,
			want:  `{"content": "Use resource \"foo\" {\n  name = \"bar\"\n}\nfor config", "cost": 100}`,
		},
		{
			name:  "hosting plan JSON with markdown containing code blocks",
			input: `{"content": "# Plan\n\n` + "```" + `hcl\nresource \"aws_instance\" \"web\" {\n  ami = \"abc\"\n}\n` + "```" + `\n\nDone.", "estimated_cost": {"monthly_cost_usd": 50}}`,
			want:  `{"content": "# Plan\n\n` + "```" + `hcl\nresource \"aws_instance\" \"web\" {\n  ami = \"abc\"\n}\n` + "```" + `\n\nDone.", "estimated_cost": {"monthly_cost_usd": 50}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			if got != tt.want {
				t.Errorf("extractJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMockClient_AnalyzeResourceNeed(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	result, err := mock.AnalyzeResourceNeed(ctx, "I need a database", domain.ProviderAWS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != domain.ResourceDatabase {
		t.Errorf("Kind = %q, want %q", result.Kind, domain.ResourceDatabase)
	}
	if result.Name != "mock-database" {
		t.Errorf("Name = %q, want %q", result.Name, "mock-database")
	}
	if _, ok := result.Mappings[domain.ProviderAWS]; !ok {
		t.Error("missing AWS mapping")
	}
	if _, ok := result.Mappings[domain.ProviderGCP]; !ok {
		t.Error("missing GCP mapping")
	}
}

func TestMockClient_AnalyzeResourceNeed_Custom(t *testing.T) {
	mock := &MockClient{
		AnalyzeResourceNeedFn: func(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error) {
			return ResourceRecommendation{
				Kind: domain.ResourceCache,
				Name: "session-cache",
				Spec: json.RawMessage(`{"engine": "redis"}`),
				Mappings: map[domain.CloudProvider]domain.ProviderResource{
					domain.ProviderAWS: {ServiceName: "ElastiCache"},
				},
			}, nil
		},
	}
	ctx := context.Background()

	result, err := mock.AnalyzeResourceNeed(ctx, "I need a Redis cache", domain.ProviderAWS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != domain.ResourceCache {
		t.Errorf("Kind = %q, want %q", result.Kind, domain.ResourceCache)
	}
	if result.Name != "session-cache" {
		t.Errorf("Name = %q, want %q", result.Name, "session-cache")
	}
}

func TestMockClient_GenerateHostingPlan(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()
	app := domain.Application{ID: uuid.New(), Name: "test-app", Provider: domain.ProviderAWS}

	result, err := mock.GenerateHostingPlan(ctx, app, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content == "" {
		t.Error("Content should not be empty")
	}
	if result.EstimatedCost == nil {
		t.Fatal("EstimatedCost should not be nil")
	}
	if result.EstimatedCost.MonthlyCostUSD != 100.00 {
		t.Errorf("MonthlyCostUSD = %f, want 100.00", result.EstimatedCost.MonthlyCostUSD)
	}
}

func TestMockClient_GenerateMigrationPlan(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()
	app := domain.Application{ID: uuid.New(), Name: "test-app", Provider: domain.ProviderAWS}

	result, err := mock.GenerateMigrationPlan(ctx, app, nil, domain.ProviderAWS, domain.ProviderGCP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content == "" {
		t.Error("Content should not be empty")
	}
	if result.EstimatedCost == nil {
		t.Fatal("EstimatedCost should not be nil")
	}
	if result.EstimatedCost.MonthlyCostUSD != 120.00 {
		t.Errorf("MonthlyCostUSD = %f, want 120.00", result.EstimatedCost.MonthlyCostUSD)
	}
}

func TestMockClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*MockClient)(nil)
}

func TestAnthropicClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*AnthropicClient)(nil)
}
