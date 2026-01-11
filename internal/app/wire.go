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
	"github.com/uniedit/server/internal/adapter/inbound/http/authproto"
	billinghttp "github.com/uniedit/server/internal/adapter/inbound/http/billing"
	collaborationhttp "github.com/uniedit/server/internal/adapter/inbound/http/collaboration"
	githttp "github.com/uniedit/server/internal/adapter/inbound/http/git"
	mediahttp "github.com/uniedit/server/internal/adapter/inbound/http/media"
	"github.com/uniedit/server/internal/adapter/inbound/http/orderproto"
	paymenthttp "github.com/uniedit/server/internal/adapter/inbound/http/payment"
	"github.com/uniedit/server/internal/adapter/inbound/http/pingproto"
	"github.com/uniedit/server/internal/adapter/inbound/http/userproto"

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

	// AI HTTP Handlers
	AIChatHandler          *aihttp.ChatHandler
	AIProviderAdminHandler *aihttp.ProviderAdminHandler
	AIModelAdminHandler    *aihttp.ModelAdminHandler
	AIPublicHandler        *aihttp.PublicHandler

	// Proto-defined HTTP Handlers (generated from google.api.http)
	PingProtoHandler  *pingproto.Handler
	AuthProtoHandler  *authproto.Handler
	UserProtoHandler  *userproto.Handler
	OrderProtoHandler *orderproto.Handler

	// Billing HTTP Handlers
	SubscriptionHandler *billinghttp.SubscriptionHandler
	QuotaHandler        *billinghttp.QuotaHandler
	CreditsHandler      *billinghttp.CreditsHandler
	UsageHandler        *billinghttp.UsageHandler

	// Payment HTTP Handlers
	PaymentHandler *paymenthttp.PaymentHandler
	RefundHandler  *paymenthttp.RefundHandler
	WebhookHandler *paymenthttp.WebhookHandler

	// Git HTTP Handlers
	GitHandler *githttp.Handler

	// Collaboration HTTP Handlers
	CollaborationHandler *collaborationhttp.Handler

	// Media HTTP Handlers
	MediaHandler *mediahttp.Handler
}

// InitializeDependencies creates all dependencies using Wire.
func InitializeDependencies(cfg *config.Config) (*Dependencies, func(), error) {
	wire.Build(
		AppSet,
		wire.Struct(new(Dependencies), "*"),
	)
	return nil, nil, nil
}
