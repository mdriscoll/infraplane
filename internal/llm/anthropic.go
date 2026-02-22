package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

func (c *AnthropicClient) GenerateHostingPlan(ctx context.Context, app domain.Application, resources []domain.Resource, complianceContext string) (HostingPlanResult, error) {
	prompt := buildHostingPlanPrompt(app, resources, complianceContext)

	resp, err := c.sendMessage(ctx, prompt, hostingPlanSystemPrompt, 12288)
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

func (c *AnthropicClient) GenerateTerraformHCL(ctx context.Context, resource domain.Resource, provider domain.CloudProvider, complianceContext string) (TerraformHCLResult, error) {
	prompt := buildTerraformHCLPrompt(resource, provider, complianceContext)

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
// The SDK auto-calculates an appropriate timeout based on maxTokens (up to 10 min).
// We do NOT set a manual context timeout — the SDK handles this correctly.
func (c *AnthropicClient) sendMessage(ctx context.Context, userPrompt, systemPrompt string, maxTokens int64) (string, error) {
	start := time.Now()
	log.Printf("[llm] sending request: model=%s max_tokens=%d prompt_len=%d system_len=%d",
		c.model, maxTokens, len(userPrompt), len(systemPrompt))

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
	elapsed := time.Since(start)
	if err != nil {
		log.Printf("[llm] API error after %s: %v", elapsed.Round(time.Millisecond), err)
		return "", fmt.Errorf("anthropic API call: %w", err)
	}

	log.Printf("[llm] response received in %s: stop_reason=%s input_tokens=%d output_tokens=%d",
		elapsed.Round(time.Millisecond), resp.StopReason,
		resp.Usage.InputTokens, resp.Usage.OutputTokens)

	// Check if the response was truncated
	if resp.StopReason == "max_tokens" {
		log.Printf("[llm] WARNING: response truncated at %d output tokens", resp.Usage.OutputTokens)
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
// markdown fences or surrounding prose. It tries multiple strategies and
// validates each extraction attempt with json.Valid before returning.
func extractJSON(text string) string {
	// Strategy 1: Try to find JSON between ```json and ``` markers.
	// Be careful not to match ``` inside the JSON content — we look for
	// the outermost closing ``` (searching from the end).
	start := -1
	for i := 0; i < len(text)-6; i++ {
		if text[i:i+7] == "```json" {
			start = i + 7
			for start < len(text) && (text[start] == '\n' || text[start] == '\r') {
				start++
			}
			break
		}
	}
	if start >= 0 {
		// Search backwards from the end for closing ```
		for end := len(text) - 3; end > start; end-- {
			if text[end:end+3] == "```" {
				candidate := strings.TrimSpace(text[start:end])
				if json.Valid([]byte(candidate)) {
					return candidate
				}
				break
			}
		}
	}

	// Strategy 2: If the text is already valid JSON, return as-is.
	trimmed := strings.TrimSpace(text)
	if json.Valid([]byte(trimmed)) {
		return trimmed
	}

	// Strategy 3: Find matching braces/brackets using depth tracking.
	// Try both { and [ and pick whichever comes first.
	firstBrace := strings.IndexByte(text, '{')
	firstBracket := strings.IndexByte(text, '[')

	type extraction struct {
		openChar  byte
		closeChar byte
		startPos  int
	}
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
		if result := matchBraces(text, ext.startPos, ext.openChar, ext.closeChar); result != "" {
			return result
		}
	}

	return text
}

// matchBraces extracts a balanced JSON structure from text starting at startPos.
// It tracks string state and escape sequences to avoid false matches.
func matchBraces(text string, startPos int, openChar, closeChar byte) string {
	depth := 0
	inString := false
	escaped := false

	for i := startPos; i < len(text); i++ {
		ch := text[i]
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
		if ch == openChar {
			depth++
		}
		if ch == closeChar {
			depth--
			if depth == 0 {
				candidate := text[startPos : i+1]
				if json.Valid([]byte(candidate)) {
					return candidate
				}
				// If not valid, keep looking (might be a false match)
				return candidate // Return best effort
			}
		}
	}

	return ""
}
