package model

import "time"

// VMStatus represents the status of a VM
type VMStatus string

const (
	VMStatusCreating VMStatus = "creating"
	VMStatusRunning  VMStatus = "running"
	VMStatusDeleting VMStatus = "deleting"
	VMStatusFailed   VMStatus = "failed"
)

// VM represents a virtual machine
type VM struct {
	Name          string    `json:"name"`
	VsphereVMName string    `json:"vsphere_vm_name"`
	Owner         string    `json:"owner"`
	OS            string    `json:"os"`
	LoginUser     string    `json:"login_user"`
	Spec          string    `json:"spec"`
	IPAddress     string    `json:"ip_address"`
	Status        VMStatus  `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateVMInput represents the input for creating a VM
type CreateVMInput struct {
	Name  string `json:"name"`
	OS    string `json:"os"`
	Spec  string `json:"spec"`
	Count int    `json:"count,omitempty"`
}

// Validate validates the VM creation input
func (v *CreateVMInput) Validate() []string {
	var errors []string

	if v.Name == "" {
		errors = append(errors, "name is required")
	} else if !isValidVMName(v.Name) {
		errors = append(errors, "name must be 1-30 characters, lowercase letters, numbers, and hyphens only")
	}

	if v.OS == "" {
		errors = append(errors, "os is required")
	}

	if v.Spec == "" {
		errors = append(errors, "spec is required")
	}

	if v.Count < 0 || v.Count > 10 {
		errors = append(errors, "count must be between 1 and 10")
	}

	return errors
}

// isValidVMName checks if VM name is valid
func isValidVMName(name string) bool {
	if len(name) < 1 || len(name) > 30 {
		return false
	}
	for i, c := range name {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i > 0 && i < len(name)-1 {
			continue
		}
		return false
	}
	return true
}

// DeleteVMInput represents the input for deleting a VM
type DeleteVMInput struct {
	Force bool `json:"force,omitempty"`
}

// VMListResponse represents the response for listing VMs
type VMListResponse struct {
	VMs   []VM `json:"vms"`
	Total int  `json:"total"`
	Quota Quota `json:"quota"`
}

// Quota represents user's resource quota
type Quota struct {
	MaxVMs     int `json:"max_vms"`
	UsedVMs    int `json:"used_vms"`
	MaxIPs     int `json:"max_ips"`
	UsedIPs    int `json:"used_ips"`
}

// CreateVMResponse represents the response for creating VMs
type CreateVMResponse struct {
	VMs     []VM   `json:"vms"`
	Created int    `json:"created"`
	Failed  int    `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}
