package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matthewdriscoll/infraplane/internal/domain"
)

// AnalysisRunRepo implements repository.AnalysisRunRepo with PostgreSQL.
type AnalysisRunRepo struct {
	pool *pgxpool.Pool
}

// NewAnalysisRunRepo creates a new AnalysisRunRepo.
func NewAnalysisRunRepo(pool *pgxpool.Pool) *AnalysisRunRepo {
	return &AnalysisRunRepo{pool: pool}
}

func (r *AnalysisRunRepo) Create(ctx context.Context, run domain.AnalysisRun) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO analysis_runs (id, application_id, trigger, resource_count, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		run.ID, run.ApplicationID, run.Trigger, run.ResourceCount, run.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert analysis run: %w", err)
	}
	return nil
}

func (r *AnalysisRunRepo) ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.AnalysisRun, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, trigger, resource_count, created_at
		 FROM analysis_runs
		 WHERE application_id = $1
		 ORDER BY created_at DESC`,
		appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list analysis runs: %w", err)
	}
	defer rows.Close()

	var runs []domain.AnalysisRun
	for rows.Next() {
		var run domain.AnalysisRun
		if err := rows.Scan(&run.ID, &run.ApplicationID, &run.Trigger, &run.ResourceCount, &run.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan analysis run: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate analysis runs: %w", err)
	}

	return runs, nil
}

func (r *AnalysisRunRepo) UpdateResourceCount(ctx context.Context, id uuid.UUID, count int) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE analysis_runs SET resource_count = $2 WHERE id = $1`,
		id, count,
	)
	if err != nil {
		return fmt.Errorf("update analysis run resource count: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// scanAnalysisRun is unused but kept for pattern consistency; the inline Scan in
// ListByApplicationID is more ergonomic for the current query set.
func scanAnalysisRun(row pgx.Row) (domain.AnalysisRun, error) {
	var run domain.AnalysisRun
	err := row.Scan(&run.ID, &run.ApplicationID, &run.Trigger, &run.ResourceCount, &run.CreatedAt)
	return run, err
}
