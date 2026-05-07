// Package store defines the storage interface (Repository Pattern).
// Business logic depends on this interface, not concrete implementations.
package store

import "github.com/n3055/backend-project/internal/domain"

// Store is the interface for session persistence.
// Implementing this interface allows swapping storage backends
// (memory → Redis → PostgreSQL) without touching business logic.
type Store interface {
	// CreateSession creates a new session and returns it.
	CreateSession(id, instructions string) (*domain.Session, error)

	// GetSession retrieves a session by ID. Returns nil, ErrNotFound if not found.
	GetSession(id string) (*domain.Session, error)

	// UpdateSession persists changes to an existing session.
	UpdateSession(session *domain.Session) error

	// DeleteSession removes a session by ID.
	DeleteSession(id string) error

	// SessionExists checks if a session with the given ID exists.
	SessionExists(id string) (bool, error)
}

// ErrNotFound is returned when a session is not found.
type ErrNotFound struct {
	ID string
}

func (e *ErrNotFound) Error() string {
	return "session not found: " + e.ID
}
