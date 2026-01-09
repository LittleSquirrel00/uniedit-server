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
	config    *config.Config
	db        *gorm.DB
	redis     goredis.UniversalClient
	router    *gin.Engine
	logger    *logger.Logger
	zapLogger *zap.Logger
	metrics   *metrics.Metrics

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
		userDomain:          deps.UserDomain,
		authDomain:          deps.AuthDomain,
		billingDomain:       deps.BillingDomain,
		orderDomain:         deps.OrderDomain,
		paymentDomain:       deps.PaymentDomain,
		aiDomain:            deps.AIDomain,
		gitDomain:           deps.GitDomain,
		collaborationDomain: deps.CollaborationDomain,
		mediaDomain:         deps.MediaDomain,
		aiChatHandler:       deps.AIChatHandler,
		cleanupFuncs:        []func(){cleanup},
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
