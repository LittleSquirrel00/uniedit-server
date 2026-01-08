package media

import (
	"context"

	"github.com/google/uuid"
)

// TaskRepository defines the interface for media task persistence.
type TaskRepository interface {
	// Create creates a new task.
	Create(ctx context.Context, task *Task) error

	// GetByID retrieves a task by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)

	// Update updates a task.
	Update(ctx context.Context, task *Task) error

	// ListByOwner lists tasks by owner.
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*Task, error)

	// ListPending lists pending tasks.
	ListPending(ctx context.Context, limit int) ([]*Task, error)
}

// ProviderRegistry provides access to media providers and models.
// This is a port interface - actual implementation depends on AI module.
type ProviderRegistry interface {
	// GetProvider returns a provider by ID.
	GetProvider(id uuid.UUID) (*Provider, bool)

	// GetModelWithProvider returns a model and its provider.
	GetModelWithProvider(modelID string) (*Model, *Provider, bool)

	// GetModelsByCapability returns models with a specific capability.
	GetModelsByCapability(cap Capability) []*Model
}

// HealthChecker checks provider health status.
// This is a port interface - actual implementation depends on AI module.
type HealthChecker interface {
	// IsHealthy checks if a provider is healthy.
	IsHealthy(providerID uuid.UUID) bool
}

// Adapter defines the interface for media generation adapters.
type Adapter interface {
	// Type returns the provider type this adapter supports.
	Type() ProviderType

	// GenerateImage generates images from a text prompt.
	GenerateImage(ctx context.Context, req *ImageGenerationRequest, model *Model, provider *Provider) (*ImageResult, error)

	// GenerateVideo generates videos from input.
	GenerateVideo(ctx context.Context, req *VideoGenerationRequest, model *Model, provider *Provider) (*VideoGenerationResult, error)

	// GetVideoStatus checks the status of a video generation task.
	GetVideoStatus(ctx context.Context, taskID string, provider *Provider) (*VideoStatusResult, error)

	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap Capability) bool

	// HealthCheck performs a health check on the provider.
	HealthCheck(ctx context.Context, provider *Provider) error
}

// AdapterRegistry manages media adapters.
type AdapterRegistry interface {
	// Register registers an adapter.
	Register(adapter Adapter)

	// Get returns an adapter by provider type.
	Get(providerType ProviderType) (Adapter, error)

	// GetForProvider returns an adapter for a provider.
	GetForProvider(provider *Provider) (Adapter, error)
}

// ImageGenerationRequest represents an image generation request.
type ImageGenerationRequest struct {
	Prompt         string
	NegativePrompt string
	N              int
	Size           string
	Quality        string
	Style          string
	ResponseFormat string
	Model          string
}

// VideoGenerationRequest represents a video generation request.
type VideoGenerationRequest struct {
	Prompt      string
	InputImage  string
	InputVideo  string
	Duration    int
	AspectRatio string
	Resolution  string
	FPS         int
	Model       string
	Seed        *int64
}

// VideoGenerationResult represents the result of video generation submission.
type VideoGenerationResult struct {
	TaskID    string
	Status    VideoState
	CreatedAt int64
}

// VideoStatusResult represents the status of a video generation task.
type VideoStatusResult struct {
	TaskID    string
	Status    VideoState
	Progress  int
	Video     *GeneratedVideo
	Error     string
	CreatedAt int64
	UpdatedAt int64
}
