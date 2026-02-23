// Package credentials provides thread-safe credential management for cloud providers.
package credentials

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
// It persists the key to a stable file path so it survives server restarts.
type Store struct {
	mu       sync.RWMutex
	filePath string          // stable path holding the key
	info     *CredentialInfo // parsed metadata from the key JSON
}

// dataDir returns the directory used to persist credential files.
// It uses $HOME/.infraplane/credentials/ and creates it if needed.
func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	dir := filepath.Join(home, ".infraplane", "credentials")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create credentials directory: %w", err)
	}
	return dir, nil
}

// keyFilePath returns the stable path for the GCP service account key.
func keyFilePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gcp-service-account.json"), nil
}

// NewStore creates a new credential store and loads any previously persisted credentials.
func NewStore() *Store {
	s := &Store{}
	// Attempt to load persisted credentials from disk
	if err := s.loadFromDisk(); err != nil {
		log.Printf("[credentials] no persisted credentials found: %v", err)
	}
	return s
}

// loadFromDisk checks for an existing credential file and restores state from it.
func (s *Store) loadFromDisk() error {
	path, err := keyFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read credential file: %w", err)
	}

	var key serviceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return fmt.Errorf("parse credential file: %w", err)
	}

	if key.Type != "service_account" || key.ProjectID == "" || key.ClientEmail == "" {
		return fmt.Errorf("invalid persisted credential file")
	}

	// Get file mod time as the "uploaded at" timestamp
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat credential file: %w", err)
	}

	s.filePath = path
	s.info = &CredentialInfo{
		ProjectID:   key.ProjectID,
		ClientEmail: key.ClientEmail,
		UploadedAt:  stat.ModTime().UTC(),
	}

	// Set env var so all GCP SDK clients use this key
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", s.filePath)
	log.Printf("[credentials] restored GCP credentials for %s (project: %s)", key.ClientEmail, key.ProjectID)

	return nil
}

// Save validates and stores a GCP service account key JSON.
// It writes the key to a stable file path and sets GOOGLE_APPLICATION_CREDENTIALS
// so all GCP SDK clients pick it up. The key persists across server restarts.
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

	// Determine the stable file path
	path, err := keyFilePath()
	if err != nil {
		return nil, fmt.Errorf("get key file path: %w", err)
	}

	// Write atomically: write to temp file in same dir, then rename
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".gcp-key-tmp-*.json")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("set file permissions: %w", err)
	}
	if _, err := tmpFile.Write(keyJSON); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("write key file: %w", err)
	}
	tmpFile.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("persist key file: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.filePath = path
	s.info = &CredentialInfo{
		ProjectID:   key.ProjectID,
		ClientEmail: key.ClientEmail,
		UploadedAt:  time.Now().UTC(),
	}

	// Set env var so all GCP SDK clients use this key
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", s.filePath)
	log.Printf("[credentials] saved GCP credentials for %s (project: %s)", key.ClientEmail, key.ProjectID)

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
	log.Printf("[credentials] deleted GCP credentials")
	return nil
}

// FilePath returns the path to the current key file, or empty if none.
func (s *Store) FilePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filePath
}
