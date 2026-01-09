package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	// Domains
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/domain/git"
	"github.com/uniedit/server/internal/domain/media"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"

	// Inbound adapters (HTTP handlers)
	aihttp "github.com/uniedit/server/internal/adapter/inbound/http/ai"

	// Inbound ports
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"

	// Outbound adapters
	gitdb "github.com/uniedit/server/internal/adapter/database"
	"github.com/uniedit/server/internal/adapter/outbound/oauth"
	"github.com/uniedit/server/internal/adapter/outbound/postgres"
	redisadapter "github.com/uniedit/server/internal/adapter/outbound/redis"
	"github.com/uniedit/server/internal/adapter/outbound/vendor"

	// Shared infrastructure
	_ "github.com/uniedit/server/cmd/server/docs" // swagger docs
	sharedcache "github.com/uniedit/server/internal/shared/cache"
	"github.com/uniedit/server/internal/shared/config"
	"github.com/uniedit/server/internal/shared/database"
	"github.com/uniedit/server/internal/shared/logger"
	"github.com/uniedit/server/internal/shared/middleware"
)

// AppV2 represents the application using new architecture.
type AppV2 struct {
	config    *config.Config
	db        *gorm.DB
	redis     goredis.UniversalClient
	router    *gin.Engine
	logger    *logger.Logger
	zapLogger *zap.Logger

	// Domain services
	userDomain          user.UserDomain
	authDomain          auth.AuthDomain
	billingDomain       billing.BillingDomain
	orderDomain         order.OrderDomain
	paymentDomain       payment.PaymentDomain
	aiDomain            ai.AIDomain
	gitDomain           inbound.GitDomain
	collaborationDomain inbound.CollaborationDomain
	mediaDomain         inbound.MediaDomain

	// HTTP handlers (inbound adapters)
	aiChatHandler *aihttp.ChatHandler

	// Cleanup functions
	cleanupFuncs []func()
}

// NewV2 creates a new application instance using hexagonal architecture.
func NewV2(cfg *config.Config) (*AppV2, error) {
	// Initialize logger
	log := logger.New(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})

	zapLog, err := logger.NewZapLogger(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("init zap logger: %w", err)
	}

	app := &AppV2{
		config:       cfg,
		logger:       log,
		zapLogger:    zapLog,
		cleanupFuncs: make([]func(), 0),
	}

	// Initialize infrastructure
	if err := app.initInfrastructure(); err != nil {
		return nil, fmt.Errorf("init infrastructure: %w", err)
	}

	// Initialize router
	app.router = app.setupRouter()

	// Initialize domains with adapters
	if err := app.initDomains(); err != nil {
		return nil, fmt.Errorf("init domains: %w", err)
	}

	// Register routes
	app.registerRoutes()

	return app, nil
}

// initInfrastructure initializes database and cache connections.
func (a *AppV2) initInfrastructure() error {
	// Initialize database
	db, err := database.New(&a.config.Database)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	a.db = db

	// Initialize Redis (optional)
	if a.config.Redis.Address != "" {
		redisClient, err := sharedcache.NewRedisClient(&a.config.Redis)
		if err != nil {
			a.zapLogger.Warn("Redis connection failed, continuing without cache", zap.Error(err))
		} else {
			a.redis = redisClient
		}
	}

	return nil
}

// getRedisClient returns a *redis.Client from UniversalClient if possible.
func (a *AppV2) getRedisClient() *goredis.Client {
	if a.redis == nil {
		return nil
	}
	if client, ok := a.redis.(*goredis.Client); ok {
		return client
	}
	return nil
}

// initDomains initializes all domain services with their adapters.
func (a *AppV2) initDomains() error {
	// Initialize domains in dependency order
	if err := a.initUserDomain(); err != nil {
		return fmt.Errorf("init user domain: %w", err)
	}

	if err := a.initAuthDomain(); err != nil {
		return fmt.Errorf("init auth domain: %w", err)
	}

	if err := a.initBillingDomain(); err != nil {
		return fmt.Errorf("init billing domain: %w", err)
	}

	if err := a.initOrderDomain(); err != nil {
		return fmt.Errorf("init order domain: %w", err)
	}

	if err := a.initPaymentDomain(); err != nil {
		return fmt.Errorf("init payment domain: %w", err)
	}

	if err := a.initAIDomain(); err != nil {
		return fmt.Errorf("init ai domain: %w", err)
	}

	if err := a.initGitDomain(); err != nil {
		return fmt.Errorf("init git domain: %w", err)
	}

	if err := a.initCollaborationDomain(); err != nil {
		return fmt.Errorf("init collaboration domain: %w", err)
	}

	if err := a.initMediaDomain(); err != nil {
		return fmt.Errorf("init media domain: %w", err)
	}

	return nil
}

// initUserDomain initializes the user domain with its adapters.
func (a *AppV2) initUserDomain() error {
	// Create outbound adapters
	userDB := postgres.NewUserAdapter(a.db)
	verificationDB := postgres.NewVerificationAdapter(a.db)

	// Create domain
	a.userDomain = user.NewUserDomain(
		userDB,
		verificationDB,
		nil, // profileDB - optional
		nil, // prefsDB - optional
		nil, // avatarStorage - optional
		nil, // emailSender - optional
		a.zapLogger,
	)

	return nil
}

// initAuthDomain initializes the auth domain with its adapters.
func (a *AppV2) initAuthDomain() error {
	// Create outbound adapters
	tokenRepo := postgres.NewRefreshTokenAdapter(a.db)
	userAPIKeyRepo := postgres.NewUserAPIKeyAdapter(a.db)
	systemAPIKeyRepo := postgres.NewSystemAPIKeyAdapter(a.db)

	// Create OAuth registry
	oauthRegistry := oauth.NewRegistry()
	if a.config.Auth.OAuth.GitHub.ClientID != "" {
		oauthRegistry.RegisterGitHub(
			a.config.Auth.OAuth.GitHub.ClientID,
			a.config.Auth.OAuth.GitHub.ClientSecret,
			a.config.Auth.OAuth.GitHub.RedirectURL,
		)
	}
	if a.config.Auth.OAuth.Google.ClientID != "" {
		oauthRegistry.RegisterGoogle(
			a.config.Auth.OAuth.Google.ClientID,
			a.config.Auth.OAuth.Google.ClientSecret,
			a.config.Auth.OAuth.Google.RedirectURL,
		)
	}

	// Create state store
	var stateStore outbound.OAuthStateStorePort
	if redisClient := a.getRedisClient(); redisClient != nil {
		stateStore = redisadapter.NewOAuthStateStore(redisClient)
	} else {
		stateStore = oauth.NewInMemoryStateStore()
	}

	// Create JWT manager
	jwtManager := oauth.NewJWTManager(&oauth.JWTConfig{
		Secret:             a.config.Auth.JWTSecret,
		AccessTokenExpiry:  a.config.Auth.AccessTokenExpiry,
		RefreshTokenExpiry: a.config.Auth.RefreshTokenExpiry,
	})

	// Create crypto adapter
	cryptoAdapter := oauth.NewCryptoAdapter(a.config.Auth.MasterKey)

	// Create domain
	a.authDomain = auth.NewAuthDomain(
		a.userDomain,
		tokenRepo,
		userAPIKeyRepo,
		systemAPIKeyRepo,
		oauthRegistry,
		stateStore,
		jwtManager,
		cryptoAdapter,
		&auth.Config{MaxAPIKeysPerUser: 10},
		a.zapLogger,
	)

	return nil
}

// initBillingDomain initializes the billing domain with its adapters.
func (a *AppV2) initBillingDomain() error {
	// Create outbound adapters
	planDB := postgres.NewPlanAdapter(a.db)
	subscriptionDB := postgres.NewSubscriptionAdapter(a.db)
	usageDB := postgres.NewUsageRecordAdapter(a.db)

	// Create quota cache if Redis available
	var quotaCache outbound.QuotaCachePort
	if redisClient := a.getRedisClient(); redisClient != nil {
		quotaCache = redisadapter.NewQuotaCache(redisClient)
	}

	// Create domain
	a.billingDomain = billing.NewBillingDomain(
		planDB,
		subscriptionDB,
		usageDB,
		quotaCache,
		a.zapLogger,
	)

	return nil
}

// initOrderDomain initializes the order domain with its adapters.
func (a *AppV2) initOrderDomain() error {
	// Create outbound adapters
	orderDB := postgres.NewOrderAdapter(a.db)
	orderItemDB := postgres.NewOrderItemAdapter(a.db)
	invoiceDB := postgres.NewInvoiceAdapter(a.db)
	planDB := postgres.NewPlanAdapter(a.db)

	// Create domain
	a.orderDomain = order.NewOrderDomain(
		orderDB,
		orderItemDB,
		invoiceDB,
		planDB,
		a.zapLogger,
	)

	return nil
}

// initPaymentDomain initializes the payment domain with its adapters.
func (a *AppV2) initPaymentDomain() error {
	// Create outbound adapters
	paymentDB := postgres.NewPaymentAdapter(a.db)
	webhookDB := postgres.NewWebhookEventAdapter(a.db)

	// Create order reader adapter
	orderReader := newOrderReaderAdapter(a.orderDomain)

	// Create billing reader adapter
	billingReader := newBillingReaderAdapter(a.billingDomain)

	// Create event publisher adapter (no-op for now)
	eventPublisher := newNoOpEventPublisher()

	// Determine notify base URL
	notifyBaseURL := a.config.Server.Address
	if a.config.Email.BaseURL != "" {
		notifyBaseURL = a.config.Email.BaseURL
	}

	// Create domain
	a.paymentDomain = payment.NewPaymentDomain(
		paymentDB,
		webhookDB,
		nil, // providerRegistry - will be implemented
		orderReader,
		billingReader,
		eventPublisher,
		notifyBaseURL,
		a.zapLogger,
	)

	return nil
}

// initAIDomain initializes the AI domain with its adapters.
func (a *AppV2) initAIDomain() error {
	// Create outbound adapters
	providerDB := postgres.NewAIProviderAdapter(a.db)
	modelDB := postgres.NewAIModelAdapter(a.db)
	accountDB := postgres.NewAIProviderAccountAdapter(a.db)
	groupDB := postgres.NewAIModelGroupAdapter(a.db)

	// Create cache adapters
	var healthCache outbound.AIProviderHealthCachePort
	var embeddingCache outbound.AIEmbeddingCachePort
	if redisClient := a.getRedisClient(); redisClient != nil {
		healthCache = redisadapter.NewAIProviderHealthCacheAdapter(redisClient)
		embeddingCache = redisadapter.NewAIEmbeddingCacheAdapter(redisClient)
	}

	// Create vendor registry
	vendorRegistry := vendor.NewRegistry()
	vendorRegistry.RegisterDefaults()

	// Create crypto adapter
	cryptoAdapter := vendor.NewCryptoAdapter(a.config.Auth.MasterKey)

	// Create domain
	a.aiDomain = ai.NewAIDomain(
		providerDB,
		modelDB,
		accountDB,
		groupDB,
		healthCache,
		embeddingCache,
		vendorRegistry,
		cryptoAdapter,
		nil, // usageRecorder
		nil, // config
		a.zapLogger,
	)

	// Start health monitoring
	ctx := context.Background()
	a.aiDomain.StartHealthMonitor(ctx)
	a.cleanupFuncs = append(a.cleanupFuncs, func() {
		a.aiDomain.StopHealthMonitor()
	})

	// Create HTTP handlers
	a.aiChatHandler = aihttp.NewChatHandler(a.aiDomain)

	return nil
}

// initGitDomain initializes the Git domain with its adapters.
func (a *AppV2) initGitDomain() error {
	// Create outbound adapters (database)
	repoDB := gitdb.NewGitRepoDatabaseAdapter(a.db)
	collabDB := gitdb.NewGitCollaboratorDatabaseAdapter(a.db)
	prDB := gitdb.NewGitPullRequestDatabaseAdapter(a.db)
	lfsObjDB := gitdb.NewGitLFSObjectDatabaseAdapter(a.db)
	lfsLockDB := gitdb.NewGitLFSLockDatabaseAdapter(a.db)

	// Create storage adapters (requires S3 client - optional for now)
	// TODO: Initialize S3 client from config when available
	var storage outbound.GitStoragePort
	var lfsStorage outbound.GitLFSStoragePort

	// Create Git config
	gitCfg := git.DefaultConfig()
	if a.config.Git.RepoPrefix != "" {
		gitCfg.RepoPrefix = a.config.Git.RepoPrefix
	}

	// Create domain
	a.gitDomain = git.NewDomain(
		repoDB,
		collabDB,
		prDB,
		lfsObjDB,
		lfsLockDB,
		storage,
		lfsStorage,
		nil, // quotaChecker - optional for now
		gitCfg,
		a.zapLogger,
	)

	return nil
}

// initCollaborationDomain initializes the collaboration domain with its adapters.
func (a *AppV2) initCollaborationDomain() error {
	// Create outbound adapters
	teamDB := postgres.NewTeamAdapter(a.db)
	memberDB := postgres.NewTeamMemberAdapter(a.db)
	invitationDB := postgres.NewTeamInvitationAdapter(a.db)
	userLookup := postgres.NewCollaborationUserLookupAdapter(a.db)
	txAdapter := postgres.NewCollaborationTransactionAdapter(a.db)

	// Create config
	collabCfg := collaboration.DefaultConfig()
	if a.config.Email.BaseURL != "" {
		collabCfg.BaseURL = a.config.Email.BaseURL
	}

	// Create domain
	a.collaborationDomain = collaboration.NewDomain(
		teamDB,
		memberDB,
		invitationDB,
		userLookup,
		txAdapter,
		collabCfg,
		a.zapLogger,
	)

	return nil
}

// initMediaDomain initializes the media domain with its adapters.
func (a *AppV2) initMediaDomain() error {
	// Create outbound adapters
	providerDB := postgres.NewMediaProviderDBAdapter(a.db)
	modelDB := postgres.NewMediaModelDBAdapter(a.db)
	taskDB := postgres.NewMediaTaskDBAdapter(a.db)

	// Create health cache (optional)
	var healthCache outbound.MediaProviderHealthCachePort
	if a.redis != nil {
		healthCache = redisadapter.NewMediaHealthCacheAdapter(a.redis)
	}

	// Create crypto adapter
	cryptoAdapter := vendor.NewCryptoAdapter(a.config.Auth.MasterKey)

	// Create vendor registry
	// TODO: Initialize media vendor registry when available

	// Create config
	mediaCfg := media.DefaultConfig()

	// Create domain
	a.mediaDomain = media.NewDomain(
		providerDB,
		modelDB,
		taskDB,
		healthCache,
		nil, // vendorRegistry - TODO: implement when available
		cryptoAdapter,
		mediaCfg,
		a.zapLogger,
	)

	return nil
}

// setupRouter creates and configures the Gin router.
func (a *AppV2) setupRouter() *gin.Engine {
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
		c.JSON(200, gin.H{"status": "ok", "version": "v2"})
	})

	// Swagger documentation endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return r
}

// registerRoutes registers all HTTP routes.
func (a *AppV2) registerRoutes() {
	// API v1 group
	v1 := a.router.Group("/api/v1")

	// Protected routes (requires auth) - will add auth middleware later
	protectedRouter := v1.Group("")

	// Register AI routes
	if a.aiChatHandler != nil {
		aiGroup := protectedRouter.Group("/ai")
		{
			aiGroup.POST("/chat", a.aiChatHandler.Chat)
			aiGroup.POST("/chat/stream", a.aiChatHandler.ChatStream)
		}
	}

	// TODO: Register other domain routes as their handlers are migrated
}

// Router returns the HTTP router.
func (a *AppV2) Router() *gin.Engine {
	return a.router
}

// Stop stops the application and releases resources.
func (a *AppV2) Stop() {
	// Run cleanup functions
	for _, cleanup := range a.cleanupFuncs {
		cleanup()
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

// ===== Adapter Implementations =====

// orderReaderAdapter adapts OrderDomain to outbound.OrderReaderPort.
type orderReaderAdapter struct {
	domain order.OrderDomain
}

func newOrderReaderAdapter(domain order.OrderDomain) outbound.OrderReaderPort {
	return &orderReaderAdapter{domain: domain}
}

func (a *orderReaderAdapter) GetOrder(ctx context.Context, id uuid.UUID) (*outbound.PaymentOrderInfo, error) {
	ord, err := a.domain.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	planID := ""
	if ord.PlanID != nil {
		planID = *ord.PlanID
	}
	return &outbound.PaymentOrderInfo{
		ID:            ord.ID,
		UserID:        ord.UserID,
		Type:          string(ord.Type),
		Status:        string(ord.Status),
		Total:         ord.Total,
		Currency:      ord.Currency,
		CreditsAmount: ord.CreditsAmount,
		PlanID:        planID,
	}, nil
}

func (a *orderReaderAdapter) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*outbound.PaymentOrderInfo, error) {
	ord, err := a.domain.GetOrderByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return nil, err
	}
	planID := ""
	if ord.PlanID != nil {
		planID = *ord.PlanID
	}
	return &outbound.PaymentOrderInfo{
		ID:            ord.ID,
		UserID:        ord.UserID,
		Type:          string(ord.Type),
		Status:        string(ord.Status),
		Total:         ord.Total,
		Currency:      ord.Currency,
		CreditsAmount: ord.CreditsAmount,
		PlanID:        planID,
	}, nil
}

func (a *orderReaderAdapter) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	// Map status string to appropriate domain method
	switch status {
	case "paid":
		return a.domain.MarkAsPaid(ctx, id)
	case "failed":
		return a.domain.MarkAsFailed(ctx, id)
	case "canceled":
		return a.domain.CancelOrder(ctx, id, "")
	case "refunded":
		return a.domain.MarkAsRefunded(ctx, id)
	default:
		return fmt.Errorf("unsupported status: %s", status)
	}
}

func (a *orderReaderAdapter) SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error {
	return a.domain.SetStripePaymentIntentID(ctx, orderID, paymentIntentID)
}

// billingReaderAdapter adapts BillingDomain to outbound.BillingReaderPort.
type billingReaderAdapter struct {
	domain billing.BillingDomain
}

func newBillingReaderAdapter(domain billing.BillingDomain) outbound.BillingReaderPort {
	return &billingReaderAdapter{domain: domain}
}

func (a *billingReaderAdapter) GetSubscription(ctx context.Context, userID uuid.UUID) (*outbound.PaymentSubscriptionInfo, error) {
	sub, err := a.domain.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	return &outbound.PaymentSubscriptionInfo{
		UserID:           sub.UserID,
		PlanID:           sub.PlanID,
		Status:           string(sub.Status),
		StripeCustomerID: sub.StripeCustomerID,
	}, nil
}

func (a *billingReaderAdapter) AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error {
	return a.domain.AddCredits(ctx, userID, amount, source)
}

// noOpEventPublisher is a no-op implementation of outbound.EventPublisherPort.
type noOpEventPublisher struct{}

func newNoOpEventPublisher() outbound.EventPublisherPort {
	return &noOpEventPublisher{}
}

func (p *noOpEventPublisher) Publish(ctx context.Context, event interface{}) error {
	return nil
}
