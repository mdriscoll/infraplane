// Package credentials provides thread-safe credential management for cloud providers.
package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// serviceAccountKey is the expected structure of a GCP service account key JSON file.
type serviceAccountKey struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	ClientEmail  string `json:"client_email"`
	PrivateKeyID string `json:"private_key_id"`
}

// CredentialInfo holds metadata about the currently loaded credential.
type CredentialInfo struct {
	ProjectID   string    `json:"project_id"`
	ClientEmail string    `json:"client_email"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// Store manages the lifecycle of an uploaded GCP service account key.
// It is process-scoped (not persisted to a database) and thread-safe.
type Store struct {
	mu       sync.RWMutex
	filePath string          // path to temp file holding the key
	info     *CredentialInfo // parsed metadata from the key JSON
}

// NewStore creates a new credential store.
func NewStore() *Store {
	return &Store{}
}

// Save validates and stores a GCP service account key JSON.
// It writes the key to a temp file and sets GOOGLE_APPLICATION_CREDENTIALS so all GCP SDK clients pick it up.
func (s *Store) Save(keyJSON []byte) (*CredentialInfo, error) {
	// Parse and validate the key
	var key serviceAccountKey
	if err := json.Unmarshal(keyJSON, &key); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if key.Type != "service_account" {
		return nil, fmt.Errorf("invalid key type %q: expected \"service_account\"", key.Type)
	}
	if key.ProjectID == "" {
		return nil, fmt.Errorf("service account key missing project_id")
	}
	if key.ClientEmail == "" {
		return nil, fmt.Errorf("service account key missing client_email")
	}

	// Write to a temp file with restrictive permissions
	tmpFile, err := os.CreateTemp("", "infraplane-gcp-key-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("set file permissions: %w", err)
	}
	if _, err := tmpFile.Write(keyJSON); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("write key file: %w", err)
	}
	tmpFile.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clean up any existing key file
	if s.filePath != "" {
		os.Remove(s.filePath)
	}

	s.filePath = tmpFile.Name()
	s.info = &CredentialInfo{
		ProjectID:   key.ProjectID,
		ClientEmail: key.ClientEmail,
		UploadedAt:  time.Now().UTC(),
	}

	// Set env var so all GCP SDK clients use this key
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", s.filePath)

	return s.info, nil
}

// GetInfo returns the current credential metadata, or nil if no credential is loaded.
func (s *Store) GetInfo() *CredentialInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info
}

// Delete removes the stored credential and unsets the environment variable.
func (s *Store) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filePath != "" {
		os.Remove(s.filePath)
	}
	s.filePath = ""
	s.info = nil

	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	return nil
}

// FilePath returns the path to the current key file, or empty if none.
func (s *Store) FilePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filePath
}
