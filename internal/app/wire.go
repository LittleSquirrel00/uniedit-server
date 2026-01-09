//go:build wireinject
// +build wireinject

package app

import (
	"net/http"

	"github.com/google/wire"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	// Domains
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"

	// Inbound adapters
	aihttp "github.com/uniedit/server/internal/adapter/inbound/http/ai"
	authhttp "github.com/uniedit/server/internal/adapter/inbound/http/auth"
	billinghttp "github.com/uniedit/server/internal/adapter/inbound/http/billing"
	orderhttp "github.com/uniedit/server/internal/adapter/inbound/http/order"
	paymenthttp "github.com/uniedit/server/internal/adapter/inbound/http/payment"
	userhttp "github.com/uniedit/server/internal/adapter/inbound/http/user"

	// Ports
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"

	// Infrastructure
	"github.com/uniedit/server/internal/infra/config"

	// Utils
	"github.com/uniedit/server/internal/utils/logger"
	"github.com/uniedit/server/internal/utils/metrics"
)

// Dependencies holds all injected dependencies.
type Dependencies struct {
	Config      *config.Config
	DB          *gorm.DB
	Redis       goredis.UniversalClient
	HTTPClient  *http.Client
	RateLimiter outbound.RateLimiterPort
	Logger      *logger.Logger
	ZapLogger   *zap.Logger
	Metrics     *metrics.Metrics

	// Domains
	UserDomain          user.UserDomain
	AuthDomain          auth.AuthDomain
	BillingDomain       billing.BillingDomain
	OrderDomain         order.OrderDomain
	PaymentDomain       payment.PaymentDomain
	AIDomain            ai.AIDomain
	GitDomain           inbound.GitDomain
	CollaborationDomain inbound.CollaborationDomain
	MediaDomain         inbound.MediaDomain

	// HTTP Handlers
	AIChatHandler *aihttp.ChatHandler

	// Auth HTTP Handlers
	OAuthHandler          *authhttp.OAuthHandler
	APIKeyHandler         *authhttp.APIKeyHandler
	SystemAPIKeyHandler   *authhttp.SystemAPIKeyHandler

	// User HTTP Handlers
	ProfileHandler      *userhttp.ProfileHandler
	RegistrationHandler *userhttp.RegistrationHandler
	UserAdminHandler    *userhttp.AdminHandler

	// Billing HTTP Handlers
	SubscriptionHandler *billinghttp.SubscriptionHandler
	QuotaHandler        *billinghttp.QuotaHandler
	CreditsHandler      *billinghttp.CreditsHandler
	UsageHandler        *billinghttp.UsageHandler

	// Order HTTP Handlers
	OrderHandler   *orderhttp.OrderHandler
	InvoiceHandler *orderhttp.InvoiceHandler

	// Payment HTTP Handlers
	PaymentHandler *paymenthttp.PaymentHandler
	RefundHandler  *paymenthttp.RefundHandler
	WebhookHandler *paymenthttp.WebhookHandler
}

// InitializeDependencies creates all dependencies using Wire.
func InitializeDependencies(cfg *config.Config) (*Dependencies, func(), error) {
	wire.Build(
		AppSet,
		wire.Struct(new(Dependencies), "*"),
	)
	return nil, nil, nil
}
