package aiprovider

import (
	"github.com/uniedit/server/internal/model"
)

// BaseAdapter provides common functionality for vendor adapters.
type BaseAdapter struct {
	capabilities map[model.AICapability]bool
}

// NewBaseAdapter creates a new base adapter with specified capabilities.
func NewBaseAdapter(caps ...model.AICapability) *BaseAdapter {
	capMap := make(map[model.AICapability]bool)
	for _, cap := range caps {
		capMap[cap] = true
	}
	return &BaseAdapter{capabilities: capMap}
}

// SupportsCapability checks if the adapter supports a capability.
func (b *BaseAdapter) SupportsCapability(cap model.AICapability) bool {
	return b.capabilities[cap]
}

// AddCapability adds a capability to the adapter.
func (b *BaseAdapter) AddCapability(cap model.AICapability) {
	b.capabilities[cap] = true
}

// RemoveCapability removes a capability from the adapter.
func (b *BaseAdapter) RemoveCapability(cap model.AICapability) {
	delete(b.capabilities, cap)
}

// Capabilities returns all supported capabilities.
func (b *BaseAdapter) Capabilities() []model.AICapability {
	caps := make([]model.AICapability, 0, len(b.capabilities))
	for cap := range b.capabilities {
		caps = append(caps, cap)
	}
	return caps
}
