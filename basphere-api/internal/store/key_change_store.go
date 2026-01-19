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

// KeyChangeStore implements storage for key change requests
type KeyChangeStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewKeyChangeStore creates a new key change store
func NewKeyChangeStore(baseDir string) (*KeyChangeStore, error) {
	keyChangeDir := filepath.Join(baseDir, "key-changes")
	if err := os.MkdirAll(keyChangeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create key change directory: %w", err)
	}

	return &KeyChangeStore{
		baseDir: keyChangeDir,
	}, nil
}

func (s *KeyChangeStore) filePath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}

// Create creates a new key change request
func (s *KeyChangeStore) Create(req *model.KeyChangeRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if username already has pending key change request
	exists, err := s.existsUsernameUnsafe(req.Username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("사용자 '%s'의 키 변경 요청이 이미 진행 중입니다", req.Username)
	}

	return s.writeRequest(req)
}

// Get retrieves a key change request by ID
func (s *KeyChangeStore) Get(id string) (*model.KeyChangeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readRequest(id)
}

// GetByUsername retrieves a key change request by username
func (s *KeyChangeStore) GetByUsername(username string) (*model.KeyChangeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests, err := s.listAllUnsafe()
	if err != nil {
		return nil, err
	}

	for _, req := range requests {
		if req.Username == username && req.Status == model.StatusPending {
			return req, nil
		}
	}

	return nil, fmt.Errorf("key change request not found for username: %s", username)
}

// List returns all key change requests with optional status filter
func (s *KeyChangeStore) List(status *model.RequestStatus) ([]*model.KeyChangeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requests, err := s.listAllUnsafe()
	if err != nil {
		return nil, err
	}

	if status == nil {
		return requests, nil
	}

	var filtered []*model.KeyChangeRequest
	for _, req := range requests {
		if req.Status == *status {
			filtered = append(filtered, req)
		}
	}

	return filtered, nil
}

// Update updates a key change request
func (s *KeyChangeStore) Update(req *model.KeyChangeRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.readRequest(req.ID); err != nil {
		return err
	}

	return s.writeRequest(req)
}

// Delete deletes a key change request
func (s *KeyChangeStore) Delete(id string) error {
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

// Internal methods

func (s *KeyChangeStore) readRequest(id string) (*model.KeyChangeRequest, error) {
	path := s.filePath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("request not found: %s", id)
		}
		return nil, err
	}

	var req model.KeyChangeRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	return &req, nil
}

func (s *KeyChangeStore) writeRequest(req *model.KeyChangeRequest) error {
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

func (s *KeyChangeStore) listAllUnsafe() ([]*model.KeyChangeRequest, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %w", err)
	}

	var requests []*model.KeyChangeRequest
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5]
		req, err := s.readRequest(id)
		if err != nil {
			continue
		}
		requests = append(requests, req)
	}

	sort.Slice(requests, func(i, j int) bool {
		return requests[i].CreatedAt.After(requests[j].CreatedAt)
	})

	return requests, nil
}

func (s *KeyChangeStore) existsUsernameUnsafe(username string) (bool, error) {
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
