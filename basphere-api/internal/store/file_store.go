package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/basphere/basphere-api/internal/model"
)

// FileStore implements Store interface using JSON files
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFileStore creates a new file-based store
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStore{
		baseDir: baseDir,
	}, nil
}

// filePath returns the file path for a given request ID
func (s *FileStore) filePath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}

// Create creates a new registration request
func (s *FileStore) Create(req *model.RegistrationRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if username already exists
	exists, err := s.existsUsernameUnsafe(req.Username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("username %s already has a pending request", req.Username)
	}

	return s.writeRequest(req)
}

// Get retrieves a registration request by ID
func (s *FileStore) Get(id string) (*model.RegistrationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readRequest(id)
}

// GetByUsername retrieves a registration request by username
func (s *FileStore) GetByUsername(username string) (*model.RegistrationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests, err := s.listAllUnsafe()
	if err != nil {
		return nil, err
	}

	for _, req := range requests {
		if req.Username == username {
			return req, nil
		}
	}

	return nil, fmt.Errorf("request not found for username: %s", username)
}

// List returns all registration requests with optional status filter
func (s *FileStore) List(status *model.RequestStatus) ([]*model.RegistrationRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests, err := s.listAllUnsafe()
	if err != nil {
		return nil, err
	}

	if status == nil {
		return requests, nil
	}

	// Filter by status
	var filtered []*model.RegistrationRequest
	for _, req := range requests {
		if req.Status == *status {
			filtered = append(filtered, req)
		}
	}

	return filtered, nil
}

// Update updates a registration request
func (s *FileStore) Update(req *model.RegistrationRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if exists
	if _, err := s.readRequest(req.ID); err != nil {
		return err
	}

	return s.writeRequest(req)
}

// Delete deletes a registration request
func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.filePath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("request not found: %s", id)
		}
		return err
	}
	return nil
}

// ExistsUsername checks if a username already has a pending request
func (s *FileStore) ExistsUsername(username string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.existsUsernameUnsafe(username)
}

// ExistsEmail checks if an email already has a pending request
func (s *FileStore) ExistsEmail(email string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.existsEmailUnsafe(email)
}

// Internal methods (not thread-safe, must be called with lock held)

func (s *FileStore) readRequest(id string) (*model.RegistrationRequest, error) {
	path := s.filePath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("request not found: %s", id)
		}
		return nil, err
	}

	var req model.RegistrationRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	return &req, nil
}

func (s *FileStore) writeRequest(req *model.RegistrationRequest) error {
	path := s.filePath(req.ID)
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

func (s *FileStore) listAllUnsafe() ([]*model.RegistrationRequest, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var requests []*model.RegistrationRequest
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // Remove .json
		req, err := s.readRequest(id)
		if err != nil {
			continue // Skip invalid files
		}
		requests = append(requests, req)
	}

	// Sort by created_at (newest first)
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].CreatedAt.After(requests[j].CreatedAt)
	})

	return requests, nil
}

func (s *FileStore) existsUsernameUnsafe(username string) (bool, error) {
	requests, err := s.listAllUnsafe()
	if err != nil {
		return false, err
	}

	for _, req := range requests {
		if req.Username == username && req.Status == model.StatusPending {
			return true, nil
		}
	}

	return false, nil
}

func (s *FileStore) existsEmailUnsafe(email string) (bool, error) {
	requests, err := s.listAllUnsafe()
	if err != nil {
		return false, err
	}

	for _, req := range requests {
		if req.Email == email && req.Status == model.StatusPending {
			return true, nil
		}
	}

	return false, nil
}
