package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
	"github.com/matthewdriscoll/infraplane/internal/api"
	gcpcloud "github.com/matthewdriscoll/infraplane/internal/cloud/gcp"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	mcpserver "github.com/matthewdriscoll/infraplane/internal/mcp"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
	"github.com/matthewdriscoll/infraplane/internal/repository/postgres"
	"github.com/matthewdriscoll/infraplane/internal/service"
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

	// Build GCP Cloud Asset Inventory client (optional — works without it)
	var assetClient *gcpcloud.AssetClient
	ac, acErr := gcpcloud.NewAssetClient(context.Background())
	if acErr != nil {
		log.Printf("GCP Asset Inventory unavailable: %v (targeted CLI discovery still works)", acErr)
	} else {
		assetClient = ac
		defer assetClient.Close()
	}

	// Build repositories and services
	var appSvc *service.ApplicationService
	var resSvc *service.ResourceService
	var planSvc *service.PlannerService
	var depSvc *service.DeploymentService
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

		appSvc = service.NewApplicationService(appRepo, resRepo, llmClient)
		resSvc = service.NewResourceService(resRepo, appRepo, llmClient)
		planSvc = service.NewPlannerService(planRepo, appRepo, resRepo, llmClient)
		depSvc = service.NewDeploymentService(depRepo, appRepo)
		graphSvc = service.NewGraphService(graphRepo, appRepo, resRepo, llmClient)
		discSvc = service.NewDiscoveryService(appRepo, llmClient, assetClient)

		log.Println("Using PostgreSQL storage")
	} else {
		// In-memory mode (for development/testing without a database)
		appRepo := mock.NewApplicationRepo()
		resRepo := mock.NewResourceRepo()
		depRepo := mock.NewDeploymentRepo()
		planRepo := mock.NewPlanRepo()
		graphRepo := mock.NewGraphRepo()

		appSvc = service.NewApplicationService(appRepo, resRepo, llmClient)
		resSvc = service.NewResourceService(resRepo, appRepo, llmClient)
		planSvc = service.NewPlannerService(planRepo, appRepo, resRepo, llmClient)
		depSvc = service.NewDeploymentService(depRepo, appRepo)
		graphSvc = service.NewGraphService(graphRepo, appRepo, resRepo, llmClient)
		discSvc = service.NewDiscoveryService(appRepo, llmClient, assetClient)

		log.Println("Using in-memory storage (set DATABASE_URL for PostgreSQL)")
	}

	if mode == "http" {
		// HTTP REST API mode for the dashboard
		router := api.NewRouter(appSvc, resSvc, planSvc, depSvc, graphSvc, discSvc)
		log.Printf("Infraplane REST API starting on :%s...", port)
		if err := http.ListenAndServe(":"+port, router); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	} else {
		// Default: MCP server on stdio for Claude Code
		mcpSrv := mcpserver.NewServer(appSvc, resSvc, planSvc, depSvc, graphSvc, discSvc)
		log.Println("Infraplane MCP server starting on stdio...")
		if err := server.ServeStdio(mcpSrv); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}
}
