// Package gcp provides GCP integration for discovering live resources and listing projects.
package gcp

import (
	"context"
	"fmt"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"google.golang.org/api/iterator"
)

// GCPProject represents a simplified GCP project for the frontend.
type GCPProject struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
}

// ProjectClient wraps the GCP Resource Manager API client for listing projects.
type ProjectClient struct {
	client *resourcemanager.ProjectsClient
}

// NewProjectClient creates a new GCP Resource Manager projects client.
// Uses Application Default Credentials (gcloud auth application-default login).
func NewProjectClient(ctx context.Context) (*ProjectClient, error) {
	client, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create projects client: %w", err)
	}
	return &ProjectClient{client: client}, nil
}

// Close releases the underlying gRPC connection.
func (c *ProjectClient) Close() error {
	return c.client.Close()
}

// ListProjects returns all GCP projects accessible to the current credentials.
// Only ACTIVE projects are returned.
func (c *ProjectClient) ListProjects(ctx context.Context) ([]GCPProject, error) {
	req := &resourcemanagerpb.SearchProjectsRequest{}

	var projects []GCPProject
	it := c.client.SearchProjects(ctx, req)
	for {
		proj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return projects, fmt.Errorf("list projects: %w", err)
		}

		// Only include ACTIVE projects
		if proj.State != resourcemanagerpb.Project_ACTIVE {
			continue
		}

		projects = append(projects, GCPProject{
			ProjectID:   proj.ProjectId,
			Name:        proj.Name,
			DisplayName: proj.DisplayName,
			State:       proj.State.String(),
		})
	}

	return projects, nil
}
