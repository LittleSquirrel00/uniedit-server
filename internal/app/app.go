package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	// Domains
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"

	// Inbound adapters (HTTP handlers)
	"github.com/uniedit/server/internal/adapter/inbound/http/aiproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/authproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/billingproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/collaborationproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/gitproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/mediaproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/orderproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/paymentproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/pingproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/userproto"

	// Inbound ports
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"

	// Infrastructure
	"github.com/uniedit/server/internal/infra/config"
	"github.com/uniedit/server/internal/infra/database"

	// Utils
	"github.com/uniedit/server/internal/utils/billingflow"
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
	billingDomain       inbound.BillingDomain
	orderDomain         order.OrderDomain
	paymentDomain       payment.PaymentDomain
	aiDomain            ai.AIDomain
	gitDomain           inbound.GitDomain
	collaborationDomain inbound.CollaborationDomain
	mediaDomain         inbound.MediaDomain

	// Proto-defined HTTP handlers (from ./api/protobuf_spec)
	pingProtoHandler  *pingproto.Handler
	authProtoHandler  *authproto.Handler
	userProtoHandler  *userproto.Handler
	orderProtoHandler *orderproto.Handler
	billingProtoHandler *billingproto.Handler
	aiProtoHandler      *aiproto.Handler
	collaborationProtoHandler *collaborationproto.Handler
	paymentProtoHandler *paymentproto.Handler
	gitProtoHandler     *gitproto.Handler
	mediaProtoHandler   *mediaproto.Handler

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
		pingProtoHandler:       deps.PingProtoHandler,
		authProtoHandler:       deps.AuthProtoHandler,
		userProtoHandler:       deps.UserProtoHandler,
		orderProtoHandler:      deps.OrderProtoHandler,
		billingProtoHandler:    deps.BillingProtoHandler,
		aiProtoHandler:         deps.AIProtoHandler,
		collaborationProtoHandler: deps.CollaborationProtoHandler,
		paymentProtoHandler:    deps.PaymentProtoHandler,
		gitProtoHandler:        deps.GitProtoHandler,
		mediaProtoHandler:      deps.MediaProtoHandler,
		cleanupFuncs:           []func(){cleanup},
	}

	if biller, ok := deps.BillingDomain.(billingflow.UsageBiller); ok {
		if setter, ok := deps.AIDomain.(interface{ SetUsageBiller(billingflow.UsageBiller) }); ok {
			setter.SetUsageBiller(biller)
		}
		if setter, ok := deps.MediaDomain.(interface{ SetUsageBiller(billingflow.UsageBiller) }); ok {
			setter.SetUsageBiller(biller)
		}
	}
	if setter, ok := deps.MediaDomain.(interface{ SetPricing(imageUSDPerCredit, videoUSDPerMinute float64) }); ok {
		setter.SetPricing(deps.Config.Media.ImageUSDPerCredit, deps.Config.Media.VideoUSDPerMinute)
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

	return r
}

// registerRoutes registers all HTTP routes.
func (a *App) registerRoutes() {
	// Create JWT validator adapter for auth middleware
	jwtValidator := middleware.NewAuthDomainValidator(a.authDomain.ValidateAccessToken)
	authMiddleware := middleware.RequireAuth(jwtValidator)

	roleAuthorizer := middleware.NewSystemRoleAuthorizer(
		a.config.AccessControl.AdminEmails,
		a.config.AccessControl.SREEmails,
		a.config.AccessControl.AdminUserIDs,
		a.config.AccessControl.SREUserIDs,
		middleware.WithUserAdminChecker(func(ctx context.Context, userID uuid.UUID) (bool, error) {
			u, err := a.userDomain.GetUser(ctx, userID)
			if err != nil {
				// Not found means no privilege; other errors should fail closed.
				if errors.Is(err, user.ErrUserNotFound) {
					return false, nil
				}
				return false, err
			}
			if u == nil {
				return false, nil
			}
			return u.IsAdmin, nil
		}),
	)

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

	// ===== Protected Routes (requires auth) =====
	protectedRouter := v1.Group("")
	protectedRouter.Use(authMiddleware)

	// ===== Admin Routes (requires admin auth) =====
	adminRouter := protectedRouter.Group("")
	adminRouter.Use(middleware.RequireAdminOrSRE(roleAuthorizer))

	// Proto-defined routes (from ./api/protobuf_spec)
	a.registerProtoRoutes(v1, protectedRouter, adminRouter)
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
	subscriptionDB outbound.SubscriptionDatabasePort
	logger         *zap.Logger
}

func newBillingReaderAdapter(subscriptionDB outbound.SubscriptionDatabasePort, logger *zap.Logger) outbound.BillingReaderPort {
	return &billingReaderAdapter{subscriptionDB: subscriptionDB, logger: logger}
}

func (a *billingReaderAdapter) GetSubscription(ctx context.Context, userID uuid.UUID) (*outbound.PaymentSubscriptionInfo, error) {
	sub, err := a.subscriptionDB.GetByUserID(ctx, userID)
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
	if amount <= 0 {
		return fmt.Errorf("invalid credits amount: %d", amount)
	}
	sub, err := a.subscriptionDB.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}
	if err := a.subscriptionDB.UpdateCredits(ctx, userID, amount); err != nil {
		return fmt.Errorf("update credits: %w", err)
	}
	if a.logger != nil {
		a.logger.Info("credits added",
			zap.String("user_id", userID.String()),
			zap.Int64("amount", amount),
			zap.String("source", source),
		)
	}
	return nil
}

// noOpEventPublisher is a no-op implementation of outbound.EventPublisherPort.
type noOpEventPublisher struct{}

func newNoOpEventPublisher() outbound.EventPublisherPort {
	return &noOpEventPublisher{}
}

func (p *noOpEventPublisher) Publish(ctx context.Context, event interface{}) error {
	return nil
}
