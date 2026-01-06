package pool

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
)

// ErrNoCandidate is returned when no account is available for selection.
var ErrNoCandidate = errors.New("no candidate account available")

// SchedulerType represents the scheduling strategy.
type SchedulerType string

const (
	SchedulerRoundRobin     SchedulerType = "round_robin"
	SchedulerWeightedRandom SchedulerType = "weighted"
	SchedulerPriority       SchedulerType = "priority"
)

// Scheduler defines the interface for account selection.
type Scheduler interface {
	// Select chooses an account from candidates based on strategy.
	Select(ctx context.Context, candidates []*ProviderAccount) (*ProviderAccount, error)
}

// RoundRobinScheduler implements round-robin account selection.
type RoundRobinScheduler struct {
	counter uint64
}

// NewRoundRobinScheduler creates a new round-robin scheduler.
func NewRoundRobinScheduler() *RoundRobinScheduler {
	return &RoundRobinScheduler{}
}

func (s *RoundRobinScheduler) Select(_ context.Context, candidates []*ProviderAccount) (*ProviderAccount, error) {
	if len(candidates) == 0 {
		return nil, ErrNoCandidate
	}

	idx := atomic.AddUint64(&s.counter, 1) % uint64(len(candidates))
	return candidates[idx], nil
}

// WeightedRandomScheduler implements weighted random account selection.
type WeightedRandomScheduler struct {
	mu sync.Mutex
}

// NewWeightedRandomScheduler creates a new weighted random scheduler.
func NewWeightedRandomScheduler() *WeightedRandomScheduler {
	return &WeightedRandomScheduler{}
}

func (s *WeightedRandomScheduler) Select(_ context.Context, candidates []*ProviderAccount) (*ProviderAccount, error) {
	if len(candidates) == 0 {
		return nil, ErrNoCandidate
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate total weight
	totalWeight := 0
	for _, c := range candidates {
		weight := c.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return candidates[0], nil
	}

	// Random selection based on weight
	r := rand.Intn(totalWeight)
	cumulative := 0
	for _, c := range candidates {
		weight := c.Weight
		if weight <= 0 {
			weight = 1
		}
		cumulative += weight
		if r < cumulative {
			return c, nil
		}
	}

	// Fallback to first candidate
	return candidates[0], nil
}

// PriorityScheduler implements priority-based account selection.
// Selects the highest priority account that is healthy.
type PriorityScheduler struct{}

// NewPriorityScheduler creates a new priority scheduler.
func NewPriorityScheduler() *PriorityScheduler {
	return &PriorityScheduler{}
}

func (s *PriorityScheduler) Select(_ context.Context, candidates []*ProviderAccount) (*ProviderAccount, error) {
	if len(candidates) == 0 {
		return nil, ErrNoCandidate
	}

	// Candidates are already sorted by priority DESC
	// First healthy account wins
	for _, c := range candidates {
		if c.HealthStatus.IsHealthy() {
			return c, nil
		}
	}

	// If no healthy accounts, return first degraded
	for _, c := range candidates {
		if c.HealthStatus == HealthStatusDegraded {
			return c, nil
		}
	}

	// Fallback to first candidate (may be unhealthy)
	return candidates[0], nil
}

// NewScheduler creates a scheduler based on type.
func NewScheduler(schedulerType SchedulerType) Scheduler {
	switch schedulerType {
	case SchedulerWeightedRandom:
		return NewWeightedRandomScheduler()
	case SchedulerPriority:
		return NewPriorityScheduler()
	case SchedulerRoundRobin:
		fallthrough
	default:
		return NewRoundRobinScheduler()
	}
}
