package domain

import (
	"time"

	"github.com/google/uuid"
)

// GraphNodeKind represents the type of node in an infrastructure topology graph.
type GraphNodeKind string

const (
	GraphNodeInternet GraphNodeKind = "internet"
	GraphNodeCompute  GraphNodeKind = "compute"
	GraphNodeDatabase GraphNodeKind = "database"
	GraphNodeCache    GraphNodeKind = "cache"
	GraphNodeQueue    GraphNodeKind = "queue"
	GraphNodeStorage  GraphNodeKind = "storage"
	GraphNodeCDN      GraphNodeKind = "cdn"
	GraphNodeNetwork  GraphNodeKind = "network"
	GraphNodeSecrets  GraphNodeKind = "secrets"
	GraphNodePolicy   GraphNodeKind = "policy"
)

// GraphNode represents a resource (or the internet) in a topology graph.
type GraphNode struct {
	ID      string        `json:"id"`
	Label   string        `json:"label"`
	Kind    GraphNodeKind `json:"kind"`
	Service string        `json:"service"` // Provider-specific service name (e.g. "Cloud Run", "RDS")
}

// GraphEdge represents a connection between two nodes in a topology graph.
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"` // Source node ID
	Target string `json:"target"` // Target node ID
	Label  string `json:"label"`  // Connection description (e.g. "HTTPS", "TCP/5432")
}

// InfraGraph is an LLM-generated infrastructure topology graph for an application.
type InfraGraph struct {
	ID            uuid.UUID   `json:"id"`
	ApplicationID uuid.UUID   `json:"application_id"`
	Nodes         []GraphNode `json:"nodes"`
	Edges         []GraphEdge `json:"edges"`
	CreatedAt     time.Time   `json:"created_at"`
}

// NewInfraGraph creates a new infrastructure topology graph.
func NewInfraGraph(appID uuid.UUID, nodes []GraphNode, edges []GraphEdge) InfraGraph {
	return InfraGraph{
		ID:            uuid.New(),
		ApplicationID: appID,
		Nodes:         nodes,
		Edges:         edges,
		CreatedAt:     time.Now().UTC(),
	}
}

// Validate checks that the graph has valid required fields.
func (g InfraGraph) Validate() error {
	if g.ApplicationID == uuid.Nil {
		return ErrValidation("application ID is required")
	}
	if len(g.Nodes) == 0 {
		return ErrValidation("graph must have at least one node")
	}
	return nil
}
