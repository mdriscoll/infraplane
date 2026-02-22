package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/matthewdriscoll/infraplane/internal/compliance"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// NewRouter creates the REST API router with all routes registered.
func NewRouter(
	appSvc *service.ApplicationService,
	resSvc *service.ResourceService,
	planSvc *service.PlannerService,
	depSvc *service.DeploymentService,
	infraSvc *service.InfraService,
	graphSvc *service.GraphService,
	discSvc *service.DiscoveryService,
	complianceRegistry *compliance.Registry,
) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware)
	r.Use(middleware.Recoverer)

	h := NewHandlers(appSvc, resSvc, planSvc, depSvc, infraSvc, graphSvc, discSvc, complianceRegistry)

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Compliance
		r.Get("/compliance/frameworks", h.ListComplianceFrameworks)

		// Applications
		r.Post("/applications/onboard", h.OnboardApplication)
		r.Post("/applications", h.RegisterApplication)
		r.Get("/applications", h.ListApplications)
		r.Get("/applications/{name}", h.GetApplication)
		r.Delete("/applications/{name}", h.DeleteApplication)

		// Reanalyze
		r.Post("/applications/{name}/reanalyze", h.ReanalyzeSource)
		r.Post("/applications/{name}/analyze-upload", h.AnalyzeUpload)

		// Resources
		r.Post("/applications/{name}/resources", h.AddResource)
		r.Get("/applications/{name}/resources", h.ListResources)
		r.Delete("/resources/{id}", h.RemoveResource)
		r.Post("/resources/{id}/terraform", h.GenerateTerraformHCL)

		// Plans
		r.Post("/applications/{name}/hosting-plan", h.GenerateHostingPlan)
		r.Post("/applications/{name}/migration-plan", h.GenerateMigrationPlan)
		r.Get("/applications/{name}/plans", h.ListPlans)

		// Graphs
		r.Post("/applications/{name}/graph", h.GenerateGraph)
		r.Get("/applications/{name}/graph", h.GetLatestGraph)

		// Live Resources (POST because discovery actively queries cloud APIs)
		r.Post("/applications/{name}/live-resources", h.GetLiveResources)

		// Deployments
		r.Post("/applications/{name}/deploy", h.Deploy)
		r.Get("/applications/{name}/deployments", h.ListDeployments)
		r.Get("/applications/{name}/deployments/latest", h.GetLatestDeployment)
		r.Get("/deployments/{id}", h.GetDeploymentStatus)

		// Deployment SSE stream (real-time execution logs)
		r.Get("/deployments/{id}/stream", h.DeployStream)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
