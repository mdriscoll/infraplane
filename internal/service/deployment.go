package service

import (
	"context"
	"encoding/json"
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
	eventStore  *deploymentEventStore
}

// NewDeploymentService creates a new DeploymentService.
func NewDeploymentService(deployments repository.DeploymentRepo, apps repository.ApplicationRepo) *DeploymentService {
	return &DeploymentService{
		deployments: deployments,
		apps:        apps,
		eventStore:  newDeploymentEventStore(),
	}
}

// GetDeploymentEvents returns stored events for a deployment.
func (s *DeploymentService) GetDeploymentEvents(deploymentID uuid.UUID) []domain.DeploymentEvent {
	return s.eventStore.GetEvents(deploymentID)
}

// SubscribeEvents returns stored events and a live channel for an in-progress deployment.
// Returns (storedEvents, liveChan, alreadyComplete).
// The liveChan will be closed when the deployment finishes. If alreadyComplete is true, liveChan is nil.
func (s *DeploymentService) SubscribeEvents(deploymentID uuid.UUID) ([]domain.DeploymentEvent, chan domain.DeploymentEvent, bool) {
	return s.eventStore.Subscribe(deploymentID)
}

// UnsubscribeEvents removes a subscriber channel.
func (s *DeploymentService) UnsubscribeEvents(deploymentID uuid.UUID, ch chan domain.DeploymentEvent) {
	s.eventStore.Unsubscribe(deploymentID, ch)
}

// Deploy creates a new deployment for an application, optionally linked to a plan.
func (s *DeploymentService) Deploy(ctx context.Context, appID uuid.UUID, gitCommit, gitBranch string, planID *uuid.UUID, target *domain.DeployTarget) (domain.Deployment, error) {
	app, err := s.apps.GetByID(ctx, appID)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("get application: %w", err)
	}

	d := domain.NewDeployment(appID, app.Provider, gitCommit, gitBranch, planID, target)
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

// Execute runs phase 1 of a deployment: generates Terraform and pauses for approval.
// It sends DeploymentEvent values to the events channel and closes it when done.
// Events are also stored in the event store for reconnection by late-joining clients.
// After HCL generation, the deployment enters "awaiting_approval" status and the
// stream closes. Use Resume() after user approval to run the apply phase.
func (s *DeploymentService) Execute(
	ctx context.Context,
	deploymentID uuid.UUID,
	infra *InfraService,
	events chan<- domain.DeploymentEvent,
) {
	defer close(events)

	emit := func(step domain.DeploymentStep, msg string, status domain.DeploymentStatus, detail string) {
		event := domain.DeploymentEvent{
			Step:      step,
			Message:   msg,
			Timestamp: time.Now().UTC(),
			Status:    status,
			Detail:    detail,
		}
		// Store event for reconnection
		s.eventStore.Append(deploymentID, event)
		// Send to primary SSE channel
		select {
		case events <- event:
		case <-ctx.Done():
		}
	}

	// 1. Look up the deployment
	d, err := s.deployments.GetByID(ctx, deploymentID)
	if err != nil {
		emit(domain.StepFailed, "Deployment not found: "+err.Error(), domain.DeploymentFailed, "")
		s.eventStore.MarkComplete(deploymentID)
		return
	}

	// Guard: only execute pending deployments
	if d.Status != domain.DeploymentPending {
		emit(domain.StepFailed, fmt.Sprintf("Deployment is %s, not pending", d.Status), d.Status, "")
		s.eventStore.MarkComplete(deploymentID)
		return
	}

	// 2. Mark in_progress
	d.Status = domain.DeploymentInProgress
	_ = s.deployments.Update(ctx, d)
	emit(domain.StepInitializing, "Deployment started. Initializing workspace...", domain.DeploymentInProgress, "")

	if ctx.Err() != nil {
		s.failDeploy(ctx, &d)
		s.eventStore.MarkComplete(deploymentID)
		return
	}

	// 3. Generate Terraform with per-resource breakdown
	emit(domain.StepGeneratingTerraform, "Generating Terraform configuration...", domain.DeploymentInProgress, "")

	hcl, resourceHCLs, err := infra.GenerateTerraformWithResources(ctx, d.ApplicationID, d.DeployTarget)
	if err != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Terraform generation failed: "+err.Error(), domain.DeploymentFailed, "")
		s.eventStore.MarkComplete(deploymentID)
		return
	}

	// Emit per-resource HCL events for the review UI
	for _, rh := range resourceHCLs {
		detailJSON, _ := json.Marshal(rh)
		emit(domain.StepGeneratingTerraform,
			fmt.Sprintf("Generated HCL for %s (%s)", rh.ResourceName, rh.ServiceName),
			domain.DeploymentInProgress, string(detailJSON))
	}

	lineCount := len(hcl) / 40 // rough line estimate
	emit(domain.StepGeneratingTerraform,
		fmt.Sprintf("Terraform configuration generated (%d chars, ~%d lines).", len(hcl), lineCount),
		domain.DeploymentInProgress, "")

	// 4. Pause for user review/approval
	d.Status = domain.DeploymentAwaitingApproval
	d.TerraformPlan = hcl
	_ = s.deployments.Update(ctx, d)

	emit(domain.StepAwaitingApproval,
		"Terraform review ready. Approve to apply or reject to cancel.",
		domain.DeploymentAwaitingApproval, "")

	// Mark the event store as paused (not complete) so reconnecting clients
	// see the stored events and know the deployment is waiting for approval.
	s.eventStore.MarkPaused(deploymentID)
}

// Resume runs phase 2 of a deployment after user approval: validates credentials and applies Terraform.
// It sends DeploymentEvent values to the events channel and closes it when done.
func (s *DeploymentService) Resume(
	ctx context.Context,
	deploymentID uuid.UUID,
	infra *InfraService,
	events chan<- domain.DeploymentEvent,
) {
	defer close(events)
	defer s.eventStore.MarkComplete(deploymentID)

	// Unpause the event store so new subscribers get live events
	s.eventStore.MarkActive(deploymentID)

	emit := func(step domain.DeploymentStep, msg string, status domain.DeploymentStatus, detail string) {
		event := domain.DeploymentEvent{
			Step:      step,
			Message:   msg,
			Timestamp: time.Now().UTC(),
			Status:    status,
			Detail:    detail,
		}
		s.eventStore.Append(deploymentID, event)
		select {
		case events <- event:
		case <-ctx.Done():
		}
	}

	d, err := s.deployments.GetByID(ctx, deploymentID)
	if err != nil {
		emit(domain.StepFailed, "Deployment not found: "+err.Error(), domain.DeploymentFailed, "")
		return
	}

	if d.Status != domain.DeploymentAwaitingApproval {
		emit(domain.StepFailed, fmt.Sprintf("Deployment is %s, not awaiting_approval", d.Status), d.Status, "")
		return
	}

	// Mark in_progress for the apply phase
	d.Status = domain.DeploymentInProgress
	_ = s.deployments.Update(ctx, d)

	hcl := d.TerraformPlan // Stored from phase 1

	// 1. Validate Credentials
	emit(domain.StepValidatingCredentials, "Validating cloud credentials...", domain.DeploymentInProgress, "")

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

	if err := adapter.ValidateCredentials(ctx, d.DeployTarget); err != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Credential validation failed: "+err.Error(), domain.DeploymentFailed, "")
		return
	}

	emit(domain.StepValidatingCredentials, "Credentials validated successfully.", domain.DeploymentInProgress, "")

	// 2. Validate Terraform
	emit(domain.StepValidating, "Configuration validated.", domain.DeploymentInProgress, "")

	// 3. Apply — stream real terraform output
	emit(domain.StepApplying, "Running terraform init/plan/apply...", domain.DeploymentInProgress, "")

	lineCallback := func(line string) {
		emit(domain.StepApplying, line, domain.DeploymentInProgress, "")
	}

	planOutput, applyErr := adapter.ApplyTerraform(ctx, hcl, d.DeployTarget, lineCallback)
	if applyErr != nil {
		s.failDeploy(ctx, &d)
		emit(domain.StepFailed, "Terraform apply failed: "+applyErr.Error(), domain.DeploymentFailed, "")
		return
	}

	emit(domain.StepApplying, "Apply complete! Resources created.", domain.DeploymentInProgress, "")

	// 4. Mark succeeded
	now := time.Now().UTC()
	d.Status = domain.DeploymentSucceeded
	d.CompletedAt = &now
	if err := s.deployments.Update(ctx, d); err != nil {
		log.Printf("[deploy] failed to update deployment status: %v", err)
	}

	emit(domain.StepComplete, "Deployment succeeded.", domain.DeploymentSucceeded, planOutput)
}

// Reject cancels a deployment that is awaiting approval.
func (s *DeploymentService) Reject(ctx context.Context, deploymentID uuid.UUID) error {
	d, err := s.deployments.GetByID(ctx, deploymentID)
	if err != nil {
		return err
	}
	if d.Status != domain.DeploymentAwaitingApproval {
		return fmt.Errorf("deployment is %s, not awaiting_approval", d.Status)
	}

	now := time.Now().UTC()
	d.Status = domain.DeploymentFailed
	d.CompletedAt = &now
	if err := s.deployments.Update(ctx, d); err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}

	// Emit a rejection event and mark complete
	event := domain.DeploymentEvent{
		Step:      domain.StepFailed,
		Message:   "Deployment rejected by user.",
		Timestamp: now,
		Status:    domain.DeploymentFailed,
	}
	s.eventStore.Append(deploymentID, event)
	s.eventStore.MarkComplete(deploymentID)

	return nil
}

func (s *DeploymentService) failDeploy(ctx context.Context, d *domain.Deployment) {
	now := time.Now().UTC()
	d.Status = domain.DeploymentFailed
	d.CompletedAt = &now
	_ = s.deployments.Update(ctx, *d)
}
