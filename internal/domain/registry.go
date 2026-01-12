package domain

import (
	"github.com/uniedit/server/internal/domain/ai"
	"github.com/uniedit/server/internal/domain/auth"
	"github.com/uniedit/server/internal/domain/billing"
	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/domain/git"
	"github.com/uniedit/server/internal/domain/media"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/domain/payment"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/port/inbound"
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
	Billing inbound.BillingDomain

	// Order handles order management.
	Order order.OrderDomain

	// Payment handles payment processing.
	Payment payment.PaymentDomain

	// AI handles AI proxy and routing.
	AI ai.AIDomain

	// Git handles git repository operations.
	Git *git.Domain

	// GitLFS handles Git LFS operations.
	GitLFS *git.LFSDomain

	// GitLFSLock handles Git LFS locking operations.
	GitLFSLock *git.LFSLockDomain

	// Media handles media file operations.
	Media *media.Domain

	// Collaboration handles team collaboration.
	Collaboration *collaboration.Domain
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

	// AI ports
	AIProviderDB     outbound.AIProviderDatabasePort
	AIModelDB        outbound.AIModelDatabasePort
	AIAccountDB      outbound.AIProviderAccountDatabasePort
	AIGroupDB        outbound.AIModelGroupDatabasePort
	AIHealthCache    outbound.AIProviderHealthCachePort
	AIEmbeddingCache outbound.AIEmbeddingCachePort
	AIVendorRegistry outbound.AIVendorRegistryPort
	AICrypto         outbound.AICryptoPort
	AIUsageRecorder  outbound.AIUsageRecorderPort

	// Git ports
	GitRepoDB       outbound.GitRepoDatabasePort
	GitCollabDB     outbound.GitCollaboratorDatabasePort
	GitPRDB         outbound.GitPullRequestDatabasePort
	GitLFSObjDB     outbound.GitLFSObjectDatabasePort
	GitLFSLockDB    outbound.GitLFSLockDatabasePort
	GitStorage      outbound.GitStoragePort
	GitLFSStorage   outbound.GitLFSStoragePort
	GitAccessCtrl   outbound.GitAccessControlPort

	// Media ports
	MediaProviderDB  outbound.MediaProviderDatabasePort
	MediaModelDB     outbound.MediaModelDatabasePort
	MediaTaskDB      outbound.MediaTaskDatabasePort
	MediaHealthCache outbound.MediaProviderHealthCachePort
	MediaVendorReg   outbound.MediaVendorRegistryPort
	MediaCrypto      outbound.MediaCryptoPort

	// Collaboration ports
	CollabTeamDB       outbound.TeamDatabasePort
	CollabMemberDB     outbound.TeamMemberDatabasePort
	CollabInvitationDB outbound.TeamInvitationDatabasePort
	CollabUserLookup   outbound.CollaborationUserLookupPort
	CollabTransaction  outbound.CollaborationTransactionPort

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

// AIConfig holds AI domain configuration.
type AIConfig = ai.Config

// DefaultAIConfig returns default AI configuration.
func DefaultAIConfig() *AIConfig {
	return ai.DefaultConfig()
}

// GitConfig holds Git domain configuration.
type GitConfig = git.Config

// DefaultGitConfig returns default Git configuration.
func DefaultGitConfig() *GitConfig {
	return git.DefaultConfig()
}

// MediaConfig holds Media domain configuration.
type MediaConfig = media.Config

// DefaultMediaConfig returns default Media configuration.
func DefaultMediaConfig() *MediaConfig {
	return media.DefaultConfig()
}

// CollaborationConfig holds Collaboration domain configuration.
type CollaborationConfig = collaboration.Config

// DefaultCollaborationConfig returns default Collaboration configuration.
func DefaultCollaborationConfig() *CollaborationConfig {
	return collaboration.DefaultConfig()
}

// NewDomain creates domain services with dependencies.
func NewDomain(ports *OutboundPorts, authConfig *AuthConfig, paymentConfig *PaymentConfig, aiConfig *AIConfig, gitConfig *GitConfig, mediaConfig *MediaConfig, collabConfig *CollaborationConfig, logger *zap.Logger) *Domain {
	if authConfig == nil {
		authConfig = DefaultAuthConfig()
	}
	if paymentConfig == nil {
		paymentConfig = DefaultPaymentConfig()
	}
	if aiConfig == nil {
		aiConfig = DefaultAIConfig()
	}
	if gitConfig == nil {
		gitConfig = DefaultGitConfig()
	}
	if mediaConfig == nil {
		mediaConfig = DefaultMediaConfig()
	}
	if collabConfig == nil {
		collabConfig = DefaultCollaborationConfig()
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
		AI: ai.NewAIDomain(
			ports.AIProviderDB,
			ports.AIModelDB,
			ports.AIAccountDB,
			ports.AIGroupDB,
			ports.AIHealthCache,
			ports.AIEmbeddingCache,
			ports.AIVendorRegistry,
			ports.AICrypto,
			ports.AIUsageRecorder,
			aiConfig,
			logger.Named("ai"),
		),
		Git: git.NewDomain(
			ports.GitRepoDB,
			ports.GitCollabDB,
			ports.GitPRDB,
			ports.GitLFSObjDB,
			ports.GitLFSLockDB,
			ports.GitStorage,
			ports.GitLFSStorage,
			nil, // quotaChecker - will be set after billing domain is available
			gitConfig,
			logger.Named("git"),
		),
		GitLFS: git.NewLFSDomain(
			ports.GitRepoDB,
			ports.GitLFSObjDB,
			ports.GitLFSStorage,
			ports.GitAccessCtrl,
			gitConfig,
			logger.Named("git.lfs"),
		),
		GitLFSLock: git.NewLFSLockDomain(
			ports.GitRepoDB,
			ports.GitLFSLockDB,
			ports.GitAccessCtrl,
			logger.Named("git.lfs.lock"),
		),
		Media: media.NewDomain(
			ports.MediaProviderDB,
			ports.MediaModelDB,
			ports.MediaTaskDB,
			ports.MediaHealthCache,
			ports.MediaVendorReg,
			ports.MediaCrypto,
			mediaConfig,
			logger.Named("media"),
		),
		Collaboration: collaboration.NewDomain(
			ports.CollabTeamDB,
			ports.CollabMemberDB,
			ports.CollabInvitationDB,
			ports.CollabUserLookup,
			ports.CollabTransaction,
			collabConfig,
			logger.Named("collaboration"),
		),
	}
}
