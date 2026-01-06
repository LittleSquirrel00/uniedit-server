package ai

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/module/ai/adapter"
	"github.com/uniedit/server/internal/module/ai/cache"
	"github.com/uniedit/server/internal/module/ai/group"
	"github.com/uniedit/server/internal/module/ai/handler"
	"github.com/uniedit/server/internal/module/ai/llm"
	"github.com/uniedit/server/internal/module/ai/media"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/provider/pool"
	"github.com/uniedit/server/internal/module/ai/routing"
	"github.com/uniedit/server/internal/module/ai/task"
	"gorm.io/gorm"
)

// Module represents the AI module.
type Module struct {
	// Repositories
	providerRepo provider.Repository
	groupRepo    group.Repository
	taskRepo     task.Repository

	// Core components
	registry        *provider.Registry
	healthMonitor   *provider.HealthMonitor
	adapterRegistry *adapter.Registry
	routingManager  *routing.Manager
	groupManager    *group.Manager
	taskManager     *task.Manager
	embeddingCache  *cache.EmbeddingCache

	// Services
	llmService   *llm.Service
	mediaService *media.Service

	// Handlers
	handlers *handler.Handlers
}

// Config contains module configuration.
type Config struct {
	// Database connection
	DB *gorm.DB

	// Redis client for caching
	Redis redis.UniversalClient

	// Health check configuration
	HealthCheckConfig *provider.HealthMonitorConfig

	// Task manager configuration
	TaskManagerConfig *task.ManagerConfig

	// Embedding cache configuration
	EmbeddingCacheConfig *cache.EmbeddingCacheConfig
}

// NewModule creates a new AI module.
func NewModule(config *Config) (*Module, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("database connection required")
	}

	m := &Module{}

	// Initialize repositories
	m.providerRepo = provider.NewRepository(config.DB)
	m.groupRepo = group.NewRepository(config.DB)
	m.taskRepo = task.NewRepository(config.DB)

	// Initialize adapter registry
	m.adapterRegistry = adapter.GetRegistry()

	// Initialize provider registry
	m.registry = provider.NewRegistry(m.providerRepo, nil)

	// Initialize health monitor (adapter registry implements HealthChecker)
	m.healthMonitor = provider.NewHealthMonitor(m.registry, m.adapterRegistry, config.HealthCheckConfig)

	// Initialize group manager
	m.groupManager = group.NewManager(m.groupRepo)

	// Initialize routing manager
	m.routingManager = routing.NewManager(m.registry, m.healthMonitor, m.groupManager, nil)

	// Initialize task manager
	m.taskManager = task.NewManager(m.taskRepo, config.TaskManagerConfig)

	// Initialize embedding cache (optional)
	if config.Redis != nil {
		m.embeddingCache = cache.NewEmbeddingCache(config.Redis, config.EmbeddingCacheConfig)
	}

	// Initialize services
	m.llmService = llm.NewService(m.registry, m.healthMonitor, m.routingManager)
	m.mediaService = media.NewService(m.registry, m.healthMonitor, m.taskManager)

	// Initialize handlers
	m.handlers = handler.NewHandlers(
		m.llmService,
		m.mediaService,
		m.taskManager,
		m.providerRepo,
		m.groupRepo,
		m.registry,
		m.groupManager,
	)

	return m, nil
}

// Start starts the AI module.
func (m *Module) Start(ctx context.Context) error {
	// Load provider registry
	if err := m.registry.Refresh(ctx); err != nil {
		return fmt.Errorf("refresh registry: %w", err)
	}

	// Start health monitor
	if err := m.healthMonitor.Start(ctx); err != nil {
		return fmt.Errorf("start health monitor: %w", err)
	}

	// Load group manager
	if err := m.groupManager.Start(ctx); err != nil {
		return fmt.Errorf("start group manager: %w", err)
	}

	// Start task manager
	if err := m.taskManager.Start(ctx); err != nil {
		return fmt.Errorf("start task manager: %w", err)
	}

	return nil
}

// Stop stops the AI module.
func (m *Module) Stop() {
	m.healthMonitor.Stop()
	m.taskManager.Stop()
}

// RegisterRoutes registers AI routes.
func (m *Module) RegisterRoutes(publicRouter *gin.RouterGroup, adminRouter *gin.RouterGroup) {
	m.handlers.RegisterRoutes(publicRouter, adminRouter)
}

// LLMService returns the LLM service.
func (m *Module) LLMService() *llm.Service {
	return m.llmService
}

// MediaService returns the media service.
func (m *Module) MediaService() *media.Service {
	return m.mediaService
}

// TaskManager returns the task manager.
func (m *Module) TaskManager() *task.Manager {
	return m.taskManager
}

// Registry returns the provider registry.
func (m *Module) Registry() *provider.Registry {
	return m.registry
}

// HealthMonitor returns the health monitor.
func (m *Module) HealthMonitor() *provider.HealthMonitor {
	return m.healthMonitor
}

// GroupManager returns the group manager.
func (m *Module) GroupManager() *group.Manager {
	return m.groupManager
}

// EmbeddingCache returns the embedding cache.
func (m *Module) EmbeddingCache() *cache.EmbeddingCache {
	return m.embeddingCache
}

// SetAccountPool sets the account pool manager for routing.
func (m *Module) SetAccountPool(accountPool *pool.Manager) {
	m.routingManager.SetAccountPool(accountPool)
}
