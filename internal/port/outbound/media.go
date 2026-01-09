package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// MediaProviderDatabasePort defines media provider persistence.
type MediaProviderDatabasePort interface {
	// Create creates a new provider.
	Create(ctx context.Context, provider *model.MediaProvider) error

	// FindByID finds a provider by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.MediaProvider, error)

	// FindAll finds all providers.
	FindAll(ctx context.Context) ([]*model.MediaProvider, error)

	// FindEnabled finds all enabled providers.
	FindEnabled(ctx context.Context) ([]*model.MediaProvider, error)

	// Update updates a provider.
	Update(ctx context.Context, provider *model.MediaProvider) error

	// Delete deletes a provider.
	Delete(ctx context.Context, id uuid.UUID) error
}

// MediaModelDatabasePort defines media model persistence.
type MediaModelDatabasePort interface {
	// Create creates a new model.
	Create(ctx context.Context, m *model.MediaModel) error

	// FindByID finds a model by ID.
	FindByID(ctx context.Context, id string) (*model.MediaModel, error)

	// FindByProvider finds all models for a provider.
	FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.MediaModel, error)

	// FindByCapability finds all models with a capability.
	FindByCapability(ctx context.Context, capability model.MediaCapability) ([]*model.MediaModel, error)

	// FindEnabled finds all enabled models.
	FindEnabled(ctx context.Context) ([]*model.MediaModel, error)

	// Update updates a model.
	Update(ctx context.Context, m *model.MediaModel) error

	// Delete deletes a model.
	Delete(ctx context.Context, id string) error
}

// MediaTaskDatabasePort defines media task persistence.
type MediaTaskDatabasePort interface {
	// Create creates a new task.
	Create(ctx context.Context, task *model.MediaTask) error

	// FindByID finds a task by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.MediaTask, error)

	// FindByOwner finds tasks by owner.
	FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.MediaTask, error)

	// FindPending finds pending tasks.
	FindPending(ctx context.Context, limit int) ([]*model.MediaTask, error)

	// Update updates a task.
	Update(ctx context.Context, task *model.MediaTask) error

	// UpdateStatus updates task status and progress.
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.MediaTaskStatus, progress int, output, errMsg string) error

	// Delete deletes a task.
	Delete(ctx context.Context, id uuid.UUID) error
}

// MediaProviderHealthCachePort defines provider health caching.
type MediaProviderHealthCachePort interface {
	// GetHealth gets provider health status.
	GetHealth(ctx context.Context, providerID uuid.UUID) (bool, error)

	// SetHealth sets provider health status.
	SetHealth(ctx context.Context, providerID uuid.UUID, healthy bool) error
}

// MediaVendorAdapterPort defines the interface for media vendor adapters.
type MediaVendorAdapterPort interface {
	// Type returns the provider type this adapter supports.
	Type() model.MediaProviderType

	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap model.MediaCapability) bool

	// HealthCheck performs a health check on the provider.
	HealthCheck(ctx context.Context, prov *model.MediaProvider, apiKey string) error

	// GenerateImage generates images from a text prompt.
	GenerateImage(ctx context.Context, req *model.ImageRequest, m *model.MediaModel, prov *model.MediaProvider, apiKey string) (*model.ImageResponse, error)

	// GenerateVideo generates videos from input.
	GenerateVideo(ctx context.Context, req *model.VideoRequest, m *model.MediaModel, prov *model.MediaProvider, apiKey string) (*model.VideoResponse, error)

	// GetVideoStatus checks the status of a video generation task.
	GetVideoStatus(ctx context.Context, taskID string, prov *model.MediaProvider, apiKey string) (*model.VideoStatus, error)
}

// MediaVendorRegistryPort defines vendor adapter registry.
type MediaVendorRegistryPort interface {
	// Register registers an adapter.
	Register(adapter MediaVendorAdapterPort)

	// Get returns an adapter by provider type.
	Get(providerType model.MediaProviderType) (MediaVendorAdapterPort, error)

	// GetForProvider returns an adapter for a provider.
	GetForProvider(prov *model.MediaProvider) (MediaVendorAdapterPort, error)

	// All returns all registered adapters.
	All() []MediaVendorAdapterPort

	// SupportedTypes returns all supported provider types.
	SupportedTypes() []model.MediaProviderType
}

// MediaCryptoPort defines encryption for API keys.
type MediaCryptoPort interface {
	// Encrypt encrypts a plaintext string.
	Encrypt(plaintext string) (string, error)

	// Decrypt decrypts a ciphertext string.
	Decrypt(ciphertext string) (string, error)
}
