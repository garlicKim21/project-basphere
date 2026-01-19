package store

import (
	"github.com/basphere/basphere-api/internal/model"
)

// Store defines the interface for storing registration requests
// This interface allows easy swapping between file-based and database storage
type Store interface {
	// Create creates a new registration request
	Create(req *model.RegistrationRequest) error

	// Get retrieves a registration request by ID
	Get(id string) (*model.RegistrationRequest, error)

	// GetByUsername retrieves a registration request by username
	GetByUsername(username string) (*model.RegistrationRequest, error)

	// List returns all registration requests with optional status filter
	List(status *model.RequestStatus) ([]*model.RegistrationRequest, error)

	// Update updates a registration request
	Update(req *model.RegistrationRequest) error

	// Delete deletes a registration request
	Delete(id string) error

	// ExistsUsername checks if a username already has a pending request
	ExistsUsername(username string) (bool, error)

	// ExistsEmail checks if an email already has a pending request
	ExistsEmail(email string) (bool, error)
}
