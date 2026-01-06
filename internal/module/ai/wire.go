//go:build wireinject
// +build wireinject

package ai

import (
	"github.com/google/wire"
	"github.com/uniedit/server/internal/module/ai/adapter"
	"github.com/uniedit/server/internal/module/ai/cache"
	"github.com/uniedit/server/internal/module/ai/group"
	"github.com/uniedit/server/internal/module/ai/handler"
	"github.com/uniedit/server/internal/module/ai/llm"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/routing"
	"github.com/uniedit/server/internal/module/media"
	sharedtask "github.com/uniedit/server/internal/shared/task"
	"go.uber.org/zap"
)

// RepositorySet contains all repository providers.
// Note: NewRepository functions return interfaces directly, no binding needed.
var RepositorySet = wire.NewSet(
	provider.NewRepository,
	group.NewRepository,
	sharedtask.NewRepository,
)

// CoreSet contains core component providers.
var CoreSet = wire.NewSet(
	adapter.GetRegistry,
	ProvideProviderRegistry,
	ProvideHealthMonitor,
	group.NewManager,
	ProvideRoutingManager,
	ProvideTaskManager,
)

// ServiceSet contains service providers.
var ServiceSet = wire.NewSet(
	llm.NewService,
	ProvideMediaService,
)

// HandlerSet contains handler providers.
var HandlerSet = wire.NewSet(
	handler.NewChatHandler,
	handler.NewMediaHandler,
	handler.NewTaskHandler,
	handler.NewAdminHandler,
	ProvideHandlers,
)

// ProvideProviderRegistry creates a provider registry.
func ProvideProviderRegistry(repo provider.Repository) *provider.Registry {
	return provider.NewRegistry(repo, nil)
}

// ProvideHealthMonitor creates a health monitor.
func ProvideHealthMonitor(
	registry *provider.Registry,
	adapterRegistry *adapter.Registry,
	config *Config,
) *provider.HealthMonitor {
	return provider.NewHealthMonitor(registry, adapterRegistry, config.HealthCheckConfig)
}

// ProvideRoutingManager creates a routing manager.
func ProvideRoutingManager(
	registry *provider.Registry,
	healthMonitor *provider.HealthMonitor,
	groupManager *group.Manager,
) *routing.Manager {
	return routing.NewManager(registry, healthMonitor, groupManager, nil)
}

// ProvideTaskManager creates a task manager.
func ProvideTaskManager(repo sharedtask.Repository, config *Config) *sharedtask.Manager {
	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return sharedtask.NewManager(repo, logger, config.TaskManagerConfig)
}

// ProvideMediaService creates a media service.
func ProvideMediaService(
	registry *provider.Registry,
	healthMonitor *provider.HealthMonitor,
	taskManager *sharedtask.Manager,
) *media.Service {
	return media.NewService(&media.ServiceConfig{
		ProviderRegistry: newMediaProviderAdapter(registry),
		HealthChecker:    newMediaHealthAdapter(healthMonitor),
		AdapterRegistry:  media.NewAdapterRegistry(),
		TaskManager:      newMediaTaskAdapter(taskManager),
	})
}

// ProvideHandlers creates the handlers struct.
func ProvideHandlers(
	chat *handler.ChatHandler,
	mediaHandler *handler.MediaHandler,
	taskHandler *handler.TaskHandler,
	admin *handler.AdminHandler,
) *handler.Handlers {
	return &handler.Handlers{
		Chat:  chat,
		Media: mediaHandler,
		Task:  taskHandler,
		Admin: admin,
	}
}

// ProvideEmbeddingCache creates an embedding cache if Redis is available.
func ProvideEmbeddingCache(config *Config) *cache.EmbeddingCache {
	if config.Redis == nil {
		return nil
	}
	return cache.NewEmbeddingCache(config.Redis, config.EmbeddingCacheConfig)
}

// ProvideModule assembles the AI module from its components.
func ProvideModule(
	providerRepo provider.Repository,
	groupRepo group.Repository,
	taskRepo sharedtask.Repository,
	registry *provider.Registry,
	healthMonitor *provider.HealthMonitor,
	adapterRegistry *adapter.Registry,
	routingManager *routing.Manager,
	groupManager *group.Manager,
	taskManager *sharedtask.Manager,
	embeddingCache *cache.EmbeddingCache,
	llmService *llm.Service,
	mediaService *media.Service,
	handlers *handler.Handlers,
	config *Config,
) *Module {
	logger := config.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Module{
		providerRepo:    providerRepo,
		groupRepo:       groupRepo,
		taskRepo:        taskRepo,
		registry:        registry,
		healthMonitor:   healthMonitor,
		adapterRegistry: adapterRegistry,
		routingManager:  routingManager,
		groupManager:    groupManager,
		taskManager:     taskManager,
		embeddingCache:  embeddingCache,
		llmService:      llmService,
		mediaService:    mediaService,
		handlers:        handlers,
		logger:          logger,
	}
}

// ModuleSet contains all providers for the AI module.
var ModuleSet = wire.NewSet(
	RepositorySet,
	CoreSet,
	ServiceSet,
	HandlerSet,
	ProvideEmbeddingCache,
	ProvideModule,
)

// InitializeModule is the wire injector for the AI module.
// Wire will generate the implementation in wire_gen.go.
func InitializeModule(config *Config) (*Module, error) {
	wire.Build(
		wire.FieldsOf(new(*Config), "DB"),
		ModuleSet,
	)
	return nil, nil
}
