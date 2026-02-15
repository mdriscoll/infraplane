package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// AnthropicClient implements Client using the Anthropic API with Claude Sonnet 4.5.
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
		model:  anthropic.ModelClaudeSonnet4_5,
	}
}

func (c *AnthropicClient) AnalyzeResourceNeed(ctx context.Context, description string, provider domain.CloudProvider) (ResourceRecommendation, error) {
	prompt := buildResourceAnalysisPrompt(description, provider)

	resp, err := c.sendMessage(ctx, prompt, resourceAnalysisSystemPrompt, 4096)
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

	resp, err := c.sendMessage(ctx, prompt, codebaseAnalysisSystemPrompt, 8192)
	if err != nil {
		return nil, fmt.Errorf("analyze codebase: %w", err)
	}

	// The LLM may return:
	// 1. A bare JSON array: [...]
	// 2. A wrapper object: {"resources": [...]}
	// 3. A single resource object: {"kind": ..., "name": ..., ...}
	// Note: extractJSON may have extracted only the first {...} from an array
	// response, so we handle all cases robustly.
	var results []ResourceRecommendation
	if err := json.Unmarshal([]byte(resp), &results); err != nil {
		// Not a bare array — try as an object
		var wrapper map[string]json.RawMessage
		if wrapErr := json.Unmarshal([]byte(resp), &wrapper); wrapErr == nil {
			// Case 2: Wrapper object with an array value like {"resources": [...]}
			for _, key := range []string{"resources", "recommendations", "results"} {
				if raw, ok := wrapper[key]; ok {
					if innerErr := json.Unmarshal(raw, &results); innerErr == nil {
						return results, nil
					}
				}
			}
			// Try any key that contains an array
			for _, raw := range wrapper {
				if innerErr := json.Unmarshal(raw, &results); innerErr == nil {
					return results, nil
				}
			}

			// Case 3: Single resource object — the LLM returned one resource
			// or extractJSON grabbed just the first object from an array.
			if _, hasKind := wrapper["kind"]; hasKind {
				var single ResourceRecommendation
				if singleErr := json.Unmarshal([]byte(resp), &single); singleErr == nil {
					log.Printf("codebase analysis: LLM returned single resource, wrapping as array")
					return []ResourceRecommendation{single}, nil
				}
			}
		}
		return nil, fmt.Errorf("parse codebase analysis: %w", err)
	}
	return results, nil
}

func (c *AnthropicClient) GenerateHostingPlan(ctx context.Context, app domain.Application, resources []domain.Resource) (HostingPlanResult, error) {
	prompt := buildHostingPlanPrompt(app, resources)

	resp, err := c.sendMessage(ctx, prompt, hostingPlanSystemPrompt, 16384)
	if err != nil {
		return HostingPlanResult{}, fmt.Errorf("generate hosting plan: %w", err)
	}

	if resp == "" {
		return HostingPlanResult{}, fmt.Errorf("generate hosting plan: LLM returned empty response")
	}

	var result HostingPlanResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// Log for debugging — the extractJSON function may have truncated
		log.Printf("hosting plan parse error for %s: resp length=%d, first 200 chars: %.200s", app.Name, len(resp), resp)
		return HostingPlanResult{}, fmt.Errorf("parse hosting plan: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) GenerateMigrationPlan(ctx context.Context, app domain.Application, resources []domain.Resource, from, to domain.CloudProvider) (MigrationPlanResult, error) {
	prompt := buildMigrationPlanPrompt(app, resources, from, to)

	resp, err := c.sendMessage(ctx, prompt, migrationPlanSystemPrompt, 16384)
	if err != nil {
		return MigrationPlanResult{}, fmt.Errorf("generate migration plan: %w", err)
	}

	var result MigrationPlanResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return MigrationPlanResult{}, fmt.Errorf("parse migration plan: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) GenerateGraph(ctx context.Context, app domain.Application, resources []domain.Resource) (GraphResult, error) {
	prompt := buildGraphPrompt(app, resources)

	resp, err := c.sendMessage(ctx, prompt, graphSystemPrompt, 4096)
	if err != nil {
		return GraphResult{}, fmt.Errorf("generate graph: %w", err)
	}

	var result GraphResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return GraphResult{}, fmt.Errorf("parse graph: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) GenerateTerraformHCL(ctx context.Context, resource domain.Resource, provider domain.CloudProvider) (TerraformHCLResult, error) {
	prompt := buildTerraformHCLPrompt(resource, provider)

	resp, err := c.sendMessage(ctx, prompt, terraformHCLSystemPrompt, 8192)
	if err != nil {
		return TerraformHCLResult{}, fmt.Errorf("generate terraform HCL: %w", err)
	}

	var result TerraformHCLResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return TerraformHCLResult{}, fmt.Errorf("parse terraform HCL: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) GenerateDiscoveryCommands(ctx context.Context, app domain.Application, codeCtx analyzer.CodeContext) (DiscoveryCommandResult, error) {
	prompt := buildDiscoveryCommandsPrompt(app, codeCtx)

	resp, err := c.sendMessage(ctx, prompt, discoveryCommandsSystemPrompt, 4096)
	if err != nil {
		return DiscoveryCommandResult{}, fmt.Errorf("generate discovery commands: %w", err)
	}

	var result DiscoveryCommandResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return DiscoveryCommandResult{}, fmt.Errorf("parse discovery commands: %w", err)
	}
	return result, nil
}

func (c *AnthropicClient) ParseDiscoveryOutput(ctx context.Context, app domain.Application, commandOutputs []CommandOutput) (LiveResourceParseResult, error) {
	prompt := buildDiscoveryOutputParsePrompt(app, commandOutputs)

	resp, err := c.sendMessage(ctx, prompt, discoveryOutputParseSystemPrompt, 8192)
	if err != nil {
		return LiveResourceParseResult{}, fmt.Errorf("parse discovery output: %w", err)
	}

	var result LiveResourceParseResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return LiveResourceParseResult{}, fmt.Errorf("parse discovery result: %w", err)
	}
	return result, nil
}

// sendMessage sends a message to the Anthropic API and extracts the text response.
func (c *AnthropicClient) sendMessage(ctx context.Context, userPrompt, systemPrompt string, maxTokens int64) (string, error) {
	// Apply a 90-second timeout so requests don't hang forever
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: maxTokens,
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

	// Check if the response was truncated
	if resp.StopReason == "max_tokens" {
		return "", fmt.Errorf("response truncated (hit %d token limit) — try reducing prompt complexity", maxTokens)
	}

	// Extract text from response
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

	// Find the first occurrence of { and [ to decide which to extract.
	// This matters because [{"a":1}] should extract the array, not the first object.
	firstBrace := -1
	firstBracket := -1
	for i, ch := range text {
		if ch == '{' && firstBrace == -1 {
			firstBrace = i
		}
		if ch == '[' && firstBracket == -1 {
			firstBracket = i
		}
		if firstBrace >= 0 && firstBracket >= 0 {
			break
		}
	}

	// Extract whichever comes first (array or object)
	type extraction struct {
		openChar  rune
		closeChar rune
		startPos  int
	}
	// Build ordered list of extractions to try, preferring whichever appears first
	var extractions []extraction
	if firstBracket >= 0 && (firstBrace < 0 || firstBracket < firstBrace) {
		extractions = append(extractions, extraction{'[', ']', firstBracket})
		if firstBrace >= 0 {
			extractions = append(extractions, extraction{'{', '}', firstBrace})
		}
	} else if firstBrace >= 0 {
		extractions = append(extractions, extraction{'{', '}', firstBrace})
		if firstBracket >= 0 {
			extractions = append(extractions, extraction{'[', ']', firstBracket})
		}
	}

	for _, ext := range extractions {
		depth := 0
		inString := false
		escaped := false
		for i, ch := range text[ext.startPos:] {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && inString {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			if ch == ext.openChar {
				depth++
			}
			if ch == ext.closeChar {
				depth--
				if depth == 0 {
					return text[ext.startPos : ext.startPos+i+1]
				}
			}
		}
	}

	return text
}
