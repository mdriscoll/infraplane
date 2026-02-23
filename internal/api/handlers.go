package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/analyzer"
	gcpcloud "github.com/matthewdriscoll/infraplane/internal/cloud/gcp"
	"github.com/matthewdriscoll/infraplane/internal/compliance"
	"github.com/matthewdriscoll/infraplane/internal/credentials"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

// Handlers holds references to all services and provides HTTP handlers.
type Handlers struct {
	apps       *service.ApplicationService
	resources  *service.ResourceService
	planner    *service.PlannerService
	deployments *service.DeploymentService
	infra      *service.InfraService
	graphs     *service.GraphService
	discovery  *service.DiscoveryService
	compliance *compliance.Registry
	gcpManager *gcpcloud.Manager
	credStore  *credentials.Store
}

// NewHandlers creates a new Handlers.
func NewHandlers(
	apps *service.ApplicationService,
	resources *service.ResourceService,
	planner *service.PlannerService,
	deployments *service.DeploymentService,
	infra *service.InfraService,
	graphs *service.GraphService,
	discovery *service.DiscoveryService,
	complianceRegistry *compliance.Registry,
	gcpManager *gcpcloud.Manager,
	credStore *credentials.Store,
) *Handlers {
	return &Handlers{
		apps:       apps,
		resources:  resources,
		planner:    planner,
		deployments: deployments,
		infra:      infra,
		graphs:     graphs,
		discovery:  discovery,
		compliance: complianceRegistry,
		gcpManager: gcpManager,
		credStore:  credStore,
	}
}

// --- Request/Response Types ---

type registerAppRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	GitRepoURL           string                 `json:"git_repo_url"`
	SourcePath           string                 `json:"source_path"`
	Provider             string                 `json:"provider"`
	ComplianceFrameworks []string               `json:"compliance_frameworks,omitempty"`
	Files                []analyzer.FileContent `json:"files,omitempty"`
}

type analyzeUploadRequest struct {
	Files []analyzer.FileContent `json:"files"`
}

type addResourceRequest struct {
	Description string `json:"description"`
}

type migrationPlanRequest struct {
	FromProvider string `json:"from_provider"`
	ToProvider   string `json:"to_provider"`
}

type deployRequest struct {
	GitBranch    string        `json:"git_branch"`
	GitCommit    string        `json:"git_commit"`
	PlanID       string        `json:"plan_id"`
	DeployTarget *deployTarget `json:"deploy_target,omitempty"`
}

type deployTarget struct {
	AWSRegion    string `json:"aws_region,omitempty"`
	AWSAccountID string `json:"aws_account_id,omitempty"`
	AWSRoleARN   string `json:"aws_role_arn,omitempty"`
	GCPProjectID string `json:"gcp_project_id,omitempty"`
	GCPRegion    string `json:"gcp_region,omitempty"`
	GCPFolder    string `json:"gcp_folder,omitempty"`
}

type validateTargetRequest struct {
	Provider     string       `json:"provider"`
	DeployTarget deployTarget `json:"deploy_target"`
}

type onboardRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Provider             string                 `json:"provider"`
	SourcePath           string                 `json:"source_path"`
	ComplianceFrameworks []string               `json:"compliance_frameworks,omitempty"`
	Files                []analyzer.FileContent `json:"files,omitempty"`
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

	var opts *service.RegisterOpts
	if len(req.Files) > 0 {
		opts = &service.RegisterOpts{
			UploadedFiles: &analyzer.CodeContext{
				Files:   req.Files,
				Summary: "Uploaded from browser",
			},
		}
	}

	app, err := h.apps.Register(r.Context(), req.Name, req.Description, req.GitRepoURL, req.SourcePath, domain.CloudProvider(req.Provider), req.ComplianceFrameworks, opts)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, app)
}

func (h *Handlers) OnboardApplication(w http.ResponseWriter, r *http.Request) {
	var req onboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	var opts *service.RegisterOpts
	if len(req.Files) > 0 {
		opts = &service.RegisterOpts{
			UploadedFiles: &analyzer.CodeContext{
				Files:   req.Files,
				Summary: "Uploaded from onboarding wizard",
			},
		}
	}

	result, err := h.apps.Onboard(
		r.Context(),
		req.Name, req.Description, req.SourcePath,
		domain.CloudProvider(req.Provider),
		req.ComplianceFrameworks,
		opts,
		h.planner,
	)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
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

// --- Reanalyze Handler ---

func (h *Handlers) ReanalyzeSource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	if err := h.apps.ReanalyzeSource(r.Context(), app.ID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Analyze Upload Handler ---

func (h *Handlers) AnalyzeUpload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req analyzeUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Files) == 0 {
		writeError(w, http.StatusBadRequest, "no files provided")
		return
	}

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	codeCtx := analyzer.CodeContext{
		Files:   req.Files,
		Summary: "Uploaded from browser",
	}

	if err := h.apps.AnalyzeUploadedFiles(r.Context(), app.ID, codeCtx); err != nil {
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

// --- Analysis Run Handlers ---

func (h *Handlers) ListAnalysisRuns(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	runs, err := h.resources.ListAnalysisRuns(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if runs == nil {
		runs = []domain.AnalysisRun{}
	}

	writeJSON(w, http.StatusOK, runs)
}

func (h *Handlers) ListResourcesByRun(w http.ResponseWriter, r *http.Request) {
	runIDStr := chi.URLParam(r, "runId")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	resources, err := h.resources.ListResourcesByRun(r.Context(), runID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if resources == nil {
		resources = []domain.Resource{}
	}

	writeJSON(w, http.StatusOK, resources)
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

	var planID *uuid.UUID
	if req.PlanID != "" {
		id, err := uuid.Parse(req.PlanID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid plan_id: "+req.PlanID)
			return
		}
		planID = &id
	}

	var target *domain.DeployTarget
	if req.DeployTarget != nil {
		target = &domain.DeployTarget{
			AWSRegion:    req.DeployTarget.AWSRegion,
			AWSAccountID: req.DeployTarget.AWSAccountID,
			AWSRoleARN:   req.DeployTarget.AWSRoleARN,
			GCPProjectID: req.DeployTarget.GCPProjectID,
			GCPRegion:    req.DeployTarget.GCPRegion,
			GCPFolder:    req.DeployTarget.GCPFolder,
		}
	}

	d, err := h.deployments.Deploy(r.Context(), app.ID, req.GitCommit, req.GitBranch, planID, target)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

// ValidateDeployTarget validates cloud credentials for a given deploy target before deployment.
func (h *Handlers) ValidateDeployTarget(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req validateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	// Verify the application exists
	_, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	target := &domain.DeployTarget{
		AWSRegion:    req.DeployTarget.AWSRegion,
		AWSAccountID: req.DeployTarget.AWSAccountID,
		AWSRoleARN:   req.DeployTarget.AWSRoleARN,
		GCPProjectID: req.DeployTarget.GCPProjectID,
		GCPRegion:    req.DeployTarget.GCPRegion,
		GCPFolder:    req.DeployTarget.GCPFolder,
	}

	if err := h.infra.ValidateProvider(r.Context(), domain.CloudProvider(req.Provider), target); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid":   false,
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid":   true,
		"message": "Credentials validated successfully.",
	})
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

// DeployStream executes a pending deployment and streams real-time log events via SSE.
func (h *Handlers) DeployStream(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment ID")
		return
	}

	// Verify the deployment exists and is pending
	d, err := h.deployments.GetStatus(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if d.Status != domain.DeploymentPending {
		writeError(w, http.StatusConflict, fmt.Sprintf("deployment is %s, not pending", d.Status))
		return
	}

	// SSE requires http.Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events := make(chan domain.DeploymentEvent, 32)

	// Execute deployment in background goroutine
	go h.deployments.Execute(r.Context(), id, h.infra, events)

	// Stream events to client
	for event := range events {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// GetDeploymentEvents returns stored events for a deployment as JSON.
// Works for any deployment status: in_progress, succeeded, or failed.
func (h *Handlers) GetDeploymentEvents(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment ID")
		return
	}

	events := h.deployments.GetDeploymentEvents(id)
	writeJSON(w, http.StatusOK, events)
}

// DeployStreamReconnect streams stored + live deployment events via SSE.
// For in-progress deployments: sends stored events, then streams live events.
// For completed deployments: sends stored events and closes.
// This allows clients to reconnect after navigating away.
func (h *Handlers) DeployStreamReconnect(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment ID")
		return
	}

	// Verify the deployment exists
	_, err = h.deployments.GetStatus(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// SSE requires http.Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Get stored events and optionally subscribe for live ones
	storedEvents, liveChan, complete := h.deployments.SubscribeEvents(id)

	// Send stored events first
	for _, event := range storedEvents {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	flusher.Flush()

	// If already complete, we're done
	if complete {
		return
	}

	// Stream live events until deployment completes or client disconnects
	defer h.deployments.UnsubscribeEvents(id, liveChan)
	for {
		select {
		case event, ok := <-liveChan:
			if !ok {
				return // Channel closed — deployment complete
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return // Client disconnected
		}
	}
}

// ApproveDeployment resumes a deployment that is awaiting approval.
// This is a GET endpoint because EventSource (SSE) only supports GET.
// Connecting to this endpoint starts the apply phase and streams events.
func (h *Handlers) ApproveDeployment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment ID")
		return
	}

	// Verify the deployment exists and is awaiting approval
	d, err := h.deployments.GetStatus(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if d.Status != domain.DeploymentAwaitingApproval {
		writeError(w, http.StatusConflict, fmt.Sprintf("deployment is %s, not awaiting_approval", d.Status))
		return
	}

	// SSE requires http.Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	events := make(chan domain.DeploymentEvent, 32)

	// Resume deployment in background goroutine
	go h.deployments.Resume(r.Context(), id, h.infra, events)

	// Stream events to client
	for event := range events {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// RejectDeployment cancels a deployment that is awaiting approval.
func (h *Handlers) RejectDeployment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid deployment ID")
		return
	}

	if err := h.deployments.Reject(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// --- Terraform HCL Handler ---

func (h *Handlers) GenerateTerraformHCL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid resource ID")
		return
	}

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	hcl, err := h.resources.GenerateTerraformHCL(r.Context(), id, domain.CloudProvider(req.Provider))
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"hcl": hcl})
}

// --- Graph Handlers ---

func (h *Handlers) GenerateGraph(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	graph, err := h.graphs.GenerateGraph(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, graph)
}

func (h *Handlers) GetLatestGraph(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	graph, err := h.graphs.GetLatest(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, graph)
}

func (h *Handlers) GetLiveResources(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	app, err := h.apps.GetByName(r.Context(), name)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	result, err := h.discovery.DiscoverLiveResources(r.Context(), app.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Compliance Handlers ---

func (h *Handlers) ListComplianceFrameworks(w http.ResponseWriter, r *http.Request) {
	providerFilter := r.URL.Query().Get("provider")

	var frameworks []compliance.FrameworkInfo
	if providerFilter != "" {
		frameworks = h.compliance.ListFrameworksForProvider(domain.CloudProvider(providerFilter))
	} else {
		frameworks = h.compliance.ListFrameworks()
	}
	if frameworks == nil {
		frameworks = []compliance.FrameworkInfo{}
	}

	writeJSON(w, http.StatusOK, frameworks)
}

// --- GCP Project Handlers ---

// ListGCPProjects returns the GCP projects accessible to the current credentials.
func (h *Handlers) ListGCPProjects(w http.ResponseWriter, r *http.Request) {
	pc := h.gcpManager.ProjectClient()
	if pc == nil {
		writeError(w, http.StatusServiceUnavailable, "GCP credentials not configured — cannot list projects")
		return
	}

	projects, err := pc.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list GCP projects: %v", err))
		return
	}
	if projects == nil {
		projects = []gcpcloud.GCPProject{}
	}

	writeJSON(w, http.StatusOK, projects)
}

// --- GCP Credential Handlers ---

// UploadGCPCredentials handles uploading a GCP service account key JSON file.
func (h *Handlers) UploadGCPCredentials(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing or invalid file upload — send a 'file' field with multipart/form-data")
		return
	}
	defer file.Close()

	keyJSON, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to read uploaded file: %v", err))
		return
	}

	info, err := h.credStore.Save(keyJSON)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Re-initialize GCP clients with the new credentials
	if reinitErr := h.gcpManager.Reinitialize(r.Context()); reinitErr != nil {
		// Non-fatal — credentials are saved but clients may not fully refresh
		fmt.Printf("warning: GCP client reinit after credential upload: %v\n", reinitErr)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"configured":   true,
		"project_id":   info.ProjectID,
		"client_email": info.ClientEmail,
		"uploaded_at":  info.UploadedAt,
	})
}

// GetGCPCredentialStatus returns the current GCP credential status.
func (h *Handlers) GetGCPCredentialStatus(w http.ResponseWriter, r *http.Request) {
	info := h.credStore.GetInfo()
	if info == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"configured": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"configured":   true,
		"project_id":   info.ProjectID,
		"client_email": info.ClientEmail,
		"uploaded_at":  info.UploadedAt,
	})
}

// DeleteGCPCredentials removes the stored GCP credential.
func (h *Handlers) DeleteGCPCredentials(w http.ResponseWriter, r *http.Request) {
	if err := h.credStore.Delete(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete credentials: %v", err))
		return
	}

	// Re-initialize GCP clients (they will become nil since creds are gone)
	h.gcpManager.Reinitialize(r.Context())

	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
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
