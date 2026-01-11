package oauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/uniedit/server/internal/port/outbound"
)

// StateStore is an alias for the interface implemented by state stores.
type StateStore = outbound.OAuthStateStorePort

// inMemoryStateStore implements outbound.OAuthStateStorePort using in-memory storage.
type inMemoryStateStore struct {
	mu     sync.RWMutex
	states map[string]stateEntry
	ttl    time.Duration
}

type stateEntry struct {
	provider  string
	expiresAt time.Time
}

// NewInMemoryStateStore creates a new in-memory OAuth state store.
func NewInMemoryStateStore() outbound.OAuthStateStorePort {
	store := &inMemoryStateStore{
		states: make(map[string]stateEntry),
		ttl:    10 * time.Minute,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// Set stores an OAuth state with provider info.
func (s *inMemoryStateStore) Set(ctx context.Context, state string, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state] = stateEntry{
		provider:  provider,
		expiresAt: time.Now().Add(s.ttl),
	}
	return nil
}

// Get retrieves the provider for a state.
func (s *inMemoryStateStore) Get(ctx context.Context, state string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.states[state]
	if !ok {
		return "", fmt.Errorf("state not found")
	}

	if time.Now().After(entry.expiresAt) {
		return "", fmt.Errorf("state expired")
	}

	return entry.provider, nil
}

// Delete removes a state.
func (s *inMemoryStateStore) Delete(ctx context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, state)
	return nil
}

// cleanup periodically removes expired states.
func (s *inMemoryStateStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for state, entry := range s.states {
			if now.After(entry.expiresAt) {
				delete(s.states, state)
			}
		}
		s.mu.Unlock()
	}
}

// Compile-time check
var _ outbound.OAuthStateStorePort = (*inMemoryStateStore)(nil)
