//go:build wireinject
// +build wireinject

package app

import (
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

	// Ports
	"github.com/uniedit/server/internal/port/inbound"

	// Infrastructure
	"github.com/uniedit/server/internal/infra/config"

	// Utils
	"github.com/uniedit/server/internal/utils/logger"
	"github.com/uniedit/server/internal/utils/metrics"
)

// Dependencies holds all injected dependencies.
type Dependencies struct {
	Config    *config.Config
	DB        *gorm.DB
	Redis     goredis.UniversalClient
	Logger    *logger.Logger
	ZapLogger *zap.Logger
	Metrics   *metrics.Metrics

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
}

// InitializeDependencies creates all dependencies using Wire.
func InitializeDependencies(cfg *config.Config) (*Dependencies, func(), error) {
	wire.Build(
		AppSet,
		wire.Struct(new(Dependencies), "*"),
	)
	return nil, nil, nil
}
