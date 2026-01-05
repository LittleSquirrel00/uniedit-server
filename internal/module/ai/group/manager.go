package group

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
)

// Manager manages AI model groups.
type Manager struct {
	mu     sync.RWMutex
	repo   Repository
	groups map[string]*Group

	// Round-robin state
	rrIndex map[string]*atomic.Uint64
}

// NewManager creates a new group manager.
func NewManager(repo Repository) *Manager {
	return &Manager{
		repo:    repo,
		groups:  make(map[string]*Group),
		rrIndex: make(map[string]*atomic.Uint64),
	}
}

// Start loads groups from database.
func (m *Manager) Start(ctx context.Context) error {
	return m.Refresh(ctx)
}

// Refresh reloads groups from database.
func (m *Manager) Refresh(ctx context.Context) error {
	groups, err := m.repo.List(ctx, true)
	if err != nil {
		return fmt.Errorf("load groups: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.groups = make(map[string]*Group, len(groups))
	for _, g := range groups {
		m.groups[g.ID] = g
		if _, ok := m.rrIndex[g.ID]; !ok {
			m.rrIndex[g.ID] = &atomic.Uint64{}
		}
	}

	return nil
}

// Get returns a group by ID.
func (m *Manager) Get(ctx context.Context, id string) (*Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

// GetByTaskType returns groups by task type.
func (m *Manager) GetByTaskType(taskType TaskType) []*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Group
	for _, g := range m.groups {
		if g.TaskType == taskType {
			result = append(result, g)
		}
	}
	return result
}

// GetDefault returns the default group for a task type.
func (m *Manager) GetDefault(taskType TaskType) (*Group, error) {
	groups := m.GetByTaskType(taskType)
	if len(groups) == 0 {
		return nil, fmt.Errorf("no default group for task type: %s", taskType)
	}

	// Return first group (usually named "default")
	for _, g := range groups {
		if g.ID == "default" || g.ID == string(taskType)+"-default" {
			return g, nil
		}
	}

	return groups[0], nil
}

// SelectModel selects a model from the group based on its strategy.
func (m *Manager) SelectModel(group *Group, availableModels []string) (string, error) {
	if len(availableModels) == 0 {
		return "", fmt.Errorf("no available models")
	}

	// Filter to models in group
	var candidates []string
	for _, modelID := range group.Models {
		for _, available := range availableModels {
			if modelID == available {
				candidates = append(candidates, modelID)
				break
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no matching models in group")
	}

	switch group.Strategy.Type {
	case StrategyPriority:
		return m.selectPriority(candidates)
	case StrategyRoundRobin:
		return m.selectRoundRobin(group.ID, candidates)
	case StrategyWeighted:
		return m.selectWeighted(candidates, group.Strategy.Weights)
	default:
		return m.selectPriority(candidates)
	}
}

// selectPriority selects the first model.
func (m *Manager) selectPriority(candidates []string) (string, error) {
	return candidates[0], nil
}

// selectRoundRobin selects models in round-robin fashion.
func (m *Manager) selectRoundRobin(groupID string, candidates []string) (string, error) {
	m.mu.RLock()
	idx, ok := m.rrIndex[groupID]
	m.mu.RUnlock()

	if !ok {
		idx = &atomic.Uint64{}
		m.mu.Lock()
		m.rrIndex[groupID] = idx
		m.mu.Unlock()
	}

	current := idx.Add(1) - 1
	return candidates[int(current)%len(candidates)], nil
}

// selectWeighted selects models based on weights.
func (m *Manager) selectWeighted(candidates []string, weights map[string]int) (string, error) {
	if len(weights) == 0 {
		// Equal weights
		return candidates[rand.Intn(len(candidates))], nil
	}

	// Calculate total weight
	var totalWeight int
	for _, modelID := range candidates {
		w := weights[modelID]
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	// Random selection based on weight
	r := rand.Intn(totalWeight)
	var cumulative int
	for _, modelID := range candidates {
		w := weights[modelID]
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if r < cumulative {
			return modelID, nil
		}
	}

	return candidates[0], nil
}

// All returns all groups.
func (m *Manager) All() []*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Group, 0, len(m.groups))
	for _, g := range m.groups {
		result = append(result, g)
	}
	return result
}
