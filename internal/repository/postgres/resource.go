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

// ResourceRepo implements repository.ResourceRepo with PostgreSQL.
type ResourceRepo struct {
	pool *pgxpool.Pool
}

// NewResourceRepo creates a new PostgreSQL-backed resource repository.
func NewResourceRepo(pool *pgxpool.Pool) *ResourceRepo {
	return &ResourceRepo{pool: pool}
}

func (r *ResourceRepo) Create(ctx context.Context, res domain.Resource) error {
	mappingsJSON, err := json.Marshal(res.ProviderMappings)
	if err != nil {
		return fmt.Errorf("marshal provider mappings: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO resources (id, application_id, analysis_run_id, kind, name, spec, provider_mappings, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		res.ID, res.ApplicationID, res.AnalysisRunID, res.Kind, res.Name, res.Spec, mappingsJSON, res.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert resource: %w", err)
	}
	return nil
}

func (r *ResourceRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Resource, error) {
	var res domain.Resource
	var mappingsJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, application_id, analysis_run_id, kind, name, spec, provider_mappings, created_at
		 FROM resources WHERE id = $1`, id,
	).Scan(&res.ID, &res.ApplicationID, &res.AnalysisRunID, &res.Kind, &res.Name, &res.Spec, &mappingsJSON, &res.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, domain.ErrNotFound
		}
		return res, fmt.Errorf("get resource by id: %w", err)
	}
	if err := json.Unmarshal(mappingsJSON, &res.ProviderMappings); err != nil {
		return res, fmt.Errorf("unmarshal provider mappings: %w", err)
	}
	return res, nil
}

func (r *ResourceRepo) ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.Resource, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, analysis_run_id, kind, name, spec, provider_mappings, created_at
		 FROM resources WHERE application_id = $1 ORDER BY created_at`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	return scanResources(rows)
}

// ListCurrentByApplicationID returns resources from the latest analysis run
// plus any manually-added resources (where analysis_run_id IS NULL).
func (r *ResourceRepo) ListCurrentByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.Resource, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, analysis_run_id, kind, name, spec, provider_mappings, created_at
		 FROM resources
		 WHERE application_id = $1
		   AND (
		     analysis_run_id IS NULL
		     OR analysis_run_id = (
		       SELECT id FROM analysis_runs
		       WHERE application_id = $1
		       ORDER BY created_at DESC
		       LIMIT 1
		     )
		   )
		 ORDER BY created_at`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list current resources: %w", err)
	}
	defer rows.Close()

	return scanResources(rows)
}

// ListByAnalysisRunID returns all resources belonging to a specific analysis run.
func (r *ResourceRepo) ListByAnalysisRunID(ctx context.Context, runID uuid.UUID) ([]domain.Resource, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, analysis_run_id, kind, name, spec, provider_mappings, created_at
		 FROM resources
		 WHERE analysis_run_id = $1
		 ORDER BY created_at`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list resources by run: %w", err)
	}
	defer rows.Close()

	return scanResources(rows)
}

func (r *ResourceRepo) Update(ctx context.Context, res domain.Resource) error {
	mappingsJSON, err := json.Marshal(res.ProviderMappings)
	if err != nil {
		return fmt.Errorf("marshal provider mappings: %w", err)
	}

	result, err := r.pool.Exec(ctx,
		`UPDATE resources SET kind = $2, name = $3, spec = $4, provider_mappings = $5
		 WHERE id = $1`,
		res.ID, res.Kind, res.Name, res.Spec, mappingsJSON,
	)
	if err != nil {
		return fmt.Errorf("update resource: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ResourceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM resources WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// scanResources scans rows into a slice of domain.Resource.
func scanResources(rows pgx.Rows) ([]domain.Resource, error) {
	var resources []domain.Resource
	for rows.Next() {
		var res domain.Resource
		var mappingsJSON []byte
		if err := rows.Scan(&res.ID, &res.ApplicationID, &res.AnalysisRunID, &res.Kind, &res.Name, &res.Spec, &mappingsJSON, &res.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan resource: %w", err)
		}
		if err := json.Unmarshal(mappingsJSON, &res.ProviderMappings); err != nil {
			return nil, fmt.Errorf("unmarshal provider mappings: %w", err)
		}
		resources = append(resources, res)
	}
	return resources, rows.Err()
}
