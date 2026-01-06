package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/uniedit/server/internal/module/ai"
	"github.com/uniedit/server/internal/module/ai/cache"
	"github.com/uniedit/server/internal/module/ai/provider"
	"github.com/uniedit/server/internal/module/ai/task"
	"github.com/uniedit/server/internal/module/auth"
	"github.com/uniedit/server/internal/module/billing"
	billingquota "github.com/uniedit/server/internal/module/billing/quota"
	billingusage "github.com/uniedit/server/internal/module/billing/usage"
	"github.com/uniedit/server/internal/module/collaboration"
	"github.com/uniedit/server/internal/module/git"
	"github.com/uniedit/server/internal/module/git/lfs"
	gitstorage "github.com/uniedit/server/internal/module/git/storage"
	"github.com/uniedit/server/internal/module/order"
	"github.com/uniedit/server/internal/module/payment"
	paymentprovider "github.com/uniedit/server/internal/module/payment/provider"
	"github.com/uniedit/server/internal/module/user"
	sharedcache "github.com/uniedit/server/internal/shared/cache"
	"github.com/uniedit/server/internal/shared/config"
	"github.com/uniedit/server/internal/shared/database"
	"github.com/uniedit/server/internal/shared/logger"
	"github.com/uniedit/server/internal/shared/middleware"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App represents the application.
type App struct {
	config    *config.Config
	db        *gorm.DB
	redis     redis.UniversalClient
	router    *gin.Engine
	logger    *logger.Logger
	zapLogger *zap.Logger

	// Modules
	aiModule       *ai.Module
	authHandler    *auth.Handler
	authService    *auth.Service
	rateLimiter    *auth.RateLimiter
	userHandler    *user.Handler
	userAdmin      *user.AdminHandler
	billingHandler *billing.Handler
	orderHandler   *order.Handler
	paymentHandler *payment.Handler
	webhookHandler *payment.WebhookHandler
	gitHandler     *git.Handler
	gitHTTPHandler *git.GitHandler
	lfsHandler     *lfs.BatchHandler
	collabHandler  *collaboration.Handler

	// Services (for cross-module dependencies)
	billingService billing.ServiceInterface
	billingRepo    billing.Repository
	orderService   *order.Service
	paymentService *payment.Service
	usageRecorder  *billingusage.Recorder
	quotaChecker   *billingquota.Checker
	gitService     *git.Service
	r2Client       *gitstorage.R2Client
}

// New creates a new application instance.
func New(cfg *config.Config) (*App, error) {
	// Initialize logger
	log := logger.New(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})

	// Initialize zap logger for modules that use zap
	zapLog, err := logger.NewZapLogger(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("init zap logger: %w", err)
	}

	app := &App{
		config:    cfg,
		logger:    log,
		zapLogger: zapLog,
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
	r.Use(middleware.Recovery(a.logger))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logging(a.logger))
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

	// Initialize auth module (for System API Keys)
	if err := a.initAuthModule(); err != nil {
		return fmt.Errorf("init auth module: %w", err)
	}

	// Initialize user module
	if err := a.initUserModule(); err != nil {
		return fmt.Errorf("init user module: %w", err)
	}

	// Initialize billing module
	if err := a.initBillingModule(); err != nil {
		return fmt.Errorf("init billing module: %w", err)
	}

	// Initialize order module
	if err := a.initOrderModule(); err != nil {
		return fmt.Errorf("init order module: %w", err)
	}

	// Initialize payment module
	if err := a.initPaymentModule(); err != nil {
		return fmt.Errorf("init payment module: %w", err)
	}

	// Initialize git module
	if err := a.initGitModule(); err != nil {
		return fmt.Errorf("init git module: %w", err)
	}

	// Initialize collaboration module
	if err := a.initCollaborationModule(); err != nil {
		return fmt.Errorf("init collaboration module: %w", err)
	}

	return nil
}

// initAuthModule initializes the auth module for System API Key management.
func (a *App) initAuthModule() error {
	// Create repositories
	userRepo := auth.NewUserRepository(a.db)
	tokenRepo := auth.NewRefreshTokenRepository(a.db)
	apiKeyRepo := auth.NewAPIKeyRepository(a.db)
	systemAPIKeyRepo := auth.NewSystemAPIKeyRepository(a.db)

	// Create state store (Redis-based for OAuth state)
	var stateStore auth.StateStore
	if a.redis != nil {
		stateStore = auth.NewRedisStateStore(a.redis)
	} else {
		stateStore = auth.NewInMemoryStateStore()
	}

	// Create auth service
	authService, err := auth.NewService(
		userRepo,
		tokenRepo,
		apiKeyRepo,
		systemAPIKeyRepo,
		nil, // OAuth registry (optional, user module handles OAuth)
		stateStore,
		&auth.ServiceConfig{
			JWTConfig: &auth.JWTConfig{
				Secret:             a.config.Auth.JWTSecret,
				AccessTokenExpiry:  a.config.Auth.AccessTokenExpiry,
				RefreshTokenExpiry: a.config.Auth.RefreshTokenExpiry,
			},
			MasterKey:         a.config.Auth.MasterKey,
			MaxAPIKeysPerUser: 10,
		},
	)
	if err != nil {
		return fmt.Errorf("create auth service: %w", err)
	}
	a.authService = authService

	// Create rate limiter (only if Redis is available)
	if a.redis != nil {
		a.rateLimiter = auth.NewRateLimiter(a.redis)
	}

	// Create handler
	a.authHandler = auth.NewHandler(authService)

	return nil
}

// initUserModule initializes the user module.
func (a *App) initUserModule() error {
	// Create email sender
	var emailSender user.EmailSender
	if a.config.Email.Provider == "smtp" {
		smtpConfig := &user.SMTPConfig{
			Host:        a.config.Email.SMTP.Host,
			Port:        a.config.Email.SMTP.Port,
			User:        a.config.Email.SMTP.User,
			Password:    a.config.Email.SMTP.Password,
			FromAddress: a.config.Email.FromAddress,
			FromName:    a.config.Email.FromName,
			BaseURL:     a.config.Email.BaseURL,
		}
		emailSender = user.NewSMTPEmailSender(smtpConfig, a.zapLogger)
	} else {
		emailSender = user.NewNoOpEmailSender(a.zapLogger)
	}

	// Create repositories
	userRepo := user.NewRepository(a.db)
	tokenRepo := auth.NewRefreshTokenRepository(a.db)

	// Create JWT manager
	jwtManager := auth.NewJWTManager(&auth.JWTConfig{
		Secret:             a.config.Auth.JWTSecret,
		AccessTokenExpiry:  a.config.Auth.AccessTokenExpiry,
		RefreshTokenExpiry: a.config.Auth.RefreshTokenExpiry,
	})

	// Create user service
	userService := user.NewService(
		userRepo,
		tokenRepo,
		jwtManager,
		emailSender,
		a.zapLogger,
	)

	// Create handlers
	a.userHandler = user.NewHandler(userService)
	a.userAdmin = user.NewAdminHandler(userService)

	return nil
}

// initBillingModule initializes the billing module.
func (a *App) initBillingModule() error {
	// Create billing repository
	a.billingRepo = billing.NewRepository(a.db)

	// Create quota manager (only if Redis is available)
	var quotaManager *billingquota.Manager
	if redisClient, ok := a.redis.(*redis.Client); ok && redisClient != nil {
		quotaManager = billingquota.NewManager(redisClient, a.zapLogger)
	}

	// Create billing service
	a.billingService = billing.NewService(
		a.billingRepo,
		quotaManager,
		a.zapLogger,
	)

	// Create quota checker middleware
	a.quotaChecker = billingquota.NewChecker(a.billingService, a.zapLogger)

	// Create usage recorder
	a.usageRecorder = billingusage.NewRecorder(a.billingRepo, a.zapLogger, 1000)

	// Create handler
	a.billingHandler = billing.NewHandler(a.billingService)

	return nil
}

// initOrderModule initializes the order module.
func (a *App) initOrderModule() error {
	// Create order repository
	orderRepo := order.NewRepository(a.db)

	// Create order service (needs billing.Repository)
	a.orderService = order.NewService(
		orderRepo,
		a.billingRepo,
		a.zapLogger,
	)

	// Create handler
	a.orderHandler = order.NewHandler(a.orderService)

	return nil
}

// initPaymentModule initializes the payment module.
func (a *App) initPaymentModule() error {
	// Create provider registry
	providerRegistry := payment.NewProviderRegistry()

	// Create and register Stripe provider
	if a.config.Stripe.SecretKey != "" {
		stripeProvider := paymentprovider.NewStripeProvider(&paymentprovider.StripeConfig{
			APIKey:        a.config.Stripe.SecretKey,
			WebhookSecret: a.config.Stripe.WebhookSecret,
		})
		providerRegistry.Register(stripeProvider)
	}

	// Create and register Alipay provider
	if a.config.Alipay.AppID != "" && a.config.Alipay.PrivateKey != "" {
		alipayProvider, err := paymentprovider.NewAlipayProvider(&paymentprovider.AlipayConfig{
			AppID:           a.config.Alipay.AppID,
			PrivateKey:      a.config.Alipay.PrivateKey,
			AlipayPublicKey: a.config.Alipay.AlipayPublicKey,
			IsProd:          a.config.Alipay.IsProd,
			NotifyURL:       a.config.Alipay.NotifyURL,
			ReturnURL:       a.config.Alipay.ReturnURL,
		})
		if err != nil {
			return fmt.Errorf("create alipay provider: %w", err)
		}
		providerRegistry.Register(alipayProvider)
	}

	// Create and register WeChat provider
	if a.config.Wechat.AppID != "" && a.config.Wechat.MchID != "" {
		wechatProvider, err := paymentprovider.NewWechatProvider(&paymentprovider.WechatConfig{
			AppID:                 a.config.Wechat.AppID,
			MchID:                 a.config.Wechat.MchID,
			APIKeyV3:              a.config.Wechat.APIKeyV3,
			SerialNo:              a.config.Wechat.SerialNo,
			PrivateKey:            a.config.Wechat.PrivateKey,
			WechatPublicKeySerial: a.config.Wechat.WechatPublicKeySerial,
			WechatPublicKey:       a.config.Wechat.WechatPublicKey,
			IsProd:                a.config.Wechat.IsProd,
			NotifyURL:             a.config.Wechat.NotifyURL,
		})
		if err != nil {
			return fmt.Errorf("create wechat provider: %w", err)
		}
		providerRegistry.Register(wechatProvider)
	}

	// Create payment repository
	paymentRepo := payment.NewRepository(a.db)

	// Determine notify base URL
	notifyBaseURL := a.config.Server.Address
	if a.config.Email.BaseURL != "" {
		notifyBaseURL = a.config.Email.BaseURL
	}

	// Create payment service
	a.paymentService = payment.NewService(
		paymentRepo,
		a.orderService,
		a.billingService,
		providerRegistry,
		notifyBaseURL,
		a.zapLogger,
	)

	// Create handlers
	a.paymentHandler = payment.NewHandler(a.paymentService)
	a.webhookHandler = payment.NewWebhookHandler(
		a.paymentService,
		a.billingService,
		a.zapLogger,
	)

	return nil
}

// initGitModule initializes the git module.
func (a *App) initGitModule() error {
	// Skip if storage is not configured
	if a.config.Storage.Endpoint == "" || a.config.Storage.Bucket == "" {
		fmt.Println("Warning: Git module disabled - storage not configured")
		return nil
	}

	// Create R2 client
	r2Client, err := gitstorage.NewR2Client(&gitstorage.R2Config{
		Endpoint:        a.config.Storage.Endpoint,
		Region:          a.config.Storage.Region,
		AccessKeyID:     a.config.Storage.AccessKeyID,
		SecretAccessKey: a.config.Storage.SecretAccessKey,
		Bucket:          a.config.Storage.Bucket,
	})
	if err != nil {
		return fmt.Errorf("create R2 client: %w", err)
	}
	a.r2Client = r2Client

	// Create git repository
	gitRepo := git.NewRepository(a.db)

	// Create git service
	a.gitService = git.NewService(
		gitRepo,
		r2Client,
		nil, // No quota checker for now
		&a.config.Git,
		a.zapLogger,
	)

	// Determine base URL for clone URLs
	baseURL := a.config.Email.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost" + a.config.Server.Address
	}

	// Create REST API handler
	a.gitHandler = git.NewHandler(a.gitService, baseURL)

	// Create Git HTTP handler
	a.gitHTTPHandler = git.NewGitHandler(
		a.gitService,
		r2Client,
		nil, // No authenticator for now
		a.zapLogger,
	)

	// Create LFS storage manager
	lfsStorage := lfs.NewStorageManager(r2Client, &a.config.Git)

	// Create LFS batch handler with a simple resolver
	lfsResolver := &gitLFSResolver{
		service: a.gitService,
		repo:    gitRepo,
	}
	a.lfsHandler = lfs.NewBatchHandler(
		lfsStorage,
		lfsResolver,
		baseURL,
		a.zapLogger,
	)

	return nil
}

// initCollaborationModule initializes the collaboration module.
func (a *App) initCollaborationModule() error {
	// Create repositories
	collabRepo := collaboration.NewRepository(a.db)
	userRepo := collaboration.NewUserRepository(a.db)

	// Create service
	collabService := collaboration.NewService(
		collabRepo,
		userRepo,
		a.zapLogger,
	)

	// Determine base URL
	baseURL := a.config.Email.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost" + a.config.Server.Address
	}

	// Create handler
	a.collabHandler = collaboration.NewHandler(collabService, baseURL)

	return nil
}

// gitLFSResolver implements lfs.RepoResolver interface.
type gitLFSResolver struct {
	service *git.Service
	repo    git.Repository
}

func (r *gitLFSResolver) GetRepoByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (repoID uuid.UUID, lfsEnabled bool, ownerUserID uuid.UUID, err error) {
	repo, err := r.service.GetRepoByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return uuid.Nil, false, uuid.Nil, err
	}
	return repo.ID, repo.LFSEnabled, repo.OwnerID, nil
}

func (r *gitLFSResolver) CanAccess(ctx context.Context, repoID uuid.UUID, userID *uuid.UUID, write bool) (bool, error) {
	perm := git.PermissionRead
	if write {
		perm = git.PermissionWrite
	}
	return r.service.CanAccess(ctx, repoID, userID, perm)
}

func (r *gitLFSResolver) LinkLFSObject(ctx context.Context, repoID uuid.UUID, oid string, size int64, storageKey string) error {
	// Create LFS object record if not exists
	lfsObj := &git.LFSObject{
		OID:        oid,
		Size:       size,
		StorageKey: storageKey,
	}
	if err := r.repo.CreateLFSObject(ctx, lfsObj); err != nil {
		// Ignore duplicate key errors
	}
	// Link to repo
	return r.repo.LinkLFSObject(ctx, repoID, oid)
}

func (r *gitLFSResolver) GetLFSObject(ctx context.Context, oid string) (size int64, exists bool, err error) {
	obj, err := r.repo.GetLFSObject(ctx, oid)
	if err != nil {
		if err == git.ErrLFSObjectNotFound {
			return 0, false, nil
		}
		return 0, false, err
	}
	return obj.Size, true, nil
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

	// Webhook routes (no auth required, uses signature verification)
	webhookRouter := a.router.Group("/webhooks")

	// Register AI module routes
	a.aiModule.RegisterRoutes(publicRouter, adminRouter)

	// Register auth module routes (for System API Key management)
	if a.authHandler != nil {
		a.authHandler.RegisterRoutes(publicRouter)
	}

	// Register new module routes
	a.userHandler.RegisterRoutes(publicRouter)
	a.userAdmin.RegisterRoutes(adminRouter)
	a.billingHandler.RegisterRoutes(publicRouter)
	a.orderHandler.RegisterRoutes(publicRouter)
	a.paymentHandler.RegisterRoutes(publicRouter)
	a.webhookHandler.RegisterRoutes(webhookRouter)

	// Register git module routes (if initialized)
	if a.gitHandler != nil {
		a.gitHandler.RegisterRoutes(publicRouter)
	}
	if a.gitHTTPHandler != nil {
		a.gitHTTPHandler.RegisterGitRoutes(v1)
	}
	if a.lfsHandler != nil {
		a.lfsHandler.RegisterRoutes(v1)
	}

	// Register collaboration module routes
	if a.collabHandler != nil {
		a.collabHandler.RegisterRoutes(publicRouter)
	}
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

	// Close usage recorder
	if a.usageRecorder != nil {
		a.usageRecorder.Close()
	}

	// Sync zap logger
	if a.zapLogger != nil {
		_ = a.zapLogger.Sync()
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
