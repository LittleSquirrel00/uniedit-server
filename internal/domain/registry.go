package domain

import (
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// Domain holds all domain services.
// This is the central registry for all business logic.
type Domain struct {
	// User handles user profile and preferences.
	User user.UserDomain

	// Auth handles authentication and authorization.
	Auth auth.AuthDomain

	// Billing handles subscription and quota management.
	Billing billing.BillingDomain

	// Order handles order management.
	Order order.OrderDomain

	// Payment handles payment processing.
	Payment payment.PaymentDomain

	// AI handles AI proxy and routing.
	// AI ai.AIDomain // Phase 6

	// Git handles git repository operations.
	// Git git.GitDomain // Phase 7

	// Media handles media file operations.
	// Media media.MediaDomain // Phase 8

	// Collaboration handles real-time collaboration.
	// Collaboration collaboration.CollaborationDomain // Phase 9
}

// OutboundPorts holds all outbound port implementations.
type OutboundPorts struct {
	// User ports
	UserDB         outbound.UserDatabasePort
	VerificationDB outbound.VerificationDatabasePort
	ProfileDB      outbound.ProfileDatabasePort
	PrefsDB        outbound.PreferencesDatabasePort
	AvatarStorage  outbound.AvatarStoragePort
	EmailSender    outbound.EmailSenderPort

	// Auth ports
	RefreshTokenDB  outbound.RefreshTokenDatabasePort
	UserAPIKeyDB    outbound.UserAPIKeyDatabasePort
	SystemAPIKeyDB  outbound.SystemAPIKeyDatabasePort
	OAuthStateStore outbound.OAuthStateStorePort
	OAuthRegistry   outbound.OAuthRegistryPort
	JWT             outbound.JWTPort
	Crypto          outbound.CryptoPort

	// Billing ports
	PlanDB         outbound.PlanDatabasePort
	SubscriptionDB outbound.SubscriptionDatabasePort
	UsageDB        outbound.UsageRecordDatabasePort
	QuotaCache     outbound.QuotaCachePort

	// Order ports
	OrderDB   outbound.OrderDatabasePort
	ItemDB    outbound.OrderItemDatabasePort
	InvoiceDB outbound.InvoiceDatabasePort

	// Payment ports
	PaymentDB          outbound.PaymentDatabasePort
	WebhookEventDB     outbound.WebhookEventDatabasePort
	PaymentProviderReg outbound.PaymentProviderRegistryPort
	OrderReader        outbound.OrderReaderPort
	BillingReader      outbound.BillingReaderPort
	EventPublisher     outbound.EventPublisherPort

	// Generic ports
	Database outbound.DatabasePort
	Cache    outbound.CachePort
	Storage  outbound.StoragePort
}

// AuthConfig holds auth domain configuration.
type AuthConfig struct {
	MaxAPIKeysPerUser int
}

// DefaultAuthConfig returns default auth configuration.
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		MaxAPIKeysPerUser: 10,
	}
}

// PaymentConfig holds payment domain configuration.
type PaymentConfig struct {
	NotifyBaseURL string
}

// DefaultPaymentConfig returns default payment configuration.
func DefaultPaymentConfig() *PaymentConfig {
	return &PaymentConfig{
		NotifyBaseURL: "https://api.example.com",
	}
}

// NewDomain creates domain services with dependencies.
func NewDomain(ports *OutboundPorts, authConfig *AuthConfig, paymentConfig *PaymentConfig, logger *zap.Logger) *Domain {
	if authConfig == nil {
		authConfig = DefaultAuthConfig()
	}
	if paymentConfig == nil {
		paymentConfig = DefaultPaymentConfig()
	}

	userDomain := user.NewUserDomain(
		ports.UserDB,
		ports.VerificationDB,
		ports.ProfileDB,
		ports.PrefsDB,
		ports.AvatarStorage,
		ports.EmailSender,
		logger.Named("user"),
	)

	return &Domain{
		User: userDomain,
		Auth: auth.NewAuthDomain(
			userDomain,
			ports.RefreshTokenDB,
			ports.UserAPIKeyDB,
			ports.SystemAPIKeyDB,
			ports.OAuthRegistry,
			ports.OAuthStateStore,
			ports.JWT,
			ports.Crypto,
			&auth.Config{MaxAPIKeysPerUser: authConfig.MaxAPIKeysPerUser},
			logger.Named("auth"),
		),
		Billing: billing.NewBillingDomain(
			ports.PlanDB,
			ports.SubscriptionDB,
			ports.UsageDB,
			ports.QuotaCache,
			logger.Named("billing"),
		),
		Order: order.NewOrderDomain(
			ports.OrderDB,
			ports.ItemDB,
			ports.InvoiceDB,
			ports.PlanDB,
			logger.Named("order"),
		),
		Payment: payment.NewPaymentDomain(
			ports.PaymentDB,
			ports.WebhookEventDB,
			ports.PaymentProviderReg,
			ports.OrderReader,
			ports.BillingReader,
			ports.EventPublisher,
			paymentConfig.NotifyBaseURL,
			logger.Named("payment"),
		),
	}
}
