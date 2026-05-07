package store

import (
	"sync"
	"time"

	"github.com/n3055/backend-project/internal/domain"
)

// MemoryStore is a thread-safe in-memory implementation of the Store interface.
// Suitable for development and single-instance deployments.
// For production multi-instance deployments, swap with a Redis or database-backed store.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*domain.Session
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*domain.Session),
	}
}

func (s *MemoryStore) CreateSession(id, instructions string) (*domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	session := &domain.Session{
		ID:           id,
		Instructions: instructions,
		Messages:     []domain.Message{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.sessions[id] = session

	// Return a copy to prevent external mutation.
	cp := *session
	return &cp, nil
}

func (s *MemoryStore) GetSession(id string) (*domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, &ErrNotFound{ID: id}
	}

	// Return a deep copy to prevent external mutation of internal state.
	cp := *session
	cp.Messages = make([]domain.Message, len(session.Messages))
	copy(cp.Messages, session.Messages)
	return &cp, nil
}

func (s *MemoryStore) UpdateSession(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[session.ID]; !ok {
		return &ErrNotFound{ID: session.ID}
	}

	session.UpdatedAt = time.Now().UTC()

	// Store a deep copy.
	cp := *session
	cp.Messages = make([]domain.Message, len(session.Messages))
	copy(cp.Messages, session.Messages)
	s.sessions[session.ID] = &cp

	return nil
}

func (s *MemoryStore) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; !ok {
		return &ErrNotFound{ID: id}
	}

	delete(s.sessions, id)
	return nil
}

func (s *MemoryStore) SessionExists(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.sessions[id]
	return ok, nil
}
