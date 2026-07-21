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

// ExtraDisk represents an additional disk attached to a VM (advanced option)
type ExtraDisk struct {
	SizeGB int    `json:"size_gb"`
	Mode   string `json:"mode,omitempty"` // "auto" (format+mount, default) or "raw"
}

// VM represents a virtual machine
type VM struct {
	Name             string      `json:"name"`
	VsphereVMName    string      `json:"vsphere_vm_name"`
	Owner            string      `json:"owner"`
	OS               string      `json:"os"`
	LoginUser        string      `json:"login_user"`
	Spec             string      `json:"spec"`
	IPAddress        string      `json:"ip_address"`
	ExtraDisks       []ExtraDisk `json:"extra_disks,omitempty"`
	ExtraNetworks    []string    `json:"extra_networks,omitempty"`
	UserPasswordAuth bool        `json:"user_password_auth,omitempty"`
	Status           VMStatus    `json:"status"`
	CreatedAt        time.Time   `json:"created_at"`
}

// CreateVMInput represents the input for creating a VM
type CreateVMInput struct {
	Name          string      `json:"name"`
	OS            string      `json:"os"`
	Spec          string      `json:"spec"`
	Count         int         `json:"count,omitempty"`
	ExtraDisks    []ExtraDisk `json:"extra_disks,omitempty"`
	ExtraNetworks []string    `json:"extra_networks,omitempty"`
	// UserPasswordHash, when set, enables password+key SSH for the tenant user.
	// Pre-hashed on the CLI (crypt SHA-512); plaintext never reaches the API.
	UserPasswordHash string `json:"user_password_hash,omitempty"`
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

	// 구조적 상한 검증 - 관리자 설정 한도(specs.yaml custom_options)는 create-vm이 검증
	if len(v.ExtraDisks) > 8 {
		errors = append(errors, "extra_disks: at most 8 disks allowed")
	}
	for _, d := range v.ExtraDisks {
		if d.SizeGB < 1 || d.SizeGB > 2000 {
			errors = append(errors, "extra_disks: size_gb must be between 1 and 2000")
		}
		if d.Mode != "" && d.Mode != "auto" && d.Mode != "raw" {
			errors = append(errors, "extra_disks: mode must be \"auto\" or \"raw\"")
		}
	}
	if len(v.ExtraNetworks) > 4 {
		errors = append(errors, "extra_networks: at most 4 networks allowed")
	}
	for _, n := range v.ExtraNetworks {
		if !isValidNetworkName(n) {
			errors = append(errors, "extra_networks: invalid network name: "+n)
		}
	}

	// 비번 해시는 crypt 형식($id$salt$hash)이어야 함 - 평문/주입 방지
	if v.UserPasswordHash != "" && !isValidCryptHash(v.UserPasswordHash) {
		errors = append(errors, "user_password_hash: must be a crypt-format hash")
	}

	return errors
}

// isValidCryptHash checks the string is a crypt(3) hash like $6$salt$hash with
// no shell/whitespace metacharacters (it is passed to a script as an argument).
func isValidCryptHash(s string) bool {
	if len(s) < 12 || len(s) > 200 || s[0] != '$' {
		return false
	}
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		if c == '$' || c == '.' || c == '/' {
			continue
		}
		return false
	}
	return true
}

// isValidNetworkName checks if a network catalog name is safe to pass through
func isValidNetworkName(name string) bool {
	if len(name) < 1 || len(name) > 64 {
		return false
	}
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		if c == '-' || c == '_' || c == '.' {
			continue
		}
		return false
	}
	return true
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
	VMs   []VM  `json:"vms"`
	Total int   `json:"total"`
	Quota Quota `json:"quota"`
}

// Quota represents user's resource quota
type Quota struct {
	MaxVMs  int `json:"max_vms"`
	UsedVMs int `json:"used_vms"`
	MaxIPs  int `json:"max_ips"`
	UsedIPs int `json:"used_ips"`
}

// CreateVMResponse represents the response for creating VMs
type CreateVMResponse struct {
	VMs     []VM     `json:"vms"`
	Created int      `json:"created"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}
