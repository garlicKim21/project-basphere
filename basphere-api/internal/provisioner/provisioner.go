package provisioner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/basphere/basphere-api/internal/model"
)

// Provisioner defines the interface for user provisioning
type Provisioner interface {
	// CreateUser creates a system user with the given SSH public key
	CreateUser(req *model.RegistrationRequest) error

	// UserExists checks if a system user already exists
	UserExists(username string) (bool, error)
}

// BashProvisioner implements Provisioner using bash scripts
type BashProvisioner struct {
	adminScript string
	tempDir     string
}

// NewBashProvisioner creates a new bash-based provisioner
func NewBashProvisioner(adminScript string) (*BashProvisioner, error) {
	// Check if admin script exists
	if _, err := os.Stat(adminScript); err != nil {
		return nil, fmt.Errorf("admin script not found: %s", adminScript)
	}

	tempDir := "/tmp/basphere-api"
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &BashProvisioner{
		adminScript: adminScript,
		tempDir:     tempDir,
	}, nil
}

// CreateUser creates a system user with the given SSH public key
func (p *BashProvisioner) CreateUser(req *model.RegistrationRequest) error {
	// Write public key to temp file
	pubkeyFile := filepath.Join(p.tempDir, req.Username+".pub")
	if err := os.WriteFile(pubkeyFile, []byte(req.PublicKey), 0600); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	defer os.Remove(pubkeyFile)

	// Run basphere-admin user add command
	cmd := exec.Command("sudo", p.adminScript, "user", "add", req.Username, "--pubkey", pubkeyFile)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user: %s\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	return nil
}

// UserExists checks if a system user already exists
func (p *BashProvisioner) UserExists(username string) (bool, error) {
	cmd := exec.Command("id", username)
	err := cmd.Run()
	if err != nil {
		// User does not exist
		return false, nil
	}
	return true, nil
}

// MockProvisioner is a provisioner for testing
type MockProvisioner struct {
	Users map[string]bool
}

// NewMockProvisioner creates a mock provisioner for testing
func NewMockProvisioner() *MockProvisioner {
	return &MockProvisioner{
		Users: make(map[string]bool),
	}
}

// CreateUser mock implementation
func (p *MockProvisioner) CreateUser(req *model.RegistrationRequest) error {
	if p.Users[req.Username] {
		return fmt.Errorf("user already exists: %s", req.Username)
	}
	p.Users[req.Username] = true
	return nil
}

// UserExists mock implementation
func (p *MockProvisioner) UserExists(username string) (bool, error) {
	return p.Users[username], nil
}
