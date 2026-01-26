package model

import (
	"strings"
	"testing"
)

// =============================================================================
// Username Validation Tests
// =============================================================================

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		// Valid cases
		{"valid simple", "john", true},
		{"valid with numbers", "john123", true},
		{"valid with hyphen", "john-doe", true},
		{"valid minimum length", "abc", true},
		{"valid maximum length", "abcdefghijklmnopqrst", true}, // 20 chars

		// Invalid cases - length
		{"too short", "ab", false},
		{"too long", "abcdefghijklmnopqrstu", false}, // 21 chars
		{"empty", "", false},

		// Invalid cases - characters
		{"uppercase", "John", false},
		{"with underscore", "john_doe", false},
		{"with space", "john doe", false},
		{"with special char", "john@doe", false},
		{"korean characters", "사용자", false},

		// Invalid cases - hyphen position
		{"starts with hyphen", "-john", false},
		{"ends with hyphen", "john-", false},
		{"only hyphen", "-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUsername(tt.username)
			if got != tt.want {
				t.Errorf("isValidUsername(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Email Validation Tests
// =============================================================================

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		// Valid cases
		{"valid simple", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"valid with dots in local", "user.name@example.com", true},

		// Invalid cases
		{"no at sign", "userexample.com", false},
		{"no domain", "user@", false},
		{"no local part", "@example.com", false},
		{"multiple at signs", "user@@example.com", false},
		{"no dot after at", "user@example", false},
		{"empty", "", false},
		{"dot immediately after at", "user@.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEmail(tt.email)
			if got != tt.want {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

// =============================================================================
// SSH Public Key Validation Tests
// =============================================================================

func TestIsValidSSHPublicKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		// Valid cases
		{
			"valid ssh-rsa",
			"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ... user@host",
			true,
		},
		{
			"valid ssh-ed25519",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... user@host",
			true,
		},
		{
			"valid ecdsa-sha2-nistp256",
			"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTI... user@host",
			true,
		},
		{
			"valid ecdsa-sha2-nistp384",
			"ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTI... user@host",
			true,
		},
		{
			"valid ecdsa-sha2-nistp521",
			"ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTI... user@host",
			true,
		},
		{
			"valid ssh-dss",
			"ssh-dss AAAAB3NzaC1kc3MAAACBAP... user@host",
			true,
		},

		// Invalid cases
		{"empty", "", false},
		{"random text", "not a valid key", false},
		{"only prefix", "ssh-rsa", false},
		{"invalid prefix", "ssh-invalid AAAA...", false},
		{"private key marker", "-----BEGIN RSA PRIVATE KEY-----", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidSSHPublicKey(tt.key)
			if got != tt.want {
				t.Errorf("isValidSSHPublicKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// =============================================================================
// SSH Key Sanitization Tests
// =============================================================================

func TestSanitizeSSHKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"no change needed",
			"ssh-ed25519 AAAAC3... user@host",
			"ssh-ed25519 AAAAC3... user@host",
		},
		{
			"remove Windows line endings",
			"ssh-ed25519 AAAAC3...\r\n user@host",
			"ssh-ed25519 AAAAC3...\n user@host",
		},
		{
			"remove multiple CR",
			"ssh-rsa AAAA\r\nBBBB\r\nCCCC",
			"ssh-rsa AAAA\nBBBB\nCCCC",
		},
		{
			"trim whitespace",
			"  ssh-ed25519 AAAAC3... user@host  ",
			"ssh-ed25519 AAAAC3... user@host",
		},
		{
			"trim and remove CR",
			"  ssh-ed25519 AAAAC3...\r\n  ",
			"ssh-ed25519 AAAAC3...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSSHKey(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeSSHKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// RegisterInput Validation Tests
// =============================================================================

func TestRegisterInput_Validate(t *testing.T) {
	validSSHKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample user@host"

	tests := []struct {
		name       string
		input      RegisterInput
		wantErrors []string // empty means no errors expected
	}{
		{
			"valid input",
			RegisterInput{
				Username:  "johndoe",
				Email:     "john@example.com",
				Team:      "platform",
				PublicKey: validSSHKey,
			},
			nil,
		},
		{
			"missing username",
			RegisterInput{
				Username:  "",
				Email:     "john@example.com",
				PublicKey: validSSHKey,
			},
			[]string{"username is required"},
		},
		{
			"invalid username",
			RegisterInput{
				Username:  "John_Doe",
				Email:     "john@example.com",
				PublicKey: validSSHKey,
			},
			[]string{"username must be 3-20 characters"},
		},
		{
			"missing email",
			RegisterInput{
				Username:  "johndoe",
				Email:     "",
				PublicKey: validSSHKey,
			},
			[]string{"email is required"},
		},
		{
			"invalid email",
			RegisterInput{
				Username:  "johndoe",
				Email:     "invalid-email",
				PublicKey: validSSHKey,
			},
			[]string{"invalid email format"},
		},
		{
			"missing public key",
			RegisterInput{
				Username:  "johndoe",
				Email:     "john@example.com",
				PublicKey: "",
			},
			[]string{"public_key is required"},
		},
		{
			"invalid public key",
			RegisterInput{
				Username:  "johndoe",
				Email:     "john@example.com",
				PublicKey: "not-a-valid-key",
			},
			[]string{"invalid SSH public key format"},
		},
		{
			"multiple errors",
			RegisterInput{
				Username:  "",
				Email:     "",
				PublicKey: "",
			},
			[]string{"username is required", "email is required", "public_key is required"},
		},
		{
			"sanitizes Windows line endings in SSH key",
			RegisterInput{
				Username:  "johndoe",
				Email:     "john@example.com",
				PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample\r\n user@host",
			},
			nil, // Should pass after sanitization
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid mutation between tests
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
// KeyChangeInput Validation Tests
// =============================================================================

func TestKeyChangeInput_Validate(t *testing.T) {
	validSSHKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample user@host"

	tests := []struct {
		name       string
		input      KeyChangeInput
		wantErrors int
	}{
		{
			"valid input",
			KeyChangeInput{
				Username:     "johndoe",
				Email:        "john@example.com",
				NewPublicKey: validSSHKey,
				Reason:       "Lost old key",
			},
			0,
		},
		{
			"missing username",
			KeyChangeInput{
				Username:     "",
				Email:        "john@example.com",
				NewPublicKey: validSSHKey,
			},
			1,
		},
		{
			"missing email",
			KeyChangeInput{
				Username:     "johndoe",
				Email:        "",
				NewPublicKey: validSSHKey,
			},
			1,
		},
		{
			"invalid email",
			KeyChangeInput{
				Username:     "johndoe",
				Email:        "invalid",
				NewPublicKey: validSSHKey,
			},
			1,
		},
		{
			"missing new public key",
			KeyChangeInput{
				Username:     "johndoe",
				Email:        "john@example.com",
				NewPublicKey: "",
			},
			1,
		},
		{
			"invalid new public key",
			KeyChangeInput{
				Username:     "johndoe",
				Email:        "john@example.com",
				NewPublicKey: "invalid-key",
			},
			1,
		},
		{
			"all fields missing",
			KeyChangeInput{},
			3, // username, email, new_public_key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			errors := input.Validate()

			if len(errors) != tt.wantErrors {
				t.Errorf("Validate() returned %d errors, want %d: %v",
					len(errors), tt.wantErrors, errors)
			}
		})
	}
}
