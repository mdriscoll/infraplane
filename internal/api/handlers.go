package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// Handlers holds references to all services and provides HTTP handlers.
type Handlers struct {
	apps        *service.ApplicationService
	resources   *service.ResourceService
	planner     *service.PlannerService
	deployments *service.DeploymentService
}

// NewHandlers creates a new Handlers.
func NewHandlers(
	apps *service.ApplicationService,
	resources *service.ResourceService,
	planner *service.PlannerService,
	deployments *service.DeploymentService,
) *Handlers {
	return &Handlers{
		apps:        apps,
		resources:   resources,
		planner:     planner,
		deployments: deployments,
	}
}

// --- Request/Response Types ---

type registerAppRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GitRepoURL  string `json:"git_repo_url"`
	Provider    string `json:"provider"`
}

type addResourceRequest struct {
	Description string `json:"description"`
}

type migrationPlanRequest struct {
	FromProvider string `json:"from_provider"`
	ToProvider   string `json:"to_provider"`
}

type deployRequest struct {
	GitBranch string `json:"git_branch"`
	GitCommit string `json:"git_commit"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Application Handlers ---

func (h *Handlers) RegisterApplication(w http.ResponseWriter, r *http.Request) {
	var req registerAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app, err := h.apps.Register(r.Context(), req.Name, req.Description, req.GitRepoURL, domain.CloudProvider(req.Provider))
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, app)
}

func (h *Handlers) ListApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := h.apps.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if apps == nil {
		apps = []domain.Application{}
	}
	writeJSON(w, http.StatusOK, apps)
}

func (h *Handlers) GetApplication(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	resources, _ := h.resources.ListByApplication(r.Context(), app.ID)
	if resources == nil {
		resources = []domain.Resource{}
	}

	latest, latestErr := h.deployments.GetLatest(r.Context(), app.ID)

	result := map[string]any{
		"application": app,
		"resources":   resources,
	}
	if latestErr == nil {
		result["latest_deployment"] = latest
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	if err := h.apps.Delete(r.Context(), app.ID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Resource Handlers ---

func (h *Handlers) AddResource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req addResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	resource, err := h.resources.AddFromDescription(r.Context(), app.ID, req.Description)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resource)
}

func (h *Handlers) ListResources(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	resources, err := h.resources.ListByApplication(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if resources == nil {
		resources = []domain.Resource{}
	}

	writeJSON(w, http.StatusOK, resources)
}

func (h *Handlers) RemoveResource(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource ID")
		return
	}

	if err := h.resources.Remove(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Plan Handlers ---

func (h *Handlers) GenerateHostingPlan(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	plan, err := h.planner.GenerateHostingPlan(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (h *Handlers) GenerateMigrationPlan(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req migrationPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	plan, err := h.planner.GenerateMigrationPlan(r.Context(), app.ID, domain.CloudProvider(req.FromProvider), domain.CloudProvider(req.ToProvider))
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (h *Handlers) ListPlans(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	plans, err := h.planner.ListPlansByApplication(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if plans == nil {
		plans = []domain.InfrastructurePlan{}
	}

	writeJSON(w, http.StatusOK, plans)
}

// --- Deployment Handlers ---

func (h *Handlers) Deploy(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	d, err := h.deployments.Deploy(r.Context(), app.ID, req.GitCommit, req.GitBranch)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

func (h *Handlers) ListDeployments(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	deps, err := h.deployments.ListByApplication(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if deps == nil {
		deps = []domain.Deployment{}
	}

	writeJSON(w, http.StatusOK, deps)
}

func (h *Handlers) GetLatestDeployment(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	d, err := h.deployments.GetLatest(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, d)
}

func (h *Handlers) GetDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment ID")
		return
	}

	d, err := h.deployments.GetStatus(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, d)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, domain.ErrConflict) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if domain.IsValidationError(err) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
