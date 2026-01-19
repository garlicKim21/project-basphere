package provisioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/basphere/basphere-api/internal/model"
)

// Provisioner defines the interface for user and VM provisioning
type Provisioner interface {
	// User management
	CreateUser(req *model.RegistrationRequest) error
	UserExists(username string) (bool, error)

	// VM management
	CreateVM(username string, input *model.CreateVMInput) (*model.VM, error)
	DeleteVM(username, vmName string) error
	ListVMs(username string) ([]model.VM, error)
	GetVM(username, vmName string) (*model.VM, error)
	VMExists(username, vmName string) (bool, error)

	// Quota
	GetQuota(username string) (*model.Quota, error)
}

// BashProvisioner implements Provisioner using bash scripts
type BashProvisioner struct {
	adminScript   string
	createVMScript string
	deleteVMScript string
	listVMsScript  string
	tempDir       string
	dataDir       string
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
		adminScript:    adminScript,
		createVMScript: "/usr/local/bin/create-vm",
		deleteVMScript: "/usr/local/bin/delete-vm",
		listVMsScript:  "/usr/local/bin/list-vms",
		tempDir:        tempDir,
		dataDir:        "/var/lib/basphere",
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

// CreateVM creates a new VM for the user
func (p *BashProvisioner) CreateVM(username string, input *model.CreateVMInput) (*model.VM, error) {
	// Run create-vm script with --api flag (non-interactive, JSON output)
	cmd := exec.Command(p.createVMScript,
		"--api",
		"--name", input.Name,
		"--os", input.OS,
		"--spec", input.Spec,
		"--user", username,
	)

	// Set environment to ensure proper execution
	cmd.Env = append(os.Environ(), "BASPHERE_API_MODE=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create VM: %s\nstderr: %s", err, stderr.String())
	}

	// Parse JSON output
	var vm model.VM
	if err := json.Unmarshal(stdout.Bytes(), &vm); err != nil {
		return nil, fmt.Errorf("failed to parse VM output: %w\nstdout: %s", err, stdout.String())
	}

	return &vm, nil
}

// DeleteVM deletes a VM
func (p *BashProvisioner) DeleteVM(username, vmName string) error {
	// Run delete-vm script with --api flag
	cmd := exec.Command(p.deleteVMScript,
		"--api",
		"--force",
		"--user", username,
		vmName,
	)

	cmd.Env = append(os.Environ(), "BASPHERE_API_MODE=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete VM: %s\nstderr: %s", err, stderr.String())
	}

	return nil
}

// ListVMs lists all VMs for a user
func (p *BashProvisioner) ListVMs(username string) ([]model.VM, error) {
	// Read VM metadata directly from filesystem
	tfDir := filepath.Join(p.dataDir, "terraform", username)

	entries, err := os.ReadDir(tfDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.VM{}, nil
		}
		return nil, fmt.Errorf("failed to read VM directory: %w", err)
	}

	var vms []model.VM
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_folder" {
			continue
		}

		metadataPath := filepath.Join(tfDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			continue // Skip if metadata doesn't exist
		}

		var vm model.VM
		if err := json.Unmarshal(data, &vm); err != nil {
			continue // Skip if invalid JSON
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

// GetVM gets a specific VM
func (p *BashProvisioner) GetVM(username, vmName string) (*model.VM, error) {
	metadataPath := filepath.Join(p.dataDir, "terraform", username, vmName, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("VM not found: %s", vmName)
		}
		return nil, fmt.Errorf("failed to read VM metadata: %w", err)
	}

	var vm model.VM
	if err := json.Unmarshal(data, &vm); err != nil {
		return nil, fmt.Errorf("failed to parse VM metadata: %w", err)
	}

	return &vm, nil
}

// VMExists checks if a VM exists
func (p *BashProvisioner) VMExists(username, vmName string) (bool, error) {
	metadataPath := filepath.Join(p.dataDir, "terraform", username, vmName, "metadata.json")
	_, err := os.Stat(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetQuota gets the quota for a user
func (p *BashProvisioner) GetQuota(username string) (*model.Quota, error) {
	// Get current VM count
	vms, err := p.ListVMs(username)
	if err != nil {
		return nil, err
	}

	// Count IPs (same as VMs for now)
	usedIPs := len(vms)

	// Default quotas (should come from config)
	return &model.Quota{
		MaxVMs:  10,
		UsedVMs: len(vms),
		MaxIPs:  32,
		UsedIPs: usedIPs,
	}, nil
}

// MockProvisioner is a provisioner for testing
type MockProvisioner struct {
	Users map[string]bool
	VMs   map[string][]model.VM
}

// NewMockProvisioner creates a mock provisioner for testing
func NewMockProvisioner() *MockProvisioner {
	return &MockProvisioner{
		Users: make(map[string]bool),
		VMs:   make(map[string][]model.VM),
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

// CreateVM mock implementation
func (p *MockProvisioner) CreateVM(username string, input *model.CreateVMInput) (*model.VM, error) {
	// Check if VM already exists
	for _, vm := range p.VMs[username] {
		if vm.Name == input.Name {
			return nil, fmt.Errorf("VM already exists: %s", input.Name)
		}
	}

	vm := model.VM{
		Name:          input.Name,
		VsphereVMName: username + "-" + input.Name,
		Owner:         username,
		OS:            input.OS,
		LoginUser:     username,
		Spec:          input.Spec,
		IPAddress:     fmt.Sprintf("10.254.0.%d", len(p.VMs[username])+10),
		Status:        model.VMStatusRunning,
	}

	p.VMs[username] = append(p.VMs[username], vm)
	return &vm, nil
}

// DeleteVM mock implementation
func (p *MockProvisioner) DeleteVM(username, vmName string) error {
	vms := p.VMs[username]
	for i, vm := range vms {
		if vm.Name == vmName {
			p.VMs[username] = append(vms[:i], vms[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("VM not found: %s", vmName)
}

// ListVMs mock implementation
func (p *MockProvisioner) ListVMs(username string) ([]model.VM, error) {
	return p.VMs[username], nil
}

// GetVM mock implementation
func (p *MockProvisioner) GetVM(username, vmName string) (*model.VM, error) {
	for _, vm := range p.VMs[username] {
		if vm.Name == vmName {
			return &vm, nil
		}
	}
	return nil, fmt.Errorf("VM not found: %s", vmName)
}

// VMExists mock implementation
func (p *MockProvisioner) VMExists(username, vmName string) (bool, error) {
	for _, vm := range p.VMs[username] {
		if vm.Name == vmName {
			return true, nil
		}
	}
	return false, nil
}

// GetQuota mock implementation
func (p *MockProvisioner) GetQuota(username string) (*model.Quota, error) {
	vms := p.VMs[username]
	return &model.Quota{
		MaxVMs:  10,
		UsedVMs: len(vms),
		MaxIPs:  32,
		UsedIPs: len(vms),
	}, nil
}

// Helper function to check OS support
func isValidOS(os string) bool {
	validOS := []string{"ubuntu-24.04", "rocky-10.1", "rocky-10"}
	for _, v := range validOS {
		if strings.EqualFold(os, v) {
			return true
		}
	}
	return false
}

// Helper function to check spec support
func isValidSpec(spec string) bool {
	validSpecs := []string{"tiny", "small", "medium", "large", "huge"}
	for _, v := range validSpecs {
		if strings.EqualFold(spec, v) {
			return true
		}
	}
	return false
}
