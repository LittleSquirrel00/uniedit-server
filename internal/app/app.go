package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	goredis "github.com/redis/go-redis/v9"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	// Domains
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"

	// Inbound adapters (HTTP handlers)
	aihttp "github.com/uniedit/server/internal/adapter/inbound/http/ai"
	authhttp "github.com/uniedit/server/internal/adapter/inbound/http/auth"
	billinghttp "github.com/uniedit/server/internal/adapter/inbound/http/billing"
	collaborationhttp "github.com/uniedit/server/internal/adapter/inbound/http/collaboration"
	githttp "github.com/uniedit/server/internal/adapter/inbound/http/git"
	mediahttp "github.com/uniedit/server/internal/adapter/inbound/http/media"
	orderhttp "github.com/uniedit/server/internal/adapter/inbound/http/order"
	paymenthttp "github.com/uniedit/server/internal/adapter/inbound/http/payment"
	userhttp "github.com/uniedit/server/internal/adapter/inbound/http/user"

	// Inbound ports
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"

	// Infrastructure
	_ "github.com/uniedit/server/cmd/server/docs" // swagger docs
	"github.com/uniedit/server/internal/infra/config"
	"github.com/uniedit/server/internal/infra/database"

	// Utils
	"github.com/uniedit/server/internal/utils/logger"
	"github.com/uniedit/server/internal/utils/metrics"
	"github.com/uniedit/server/internal/utils/middleware"
)

// Application is the interface for the application.
type Application interface {
	Router() *gin.Engine
	Stop()
}

// App represents the application using hexagonal architecture.
type App struct {
	config      *config.Config
	db          *gorm.DB
	redis       goredis.UniversalClient
	router      *gin.Engine
	logger      *logger.Logger
	zapLogger   *zap.Logger
	metrics     *metrics.Metrics
	rateLimiter outbound.RateLimiterPort

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

	// AI HTTP handlers
	aiChatHandler          *aihttp.ChatHandler
	aiProviderAdminHandler *aihttp.ProviderAdminHandler
	aiModelAdminHandler    *aihttp.ModelAdminHandler
	aiPublicHandler        *aihttp.PublicHandler

	// Auth HTTP handlers
	oauthHandler        *authhttp.OAuthHandler
	apiKeyHandler       *authhttp.APIKeyHandler
	systemAPIKeyHandler *authhttp.SystemAPIKeyHandler

	// User HTTP handlers
	profileHandler      *userhttp.ProfileHandler
	registrationHandler *userhttp.RegistrationHandler
	userAdminHandler    *userhttp.AdminHandler

	// Billing HTTP handlers
	subscriptionHandler *billinghttp.SubscriptionHandler
	quotaHandler        *billinghttp.QuotaHandler
	creditsHandler      *billinghttp.CreditsHandler
	usageHandler        *billinghttp.UsageHandler

	// Order HTTP handlers
	orderHandler   *orderhttp.OrderHandler
	invoiceHandler *orderhttp.InvoiceHandler

	// Payment HTTP handlers
	paymentHandler *paymenthttp.PaymentHandler
	refundHandler  *paymenthttp.RefundHandler
	webhookHandler *paymenthttp.WebhookHandler

	// Git HTTP handlers
	gitHandler *githttp.Handler

	// Collaboration HTTP handlers
	collaborationHandler *collaborationhttp.Handler

	// Media HTTP handlers
	mediaHandler *mediahttp.Handler

	// Cleanup functions
	cleanupFuncs []func()
}

// New creates a new application instance using Wire for dependency injection.
func New(cfg *config.Config) (*App, error) {
	// Use Wire to initialize all dependencies
	deps, cleanup, err := InitializeDependencies(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize dependencies: %w", err)
	}

	app := &App{
		config:              deps.Config,
		db:                  deps.DB,
		redis:               deps.Redis,
		logger:              deps.Logger,
		zapLogger:           deps.ZapLogger,
		metrics:             deps.Metrics,
		rateLimiter:         deps.RateLimiter,
		userDomain:          deps.UserDomain,
		authDomain:          deps.AuthDomain,
		billingDomain:       deps.BillingDomain,
		orderDomain:         deps.OrderDomain,
		paymentDomain:       deps.PaymentDomain,
		aiDomain:            deps.AIDomain,
		gitDomain:           deps.GitDomain,
		collaborationDomain: deps.CollaborationDomain,
		mediaDomain:         deps.MediaDomain,
		// AI HTTP handlers
		aiChatHandler:          deps.AIChatHandler,
		aiProviderAdminHandler: deps.AIProviderAdminHandler,
		aiModelAdminHandler:    deps.AIModelAdminHandler,
		aiPublicHandler:        deps.AIPublicHandler,
		// Auth HTTP handlers
		oauthHandler:        deps.OAuthHandler,
		apiKeyHandler:       deps.APIKeyHandler,
		systemAPIKeyHandler: deps.SystemAPIKeyHandler,
		profileHandler:      deps.ProfileHandler,
		registrationHandler: deps.RegistrationHandler,
		userAdminHandler:    deps.UserAdminHandler,
		subscriptionHandler: deps.SubscriptionHandler,
		quotaHandler:        deps.QuotaHandler,
		creditsHandler:      deps.CreditsHandler,
		usageHandler:        deps.UsageHandler,
		orderHandler:        deps.OrderHandler,
		invoiceHandler:      deps.InvoiceHandler,
		paymentHandler:       deps.PaymentHandler,
		refundHandler:        deps.RefundHandler,
		webhookHandler:       deps.WebhookHandler,
		gitHandler:           deps.GitHandler,
		collaborationHandler: deps.CollaborationHandler,
		mediaHandler:         deps.MediaHandler,
		cleanupFuncs:         []func(){cleanup},
	}

	// Initialize router
	app.router = app.setupRouter()

	// Start health monitoring for AI domain
	ctx := context.Background()
	app.aiDomain.StartHealthMonitor(ctx)
	app.cleanupFuncs = append(app.cleanupFuncs, func() {
		app.aiDomain.StopHealthMonitor()
	})

	// Register routes
	app.registerRoutes()

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
	r.Use(middleware.Metrics(a.metrics))
	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Apply global rate limiting (if enabled)
	if a.config.RateLimit.Enabled && a.rateLimiter != nil {
		r.Use(middleware.RateLimitByIP(
			a.rateLimiter,
			a.config.RateLimit.GlobalLimit,
			a.config.RateLimit.GlobalWindow,
		))
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "v2"})
	})

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger documentation endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return r
}

// registerRoutes registers all HTTP routes.
func (a *App) registerRoutes() {
	// Create JWT validator adapter for auth middleware
	jwtValidator := middleware.NewAuthDomainValidator(a.authDomain.ValidateAccessToken)
	authMiddleware := middleware.RequireAuth(jwtValidator)

	// API v1 group
	v1 := a.router.Group("/api/v1")

	// Apply API-level rate limiting (per user/IP)
	if a.config.RateLimit.Enabled && a.rateLimiter != nil {
		v1.Use(middleware.RateLimitByUser(
			a.rateLimiter,
			a.config.RateLimit.APILimit,
			a.config.RateLimit.APIWindow,
		))
	}

	// Apply idempotency middleware for mutation requests
	if a.redis != nil {
		v1.Use(middleware.Idempotency(a.redis, middleware.IdempotencyConfig{
			TTL:     a.config.RateLimit.IdempotencyTTL,
			Methods: []string{"POST", "PUT", "PATCH"},
		}))
	}

	// ===== Public Routes (no auth required) =====

	// Auth routes (OAuth, refresh, logout)
	if a.oauthHandler != nil {
		a.oauthHandler.RegisterRoutes(v1)
	}

	// Registration routes
	if a.registrationHandler != nil {
		a.registrationHandler.RegisterRoutes(v1)
	}

	// Billing plans (read-only, public)
	if a.subscriptionHandler != nil {
		a.subscriptionHandler.RegisterRoutes(v1)
	}

	// Payment webhooks (no auth, verified by signature)
	if a.webhookHandler != nil {
		a.webhookHandler.RegisterRoutes(v1)
	}

	// ===== Protected Routes (requires auth) =====
	protectedRouter := v1.Group("")
	protectedRouter.Use(authMiddleware)

	// AI routes
	if a.aiChatHandler != nil {
		aiGroup := protectedRouter.Group("/ai")
		{
			aiGroup.POST("/chat", a.aiChatHandler.Chat)
			aiGroup.POST("/chat/stream", a.aiChatHandler.ChatStream)
		}
	}

	// AI public routes (list models)
	if a.aiPublicHandler != nil {
		aiPublicGroup := protectedRouter.Group("/ai")
		{
			aiPublicGroup.GET("/models", a.aiPublicHandler.ListModels)
			aiPublicGroup.GET("/models/:id", a.aiPublicHandler.GetModel)
		}
	}

	// User profile routes
	if a.profileHandler != nil {
		a.profileHandler.RegisterRoutes(protectedRouter)
	}

	// API key routes
	if a.apiKeyHandler != nil {
		a.apiKeyHandler.RegisterRoutes(protectedRouter)
	}

	// Billing routes (subscription, quota, credits, usage)
	if a.quotaHandler != nil {
		a.quotaHandler.RegisterRoutes(protectedRouter)
	}
	if a.creditsHandler != nil {
		a.creditsHandler.RegisterRoutes(protectedRouter)
	}
	if a.usageHandler != nil {
		a.usageHandler.RegisterRoutes(protectedRouter)
	}

	// Order routes
	if a.orderHandler != nil {
		a.orderHandler.RegisterRoutes(protectedRouter)
	}

	// Invoice routes
	if a.invoiceHandler != nil {
		a.invoiceHandler.RegisterRoutes(protectedRouter)
	}

	// Payment routes
	if a.paymentHandler != nil {
		a.paymentHandler.RegisterRoutes(protectedRouter)
	}

	// Git routes
	if a.gitHandler != nil {
		a.gitHandler.RegisterRoutes(v1, authMiddleware)
	}

	// Collaboration routes
	if a.collaborationHandler != nil {
		a.collaborationHandler.RegisterRoutes(v1, authMiddleware)
	}

	// Media routes
	if a.mediaHandler != nil {
		a.mediaHandler.RegisterRoutes(v1, authMiddleware)
	}

	// ===== Admin Routes (requires admin auth) =====
	// TODO: Add admin middleware when available
	adminRouter := protectedRouter.Group("")
	// adminRouter.Use(middleware.RequireAdmin())

	// User admin routes
	if a.userAdminHandler != nil {
		a.userAdminHandler.RegisterRoutes(adminRouter)
	}

	// System API key routes (admin only)
	if a.systemAPIKeyHandler != nil {
		a.systemAPIKeyHandler.RegisterRoutes(adminRouter)
	}

	// Credits admin routes (add credits)
	if a.creditsHandler != nil {
		a.creditsHandler.RegisterAdminRoutes(adminRouter)
	}

	// Refund routes (admin only)
	if a.refundHandler != nil {
		a.refundHandler.RegisterRoutes(adminRouter)
	}

	// AI provider admin routes
	if a.aiProviderAdminHandler != nil {
		aiAdminGroup := adminRouter.Group("/admin/ai")
		{
			aiAdminGroup.GET("/providers", a.aiProviderAdminHandler.ListProviders)
			aiAdminGroup.POST("/providers", a.aiProviderAdminHandler.CreateProvider)
			aiAdminGroup.GET("/providers/:id", a.aiProviderAdminHandler.GetProvider)
			aiAdminGroup.PUT("/providers/:id", a.aiProviderAdminHandler.UpdateProvider)
			aiAdminGroup.DELETE("/providers/:id", a.aiProviderAdminHandler.DeleteProvider)
			aiAdminGroup.POST("/providers/:id/sync", a.aiProviderAdminHandler.SyncModels)
			aiAdminGroup.POST("/providers/:id/health", a.aiProviderAdminHandler.HealthCheck)
		}
	}

	// AI model admin routes
	if a.aiModelAdminHandler != nil {
		aiAdminGroup := adminRouter.Group("/admin/ai")
		{
			aiAdminGroup.GET("/models", a.aiModelAdminHandler.ListModels)
			aiAdminGroup.POST("/models", a.aiModelAdminHandler.CreateModel)
			aiAdminGroup.GET("/models/:id", a.aiModelAdminHandler.GetModel)
			aiAdminGroup.PUT("/models/:id", a.aiModelAdminHandler.UpdateModel)
			aiAdminGroup.DELETE("/models/:id", a.aiModelAdminHandler.DeleteModel)
		}
	}
}

// Router returns the HTTP router.
func (a *App) Router() *gin.Engine {
	return a.router
}

// Stop stops the application and releases resources.
func (a *App) Stop() {
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

// ===== Cross-Domain Adapter Implementations =====

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
