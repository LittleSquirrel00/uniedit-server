// Package ai provides adapters to integrate the independent media module with the AI module.
package ai

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/media"
	sharedtask "github.com/uniedit/server/internal/shared/task"
)

// mediaProviderAdapter adapts ai/provider.Registry to media.ProviderRegistry.
type mediaProviderAdapter struct {
	registry *provider.Registry
}

// newMediaProviderAdapter creates a new media provider adapter.
func newMediaProviderAdapter(registry *provider.Registry) media.ProviderRegistry {
	return &mediaProviderAdapter{registry: registry}
}

// GetProvider returns a provider by ID.
func (a *mediaProviderAdapter) GetProvider(id uuid.UUID) (*media.Provider, bool) {
	prov, ok := a.registry.GetProvider(id)
	if !ok {
		return nil, false
	}
	return convertProvider(prov), true
}

// GetModelWithProvider returns a model with its provider.
func (a *mediaProviderAdapter) GetModelWithProvider(modelID string) (*media.Model, *media.Provider, bool) {
	model, prov, ok := a.registry.GetModelWithProvider(modelID)
	if !ok {
		return nil, nil, false
	}
	return convertModel(model), convertProvider(prov), true
}

// GetModelsByCapability returns models with a specific capability.
func (a *mediaProviderAdapter) GetModelsByCapability(cap media.Capability) []*media.Model {
	// Convert media capability to provider capability
	provCap := convertCapabilityToProvider(cap)
	models := a.registry.GetModelsByCapability(provCap)
	result := make([]*media.Model, len(models))
	for i, m := range models {
		result[i] = convertModel(m)
	}
	return result
}

// mediaHealthAdapter adapts ai/provider.HealthMonitor to media.HealthChecker.
type mediaHealthAdapter struct {
	monitor *provider.HealthMonitor
}

// newMediaHealthAdapter creates a new media health adapter.
func newMediaHealthAdapter(monitor *provider.HealthMonitor) media.HealthChecker {
	return &mediaHealthAdapter{monitor: monitor}
}

// IsHealthy checks if a provider is healthy.
func (a *mediaHealthAdapter) IsHealthy(providerID uuid.UUID) bool {
	return a.monitor.IsHealthy(providerID)
}

// mediaTaskAdapter adapts shared/task.Manager to media.TaskManager.
type mediaTaskAdapter struct {
	manager *sharedtask.Manager
}

// newMediaTaskAdapter creates a new media task adapter.
func newMediaTaskAdapter(manager *sharedtask.Manager) media.TaskManager {
	return &mediaTaskAdapter{manager: manager}
}

// Submit submits a new task.
func (a *mediaTaskAdapter) Submit(ctx context.Context, ownerID uuid.UUID, req *media.TaskSubmitRequest) (*media.Task, error) {
	t, err := a.manager.Submit(ctx, ownerID, &sharedtask.SubmitRequest{
		Type:    req.Type,
		Payload: req.Payload,
	})
	if err != nil {
		return nil, err
	}
	return convertTask(t), nil
}

// Get retrieves a task by ID.
func (a *mediaTaskAdapter) Get(ctx context.Context, id uuid.UUID) (*media.Task, error) {
	t, err := a.manager.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return convertTask(t), nil
}

// Helper functions for type conversion

func convertProvider(p *provider.Provider) *media.Provider {
	return &media.Provider{
		ID:      p.ID,
		Name:    p.Name,
		Type:    media.ProviderType(p.Type),
		BaseURL: p.BaseURL,
		APIKey:  p.APIKey,
		Enabled: p.Enabled,
	}
}

func convertModel(m *provider.Model) *media.Model {
	caps := make([]media.Capability, 0)
	for _, c := range m.Capabilities {
		// m.Capabilities is pq.StringArray ([]string), convert to Capability first
		if mediaCap := convertCapabilityToMedia(provider.Capability(c)); mediaCap != "" {
			caps = append(caps, mediaCap)
		}
	}
	return &media.Model{
		ID:           m.ID,
		ProviderID:   m.ProviderID,
		Name:         m.Name,
		Capabilities: caps,
		Enabled:      m.Enabled,
	}
}

func convertCapabilityToMedia(c provider.Capability) media.Capability {
	switch c {
	case provider.CapabilityImage:
		return media.CapabilityImage
	case provider.CapabilityVideo:
		return media.CapabilityVideo
	case provider.CapabilityAudio:
		return media.CapabilityAudio
	default:
		return ""
	}
}

func convertCapabilityToProvider(c media.Capability) provider.Capability {
	switch c {
	case media.CapabilityImage:
		return provider.CapabilityImage
	case media.CapabilityVideo:
		return provider.CapabilityVideo
	case media.CapabilityAudio:
		return provider.CapabilityAudio
	default:
		return ""
	}
}

func convertTask(t *sharedtask.Task) *media.Task {
	result := &media.Task{
		ID:        t.ID,
		OwnerID:   t.OwnerID,
		Type:      t.Type,
		Status:    media.TaskStatus(t.Status),
		Progress:  t.Progress,
		Input:     t.Input,
		Output:    t.Output,
		CreatedAt: t.CreatedAt.Unix(),
	}
	if t.Error != nil {
		result.Error = &media.TaskError{
			Code:    t.Error.Code,
			Message: t.Error.Message,
		}
	}
	return result
}
