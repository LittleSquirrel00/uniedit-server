package ai

import (
	"context"

	"github.com/google/uuid"
)

// ProviderRepository defines the interface for provider persistence.
type ProviderRepository interface {
	// Create creates a new provider.
	Create(ctx context.Context, provider *Provider) error

	// GetByID retrieves a provider by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Provider, error)

	// List lists all providers.
	List(ctx context.Context, enabledOnly bool) ([]*Provider, error)

	// Update updates a provider.
	Update(ctx context.Context, provider *Provider) error

	// Delete deletes a provider.
	Delete(ctx context.Context, id uuid.UUID) error
}

// ModelRepository defines the interface for model persistence.
type ModelRepository interface {
	// Create creates a new model.
	Create(ctx context.Context, model *Model) error

	// GetByID retrieves a model by ID.
	GetByID(ctx context.Context, id string) (*Model, error)

	// GetByProviderID retrieves models for a provider.
	GetByProviderID(ctx context.Context, providerID uuid.UUID) ([]*Model, error)

	// List lists all models.
	List(ctx context.Context, enabledOnly bool) ([]*Model, error)

	// ListWithCapabilities lists models with specific capabilities.
	ListWithCapabilities(ctx context.Context, caps []Capability, enabledOnly bool) ([]*Model, error)

	// Update updates a model.
	Update(ctx context.Context, model *Model) error

	// Delete deletes a model.
	Delete(ctx context.Context, id string) error
}

// GroupRepository defines the interface for group persistence.
type GroupRepository interface {
	// Create creates a new group.
	Create(ctx context.Context, group *Group) error

	// GetByID retrieves a group by ID.
	GetByID(ctx context.Context, id string) (*Group, error)

	// GetByTaskType retrieves groups for a task type.
	GetByTaskType(ctx context.Context, taskType TaskType) ([]*Group, error)

	// List lists all groups.
	List(ctx context.Context, enabledOnly bool) ([]*Group, error)

	// Update updates a group.
	Update(ctx context.Context, group *Group) error

	// Delete deletes a group.
	Delete(ctx context.Context, id string) error
}

// ProviderRegistry provides runtime access to enabled providers.
type ProviderRegistry interface {
	// Get retrieves a provider by ID.
	Get(id uuid.UUID) (*Provider, bool)

	// GetByType retrieves providers by type.
	GetByType(pType ProviderType) []*Provider

	// List returns all enabled providers.
	List() []*Provider

	// Refresh reloads providers from the database.
	Refresh(ctx context.Context) error
}

// ModelRegistry provides runtime access to enabled models.
type ModelRegistry interface {
	// Get retrieves a model by ID.
	Get(id string) (*Model, bool)

	// GetForProvider retrieves models for a provider.
	GetForProvider(providerID uuid.UUID) []*Model

	// GetWithCapabilities retrieves models with specific capabilities.
	GetWithCapabilities(caps []Capability) []*Model

	// List returns all enabled models.
	List() []*Model

	// Refresh reloads models from the database.
	Refresh(ctx context.Context) error
}

// Router selects the best model/provider for a request.
type Router interface {
	// Route selects a model for the given request.
	Route(ctx context.Context, req *RouteRequest) (*RouteResult, error)
}

// RouteRequest contains routing request parameters.
type RouteRequest struct {
	ModelID            string       // Specific model ID requested
	GroupID            string       // Group ID for group-based routing
	TaskType           TaskType     // Task type for capability matching
	RequiredCapabilities []Capability // Required capabilities
	UserID             uuid.UUID    // User ID for rate limiting
	PreferLowCost      bool         // Prefer lower cost models
	PreferLowLatency   bool         // Prefer lower latency providers
}

// RouteResult contains the routing result.
type RouteResult struct {
	Model    *Model
	Provider *Provider
	Fallback []*Model // Fallback models in order
}

// TextAdapter handles text generation (chat completions).
type TextAdapter interface {
	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, req *ChatRequest, model *Model, prov *Provider) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req *ChatRequest, model *Model, prov *Provider) (<-chan *ChatChunk, error)
}

// EmbeddingAdapter handles text embeddings.
type EmbeddingAdapter interface {
	// Embed generates text embeddings.
	Embed(ctx context.Context, input []string, model *Model, prov *Provider) (*EmbedResponse, error)
}

// HealthChecker performs health checks on providers.
type HealthChecker interface {
	// HealthCheck performs a health check on a provider.
	HealthCheck(ctx context.Context, prov *Provider) error
}

// Adapter is the unified interface that combines all adapter capabilities.
type Adapter interface {
	// Type returns the adapter type identifier.
	Type() ProviderType

	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap Capability) bool

	TextAdapter
	EmbeddingAdapter
	HealthChecker
}

// AdapterRegistry provides runtime access to adapters.
type AdapterRegistry interface {
	// Get retrieves an adapter by provider type.
	Get(pType ProviderType) (Adapter, bool)

	// Register registers an adapter for a provider type.
	Register(pType ProviderType, adapter Adapter)

	// List returns all registered adapters.
	List() map[ProviderType]Adapter
}

// ChatService provides chat completion operations.
type ChatService interface {
	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, userID uuid.UUID, req *ChatRequest) (*ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, userID uuid.UUID, req *ChatRequest) (<-chan *ChatChunk, *RoutingResult, error)
}

// EmbeddingService provides text embedding operations.
type EmbeddingService interface {
	// Embed generates text embeddings.
	Embed(ctx context.Context, userID uuid.UUID, req *EmbedRequest) (*EmbedResponse, error)
}

// LLMService combines all LLM operations.
type LLMService interface {
	ChatService
	EmbeddingService
}

// HealthMonitor monitors provider health status.
type HealthMonitor interface {
	// Start starts the health monitor.
	Start(ctx context.Context) error

	// Stop stops the health monitor.
	Stop() error

	// GetHealth returns the health status of a provider.
	GetHealth(providerID uuid.UUID) bool

	// GetAllHealth returns health status of all providers.
	GetAllHealth() map[string]bool

	// MarkHealthy marks a provider as healthy.
	MarkHealthy(providerID uuid.UUID)

	// MarkUnhealthy marks a provider as unhealthy.
	MarkUnhealthy(providerID uuid.UUID)
}
