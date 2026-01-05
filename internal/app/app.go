package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/module/ai"
	"github.com/uniedit/server/internal/module/ai/cache"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/task"
	sharedcache "github.com/uniedit/server/internal/shared/cache"
	"github.com/uniedit/server/internal/shared/config"
	"github.com/uniedit/server/internal/shared/database"
	"github.com/uniedit/server/internal/shared/middleware"
	"gorm.io/gorm"
)

// App represents the application.
type App struct {
	config *config.Config
	db     *gorm.DB
	redis  redis.UniversalClient
	router *gin.Engine

	// Modules
	aiModule *ai.Module
}

// New creates a new application instance.
func New(cfg *config.Config) (*App, error) {
	app := &App{
		config: cfg,
	}

	// Initialize database
	db, err := database.New(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	app.db = db

	// Initialize Redis (optional)
	if cfg.Redis.Address != "" {
		redisClient, err := sharedcache.NewRedisClient(&cfg.Redis)
		if err != nil {
			// Redis is optional, log warning but continue
			fmt.Printf("Warning: Redis connection failed: %v\n", err)
		} else {
			app.redis = redisClient
		}
	}

	// Initialize router
	app.router = app.setupRouter()

	// Initialize modules
	if err := app.initModules(); err != nil {
		return nil, fmt.Errorf("init modules: %w", err)
	}

	// Start modules
	ctx := context.Background()
	if err := app.startModules(ctx); err != nil {
		return nil, fmt.Errorf("start modules: %w", err)
	}

	return app, nil
}

// setupRouter creates and configures the Gin router.
func (a *App) setupRouter() *gin.Engine {
	// Set Gin mode based on environment
	if a.config.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Apply global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}

// initModules initializes all application modules.
func (a *App) initModules() error {
	// Initialize AI module
	aiConfig := &ai.Config{
		DB:    a.db,
		Redis: a.redis,
		HealthCheckConfig: &provider.HealthMonitorConfig{
			CheckInterval:       a.config.AI.HealthCheckInterval,
			FailureThreshold:    a.config.AI.FailureThreshold,
			SuccessThreshold:    a.config.AI.SuccessThreshold,
			Timeout:             a.config.AI.CircuitTimeout,
			MaxHalfOpenRequests: 1,
		},
		TaskManagerConfig: &task.ManagerConfig{
			MaxConcurrent: a.config.AI.MaxConcurrentTasks,
		},
		EmbeddingCacheConfig: &cache.EmbeddingCacheConfig{
			TTL: a.config.AI.EmbeddingCacheTTL,
		},
	}

	aiModule, err := ai.NewModule(aiConfig)
	if err != nil {
		return fmt.Errorf("create ai module: %w", err)
	}
	a.aiModule = aiModule

	return nil
}

// startModules starts all application modules.
func (a *App) startModules(ctx context.Context) error {
	// Start AI module
	if err := a.aiModule.Start(ctx); err != nil {
		return fmt.Errorf("start ai module: %w", err)
	}

	// Register module routes
	a.registerRoutes()

	return nil
}

// registerRoutes registers routes for all modules.
func (a *App) registerRoutes() {
	// API v1 group
	v1 := a.router.Group("/api/v1")

	// Public routes (with rate limiting, auth middleware to be added)
	publicRouter := v1.Group("")

	// Admin routes (requires admin auth)
	adminRouter := v1.Group("/admin")

	// Register AI module routes
	a.aiModule.RegisterRoutes(publicRouter, adminRouter)
}

// Router returns the HTTP router.
func (a *App) Router() *gin.Engine {
	return a.router
}

// Stop stops the application and releases resources.
func (a *App) Stop() {
	// Stop modules
	if a.aiModule != nil {
		a.aiModule.Stop()
	}

	// Close Redis connection
	if a.redis != nil {
		_ = a.redis.Close()
	}

	// Close database connection
	if a.db != nil {
		_ = database.Close(a.db)
	}
}
