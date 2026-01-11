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
	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/domain/git"
	"github.com/uniedit/server/internal/domain/media"
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

	// Outbound adapters
	"github.com/uniedit/server/internal/adapter/outbound/aiprovider"
	"github.com/uniedit/server/internal/adapter/outbound/mediaprovider"
	"github.com/uniedit/server/internal/adapter/outbound/oauth"
	"github.com/uniedit/server/internal/adapter/outbound/postgres"
	redisadapter "github.com/uniedit/server/internal/adapter/outbound/redis"

	// Infrastructure
	"github.com/uniedit/server/internal/infra/cache"
	"github.com/uniedit/server/internal/infra/config"
	"github.com/uniedit/server/internal/infra/database"
	"github.com/uniedit/server/internal/infra/httpclient"

	// Utils
	"github.com/uniedit/server/internal/utils/logger"
	"github.com/uniedit/server/internal/utils/metrics"
)

// ===== Infrastructure Providers =====

// InfraSet provides infrastructure dependencies.
var InfraSet = wire.NewSet(
	ProvideDatabase,
	ProvideRedisClient,
	ProvideHTTPClient,
	ProvideRateLimiter,
	ProvideLogger,
	ProvideZapLogger,
	ProvideMetrics,
)

// ProvideDatabase creates a database connection.
func ProvideDatabase(cfg *config.Config) (*gorm.DB, error) {
	return database.New(&cfg.Database)
}

// ProvideRedisClient creates a Redis client.
func ProvideRedisClient(cfg *config.Config, zapLog *zap.Logger) goredis.UniversalClient {
	if cfg.Redis.Address == "" {
		return nil
	}
	client, err := cache.NewRedisClient(&cfg.Redis)
	if err != nil {
		zapLog.Warn("Redis connection failed, continuing without cache", zap.Error(err))
		return nil
	}
	return client
}

// ProvideLogger creates a logger instance.
func ProvideLogger(cfg *config.Config) *logger.Logger {
	return logger.New(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
}

// ProvideZapLogger creates a zap logger instance.
func ProvideZapLogger(cfg *config.Config) (*zap.Logger, error) {
	return logger.NewZapLogger(&logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
}

// ProvideHTTPClient creates a shared HTTP client with connection pooling.
func ProvideHTTPClient(cfg *config.Config) *http.Client {
	return httpclient.New(cfg.HTTPClient)
}

// ProvideRateLimiter creates a rate limiter.
func ProvideRateLimiter(redis goredis.UniversalClient) outbound.RateLimiterPort {
	if redis == nil {
		return nil
	}
	if client, ok := redis.(*goredis.Client); ok {
		return redisadapter.NewRateLimiter(client)
	}
	return nil
}

// ProvideMetrics creates a metrics instance.
func ProvideMetrics() *metrics.Metrics {
	return metrics.New("uniedit")
}

// ===== User Domain Providers =====

// UserSet provides user domain dependencies.
var UserSet = wire.NewSet(
	postgres.NewUserAdapter,
	postgres.NewVerificationAdapter,
	ProvideUserDomain,
)

// ProvideUserDomain creates the user domain.
func ProvideUserDomain(
	userDB outbound.UserDatabasePort,
	verificationDB outbound.VerificationDatabasePort,
	zapLog *zap.Logger,
) user.UserDomain {
	return user.NewUserDomain(
		userDB,
		verificationDB,
		nil, // profileDB - optional
		nil, // prefsDB - optional
		nil, // avatarStorage - optional
		nil, // emailSender - optional
		zapLog,
	)
}

// ===== Auth Domain Providers =====

// AuthSet provides auth domain dependencies.
var AuthSet = wire.NewSet(
	postgres.NewRefreshTokenAdapter,
	postgres.NewUserAPIKeyAdapter,
	postgres.NewSystemAPIKeyAdapter,
	ProvideOAuthRegistry,
	ProvideOAuthStateStore,
	ProvideJWTManager,
	ProvideAuthCryptoAdapter,
	ProvideAuthDomain,
)

// ProvideOAuthRegistry creates the OAuth registry.
func ProvideOAuthRegistry(cfg *config.Config) outbound.OAuthRegistryPort {
	registry := oauth.NewRegistry()
	if cfg.Auth.OAuth.GitHub.ClientID != "" {
		registry.RegisterGitHub(
			cfg.Auth.OAuth.GitHub.ClientID,
			cfg.Auth.OAuth.GitHub.ClientSecret,
			cfg.Auth.OAuth.GitHub.RedirectURL,
		)
	}
	if cfg.Auth.OAuth.Google.ClientID != "" {
		registry.RegisterGoogle(
			cfg.Auth.OAuth.Google.ClientID,
			cfg.Auth.OAuth.Google.ClientSecret,
			cfg.Auth.OAuth.Google.RedirectURL,
		)
	}
	return registry
}

// ProvideOAuthStateStore creates the OAuth state store.
func ProvideOAuthStateStore(redis goredis.UniversalClient) outbound.OAuthStateStorePort {
	if redis == nil {
		return oauth.NewInMemoryStateStore()
	}
	if client, ok := redis.(*goredis.Client); ok {
		return redisadapter.NewOAuthStateStore(client)
	}
	return oauth.NewInMemoryStateStore()
}

// ProvideJWTManager creates the JWT manager.
func ProvideJWTManager(cfg *config.Config) outbound.JWTPort {
	return oauth.NewJWTManager(&oauth.JWTConfig{
		Secret:             cfg.Auth.JWTSecret,
		AccessTokenExpiry:  cfg.Auth.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.Auth.RefreshTokenExpiry,
	})
}

// ProvideAuthCryptoAdapter creates the crypto adapter for auth.
func ProvideAuthCryptoAdapter(cfg *config.Config) outbound.CryptoPort {
	return oauth.NewCryptoAdapter(cfg.Auth.MasterKey)
}

// ProvideAuthDomain creates the auth domain.
func ProvideAuthDomain(
	userDomain user.UserDomain,
	tokenRepo outbound.RefreshTokenDatabasePort,
	userAPIKeyRepo outbound.UserAPIKeyDatabasePort,
	systemAPIKeyRepo outbound.SystemAPIKeyDatabasePort,
	oauthRegistry outbound.OAuthRegistryPort,
	stateStore outbound.OAuthStateStorePort,
	jwtManager outbound.JWTPort,
	crypto outbound.CryptoPort,
	zapLog *zap.Logger,
) auth.AuthDomain {
	return auth.NewAuthDomain(
		userDomain,
		tokenRepo,
		userAPIKeyRepo,
		systemAPIKeyRepo,
		oauthRegistry,
		stateStore,
		jwtManager,
		crypto,
		&auth.Config{MaxAPIKeysPerUser: 10},
		zapLog,
	)
}

// ===== Billing Domain Providers =====

// BillingSet provides billing domain dependencies.
var BillingSet = wire.NewSet(
	postgres.NewPlanAdapter,
	postgres.NewSubscriptionAdapter,
	postgres.NewUsageRecordAdapter,
	ProvideQuotaCache,
	ProvideBillingDomain,
)

// ProvideQuotaCache creates the quota cache.
func ProvideQuotaCache(redis goredis.UniversalClient) outbound.QuotaCachePort {
	if redis == nil {
		return nil
	}
	if client, ok := redis.(*goredis.Client); ok {
		return redisadapter.NewQuotaCache(client)
	}
	return nil
}

// ProvideBillingDomain creates the billing domain.
func ProvideBillingDomain(
	planDB outbound.PlanDatabasePort,
	subscriptionDB outbound.SubscriptionDatabasePort,
	usageDB outbound.UsageRecordDatabasePort,
	quotaCache outbound.QuotaCachePort,
	zapLog *zap.Logger,
) billing.BillingDomain {
	return billing.NewBillingDomain(
		planDB,
		subscriptionDB,
		usageDB,
		quotaCache,
		zapLog,
	)
}

// ===== Order Domain Providers =====

// OrderSet provides order domain dependencies.
var OrderSet = wire.NewSet(
	postgres.NewOrderAdapter,
	postgres.NewOrderItemAdapter,
	postgres.NewInvoiceAdapter,
	ProvideOrderDomain,
)

// ProvideOrderDomain creates the order domain.
func ProvideOrderDomain(
	orderDB outbound.OrderDatabasePort,
	orderItemDB outbound.OrderItemDatabasePort,
	invoiceDB outbound.InvoiceDatabasePort,
	planDB outbound.PlanDatabasePort,
	zapLog *zap.Logger,
) order.OrderDomain {
	return order.NewOrderDomain(
		orderDB,
		orderItemDB,
		invoiceDB,
		planDB,
		zapLog,
	)
}

// ===== Payment Domain Providers =====

// PaymentSet provides payment domain dependencies.
var PaymentSet = wire.NewSet(
	postgres.NewPaymentAdapter,
	postgres.NewWebhookEventAdapter,
	ProvideOrderReaderAdapter,
	ProvideBillingReaderAdapter,
	ProvideEventPublisher,
	ProvidePaymentDomain,
)

// ProvideOrderReaderAdapter creates the order reader adapter.
func ProvideOrderReaderAdapter(domain order.OrderDomain) outbound.OrderReaderPort {
	return newOrderReaderAdapter(domain)
}

// ProvideBillingReaderAdapter creates the billing reader adapter.
func ProvideBillingReaderAdapter(domain billing.BillingDomain) outbound.BillingReaderPort {
	return newBillingReaderAdapter(domain)
}

// ProvideEventPublisher creates the event publisher.
func ProvideEventPublisher() outbound.EventPublisherPort {
	return newNoOpEventPublisher()
}

// ProvidePaymentDomain creates the payment domain.
func ProvidePaymentDomain(
	paymentDB outbound.PaymentDatabasePort,
	webhookDB outbound.WebhookEventDatabasePort,
	orderReader outbound.OrderReaderPort,
	billingReader outbound.BillingReaderPort,
	eventPublisher outbound.EventPublisherPort,
	cfg *config.Config,
	zapLog *zap.Logger,
) payment.PaymentDomain {
	notifyBaseURL := cfg.Server.Address
	if cfg.Email.BaseURL != "" {
		notifyBaseURL = cfg.Email.BaseURL
	}
	return payment.NewPaymentDomain(
		paymentDB,
		webhookDB,
		nil, // providerRegistry
		orderReader,
		billingReader,
		eventPublisher,
		notifyBaseURL,
		zapLog,
	)
}

// ===== AI Domain Providers =====

// AISet provides AI domain dependencies.
var AISet = wire.NewSet(
	postgres.NewAIProviderAdapter,
	postgres.NewAIModelAdapter,
	postgres.NewAIProviderAccountAdapter,
	postgres.NewAIModelGroupAdapter,
	ProvideAIHealthCache,
	ProvideAIEmbeddingCache,
	ProvideVendorRegistry,
	ProvideAICryptoAdapter,
	ProvideAIDomain,
)

// ProvideAIHealthCache creates the AI health cache.
func ProvideAIHealthCache(redis goredis.UniversalClient) outbound.AIProviderHealthCachePort {
	if redis == nil {
		return nil
	}
	if client, ok := redis.(*goredis.Client); ok {
		return redisadapter.NewAIProviderHealthCacheAdapter(client)
	}
	return nil
}

// ProvideAIEmbeddingCache creates the AI embedding cache.
func ProvideAIEmbeddingCache(redis goredis.UniversalClient) outbound.AIEmbeddingCachePort {
	if redis == nil {
		return nil
	}
	if client, ok := redis.(*goredis.Client); ok {
		return redisadapter.NewAIEmbeddingCacheAdapter(client)
	}
	return nil
}

// ProvideVendorRegistry creates the vendor registry with shared HTTP client.
func ProvideVendorRegistry(client *http.Client) outbound.AIVendorRegistryPort {
	return aiprovider.NewDefaultRegistry(client)
}

// ProvideAICryptoAdapter creates the crypto adapter for AI.
func ProvideAICryptoAdapter(cfg *config.Config) outbound.AICryptoPort {
	return aiprovider.NewCryptoAdapter(cfg.Auth.MasterKey)
}

// ProvideAIDomain creates the AI domain.
func ProvideAIDomain(
	providerDB outbound.AIProviderDatabasePort,
	modelDB outbound.AIModelDatabasePort,
	accountDB outbound.AIProviderAccountDatabasePort,
	groupDB outbound.AIModelGroupDatabasePort,
	healthCache outbound.AIProviderHealthCachePort,
	embeddingCache outbound.AIEmbeddingCachePort,
	vendorRegistry outbound.AIVendorRegistryPort,
	crypto outbound.AICryptoPort,
	zapLog *zap.Logger,
) ai.AIDomain {
	return ai.NewAIDomain(
		providerDB,
		modelDB,
		accountDB,
		groupDB,
		healthCache,
		embeddingCache,
		vendorRegistry,
		crypto,
		nil, // usageRecorder
		nil, // config
		zapLog,
	)
}

// ===== Git Domain Providers =====

// GitSet provides Git domain dependencies.
var GitSet = wire.NewSet(
	postgres.NewGitRepoDatabaseAdapter,
	wire.Bind(new(outbound.GitRepoDatabasePort), new(*postgres.GitRepoDatabaseAdapter)),
	postgres.NewGitCollaboratorDatabaseAdapter,
	wire.Bind(new(outbound.GitCollaboratorDatabasePort), new(*postgres.GitCollaboratorDatabaseAdapter)),
	postgres.NewGitPullRequestDatabaseAdapter,
	wire.Bind(new(outbound.GitPullRequestDatabasePort), new(*postgres.GitPullRequestDatabaseAdapter)),
	postgres.NewGitLFSObjectDatabaseAdapter,
	wire.Bind(new(outbound.GitLFSObjectDatabasePort), new(*postgres.GitLFSObjectDatabaseAdapter)),
	postgres.NewGitLFSLockDatabaseAdapter,
	wire.Bind(new(outbound.GitLFSLockDatabasePort), new(*postgres.GitLFSLockDatabaseAdapter)),
	ProvideGitDomain,
)

// ProvideGitDomain creates the Git domain.
func ProvideGitDomain(
	repoDB outbound.GitRepoDatabasePort,
	collabDB outbound.GitCollaboratorDatabasePort,
	prDB outbound.GitPullRequestDatabasePort,
	lfsObjDB outbound.GitLFSObjectDatabasePort,
	lfsLockDB outbound.GitLFSLockDatabasePort,
	cfg *config.Config,
	zapLog *zap.Logger,
) inbound.GitDomain {
	gitCfg := git.DefaultConfig()
	if cfg.Git.RepoPrefix != "" {
		gitCfg.RepoPrefix = cfg.Git.RepoPrefix
	}
	return git.NewDomain(
		repoDB,
		collabDB,
		prDB,
		lfsObjDB,
		lfsLockDB,
		nil, // storage
		nil, // lfsStorage
		nil, // quotaChecker
		gitCfg,
		zapLog,
	)
}

// ===== Collaboration Domain Providers =====

// CollaborationSet provides collaboration domain dependencies.
var CollaborationSet = wire.NewSet(
	postgres.NewTeamAdapter,
	wire.Bind(new(outbound.TeamDatabasePort), new(*postgres.TeamAdapter)),
	postgres.NewTeamMemberAdapter,
	wire.Bind(new(outbound.TeamMemberDatabasePort), new(*postgres.TeamMemberAdapter)),
	postgres.NewTeamInvitationAdapter,
	wire.Bind(new(outbound.TeamInvitationDatabasePort), new(*postgres.TeamInvitationAdapter)),
	postgres.NewCollaborationUserLookupAdapter,
	wire.Bind(new(outbound.CollaborationUserLookupPort), new(*postgres.CollaborationUserLookupAdapter)),
	postgres.NewCollaborationTransactionAdapter,
	wire.Bind(new(outbound.CollaborationTransactionPort), new(*postgres.CollaborationTransactionAdapter)),
	ProvideCollaborationDomain,
)

// ProvideCollaborationDomain creates the collaboration domain.
func ProvideCollaborationDomain(
	teamDB outbound.TeamDatabasePort,
	memberDB outbound.TeamMemberDatabasePort,
	invitationDB outbound.TeamInvitationDatabasePort,
	userLookup outbound.CollaborationUserLookupPort,
	txAdapter outbound.CollaborationTransactionPort,
	cfg *config.Config,
	zapLog *zap.Logger,
) inbound.CollaborationDomain {
	collabCfg := collaboration.DefaultConfig()
	if cfg.Email.BaseURL != "" {
		collabCfg.BaseURL = cfg.Email.BaseURL
	}
	return collaboration.NewDomain(
		teamDB,
		memberDB,
		invitationDB,
		userLookup,
		txAdapter,
		collabCfg,
		zapLog,
	)
}

// ===== Media Domain Providers =====

// MediaSet provides media domain dependencies.
var MediaSet = wire.NewSet(
	postgres.NewMediaProviderDBAdapter,
	wire.Bind(new(outbound.MediaProviderDatabasePort), new(*postgres.MediaProviderDBAdapter)),
	postgres.NewMediaModelDBAdapter,
	wire.Bind(new(outbound.MediaModelDatabasePort), new(*postgres.MediaModelDBAdapter)),
	postgres.NewMediaTaskDBAdapter,
	wire.Bind(new(outbound.MediaTaskDatabasePort), new(*postgres.MediaTaskDBAdapter)),
	ProvideMediaHealthCache,
	ProvideMediaVendorRegistry,
	ProvideMediaCryptoAdapter,
	ProvideMediaDomain,
)

// ProvideMediaHealthCache creates the media health cache.
func ProvideMediaHealthCache(redis goredis.UniversalClient) outbound.MediaProviderHealthCachePort {
	if redis == nil {
		return nil
	}
	return redisadapter.NewMediaHealthCacheAdapter(redis)
}

// ProvideMediaCryptoAdapter creates the crypto adapter for media.
func ProvideMediaCryptoAdapter(cfg *config.Config) outbound.MediaCryptoPort {
	return aiprovider.NewCryptoAdapter(cfg.Auth.MasterKey)
}

// ProvideMediaVendorRegistry creates the media vendor registry with shared HTTP client.
func ProvideMediaVendorRegistry(client *http.Client) outbound.MediaVendorRegistryPort {
	registry := mediaprovider.NewRegistry()
	registry.Register(mediaprovider.NewOpenAIAdapter(client))
	return registry
}

// ProvideMediaDomain creates the media domain.
func ProvideMediaDomain(
	providerDB outbound.MediaProviderDatabasePort,
	modelDB outbound.MediaModelDatabasePort,
	taskDB outbound.MediaTaskDatabasePort,
	healthCache outbound.MediaProviderHealthCachePort,
	vendorRegistry outbound.MediaVendorRegistryPort,
	crypto outbound.MediaCryptoPort,
	zapLog *zap.Logger,
) inbound.MediaDomain {
	return media.NewDomain(
		providerDB,
		modelDB,
		taskDB,
		healthCache,
		vendorRegistry,
		crypto,
		media.DefaultConfig(),
		zapLog,
	)
}

// ===== HTTP Handler Providers =====

// ProtoHandlerSet provides proto-defined HTTP handlers (google.api.http).
var ProtoHandlerSet = wire.NewSet(
	pingproto.NewHandler,
	authproto.NewHandler,
	userproto.NewHandler,
	orderproto.NewHandler,
)

// BillingHandlerSet provides billing HTTP handlers.
var BillingHandlerSet = wire.NewSet(
	billinghttp.NewSubscriptionHandler,
	billinghttp.NewQuotaHandler,
	billinghttp.NewCreditsHandler,
	billinghttp.NewUsageHandler,
)

// PaymentHandlerSet provides payment HTTP handlers.
var PaymentHandlerSet = wire.NewSet(
	paymenthttp.NewPaymentHandler,
	paymenthttp.NewRefundHandler,
	paymenthttp.NewWebhookHandler,
)

// GitHandlerSet provides Git HTTP handlers.
var GitHandlerSet = wire.NewSet(
	ProvideGitHandler,
)

// ProvideGitHandler creates the Git HTTP handler.
func ProvideGitHandler(domain inbound.GitDomain, cfg *config.Config) *githttp.Handler {
	baseURL := cfg.Server.Address
	if cfg.Email.BaseURL != "" {
		baseURL = cfg.Email.BaseURL
	}
	// LFS domains are optional - pass nil for now
	return githttp.NewHandler(domain, nil, nil, baseURL)
}

// CollaborationHandlerSet provides collaboration HTTP handlers.
var CollaborationHandlerSet = wire.NewSet(
	ProvideCollaborationHandler,
)

// ProvideCollaborationHandler creates the Collaboration HTTP handler.
func ProvideCollaborationHandler(domain inbound.CollaborationDomain, cfg *config.Config) *collaborationhttp.Handler {
	baseURL := cfg.Server.Address
	if cfg.Email.BaseURL != "" {
		baseURL = cfg.Email.BaseURL
	}
	// Cast interface to concrete type
	if d, ok := domain.(*collaboration.Domain); ok {
		return collaborationhttp.NewHandler(d, baseURL)
	}
	return nil
}

// MediaHandlerSet provides media HTTP handlers.
var MediaHandlerSet = wire.NewSet(
	ProvideMediaHandler,
)

// ProvideMediaHandler creates the Media HTTP handler.
func ProvideMediaHandler(domain inbound.MediaDomain) *mediahttp.Handler {
	// Cast interface to concrete type
	if d, ok := domain.(*media.Domain); ok {
		return mediahttp.NewHandler(d)
	}
	return nil
}

// ProvideAIProviderAdminHandler creates the AI Provider admin HTTP handler.
func ProvideAIProviderAdminHandler(domain ai.AIDomain) *aihttp.ProviderAdminHandler {
	return aihttp.NewProviderAdminHandler(domain)
}

// ProvideAIModelAdminHandler creates the AI Model admin HTTP handler.
func ProvideAIModelAdminHandler(domain ai.AIDomain) *aihttp.ModelAdminHandler {
	return aihttp.NewModelAdminHandler(domain)
}

// ProvideAIPublicHandler creates the AI public HTTP handler.
func ProvideAIPublicHandler(domain ai.AIDomain) *aihttp.PublicHandler {
	return aihttp.NewPublicHandler(domain)
}

// AIHandlerSet provides AI HTTP handlers.
var AIHandlerSet = wire.NewSet(
	aihttp.NewChatHandler,
	ProvideAIProviderAdminHandler,
	ProvideAIModelAdminHandler,
	ProvideAIPublicHandler,
)

// HandlerSet provides all HTTP handlers.
var HandlerSet = wire.NewSet(
	AIHandlerSet,
	ProtoHandlerSet,
	BillingHandlerSet,
	PaymentHandlerSet,
	GitHandlerSet,
	CollaborationHandlerSet,
	MediaHandlerSet,
)

// ===== Master Set =====

// AppSet is the master provider set that includes all dependencies.
var AppSet = wire.NewSet(
	InfraSet,
	UserSet,
	AuthSet,
	BillingSet,
	OrderSet,
	PaymentSet,
	AISet,
	GitSet,
	CollaborationSet,
	MediaSet,
	HandlerSet,
)
