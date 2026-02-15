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

// GraphRepo implements repository.GraphRepo with PostgreSQL.
type GraphRepo struct {
	pool *pgxpool.Pool
}

// NewGraphRepo creates a new PostgreSQL-backed graph repository.
func NewGraphRepo(pool *pgxpool.Pool) *GraphRepo {
	return &GraphRepo{pool: pool}
}

func (r *GraphRepo) Create(ctx context.Context, g domain.InfraGraph) error {
	nodesJSON, err := json.Marshal(g.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}

	edgesJSON, err := json.Marshal(g.Edges)
	if err != nil {
		return fmt.Errorf("marshal edges: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO infrastructure_graphs (id, application_id, nodes, edges, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		g.ID, g.ApplicationID, nodesJSON, edgesJSON, g.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert graph: %w", err)
	}
	return nil
}

func (r *GraphRepo) GetLatestByApplicationID(ctx context.Context, appID uuid.UUID) (domain.InfraGraph, error) {
	var g domain.InfraGraph
	var nodesJSON, edgesJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, application_id, nodes, edges, created_at
		 FROM infrastructure_graphs WHERE application_id = $1
		 ORDER BY created_at DESC LIMIT 1`, appID,
	).Scan(&g.ID, &g.ApplicationID, &nodesJSON, &edgesJSON, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return g, domain.ErrNotFound
		}
		return g, fmt.Errorf("get latest graph: %w", err)
	}
	if err := json.Unmarshal(nodesJSON, &g.Nodes); err != nil {
		return g, fmt.Errorf("unmarshal nodes: %w", err)
	}
	if err := json.Unmarshal(edgesJSON, &g.Edges); err != nil {
		return g, fmt.Errorf("unmarshal edges: %w", err)
	}
	return g, nil
}

func (r *GraphRepo) ListByApplicationID(ctx context.Context, appID uuid.UUID) ([]domain.InfraGraph, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, application_id, nodes, edges, created_at
		 FROM infrastructure_graphs WHERE application_id = $1
		 ORDER BY created_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list graphs: %w", err)
	}
	defer rows.Close()

	var graphs []domain.InfraGraph
	for rows.Next() {
		var g domain.InfraGraph
		var nodesJSON, edgesJSON []byte
		if err := rows.Scan(&g.ID, &g.ApplicationID, &nodesJSON, &edgesJSON, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan graph: %w", err)
		}
		if err := json.Unmarshal(nodesJSON, &g.Nodes); err != nil {
			return nil, fmt.Errorf("unmarshal nodes: %w", err)
		}
		if err := json.Unmarshal(edgesJSON, &g.Edges); err != nil {
			return nil, fmt.Errorf("unmarshal edges: %w", err)
		}
		graphs = append(graphs, g)
	}
	return graphs, rows.Err()
}
