package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	frameworks, _ := json.Marshal(app.ComplianceFrameworks)
	if string(frameworks) == "null" {
		frameworks = []byte("[]")
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO applications (id, name, description, git_repo_url, source_path, provider, status, compliance_frameworks, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		app.ID, app.Name, app.Description, app.GitRepoURL, app.SourcePath, app.Provider, app.Status, frameworks, app.CreatedAt, app.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return fmt.Errorf("insert application: %w", err)
	}
	return nil
}

func (r *ApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Application, error) {
	var app domain.Application
	var frameworksJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, git_repo_url, source_path, provider, status, compliance_frameworks, created_at, updated_at
		 FROM applications WHERE id = $1`, id,
	).Scan(&app.ID, &app.Name, &app.Description, &app.GitRepoURL, &app.SourcePath, &app.Provider, &app.Status, &frameworksJSON, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app, domain.ErrNotFound
		}
		return app, fmt.Errorf("get application by id: %w", err)
	}
	if len(frameworksJSON) > 0 {
		json.Unmarshal(frameworksJSON, &app.ComplianceFrameworks)
	}
	return app, nil
}

func (r *ApplicationRepo) GetByName(ctx context.Context, name string) (domain.Application, error) {
	var app domain.Application
	var frameworksJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, git_repo_url, source_path, provider, status, compliance_frameworks, created_at, updated_at
		 FROM applications WHERE name = $1`, name,
	).Scan(&app.ID, &app.Name, &app.Description, &app.GitRepoURL, &app.SourcePath, &app.Provider, &app.Status, &frameworksJSON, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app, domain.ErrNotFound
		}
		return app, fmt.Errorf("get application by name: %w", err)
	}
	if len(frameworksJSON) > 0 {
		json.Unmarshal(frameworksJSON, &app.ComplianceFrameworks)
	}
	return app, nil
}

func (r *ApplicationRepo) List(ctx context.Context) ([]domain.Application, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, git_repo_url, source_path, provider, status, compliance_frameworks, created_at, updated_at
		 FROM applications ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		var app domain.Application
		var frameworksJSON []byte
		if err := rows.Scan(&app.ID, &app.Name, &app.Description, &app.GitRepoURL, &app.SourcePath, &app.Provider, &app.Status, &frameworksJSON, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		if len(frameworksJSON) > 0 {
			json.Unmarshal(frameworksJSON, &app.ComplianceFrameworks)
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (r *ApplicationRepo) Update(ctx context.Context, app domain.Application) error {
	frameworks, _ := json.Marshal(app.ComplianceFrameworks)
	if string(frameworks) == "null" {
		frameworks = []byte("[]")
	}

	result, err := r.pool.Exec(ctx,
		`UPDATE applications
		 SET name = $2, description = $3, git_repo_url = $4, source_path = $5, provider = $6, status = $7, compliance_frameworks = $8, updated_at = $9
		 WHERE id = $1`,
		app.ID, app.Name, app.Description, app.GitRepoURL, app.SourcePath, app.Provider, app.Status, frameworks, app.UpdatedAt,
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
