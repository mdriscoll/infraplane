package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matthewdriscoll/infraplane/internal/api"
	gcpcloud "github.com/matthewdriscoll/infraplane/internal/cloud/gcp"
	"github.com/matthewdriscoll/infraplane/internal/compliance"
	"github.com/matthewdriscoll/infraplane/internal/credentials"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	mcpserver "github.com/matthewdriscoll/infraplane/internal/mcp"
	"github.com/matthewdriscoll/infraplane/internal/provider"
	awsadapter "github.com/matthewdriscoll/infraplane/internal/provider/aws"
	gcpadapter "github.com/matthewdriscoll/infraplane/internal/provider/gcp"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
	"github.com/matthewdriscoll/infraplane/internal/repository/postgres"
	"github.com/matthewdriscoll/infraplane/internal/service"
	tfexec "github.com/matthewdriscoll/infraplane/internal/terraform"
)

func main() {
	// Load .env file (overrides existing env vars so .env is always authoritative)
	if err := godotenv.Overload(); err != nil {
		log.Printf(".env file not loaded: %v", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		fmt.Println("Migration support coming soon")
		return
	}

	databaseURL := os.Getenv("DATABASE_URL")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	mode := os.Getenv("MCP_MODE") // "mcp" (default) or "http"
	port := os.Getenv("PORT")

	if anthropicKey == "" {
		log.Println("WARNING: ANTHROPIC_API_KEY is not set — LLM features will fail")
	}
	if port == "" {
		port = "8080"
	}

	// Build LLM client
	llmClient := llm.NewAnthropicClient(anthropicKey)

	// Build credential store for GCP service account key uploads
	credStore := credentials.NewStore()

	// Build GCP client manager (manages ProjectClient + AssetClient lifecycle)
	gcpManager := gcpcloud.NewManager(context.Background())
	defer gcpManager.Close()

	// Build compliance registry
	complianceRegistry := compliance.NewRegistry()
	log.Printf("Compliance registry loaded: %d frameworks", len(complianceRegistry.ListFrameworks()))

	// Build Terraform executor
	tfBinary := os.Getenv("TERRAFORM_BINARY")
	if tfBinary == "" {
		tfBinary = "terraform"
	}
	tfExecutor := tfexec.NewExecutor(tfBinary, 10*time.Minute)

	// Build provider registry with real terraform executor
	providerRegistry := provider.NewRegistry()
	providerRegistry.Register(awsadapter.NewAdapter(tfExecutor))
	providerRegistry.Register(gcpadapter.NewAdapter(tfExecutor))

	// Build repositories and services
	var appSvc *service.ApplicationService
	var resSvc *service.ResourceService
	var planSvc *service.PlannerService
	var depSvc *service.DeploymentService
	var infraSvc *service.InfraService
	var graphSvc *service.GraphService
	var discSvc *service.DiscoveryService

	if databaseURL != "" {
		// PostgreSQL mode
		pool, err := postgres.NewPool(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("connect to database: %v", err)
		}
		defer pool.Close()

		appRepo := postgres.NewApplicationRepo(pool)
		resRepo := postgres.NewResourceRepo(pool)
		depRepo := postgres.NewDeploymentRepo(pool)
		planRepo := postgres.NewPlanRepo(pool)
		graphRepo := postgres.NewGraphRepo(pool)
		analysisRunRepo := postgres.NewAnalysisRunRepo(pool)

		// Clean up orphaned deployments from previous server crash/restart
		if n, err := depRepo.FailOrphaned(context.Background()); err != nil {
			log.Printf("WARNING: failed to clean up orphaned deployments: %v", err)
		} else if n > 0 {
			log.Printf("Cleaned up %d orphaned deployment(s) from previous run", n)
		}

		appSvc = service.NewApplicationService(appRepo, resRepo, analysisRunRepo, llmClient, complianceRegistry)
		resSvc = service.NewResourceService(resRepo, appRepo, analysisRunRepo, llmClient, complianceRegistry)
		planSvc = service.NewPlannerService(planRepo, appRepo, resRepo, llmClient, complianceRegistry)
		depSvc = service.NewDeploymentService(depRepo, appRepo)
		infraSvc = service.NewInfraService(appRepo, resRepo, depRepo, providerRegistry, llmClient, complianceRegistry)
		graphSvc = service.NewGraphService(graphRepo, appRepo, resRepo, llmClient)
		discSvc = service.NewDiscoveryService(appRepo, llmClient, gcpManager.AssetClient())

		log.Println("Using PostgreSQL storage")
	} else {
		// In-memory mode (for development/testing without a database)
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		depRepo := mock.NewDeploymentRepo()
		planRepo := mock.NewPlanRepo()
		graphRepo := mock.NewGraphRepo()
		analysisRunRepo := mock.NewAnalysisRunRepo()
		resRepo.SetAnalysisRunRepo(analysisRunRepo)

		appSvc = service.NewApplicationService(appRepo, resRepo, analysisRunRepo, llmClient, complianceRegistry)
		resSvc = service.NewResourceService(resRepo, appRepo, analysisRunRepo, llmClient, complianceRegistry)
		planSvc = service.NewPlannerService(planRepo, appRepo, resRepo, llmClient, complianceRegistry)
		depSvc = service.NewDeploymentService(depRepo, appRepo)
		infraSvc = service.NewInfraService(appRepo, resRepo, depRepo, providerRegistry, llmClient, complianceRegistry)
		graphSvc = service.NewGraphService(graphRepo, appRepo, resRepo, llmClient)
		discSvc = service.NewDiscoveryService(appRepo, llmClient, gcpManager.AssetClient())

		log.Println("Using in-memory storage (set DATABASE_URL for PostgreSQL)")
	}

	if mode == "http" {
		// HTTP REST API mode for the dashboard
		router := api.NewRouter(appSvc, resSvc, planSvc, depSvc, infraSvc, graphSvc, discSvc, complianceRegistry, gcpManager, credStore)
		log.Printf("Infraplane REST API starting on :%s...", port)
		if err := http.ListenAndServe(":"+port, router); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	} else {
		// Default: MCP server on stdio for Claude Code
		mcpSrv := mcpserver.NewServer(appSvc, resSvc, planSvc, depSvc, graphSvc, discSvc, complianceRegistry)
		log.Println("Infraplane MCP server starting on stdio...")
		if err := server.ServeStdio(mcpSrv); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}
}
