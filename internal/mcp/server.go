package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// NewServer creates and configures the MCP server with all tools registered.
func NewServer(
	appSvc *service.ApplicationService,
	resSvc *service.ResourceService,
	planSvc *service.PlannerService,
	depSvc *service.DeploymentService,
) *server.MCPServer {
	s := server.NewMCPServer(
		"infraplane",
		"0.1.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	handlers := NewToolHandlers(appSvc, resSvc, planSvc, depSvc)
	handlers.RegisterAll(s)

	return s
}
