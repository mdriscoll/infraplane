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

// ApplicationRepo implements repository.ApplicationRepo with PostgreSQL.
type ApplicationRepo struct {
	pool *pgxpool.Pool
}

// NewApplicationRepo creates a new PostgreSQL-backed application repository.
func NewApplicationRepo(pool *pgxpool.Pool) *ApplicationRepo {
	return &ApplicationRepo{pool: pool}
}

func (r *ApplicationRepo) Create(ctx context.Context, app domain.Application) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO applications (id, name, description, git_repo_url, provider, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		app.ID, app.Name, app.Description, app.GitRepoURL, app.Provider, app.Status, app.CreatedAt, app.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}
	return nil
}

func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Application, error) {
	var app domain.Application
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, git_repo_url, provider, status, created_at, updated_at
		 FROM applications WHERE id = $1`, id,
	).Scan(&app.ID, &app.Name, &app.Description, &app.GitRepoURL, &app.Provider, &app.Status, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app, domain.ErrNotFound
		}
		return app, fmt.Errorf("get application by id: %w", err)
	}
	return app, nil
}

func (r *ApplicationRepo) GetByName(ctx context.Context, name string) (domain.Application, error) {
	var app domain.Application
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, git_repo_url, provider, status, created_at, updated_at
		 FROM applications WHERE name = $1`, name,
	).Scan(&app.ID, &app.Name, &app.Description, &app.GitRepoURL, &app.Provider, &app.Status, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app, domain.ErrNotFound
		}
		return app, fmt.Errorf("get application by name: %w", err)
	}
	return app, nil
}

func (r *ApplicationRepo) List(ctx context.Context) ([]domain.Application, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, git_repo_url, provider, status, created_at, updated_at
		 FROM applications ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		var app domain.Application
		if err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.GitRepoURL, &app.Provider, &app.Status, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *ApplicationRepo) Update(ctx context.Context, app domain.Application) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE applications
		 SET name = $2, description = $3, git_repo_url = $4, provider = $5, status = $6, updated_at = $7
		 WHERE id = $1`,
		app.ID, app.Name, app.Description, app.GitRepoURL, app.Provider, app.Status, app.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update application: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ApplicationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM applications WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
