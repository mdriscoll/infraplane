package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	gcpcloud "github.com/matthewdriscoll/infraplane/internal/cloud/gcp"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/executor"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// DiscoveryService handles live cloud resource discovery.
type DiscoveryService struct {
	apps     repository.ApplicationRepo
	llm      llm.Client
	executor *executor.Executor
	assets   *gcpcloud.AssetClient // nil if GCP credentials unavailable
}

// NewDiscoveryService creates a new DiscoveryService.
// assetClient may be nil if GCP Cloud Asset Inventory is not available.
func NewDiscoveryService(apps repository.ApplicationRepo, llmClient llm.Client, assetClient *gcpcloud.AssetClient) *DiscoveryService {
	return &DiscoveryService{
		apps:     apps,
		llm:      llmClient,
		executor: executor.NewExecutor(),
		assets:   assetClient,
	}
}

// DiscoverLiveResources performs the full discovery pipeline:
// Phase A: LLM analyzes deploy scripts → generates CLI commands → executes → LLM parses results
// Phase B: GCP Cloud Asset Inventory scans the project for comprehensive coverage
// Results are merged and deduplicated.
func (s *DiscoveryService) DiscoverLiveResources(ctx context.Context, appID uuid.UUID) (domain.LiveResourceResult, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.LiveResourceResult{}, fmt.Errorf("get application: %w", err)
	}

	if app.SourcePath == "" {
		return domain.LiveResourceResult{}, domain.ErrValidation("application has no source path configured")
	}

	now := time.Now().UTC()
	var allResources []domain.LiveResource
	var nonFatalErrors []string

	// Phase A: Targeted discovery via LLM + CLI
	targeted, targetedErrs := s.targetedDiscovery(ctx, app)
	allResources = append(allResources, targeted...)
	nonFatalErrors = append(nonFatalErrors, targetedErrs...)

	// Phase B: Comprehensive discovery via Cloud Asset Inventory
	if s.assets != nil && app.Provider == domain.ProviderGCP {
		comprehensive, compErrs := s.comprehensiveDiscovery(ctx, app)
		// Merge — Phase B fills in resources Phase A might have missed
		allResources = mergeResources(allResources, comprehensive)
		nonFatalErrors = append(nonFatalErrors, compErrs...)
	}

	// Set LastChecked on all resources
	for i := range allResources {
		allResources[i].LastChecked = now
		if allResources[i].Provider == "" {
			allResources[i].Provider = app.Provider
		}
	}

	return domain.LiveResourceResult{
		Resources: allResources,
		Errors:    nonFatalErrors,
		Timestamp: now,
	}, nil
}

// targetedDiscovery uses the LLM to analyze deploy scripts, generate CLI commands,
// execute them, and parse the output.
func (s *DiscoveryService) targetedDiscovery(ctx context.Context, app domain.Application) ([]domain.LiveResource, []string) {
	var errors []string

	// Step 1: Analyze source to get deploy scripts
	codeCtx, err := analyzer.Analyze(app.SourcePath)
	if err != nil {
		return nil, []string{fmt.Sprintf("analyze source: %s", err)}
	}

	if len(codeCtx.Files) == 0 {
		return nil, []string{"no infrastructure files found at source path"}
	}

	// Step 2: Ask LLM to generate discovery commands
	cmdResult, err := s.llm.GenerateDiscoveryCommands(ctx, app, codeCtx)
	if err != nil {
		return nil, []string{fmt.Sprintf("generate discovery commands: %s", err)}
	}

	if len(cmdResult.Commands) == 0 {
		return nil, []string{"LLM generated no discovery commands from deploy scripts"}
	}

	// Step 2b: Resolve placeholder variables in commands
	// The LLM may use $GOOGLE_PROJECT, $AWS_REGION etc. when it can't extract values from scripts.
	resolveCommandPlaceholders(cmdResult.Commands)

	// Step 3: Execute commands and collect outputs
	var commandOutputs []llm.CommandOutput
	for _, cmd := range cmdResult.Commands {
		log.Printf("discovery: running %s: %s", cmd.Description, cmd.Command)
		result := s.executor.Execute(ctx, cmd.Command)

		output := llm.CommandOutput{
			Command: cmd,
			Output:  result.Stdout,
		}
		if result.Error != "" {
			// Include stderr in error message for actionable diagnostics
			errMsg := result.Error
			if result.Stderr != "" {
				// Extract first line of stderr for a concise error
				stderrLines := strings.SplitN(strings.TrimSpace(result.Stderr), "\n", 2)
				errMsg = stderrLines[0]
			}
			output.Error = errMsg
			errors = append(errors, fmt.Sprintf("%s: %s", cmd.Description, errMsg))
			log.Printf("discovery: command failed: %s (stderr: %s)", result.Error, result.Stderr)
		}

		// Only include in parse if we got some output
		if result.Stdout != "" || result.Error != "" {
			commandOutputs = append(commandOutputs, output)
		}
	}

	if len(commandOutputs) == 0 {
		return nil, errors
	}

	// Step 4: Ask LLM to parse outputs into structured data
	parseResult, err := s.llm.ParseDiscoveryOutput(ctx, app, commandOutputs)
	if err != nil {
		errors = append(errors, fmt.Sprintf("parse discovery output: %s", err))
		return nil, errors
	}

	return parseResult.Resources, errors
}

// comprehensiveDiscovery uses GCP Cloud Asset Inventory to scan the project.
func (s *DiscoveryService) comprehensiveDiscovery(ctx context.Context, app domain.Application) ([]domain.LiveResource, []string) {
	// Try to determine the GCP project ID from environment or deploy scripts
	projectID := getGCPProjectID()
	if projectID == "" {
		return nil, []string{"GCP project ID not found (set GOOGLE_PROJECT env var)"}
	}

	resources, err := s.assets.ListProjectAssets(ctx, projectID)
	if err != nil {
		return nil, []string{fmt.Sprintf("cloud asset inventory: %s", err)}
	}

	return resources, nil
}

// getGCPProjectID reads the GCP project ID from environment.
func getGCPProjectID() string {
	for _, key := range []string{"GOOGLE_PROJECT", "GOOGLE_CLOUD_PROJECT", "GCP_PROJECT_ID", "GCLOUD_PROJECT"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

// resolveCommandPlaceholders replaces placeholder variables ($GOOGLE_PROJECT, $AWS_REGION, etc.)
// with actual values from the environment or gcloud config.
func resolveCommandPlaceholders(commands []llm.DiscoveryCommand) {
	// Build replacement map from environment
	replacements := map[string]string{}

	// GCP project: try env vars, then gcloud config
	gcpProject := getGCPProjectID()
	if gcpProject == "" {
		gcpProject = getGcloudConfigProject()
	}
	if gcpProject != "" {
		replacements["$GOOGLE_PROJECT"] = gcpProject
		replacements["${GOOGLE_PROJECT}"] = gcpProject
		replacements["$GCP_PROJECT_ID"] = gcpProject
		replacements["${GCP_PROJECT_ID}"] = gcpProject
		replacements["$PROJECT_ID"] = gcpProject
		replacements["${PROJECT_ID}"] = gcpProject
	}

	// AWS region
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	if awsRegion != "" {
		replacements["$AWS_REGION"] = awsRegion
		replacements["${AWS_REGION}"] = awsRegion
	}

	if len(replacements) == 0 {
		return
	}

	for i, cmd := range commands {
		resolved := cmd.Command
		for placeholder, value := range replacements {
			resolved = strings.ReplaceAll(resolved, placeholder, value)
		}
		if resolved != cmd.Command {
			log.Printf("discovery: resolved placeholders: %s -> %s", cmd.Command, resolved)
			commands[i].Command = resolved
		}
	}
}

// getGcloudConfigProject reads the active gcloud project via `gcloud config get-value project`.
func getGcloudConfigProject() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "gcloud", "config", "get-value", "project").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// mergeResources combines Phase A (targeted) and Phase B (comprehensive) results,
// deduplicating by resource name + type.
func mergeResources(targeted, comprehensive []domain.LiveResource) []domain.LiveResource {
	// Build a set of name+type from targeted results
	seen := make(map[string]bool)
	for _, r := range targeted {
		key := r.Name + "|" + r.ResourceType
		seen[key] = true
	}

	// Add comprehensive results that aren't already in targeted
	result := make([]domain.LiveResource, len(targeted))
	copy(result, targeted)
	for _, r := range comprehensive {
		key := r.Name + "|" + r.ResourceType
		if !seen[key] {
			result = append(result, r)
			seen[key] = true
		}
	}

	return result
}
