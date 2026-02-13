package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// PlanRepo implements repository.PlanRepo with PostgreSQL.
type PlanRepo struct {
	pool *pgxpool.Pool
}

// NewPlanRepo creates a new PostgreSQL-backed plan repository.
func NewPlanRepo(pool *pgxpool.Pool) *PlanRepo {
	return &PlanRepo{pool: pool}
}

func (r *PlanRepo) Create(ctx context.Context, p domain.InfrastructurePlan) error {
	resourcesJSON, err := json.Marshal(p.Resources)
	if err != nil {
		return fmt.Errorf("marshal resources: %w", err)
	}

	var costJSON []byte
	if p.EstimatedCost != nil {
		costJSON, err = json.Marshal(p.EstimatedCost)
		if err != nil {
			return fmt.Errorf("marshal estimated cost: %w", err)
		}
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO infrastructure_plans (id, application_id, plan_type, from_provider, to_provider, content, resources, estimated_cost, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.ID, p.ApplicationID, p.PlanType, p.FromProvider, p.ToProvider, p.Content, resourcesJSON, costJSON, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert plan: %w", err)
	}
	return nil
}

func (r *PlanRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.InfrastructurePlan, error) {
	var p domain.InfrastructurePlan
	var resourcesJSON []byte
	var costJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, application_id, plan_type, from_provider, to_provider, content, resources, estimated_cost, created_at
		 FROM infrastructure_plans WHERE id = $1`, id,
	).Scan(&p.ID, &p.ApplicationID, &p.PlanType, &p.FromProvider, &p.ToProvider, &p.Content, &resourcesJSON, &costJSON, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p, domain.ErrNotFound
		}
		return p, fmt.Errorf("get plan by id: %w", err)
	}
	if err := json.Unmarshal(resourcesJSON, &p.Resources); err != nil {
		return p, fmt.Errorf("unmarshal resources: %w", err)
	}
	if costJSON != nil {
		p.EstimatedCost = &domain.CostEstimate{}
		if err := json.Unmarshal(costJSON, p.EstimatedCost); err != nil {
			return p, fmt.Errorf("unmarshal estimated cost: %w", err)
		}
	}
	return p, nil
}

func (r *PlanRepo) ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.InfrastructurePlan, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, plan_type, from_provider, to_provider, content, resources, estimated_cost, created_at
		 FROM infrastructure_plans WHERE application_id = $1 ORDER BY created_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	var plans []domain.InfrastructurePlan
	for rows.Next() {
		var p domain.InfrastructurePlan
		var resourcesJSON []byte
		var costJSON []byte
		if err := rows.Scan(&p.ID, &p.ApplicationID, &p.PlanType, &p.FromProvider, &p.ToProvider, &p.Content, &resourcesJSON, &costJSON, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		if err := json.Unmarshal(resourcesJSON, &p.Resources); err != nil {
			return nil, fmt.Errorf("unmarshal resources: %w", err)
		}
		if costJSON != nil {
			p.EstimatedCost = &domain.CostEstimate{}
			if err := json.Unmarshal(costJSON, p.EstimatedCost); err != nil {
				return nil, fmt.Errorf("unmarshal estimated cost: %w", err)
			}
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}
