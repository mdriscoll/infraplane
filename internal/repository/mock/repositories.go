package mock

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// ApplicationRepo is an in-memory mock implementation of repository.ApplicationRepo.
type ApplicationRepo struct {
	mu   sync.RWMutex
	apps map[uuid.UUID]domain.Application
}

func NewApplicationRepo() *ApplicationRepo {
	return &ApplicationRepo{apps: make(map[uuid.UUID]domain.Application)}
}

func (r *ApplicationRepo) Create(_ context.Context, app domain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.apps {
		if existing.Name == app.Name {
			return domain.ErrConflict
		}
	}
	r.apps[app.ID] = app
	return nil
}

func (r *ApplicationRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[id]
	if !ok {
		return app, domain.ErrNotFound
	}
	return app, nil
}

func (r *ApplicationRepo) GetByName(_ context.Context, name string) (domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, app := range r.apps {
		if app.Name == name {
			return app, nil
		}
	}
	return domain.Application{}, domain.ErrNotFound
}

func (r *ApplicationRepo) List(_ context.Context) ([]domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	apps := make([]domain.Application, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	return apps, nil
}

func (r *ApplicationRepo) Update(_ context.Context, app domain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.apps[app.ID]; !ok {
		return domain.ErrNotFound
	}
	r.apps[app.ID] = app
	return nil
}

func (r *ApplicationRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.apps[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.apps, id)
	return nil
}

// ResourceRepo is an in-memory mock implementation of repository.ResourceRepo.
type ResourceRepo struct {
	mu        sync.RWMutex
	resources map[uuid.UUID]domain.Resource
}

func NewResourceRepo() *ResourceRepo {
	return &ResourceRepo{resources: make(map[uuid.UUID]domain.Resource)}
}

func (r *ResourceRepo) Create(_ context.Context, res domain.Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resources[res.ID] = res
	return nil
}

func (r *ResourceRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.resources[id]
	if !ok {
		return res, domain.ErrNotFound
	}
	return res, nil
}

func (r *ResourceRepo) ListByApplicationID(_ context.Context, appID uuid.UUID) ([]domain.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var resources []domain.Resource
	for _, res := range r.resources {
		if res.ApplicationID == appID {
			resources = append(resources, res)
		}
	}
	return resources, nil
}

func (r *ResourceRepo) Update(_ context.Context, res domain.Resource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.resources[res.ID]; !ok {
		return domain.ErrNotFound
	}
	r.resources[res.ID] = res
	return nil
}

func (r *ResourceRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.resources[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.resources, id)
	return nil
}

// DeploymentRepo is an in-memory mock implementation of repository.DeploymentRepo.
type DeploymentRepo struct {
	mu          sync.RWMutex
	deployments map[uuid.UUID]domain.Deployment
}

func NewDeploymentRepo() *DeploymentRepo {
	return &DeploymentRepo{deployments: make(map[uuid.UUID]domain.Deployment)}
}

func (r *DeploymentRepo) Create(_ context.Context, d domain.Deployment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deployments[d.ID] = d
	return nil
}

func (r *DeploymentRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.deployments[id]
	if !ok {
		return d, domain.ErrNotFound
	}
	return d, nil
}

func (r *DeploymentRepo) ListByApplicationID(_ context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var deployments []domain.Deployment
	for _, d := range r.deployments {
		if d.ApplicationID == appID {
			deployments = append(deployments, d)
		}
	}
	return deployments, nil
}

func (r *DeploymentRepo) Update(_ context.Context, d domain.Deployment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.deployments[d.ID]; !ok {
		return domain.ErrNotFound
	}
	r.deployments[d.ID] = d
	return nil
}

func (r *DeploymentRepo) GetLatestByApplicationID(_ context.Context, appID uuid.UUID) (domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest domain.Deployment
	found := false
	for _, d := range r.deployments {
		if d.ApplicationID == appID {
			if !found || d.StartedAt.After(latest.StartedAt) {
				latest = d
				found = true
			}
		}
	}
	if !found {
		return latest, domain.ErrNotFound
	}
	return latest, nil
}

// PlanRepo is an in-memory mock implementation of repository.PlanRepo.
type PlanRepo struct {
	mu    sync.RWMutex
	plans map[uuid.UUID]domain.InfrastructurePlan
}

func NewPlanRepo() *PlanRepo {
	return &PlanRepo{plans: make(map[uuid.UUID]domain.InfrastructurePlan)}
}

func (r *PlanRepo) Create(_ context.Context, p domain.InfrastructurePlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plans[p.ID] = p
	return nil
}

func (r *PlanRepo) GetByID(_ context.Context, id uuid.UUID) (domain.InfrastructurePlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plans[id]
	if !ok {
		return p, domain.ErrNotFound
	}
	return p, nil
}

func (r *PlanRepo) ListByApplicationID(_ context.Context, appID uuid.UUID) ([]domain.InfrastructurePlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var plans []domain.InfrastructurePlan
	for _, p := range r.plans {
		if p.ApplicationID == appID {
			plans = append(plans, p)
		}
	}
	return plans, nil
}

// GraphRepo is an in-memory mock implementation of repository.GraphRepo.
type GraphRepo struct {
	mu     sync.RWMutex
	graphs map[uuid.UUID]domain.InfraGraph
}

func NewGraphRepo() *GraphRepo {
	return &GraphRepo{graphs: make(map[uuid.UUID]domain.InfraGraph)}
}

func (r *GraphRepo) Create(_ context.Context, g domain.InfraGraph) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.graphs[g.ID] = g
	return nil
}

func (r *GraphRepo) GetLatestByApplicationID(_ context.Context, appID uuid.UUID) (domain.InfraGraph, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest domain.InfraGraph
	found := false
	for _, g := range r.graphs {
		if g.ApplicationID == appID {
			if !found || g.CreatedAt.After(latest.CreatedAt) {
				latest = g
				found = true
			}
		}
	}
	if !found {
		return latest, domain.ErrNotFound
	}
	return latest, nil
}

func (r *GraphRepo) ListByApplicationID(_ context.Context, appID uuid.UUID) ([]domain.InfraGraph, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var graphs []domain.InfraGraph
	for _, g := range r.graphs {
		if g.ApplicationID == appID {
			graphs = append(graphs, g)
		}
	}
	return graphs, nil
}
