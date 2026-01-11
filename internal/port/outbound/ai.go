package outbound

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// ===== Provider Database Ports =====

// AIProviderDatabasePort defines provider persistence operations.
type AIProviderDatabasePort interface {
	// Create creates a new provider.
	Create(ctx context.Context, provider *model.AIProvider) error

	// FindByID finds a provider by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.AIProvider, error)

	// FindAll finds all providers.
	FindAll(ctx context.Context) ([]*model.AIProvider, error)

	// FindEnabled finds all enabled providers.
	FindEnabled(ctx context.Context) ([]*model.AIProvider, error)

	// FindByType finds providers by type.
	FindByType(ctx context.Context, providerType model.AIProviderType) ([]*model.AIProvider, error)

	// Update updates a provider.
	Update(ctx context.Context, provider *model.AIProvider) error

	// Delete deletes a provider.
	Delete(ctx context.Context, id uuid.UUID) error
}

// AIModelDatabasePort defines model persistence operations.
type AIModelDatabasePort interface {
	// Create creates a new model.
	Create(ctx context.Context, m *model.AIModel) error

	// FindByID finds a model by ID.
	FindByID(ctx context.Context, id string) (*model.AIModel, error)

	// FindByProvider finds models by provider ID.
	FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIModel, error)

	// FindEnabled finds all enabled models.
	FindEnabled(ctx context.Context) ([]*model.AIModel, error)

	// FindByCapability finds models that have a specific capability.
	FindByCapability(ctx context.Context, capability model.AICapability) ([]*model.AIModel, error)

	// FindByCapabilities finds models that have all specified capabilities.
	FindByCapabilities(ctx context.Context, capabilities []model.AICapability) ([]*model.AIModel, error)

	// Update updates a model.
	Update(ctx context.Context, m *model.AIModel) error

	// Delete deletes a model.
	Delete(ctx context.Context, id string) error

	// DeleteByProvider deletes all models for a provider.
	DeleteByProvider(ctx context.Context, providerID uuid.UUID) error
}

// ===== Provider Account (Pool) Database Ports =====

// AIProviderAccountDatabasePort defines provider account persistence operations.
type AIProviderAccountDatabasePort interface {
	// Create creates a new account.
	Create(ctx context.Context, account *model.AIProviderAccount) error

	// FindByID finds an account by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.AIProviderAccount, error)

	// FindByProvider finds all accounts for a provider.
	FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error)

	// FindActiveByProvider finds all active accounts for a provider.
	FindActiveByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error)

	// FindAvailableByProvider finds all available accounts for a provider (active and can serve requests).
	FindAvailableByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIProviderAccount, error)

	// Update updates an account.
	Update(ctx context.Context, account *model.AIProviderAccount) error

	// UpdateHealth updates account health status.
	UpdateHealth(ctx context.Context, id uuid.UUID, status model.AIHealthStatus, consecutiveFailures int) error

	// IncrementUsage increments usage statistics.
	IncrementUsage(ctx context.Context, id uuid.UUID, requests, tokens int64, cost float64) error

	// Delete deletes an account.
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByProvider deletes all accounts for a provider.
	DeleteByProvider(ctx context.Context, providerID uuid.UUID) error
}

// AIAccountUsageStatsDatabasePort defines account usage stats persistence operations.
type AIAccountUsageStatsDatabasePort interface {
	// RecordUsage records daily usage statistics.
	RecordUsage(ctx context.Context, accountID uuid.UUID, requests, tokens int64, cost float64) error

	// FindByAccount finds usage stats for an account.
	FindByAccount(ctx context.Context, accountID uuid.UUID, days int) ([]*model.AIAccountUsageStats, error)

	// FindByAccountAndDateRange finds usage stats for a date range.
	FindByAccountAndDateRange(ctx context.Context, accountID uuid.UUID, start, end time.Time) ([]*model.AIAccountUsageStats, error)
}

// ===== Model Group Database Ports =====

// AIModelGroupDatabasePort defines model group persistence operations.
type AIModelGroupDatabasePort interface {
	// Create creates a new group.
	Create(ctx context.Context, group *model.AIModelGroup) error

	// FindByID finds a group by ID.
	FindByID(ctx context.Context, id string) (*model.AIModelGroup, error)

	// FindAll finds all groups.
	FindAll(ctx context.Context) ([]*model.AIModelGroup, error)

	// FindEnabled finds all enabled groups.
	FindEnabled(ctx context.Context) ([]*model.AIModelGroup, error)

	// FindByTaskType finds groups by task type.
	FindByTaskType(ctx context.Context, taskType model.AITaskType) ([]*model.AIModelGroup, error)

	// Update updates a group.
	Update(ctx context.Context, group *model.AIModelGroup) error

	// Delete deletes a group.
	Delete(ctx context.Context, id string) error
}

// ===== Cache Ports =====

// AIProviderHealthCachePort defines provider health status caching.
type AIProviderHealthCachePort interface {
	// GetProviderHealth gets provider health status.
	GetProviderHealth(ctx context.Context, providerID uuid.UUID) (bool, error)

	// SetProviderHealth sets provider health status.
	SetProviderHealth(ctx context.Context, providerID uuid.UUID, healthy bool, ttl time.Duration) error

	// GetAccountHealth gets account health status.
	GetAccountHealth(ctx context.Context, accountID uuid.UUID) (model.AIHealthStatus, error)

	// SetAccountHealth sets account health status.
	SetAccountHealth(ctx context.Context, accountID uuid.UUID, status model.AIHealthStatus, ttl time.Duration) error

	// InvalidateProviderHealth invalidates provider health cache.
	InvalidateProviderHealth(ctx context.Context, providerID uuid.UUID) error

	// InvalidateAccountHealth invalidates account health cache.
	InvalidateAccountHealth(ctx context.Context, accountID uuid.UUID) error
}

// AIEmbeddingCachePort defines embedding caching operations.
type AIEmbeddingCachePort interface {
	// Get gets cached embedding.
	Get(ctx context.Context, key string) ([]float64, error)

	// Set sets embedding in cache.
	Set(ctx context.Context, key string, embedding []float64, ttl time.Duration) error

	// Delete deletes cached embedding.
	Delete(ctx context.Context, key string) error

	// GenerateKey generates a cache key for the given model and input.
	GenerateKey(model string, input string) string
}

// ===== Vendor Adapter Ports =====

// AIVendorAdapterPort defines the interface for AI vendor adapters.
type AIVendorAdapterPort interface {
	// Type returns the provider type this adapter supports.
	Type() model.AIProviderType

	// SupportsCapability checks if the adapter supports a capability.
	SupportsCapability(cap model.AICapability) bool

	// HealthCheck performs a health check on the provider.
	HealthCheck(ctx context.Context, provider *model.AIProvider, apiKey string) error

	// Chat sends a chat completion request.
	Chat(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIChatResponse, error)

	// ChatStream sends a streaming chat request.
	ChatStream(ctx context.Context, req *model.AIChatRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (<-chan *model.AIChatChunk, error)

	// Embed generates embeddings.
	Embed(ctx context.Context, req *model.AIEmbedRequest, m *model.AIModel, p *model.AIProvider, apiKey string) (*model.AIEmbedResponse, error)
}

// AIVendorRegistryPort defines vendor adapter registry.
type AIVendorRegistryPort interface {
	// Register registers an adapter.
	Register(adapter AIVendorAdapterPort)

	// Get gets an adapter by provider type.
	Get(providerType model.AIProviderType) (AIVendorAdapterPort, error)

	// GetForProvider gets an adapter for a provider.
	GetForProvider(provider *model.AIProvider) (AIVendorAdapterPort, error)

	// SupportedTypes returns all supported provider types.
	SupportedTypes() []model.AIProviderType
}

// ===== Crypto Port =====

// AICryptoPort defines encryption operations for API keys.
type AICryptoPort interface {
	// Encrypt encrypts plaintext.
	Encrypt(plaintext string) (string, error)

	// Decrypt decrypts ciphertext.
	Decrypt(ciphertext string) (string, error)
}

// ===== Usage Recording Port =====

// AIUsageRecorderPort defines usage recording for billing integration.
type AIUsageRecorderPort interface {
	// RecordUsage records AI usage for billing.
	RecordUsage(ctx context.Context, userID uuid.UUID, modelID string, promptTokens, completionTokens int, cost float64) error
}
