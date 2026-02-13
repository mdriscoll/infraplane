package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/llm"
	"github.com/matthewdriscoll/infraplane/internal/repository/mock"
	"github.com/matthewdriscoll/infraplane/internal/service"
)

func setupTestRouter() http.Handler {
	appRepo := mock.NewApplicationRepo()
	resRepo := mock.NewResourceRepo()
	depRepo := mock.NewDeploymentRepo()
	planRepo := mock.NewPlanRepo()
	mockLLM := &llm.MockClient{}

	appSvc := service.NewApplicationService(appRepo)
	resSvc := service.NewResourceService(resRepo, appRepo, mockLLM)
	planSvc := service.NewPlannerService(planRepo, appRepo, resRepo, mockLLM)
	depSvc := service.NewDeploymentService(depRepo, appRepo)

	return NewRouter(appSvc, resSvc, planSvc, depSvc)
}

func doRequest(router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestHealthCheck(t *testing.T) {
	router := setupTestRouter()
	w := doRequest(router, "GET", "/health", nil)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRegisterApplication(t *testing.T) {
	router := setupTestRouter()

	t.Run("successful registration", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications", registerAppRequest{
			Name:        "test-api",
			Description: "A test API",
			Provider:    "aws",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}

		var app domain.Application
		json.NewDecoder(w.Body).Decode(&app)
		if app.Name != "test-api" {
			t.Errorf("name = %q, want %q", app.Name, "test-api")
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications", registerAppRequest{
			Name:     "bad-provider",
			Provider: "azure",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications", registerAppRequest{
			Provider: "aws",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestListApplications(t *testing.T) {
	router := setupTestRouter()

	// Register two apps
	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "app-1", Provider: "aws"})
	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "app-2", Provider: "gcp"})

	w := doRequest(router, "GET", "/api/applications", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var apps []domain.Application
	json.NewDecoder(w.Body).Decode(&apps)
	if len(apps) != 2 {
		t.Errorf("len = %d, want 2", len(apps))
	}
}

func TestGetApplication(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{
		Name: "get-test", Provider: "aws", Description: "test app",
	})

	t.Run("found", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/applications/get-test", nil)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		app := resp["application"].(map[string]any)
		if app["name"] != "get-test" {
			t.Errorf("name = %v, want get-test", app["name"])
		}
	})

	t.Run("not found", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/applications/nonexistent", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestDeleteApplication(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "delete-me", Provider: "aws"})

	t.Run("successful delete", func(t *testing.T) {
		w := doRequest(router, "DELETE", "/api/applications/delete-me", nil)
		if w.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
		}

		// Verify it's gone
		w2 := doRequest(router, "GET", "/api/applications/delete-me", nil)
		if w2.Code != http.StatusNotFound {
			t.Errorf("after delete: status = %d, want %d", w2.Code, http.StatusNotFound)
		}
	})

	t.Run("not found", func(t *testing.T) {
		w := doRequest(router, "DELETE", "/api/applications/ghost", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestAddAndListResources(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "res-app", Provider: "aws"})

	t.Run("add resource", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications/res-app/resources", addResourceRequest{
			Description: "I need a PostgreSQL database",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}
	})

	t.Run("list resources", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/applications/res-app/resources", nil)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resources []domain.Resource
		json.NewDecoder(w.Body).Decode(&resources)
		if len(resources) != 1 {
			t.Errorf("len = %d, want 1", len(resources))
		}
	})

	t.Run("app not found", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications/ghost/resources", addResourceRequest{
			Description: "something",
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestRemoveResource(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "rm-app", Provider: "aws"})
	addW := doRequest(router, "POST", "/api/applications/rm-app/resources", addResourceRequest{Description: "database"})

	var resource domain.Resource
	json.NewDecoder(addW.Body).Decode(&resource)

	t.Run("successful remove", func(t *testing.T) {
		w := doRequest(router, "DELETE", "/api/resources/"+resource.ID.String(), nil)
		if w.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
		}
	})

	t.Run("invalid ID", func(t *testing.T) {
		w := doRequest(router, "DELETE", "/api/resources/not-a-uuid", nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestGenerateHostingPlan(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "plan-app", Provider: "aws"})

	w := doRequest(router, "POST", "/api/applications/plan-app/hosting-plan", nil)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var plan domain.InfrastructurePlan
	json.NewDecoder(w.Body).Decode(&plan)
	if plan.PlanType != domain.PlanTypeHosting {
		t.Errorf("PlanType = %q, want %q", plan.PlanType, domain.PlanTypeHosting)
	}
}

func TestGenerateMigrationPlan(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "migrate-app", Provider: "aws"})

	t.Run("successful migration plan", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications/migrate-app/migration-plan", migrationPlanRequest{
			FromProvider: "aws",
			ToProvider:   "gcp",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}
	})

	t.Run("same provider error", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications/migrate-app/migration-plan", migrationPlanRequest{
			FromProvider: "aws",
			ToProvider:   "aws",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestDeploy(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "deploy-app", Provider: "gcp"})

	t.Run("successful deploy", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications/deploy-app/deploy", deployRequest{
			GitBranch: "main",
			GitCommit: "abc123",
		})
		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}

		var d domain.Deployment
		json.NewDecoder(w.Body).Decode(&d)
		if d.Provider != domain.ProviderGCP {
			t.Errorf("Provider = %q, want %q", d.Provider, domain.ProviderGCP)
		}
	})

	t.Run("missing branch", func(t *testing.T) {
		w := doRequest(router, "POST", "/api/applications/deploy-app/deploy", deployRequest{
			GitCommit: "abc",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestGetLatestDeployment(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "latest-app", Provider: "aws"})
	doRequest(router, "POST", "/api/applications/latest-app/deploy", deployRequest{GitBranch: "main", GitCommit: "abc"})

	t.Run("found", func(t *testing.T) {
		w := doRequest(router, "GET", "/api/applications/latest-app/deployments/latest", nil)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("no deployments", func(t *testing.T) {
		doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "no-deps", Provider: "aws"})
		w := doRequest(router, "GET", "/api/applications/no-deps/deployments/latest", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestListPlans(t *testing.T) {
	router := setupTestRouter()

	doRequest(router, "POST", "/api/applications", registerAppRequest{Name: "plans-app", Provider: "aws"})
	doRequest(router, "POST", "/api/applications/plans-app/hosting-plan", nil)

	w := doRequest(router, "GET", "/api/applications/plans-app/plans", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var plans []domain.InfrastructurePlan
	json.NewDecoder(w.Body).Decode(&plans)
	if len(plans) != 1 {
		t.Errorf("len = %d, want 1", len(plans))
	}
}
