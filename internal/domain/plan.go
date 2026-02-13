package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlanType represents the type of infrastructure plan.
type PlanType string

const (
	PlanTypeHosting   PlanType = "hosting"
	PlanTypeMigration PlanType = "migration"
)

// CostEstimate holds estimated cloud costs.
type CostEstimate struct {
	MonthlyCostUSD float64            `json:"monthly_cost_usd"`
	Breakdown      map[string]float64 `json:"breakdown"`
}

// InfrastructurePlan is an LLM-generated hosting or migration plan.
type InfrastructurePlan struct {
	ID            uuid.UUID      `json:"id"`
	ApplicationID uuid.UUID      `json:"application_id"`
	PlanType      PlanType       `json:"plan_type"`
	FromProvider  *CloudProvider `json:"from_provider,omitempty"`
	ToProvider    *CloudProvider `json:"to_provider,omitempty"`
	Content       string         `json:"content"`
	Resources     []Resource     `json:"resources"`
	EstimatedCost *CostEstimate  `json:"estimated_cost,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

// NewHostingPlan creates a new hosting plan.
func NewHostingPlan(appID uuid.UUID, content string, resources []Resource, cost *CostEstimate) InfrastructurePlan {
	return InfrastructurePlan{
		ID:            uuid.New(),
		ApplicationID: appID,
		PlanType:      PlanTypeHosting,
		Content:       content,
		Resources:     resources,
		EstimatedCost: cost,
		CreatedAt:     time.Now().UTC(),
	}
}

// NewMigrationPlan creates a new migration plan between providers.
func NewMigrationPlan(appID uuid.UUID, from, to CloudProvider, content string, resources []Resource, cost *CostEstimate) InfrastructurePlan {
	return InfrastructurePlan{
		ID:            uuid.New(),
		ApplicationID: appID,
		PlanType:      PlanTypeMigration,
		FromProvider:  &from,
		ToProvider:    &to,
		Content:       content,
		Resources:     resources,
		EstimatedCost: cost,
		CreatedAt:     time.Now().UTC(),
	}
}

// Validate checks that the plan has valid required fields.
func (p InfrastructurePlan) Validate() error {
	if p.ApplicationID == uuid.Nil {
		return ErrValidation("application ID is required")
	}
	if p.Content == "" {
		return ErrValidation("plan content is required")
	}
	if p.PlanType == PlanTypeMigration {
		if p.FromProvider == nil || p.ToProvider == nil {
			return ErrValidation("migration plan requires from and to providers")
		}
	}
	return nil
}
