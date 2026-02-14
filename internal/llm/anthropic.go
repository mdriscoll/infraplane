package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// AnthropicClient implements Client using the Anthropic API with Claude Opus 4.6.
type AnthropicClient struct {
	client anthropic.Client
	model  anthropic.Model
}

// NewAnthropicClient creates a new Anthropic-backed LLM client.
// apiKey is the Anthropic API key. If empty, the SDK reads ANTHROPIC_API_KEY from env.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &AnthropicClient{
		client: anthropic.NewClient(opts...),
		model:  anthropic.ModelClaudeOpus4_6,
	}
}

func (c *AnthropicClient) AnalyzeResourceNeed(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error) {
	prompt := buildResourceAnalysisPrompt(description, provider)

	resp, err := c.sendMessage(ctx, prompt, resourceAnalysisSystemPrompt)
	if err != nil {
		return ResourceRecommendation{}, fmt.Errorf("analyze resource need: %w", err)
	}

	var result ResourceRecommendation
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return ResourceRecommendation{}, fmt.Errorf("parse resource recommendation: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) AnalyzeCodebase(ctx context.Context, codeCtx analyzer.CodeContext, provider domain.CloudProvider) ([]ResourceRecommendation, error) {
	prompt := buildCodebaseAnalysisPrompt(codeCtx, provider)

	resp, err := c.sendMessage(ctx, prompt, codebaseAnalysisSystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("analyze codebase: %w", err)
	}

	var results []ResourceRecommendation
	if err := json.Unmarshal([]byte(resp), &results); err != nil {
		return nil, fmt.Errorf("parse codebase analysis: %w", err)
	}
	return results, nil
}

func (c *AnthropicClient) GenerateHostingPlan(ctx context.Context, app domain.Application, resources []domain.Resource) (HostingPlanResult, error) {
	prompt := buildHostingPlanPrompt(app, resources)

	resp, err := c.sendMessage(ctx, prompt, hostingPlanSystemPrompt)
	if err != nil {
		return HostingPlanResult{}, fmt.Errorf("generate hosting plan: %w", err)
	}

	var result HostingPlanResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return HostingPlanResult{}, fmt.Errorf("parse hosting plan: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) GenerateMigrationPlan(ctx context.Context, app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) (MigrationPlanResult, error) {
	prompt := buildMigrationPlanPrompt(app, resources, from, to)

	resp, err := c.sendMessage(ctx, prompt, migrationPlanSystemPrompt)
	if err != nil {
		return MigrationPlanResult{}, fmt.Errorf("generate migration plan: %w", err)
	}

	var result MigrationPlanResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return MigrationPlanResult{}, fmt.Errorf("parse migration plan: %w", err)
	}
	return result, nil
}

// sendMessage sends a message to the Anthropic API and extracts the text response.
func (c *AnthropicClient) sendMessage(ctx context.Context, userPrompt, systemPrompt string) (string, error) {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API call: %w", err)
	}

	// Extract text from response using type switch
	for _, block := range resp.Content {
		if block.Type == "text" {
			return extractJSON(block.Text), nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

// extractJSON attempts to extract a JSON object from text that may contain
// markdown fences or surrounding prose.
func extractJSON(text string) string {
	// Try to find JSON between ```json and ``` markers
	start := -1
	end := -1

	for i := 0; i < len(text)-6; i++ {
		if text[i:i+7] == "```json" {
			start = i + 7
			// Skip whitespace after marker
			for start < len(text) && (text[start] == '\n' || text[start] == '\r') {
				start++
			}
		}
		if start >= 0 && text[i:i+3] == "```" && i > start {
			end = i
			break
		}
	}
	if start >= 0 && end > start {
		result := text[start:end]
		// Trim trailing whitespace
		for len(result) > 0 && (result[len(result)-1] == '\n' || result[len(result)-1] == '\r' || result[len(result)-1] == ' ') {
			result = result[:len(result)-1]
		}
		return result
	}

	// Try to find raw JSON object
	braceStart := -1
	braceCount := 0
	for i, ch := range text {
		if ch == '{' {
			if braceStart == -1 {
				braceStart = i
			}
			braceCount++
		}
		if ch == '}' {
			braceCount--
			if braceCount == 0 && braceStart >= 0 {
				return text[braceStart : i+1]
			}
		}
	}

	// Try to find raw JSON array
	bracketStart := -1
	bracketCount := 0
	for i, ch := range text {
		if ch == '[' {
			if bracketStart == -1 {
				bracketStart = i
			}
			bracketCount++
		}
		if ch == ']' {
			bracketCount--
			if bracketCount == 0 && bracketStart >= 0 {
				return text[bracketStart : i+1]
			}
		}
	}

	return text
}
