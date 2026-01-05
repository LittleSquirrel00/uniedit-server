package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryStateStore is an in-memory implementation of StateStore.
// Suitable for single-instance deployments.
// For production multi-instance deployments, use Redis-based implementation.
type MemoryStateStore struct {
	mu     sync.RWMutex
	states map[string]*stateEntry
	ttl    time.Duration
}

type stateEntry struct {
	data      string
	expiresAt time.Time
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore(ttl time.Duration) *MemoryStateStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	store := &MemoryStateStore{
		states: make(map[string]*stateEntry),
		ttl:    ttl,
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// Set stores a state with data.
func (s *MemoryStateStore) Set(_ context.Context, state string, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state] = &stateEntry{
		data:      data,
		expiresAt: time.Now().Add(s.ttl),
	}
	return nil
}

// Get retrieves data for a state.
func (s *MemoryStateStore) Get(_ context.Context, state string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.states[state]
	if !ok {
		return "", fmt.Errorf("state not found")
	}

	if time.Now().After(entry.expiresAt) {
		return "", fmt.Errorf("state expired")
	}

	return entry.data, nil
}

// Delete removes a state.
func (s *MemoryStateStore) Delete(_ context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, state)
	return nil
}

// cleanup periodically removes expired states.
func (s *MemoryStateStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.states {
			if now.After(entry.expiresAt) {
				delete(s.states, key)
			}
		}
		s.mu.Unlock()
	}
}
