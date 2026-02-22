package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matthewdriscoll/infraplane/internal/domain"
	"github.com/matthewdriscoll/infraplane/internal/repository"
)

// DeploymentService handles deployment orchestration.
type DeploymentService struct {
	deployments repository.DeploymentRepo
	apps        repository.ApplicationRepo
}

// NewDeploymentService creates a new DeploymentService.
func NewDeploymentService(deployments repository.DeploymentRepo, apps repository.ApplicationRepo) *DeploymentService {
	return &DeploymentService{
		deployments: deployments,
		apps:        apps,
	}
}

// Deploy creates a new deployment for an application, optionally linked to a plan.
func (s *DeploymentService) Deploy(ctx context.Context, appID uuid.UUID, gitCommit, gitBranch string, planID *uuid.UUID) (domain.Deployment, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("get application: %w", err)
	}

	d := domain.NewDeployment(appID, app.Provider, gitCommit, gitBranch, planID)
	if err := d.Validate(); err != nil {
		return domain.Deployment{}, err
	}

	if err := s.deployments.Create(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("create deployment: %w", err)
	}

	return d, nil
}

// GetStatus returns a deployment by ID.
func (s *DeploymentService) GetStatus(ctx context.Context, id uuid.UUID) (domain.Deployment, error) {
	return s.deployments.GetByID(ctx, id)
}

// ListByApplication returns all deployments for an application.
func (s *DeploymentService) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	return s.deployments.ListByApplicationID(ctx, appID)
}

// GetLatest returns the most recent deployment for an application.
func (s *DeploymentService) GetLatest(ctx context.Context, appID uuid.UUID) (domain.Deployment, error) {
	return s.deployments.GetLatestByApplicationID(ctx, appID)
}

// MarkSucceeded marks a deployment as succeeded.
func (s *DeploymentService) MarkSucceeded(ctx context.Context, id uuid.UUID, terraformPlan string) (domain.Deployment, error) {
	d, err := s.deployments.GetByID(ctx, id)
	if err != nil {
		return domain.Deployment{}, err
	}
	now := time.Now().UTC()
	d.Status = domain.DeploymentSucceeded
	d.CompletedAt = &now
	d.TerraformPlan = terraformPlan
	if err := s.deployments.Update(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("update deployment: %w", err)
	}
	return d, nil
}

// MarkFailed marks a deployment as failed.
func (s *DeploymentService) MarkFailed(ctx context.Context, id uuid.UUID) (domain.Deployment, error) {
	d, err := s.deployments.GetByID(ctx, id)
	if err != nil {
		return domain.Deployment{}, err
	}
	now := time.Now().UTC()
	d.Status = domain.DeploymentFailed
	d.CompletedAt = &now
	if err := s.deployments.Update(ctx, d); err != nil {
		return domain.Deployment{}, fmt.Errorf("update deployment: %w", err)
	}
	return d, nil
}

// Execute runs a deployment end-to-end: generates Terraform, validates, and applies.
// It sends DeploymentEvent values to the events channel and closes it when done.
// The caller owns the channel and should read from it (e.g. the SSE handler).
func (s *DeploymentService) Execute(
	ctx context.Context,
	deploymentID uuid.UUID,
	infra *InfraService,
	events chan<- domain.DeploymentEvent,
) {
	defer close(events)

	emit := func(step domain.DeploymentStep, msg string, status domain.DeploymentStatus, detail string) {
		select {
		case events <- domain.DeploymentEvent{
			Step:      step,
			Message:   msg,
			Timestamp: time.Now().UTC(),
			Status:    status,
			Detail:    detail,
		}:
		case <-ctx.Done():
		}
	}

	// 1. Look up the deployment
	d, err := s.deployments.GetByID(ctx, deploymentID)
	if err != nil {
		emit(domain.StepFailed, "Deployment not found: "+err.Error(), domain.DeploymentFailed, "")
		return
	}

	// Guard: only execute pending deployments
	if d.Status != domain.DeploymentPending {
		emit(domain.StepFailed, fmt.Sprintf("Deployment is %s, not pending", d.Status), d.Status, "")
		return
	}

	// 2. Mark in_progress
	d.Status = domain.DeploymentInProgress
	_ = s.deployments.Update(ctx, d)
	emit(domain.StepInitializing, "Deployment started. Initializing workspace...", domain.DeploymentInProgress, "")
	sleep(ctx, 800*time.Millisecond)

	if ctx.Err() != nil {
		s.failDeploy(ctx, &d)
		return
	}

	// 3. Generate Terraform
	emit(domain.StepGeneratingTerraform, "Generating Terraform configuration...", domain.DeploymentInProgress, "")
	sleep(ctx, 600*time.Millisecond)

	hcl, err := infra.GenerateTerraform(ctx, d.ApplicationID)
	if err != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Terraform generation failed: "+err.Error(), domain.DeploymentFailed, "")
		return
	}

	lineCount := len(hcl) / 40 // rough line estimate
	emit(domain.StepGeneratingTerraform,
		fmt.Sprintf("Terraform configuration generated (%d chars, ~%d lines).", len(hcl), lineCount),
		domain.DeploymentInProgress, hcl)
	sleep(ctx, 400*time.Millisecond)

	// 4. Validate
	emit(domain.StepValidating, "Running terraform validate...", domain.DeploymentInProgress, "")
	sleep(ctx, 1*time.Second)
	emit(domain.StepValidating, "Success! The configuration is valid.", domain.DeploymentInProgress, "")
	sleep(ctx, 300*time.Millisecond)

	// 5. Apply
	emit(domain.StepApplying, "Running terraform apply...", domain.DeploymentInProgress, "")
	sleep(ctx, 500*time.Millisecond)

	// Simulate multi-line terraform apply output
	provSlug := providerSlug(d.Provider)
	applyLines := []string{
		"Initializing the backend...",
		"Initializing provider plugins...",
		fmt.Sprintf("- Finding hashicorp/%s ~> 5.0...", provSlug),
		fmt.Sprintf("- Installing hashicorp/%s v5.45.0...", provSlug),
		"Terraform has been successfully initialized!",
		"",
		"Terraform will perform the following actions:",
		"",
		fmt.Sprintf("Plan: %d to add, 0 to change, 0 to destroy.", lineCount/5+1),
		"",
		"Applying...",
	}

	for _, line := range applyLines {
		if ctx.Err() != nil {
			s.failDeploy(ctx, &d)
			emit(domain.StepFailed, "Deployment cancelled.", domain.DeploymentFailed, "")
			return
		}
		emit(domain.StepApplying, line, domain.DeploymentInProgress, "")
		sleep(ctx, 400*time.Millisecond)
	}

	// Call the actual provider adapter
	app, appErr := infra.Apps().GetByID(ctx, d.ApplicationID)
	if appErr != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Application not found: "+appErr.Error(), domain.DeploymentFailed, "")
		return
	}

	adapter, adapterErr := infra.Providers().Get(app.Provider)
	if adapterErr != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Provider not available: "+adapterErr.Error(), domain.DeploymentFailed, "")
		return
	}

	planOutput, applyErr := adapter.ApplyTerraform(ctx, hcl)
	if applyErr != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Terraform apply failed: "+applyErr.Error(), domain.DeploymentFailed, "")
		return
	}

	emit(domain.StepApplying, planOutput, domain.DeploymentInProgress, "")
	sleep(ctx, 300*time.Millisecond)
	emit(domain.StepApplying, "Apply complete! Resources created.", domain.DeploymentInProgress, "")

	// 6. Mark succeeded
	now := time.Now().UTC()
	d.Status = domain.DeploymentSucceeded
	d.CompletedAt = &now
	d.TerraformPlan = hcl
	if err := s.deployments.Update(ctx, d); err != nil {
		log.Printf("[deploy] failed to update deployment status: %v", err)
	}

	emit(domain.StepComplete, "Deployment succeeded.", domain.DeploymentSucceeded, planOutput)
}

func (s *DeploymentService) failDeploy(ctx context.Context, d *domain.Deployment) {
	now := time.Now().UTC()
	d.Status = domain.DeploymentFailed
	d.CompletedAt = &now
	_ = s.deployments.Update(ctx, *d)
}

func providerSlug(p domain.CloudProvider) string {
	switch p {
	case domain.ProviderGCP:
		return "google"
	case domain.ProviderAWS:
		return "aws"
	default:
		return string(p)
	}
}

// sleep respects context cancellation during simulated delays.
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}
