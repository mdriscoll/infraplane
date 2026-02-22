package gcp

import (
	"context"
	"log"
	"sync"
)

// Manager manages the lifecycle of GCP API clients (ProjectClient, AssetClient).
// It supports re-initialization after credential changes (e.g., uploading a service account key).
type Manager struct {
	mu            sync.RWMutex
	projectClient *ProjectClient
	assetClient   *AssetClient
}

// NewManager creates a Manager and attempts an initial client initialization.
// If credentials are unavailable, clients will be nil (graceful degradation).
func NewManager(ctx context.Context) *Manager {
	m := &Manager{}
	m.tryInit(ctx)
	return m
}

// tryInit attempts to create GCP clients. Failures are logged but not fatal.
func (m *Manager) tryInit(ctx context.Context) {
	pc, pcErr := NewProjectClient(ctx)
	if pcErr != nil {
		log.Printf("GCP Project listing unavailable: %v", pcErr)
	} else {
		m.projectClient = pc
	}

	ac, acErr := NewAssetClient(ctx)
	if acErr != nil {
		log.Printf("GCP Asset Inventory unavailable: %v (targeted CLI discovery still works)", acErr)
	} else {
		m.assetClient = ac
	}
}

// Reinitialize closes existing clients and creates new ones using the current credentials.
// This should be called after uploading or deleting a GCP service account key.
func (m *Manager) Reinitialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close existing clients
	if m.projectClient != nil {
		m.projectClient.Close()
		m.projectClient = nil
	}
	if m.assetClient != nil {
		m.assetClient.Close()
		m.assetClient = nil
	}

	// Re-create clients with new credentials
	m.tryInit(ctx)
	return nil
}

// ProjectClient returns the current ProjectClient, or nil if unavailable.
func (m *Manager) ProjectClient() *ProjectClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.projectClient
}

// AssetClient returns the current AssetClient, or nil if unavailable.
func (m *Manager) AssetClient() *AssetClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.assetClient
}

// Close releases all underlying gRPC connections.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.projectClient != nil {
		m.projectClient.Close()
		m.projectClient = nil
	}
	if m.assetClient != nil {
		m.assetClient.Close()
		m.assetClient = nil
	}
}
