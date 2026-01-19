package model

import "time"

// RequestStatus represents the status of a registration request
type RequestStatus string

const (
	StatusPending  RequestStatus = "pending"
	StatusApproved RequestStatus = "approved"
	StatusRejected RequestStatus = "rejected"
)

// RegistrationRequest represents a user registration request
type RegistrationRequest struct {
	ID        string        `json:"id"`
	Username  string        `json:"username"`
	Email     string        `json:"email"`
	Team      string        `json:"team,omitempty"`
	PublicKey string        `json:"public_key"`
	Status    RequestStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	// Filled when approved or rejected
	ProcessedBy string `json:"processed_by,omitempty"`
	ProcessedAt string `json:"processed_at,omitempty"`
	RejectReason string `json:"reject_reason,omitempty"`
}

// RegisterInput represents the input for user registration
type RegisterInput struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Team      string `json:"team"`
	PublicKey string `json:"public_key"`
}

// Validate validates the registration input
func (r *RegisterInput) Validate() []string {
	var errors []string

	if r.Username == "" {
		errors = append(errors, "username is required")
	} else if !isValidUsername(r.Username) {
		errors = append(errors, "username must be 3-20 characters, lowercase letters, numbers, and hyphens only")
	}

	if r.Email == "" {
		errors = append(errors, "email is required")
	} else if !isValidEmail(r.Email) {
		errors = append(errors, "invalid email format")
	}

	if r.PublicKey == "" {
		errors = append(errors, "public_key is required")
	} else if !isValidSSHPublicKey(r.PublicKey) {
		errors = append(errors, "invalid SSH public key format")
	}

	return errors
}

// isValidUsername checks if username is valid
func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	for i, c := range username {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i > 0 && i < len(username)-1 {
			continue
		}
		return false
	}
	return true
}

// isValidEmail checks if email is valid (basic check)
func isValidEmail(email string) bool {
	hasAt := false
	hasDot := false
	atIndex := -1

	for i, c := range email {
		if c == '@' {
			if hasAt {
				return false // multiple @
			}
			hasAt = true
			atIndex = i
		}
		if c == '.' && hasAt && i > atIndex+1 {
			hasDot = true
		}
	}

	return hasAt && hasDot && atIndex > 0
}

// isValidSSHPublicKey checks if the public key is valid
func isValidSSHPublicKey(key string) bool {
	// Basic validation: should start with ssh-rsa, ssh-ed25519, ecdsa-sha2-, etc.
	validPrefixes := []string{
		"ssh-rsa",
		"ssh-ed25519",
		"ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384",
		"ecdsa-sha2-nistp521",
		"ssh-dss",
	}

	for _, prefix := range validPrefixes {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// ApproveInput represents the input for approving a request
type ApproveInput struct {
	ProcessedBy string `json:"processed_by"`
}

// RejectInput represents the input for rejecting a request
type RejectInput struct {
	ProcessedBy string `json:"processed_by"`
	Reason      string `json:"reason"`
}

// KeyChangeRequest represents a request to change SSH public key
type KeyChangeRequest struct {
	ID           string        `json:"id"`
	Username     string        `json:"username"`
	Email        string        `json:"email"`
	NewPublicKey string        `json:"new_public_key"`
	Reason       string        `json:"reason,omitempty"` // Why changing key (lost, rotation, etc.)
	Status       RequestStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	ProcessedBy  string        `json:"processed_by,omitempty"`
	ProcessedAt  string        `json:"processed_at,omitempty"`
	RejectReason string        `json:"reject_reason,omitempty"`
}

// KeyChangeInput represents the input for key change request
type KeyChangeInput struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	NewPublicKey string `json:"new_public_key"`
	Reason       string `json:"reason"`
}

// Validate validates the key change input
func (k *KeyChangeInput) Validate() []string {
	var errors []string

	if k.Username == "" {
		errors = append(errors, "사용자명을 입력해주세요")
	}

	if k.Email == "" {
		errors = append(errors, "이메일을 입력해주세요")
	} else if !isValidEmail(k.Email) {
		errors = append(errors, "올바른 이메일 형식이 아닙니다")
	}

	if k.NewPublicKey == "" {
		errors = append(errors, "새 SSH 공개키를 입력해주세요")
	} else if !isValidSSHPublicKey(k.NewPublicKey) {
		errors = append(errors, "올바른 SSH 공개키 형식이 아닙니다")
	}

	return errors
}
