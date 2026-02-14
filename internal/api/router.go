package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// NewRouter creates the REST API router with all routes registered.
func NewRouter(
	appSvc *service.ApplicationService,
	resSvc *service.ResourceService,
	planSvc *service.PlannerService,
	depSvc *service.DeploymentService,
) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	h := NewHandlers(appSvc, resSvc, planSvc, depSvc)

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Applications
		r.Post("/applications", h.RegisterApplication)
		r.Get("/applications", h.ListApplications)
		r.Get("/applications/{name}", h.GetApplication)
		r.Delete("/applications/{name}", h.DeleteApplication)

		// Reanalyze
		r.Post("/applications/{name}/reanalyze", h.ReanalyzeSource)

		// Resources
		r.Post("/applications/{name}/resources", h.AddResource)
		r.Get("/applications/{name}/resources", h.ListResources)
		r.Delete("/resources/{id}", h.RemoveResource)

		// Plans
		r.Post("/applications/{name}/hosting-plan", h.GenerateHostingPlan)
		r.Post("/applications/{name}/migration-plan", h.GenerateMigrationPlan)
		r.Get("/applications/{name}/plans", h.ListPlans)

		// Deployments
		r.Post("/applications/{name}/deploy", h.Deploy)
		r.Get("/applications/{name}/deployments", h.ListDeployments)
		r.Get("/applications/{name}/deployments/latest", h.GetLatestDeployment)
		r.Get("/deployments/{id}", h.GetDeploymentStatus)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
