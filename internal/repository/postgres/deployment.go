package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// DeploymentRepo implements repository.DeploymentRepo with PostgreSQL.
type DeploymentRepo struct {
	pool *pgxpool.Pool
}

// NewDeploymentRepo creates a new PostgreSQL-backed deployment repository.
func NewDeploymentRepo(pool *pgxpool.Pool) *DeploymentRepo {
	return &DeploymentRepo{pool: pool}
}

func (r *DeploymentRepo) Create(ctx context.Context, d domain.Deployment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO deployments (id, application_id, plan_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		d.ID, d.ApplicationID, d.PlanID, d.Provider, d.GitCommit, d.GitBranch, d.Status, d.TerraformPlan, d.StartedAt, d.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert deployment: %w", err)
	}
	return nil
}

func (r *DeploymentRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Deployment, error) {
	var d domain.Deployment
	err := r.pool.QueryRow(ctx,
		`SELECT id, application_id, plan_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at
		 FROM deployments WHERE id = $1`, id,
	).Scan(&d.ID, &d.ApplicationID, &d.PlanID, &d.Provider, &d.GitCommit, &d.GitBranch, &d.Status, &d.TerraformPlan, &d.StartedAt, &d.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d, domain.ErrNotFound
		}
		return d, fmt.Errorf("get deployment by id: %w", err)
	}
	return d, nil
}

func (r *DeploymentRepo) ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, plan_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at
		 FROM deployments WHERE application_id = $1 ORDER BY started_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []domain.Deployment
	for rows.Next() {
		var d domain.Deployment
		if err := rows.Scan(&d.ID, &d.ApplicationID, &d.PlanID, &d.Provider, &d.GitCommit, &d.GitBranch, &d.Status, &d.TerraformPlan, &d.StartedAt, &d.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan deployment: %w", err)
		}
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

func (r *DeploymentRepo) Update(ctx context.Context, d domain.Deployment) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE deployments SET status = $2, terraform_plan = $3, completed_at = $4
		 WHERE id = $1`,
		d.ID, d.Status, d.TerraformPlan, d.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update deployment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DeploymentRepo) GetLatestByApplicationID(ctx context.Context, appID uuid.UUID) (domain.Deployment, error) {
	var d domain.Deployment
	err := r.pool.QueryRow(ctx,
		`SELECT id, application_id, plan_id, provider, git_commit, git_branch, status, terraform_plan, started_at, completed_at
		 FROM deployments WHERE application_id = $1 ORDER BY started_at DESC LIMIT 1`, appID,
	).Scan(&d.ID, &d.ApplicationID, &d.PlanID, &d.Provider, &d.GitCommit, &d.GitBranch, &d.Status, &d.TerraformPlan, &d.StartedAt, &d.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return d, domain.ErrNotFound
		}
		return d, fmt.Errorf("get latest deployment: %w", err)
	}
	return d, nil
}
