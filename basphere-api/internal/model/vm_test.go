package model

import (
	"strings"
	"testing"
)

// =============================================================================
// VM Name Validation Tests
// =============================================================================

func TestIsValidVMName(t *testing.T) {
	tests := []struct {
		name   string
		vmName string
		want   bool
	}{
		// Valid cases
		{"valid simple", "webserver", true},
		{"valid with numbers", "web01", true},
		{"valid with hyphen", "web-server", true},
		{"valid single char", "a", true},
		{"valid max length", "abcdefghijklmnopqrstuvwxyz1234", true}, // 30 chars

		// Invalid cases - length
		{"empty", "", false},
		{"too long", "abcdefghijklmnopqrstuvwxyz12345", false}, // 31 chars

		// Invalid cases - characters
		{"uppercase", "WebServer", false},
		{"with underscore", "web_server", false},
		{"with space", "web server", false},
		{"with special char", "web@server", false},
		{"with dot", "web.server", false},
		{"korean characters", "서버", false},

		// Invalid cases - hyphen position
		{"starts with hyphen", "-webserver", false},
		{"ends with hyphen", "webserver-", false},
		{"only hyphen", "-", false},
		{"double hyphen", "web--server", true}, // Note: double hyphen in middle is valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidVMName(tt.vmName)
			if got != tt.want {
				t.Errorf("isValidVMName(%q) = %v, want %v", tt.vmName, got, tt.want)
			}
		})
	}
}

// =============================================================================
// CreateVMInput Validation Tests
// =============================================================================

func TestCreateVMInput_Validate(t *testing.T) {
	tests := []struct {
		name       string
		input      CreateVMInput
		wantErrors []string
	}{
		{
			"valid input",
			CreateVMInput{
				Name:  "webserver",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 1,
			},
			nil,
		},
		{
			"valid input with zero count (defaults to 1)",
			CreateVMInput{
				Name:  "webserver",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 0,
			},
			nil,
		},
		{
			"valid input max count",
			CreateVMInput{
				Name:  "webserver",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 10,
			},
			nil,
		},
		{
			"missing name",
			CreateVMInput{
				Name:  "",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 1,
			},
			[]string{"name is required"},
		},
		{
			"invalid name - uppercase",
			CreateVMInput{
				Name:  "WebServer",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 1,
			},
			[]string{"name must be 1-30 characters"},
		},
		{
			"invalid name - too long",
			CreateVMInput{
				Name:  "this-is-a-very-long-vm-name-that-exceeds-limit",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 1,
			},
			[]string{"name must be 1-30 characters"},
		},
		{
			"missing os",
			CreateVMInput{
				Name:  "webserver",
				OS:    "",
				Spec:  "small",
				Count: 1,
			},
			[]string{"os is required"},
		},
		{
			"missing spec",
			CreateVMInput{
				Name:  "webserver",
				OS:    "ubuntu-24.04",
				Spec:  "",
				Count: 1,
			},
			[]string{"spec is required"},
		},
		{
			"count too high",
			CreateVMInput{
				Name:  "webserver",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: 11,
			},
			[]string{"count must be between 1 and 10"},
		},
		{
			"negative count",
			CreateVMInput{
				Name:  "webserver",
				OS:    "ubuntu-24.04",
				Spec:  "small",
				Count: -1,
			},
			[]string{"count must be between 1 and 10"},
		},
		{
			"multiple errors",
			CreateVMInput{
				Name:  "",
				OS:    "",
				Spec:  "",
				Count: 1,
			},
			[]string{"name is required", "os is required", "spec is required"},
		},
		{
			"all fields invalid",
			CreateVMInput{
				Name:  "Invalid_Name!",
				OS:    "",
				Spec:  "",
				Count: 100,
			},
			[]string{"name must be", "os is required", "spec is required", "count must be"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			errors := input.Validate()

			if len(tt.wantErrors) == 0 {
				if len(errors) != 0 {
					t.Errorf("Validate() returned errors %v, want none", errors)
				}
				return
			}

			if len(errors) != len(tt.wantErrors) {
				t.Errorf("Validate() returned %d errors, want %d: got %v",
					len(errors), len(tt.wantErrors), errors)
				return
			}

			for _, wantErr := range tt.wantErrors {
				found := false
				for _, gotErr := range errors {
					if strings.Contains(gotErr, wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Validate() missing expected error containing %q, got %v",
						wantErr, errors)
				}
			}
		})
	}
}

// =============================================================================
// VM Status Tests
// =============================================================================

func TestVMStatus_Constants(t *testing.T) {
	// Verify status constants have expected values
	tests := []struct {
		status VMStatus
		want   string
	}{
		{VMStatusCreating, "creating"},
		{VMStatusRunning, "running"},
		{VMStatusDeleting, "deleting"},
		{VMStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("VMStatus = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

// =============================================================================
// Quota Tests
// =============================================================================

func TestQuota_Fields(t *testing.T) {
	quota := Quota{
		MaxVMs:  10,
		UsedVMs: 3,
		MaxIPs:  32,
		UsedIPs: 5,
	}

	if quota.MaxVMs != 10 {
		t.Errorf("MaxVMs = %d, want 10", quota.MaxVMs)
	}
	if quota.UsedVMs != 3 {
		t.Errorf("UsedVMs = %d, want 3", quota.UsedVMs)
	}
	if quota.MaxIPs != 32 {
		t.Errorf("MaxIPs = %d, want 32", quota.MaxIPs)
	}
	if quota.UsedIPs != 5 {
		t.Errorf("UsedIPs = %d, want 5", quota.UsedIPs)
	}
}
