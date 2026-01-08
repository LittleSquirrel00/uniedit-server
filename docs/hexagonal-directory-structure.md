# UniEdit Server 六边形架构目录结构设计

## 参考标准

基于 [Hexagonal-Architecture.md](../Hexagonal-Architecture.md) 的 Project Structure，采用**全局分层**而非模块内分层：

```
prabogo/
├── cmd/
│   └── main.go                  # Application entry point
├── internal/
│   ├── app.go                   # Application setup
│   ├── domain/                  # Domain layer (business logic)
│   │   ├── registry.go          # Domain service registry
│   │   └── {entity}/            # Domain logic for each entity
│   ├── port/                    # Interfaces (ports)
│   │   ├── inbound/             # Input ports (by entity)
│   │   └── outbound/            # Output ports (by entity)
│   ├── adapter/                 # Adapters implementing ports
│   │   ├── inbound/             # Input adapters (by technology)
│   │   └── outbound/            # Output adapters (by technology)
│   ├── model/                   # Data structures
│   └── migration/               # Database migrations
└── utils/                       # Utility functions
```

**核心设计理念**：
- `domain/` - 按业务实体划分，包含核心业务逻辑
- `port/` - 按业务实体划分，定义入站/出站接口
- `adapter/` - 按技术划分，实现具体的端口接口
- `model/` - 共享的数据结构

---

## UniEdit Server 完整目录结构

```
uniedit-server/
├── cmd/
│   └── server/
│       └── main.go                      # Application entry point
│
├── internal/
│   ├── app.go                           # Application setup & bootstrap
│   ├── config.go                        # Application configuration
│   │
│   ├── domain/                          # Domain layer (core business logic)
│   │   ├── registry.go                  # Domain service registry
│   │   ├── auth/                        # Auth domain
│   │   ├── user/                        # User domain
│   │   ├── billing/                     # Billing domain
│   │   ├── order/                       # Order domain
│   │   ├── payment/                     # Payment domain
│   │   ├── ai/                          # AI domain
│   │   ├── git/                         # Git domain
│   │   ├── media/                       # Media domain
│   │   └── collaboration/               # Collaboration domain
│   │
│   ├── port/                            # Ports (interfaces)
│   │   ├── inbound/                     # Input ports
│   │   │   ├── registry.go              # Inbound port registry
│   │   │   ├── auth.go                  # Auth service ports
│   │   │   ├── user.go                  # User service ports
│   │   │   ├── billing.go               # Billing service ports
│   │   │   ├── order.go                 # Order service ports
│   │   │   ├── payment.go               # Payment service ports
│   │   │   ├── ai.go                    # AI service ports
│   │   │   ├── git.go                   # Git service ports
│   │   │   ├── media.go                 # Media service ports
│   │   │   └── collaboration.go         # Collaboration service ports
│   │   │
│   │   └── outbound/                    # Output ports
│   │       ├── registry.go              # Outbound port registry
│   │       ├── auth.go                  # Auth repository ports
│   │       ├── user.go                  # User repository ports
│   │       ├── billing.go               # Billing repository ports
│   │       ├── order.go                 # Order repository ports
│   │       ├── payment.go               # Payment gateway ports
│   │       ├── ai.go                    # AI adapter ports
│   │       ├── git.go                   # Git storage ports
│   │       ├── media.go                 # Media storage ports
│   │       ├── collaboration.go         # Collaboration repository ports
│   │       ├── database.go              # Generic database port
│   │       ├── cache.go                 # Generic cache port
│   │       ├── storage.go               # Generic storage port
│   │       └── message.go               # Generic message port
│   │
│   ├── adapter/                         # Adapters (implementations)
│   │   ├── inbound/                     # Input adapters
│   │   │   ├── gin/                     # HTTP adapters (Gin framework)
│   │   │   │   ├── router.go            # Route registration
│   │   │   │   ├── middleware.go        # HTTP middlewares
│   │   │   │   ├── auth.go              # Auth handlers
│   │   │   │   ├── user.go              # User handlers
│   │   │   │   ├── billing.go           # Billing handlers
│   │   │   │   ├── order.go             # Order handlers
│   │   │   │   ├── payment.go           # Payment handlers
│   │   │   │   ├── ai.go                # AI handlers
│   │   │   │   ├── git.go               # Git handlers
│   │   │   │   ├── media.go             # Media handlers
│   │   │   │   └── collaboration.go     # Collaboration handlers
│   │   │   │
│   │   │   ├── grpc/                    # gRPC adapters (future)
│   │   │   │   └── ai.go
│   │   │   │
│   │   │   └── command/                 # CLI adapters
│   │   │       ├── migrate.go
│   │   │       └── seed.go
│   │   │
│   │   └── outbound/                    # Output adapters
│   │       ├── postgres/                # PostgreSQL adapters
│   │       │   ├── database.go          # Connection & transaction
│   │       │   ├── auth.go              # Auth repository impl
│   │       │   ├── user.go              # User repository impl
│   │       │   ├── billing.go           # Billing repository impl
│   │       │   ├── order.go             # Order repository impl
│   │       │   ├── payment.go           # Payment repository impl
│   │       │   ├── ai.go                # AI repository impl
│   │       │   ├── git.go               # Git repository impl
│   │       │   ├── media.go             # Media repository impl
│   │       │   └── collaboration.go     # Collaboration repository impl
│   │       │
│   │       ├── redis/                   # Redis adapters
│   │       │   ├── cache.go             # Generic cache impl
│   │       │   ├── session.go           # Session cache
│   │       │   ├── quota.go             # Quota cache
│   │       │   └── embedding.go         # Embedding cache
│   │       │
│   │       ├── r2/                      # Cloudflare R2 adapters
│   │       │   ├── storage.go           # Generic storage impl
│   │       │   ├── git.go               # Git LFS storage
│   │       │   └── media.go             # Media storage
│   │       │
│   │       ├── http/                    # HTTP client adapters
│   │       │   ├── openai.go            # OpenAI client
│   │       │   ├── anthropic.go         # Anthropic client
│   │       │   ├── stripe.go            # Stripe client
│   │       │   ├── alipay.go            # Alipay client
│   │       │   └── wechat.go            # WeChat Pay client
│   │       │
│   │       └── rabbitmq/                # Message queue adapters
│   │           └── publisher.go
│   │
│   ├── model/                           # Data structures
│   │   ├── auth.go                      # Auth models
│   │   ├── user.go                      # User models
│   │   ├── billing.go                   # Billing models
│   │   ├── order.go                     # Order models
│   │   ├── payment.go                   # Payment models
│   │   ├── ai.go                        # AI models
│   │   ├── git.go                       # Git models
│   │   ├── media.go                     # Media models
│   │   ├── collaboration.go             # Collaboration models
│   │   ├── request.go                   # Common request models
│   │   └── response.go                  # Common response models
│   │
│   └── migration/                       # Database migrations
│       └── postgres/
│           ├── 001_users.sql
│           ├── 002_auth.sql
│           ├── 003_billing.sql
│           ├── 004_orders.sql
│           ├── 005_payments.sql
│           ├── 006_ai.sql
│           ├── 007_git.sql
│           ├── 008_media.sql
│           └── 009_collaboration.sql
│
├── utils/                               # Utility functions and helpers
│   ├── crypto.go                        # Encryption utilities
│   ├── jwt.go                           # JWT utilities
│   ├── pagination.go                    # Pagination utilities
│   ├── validation.go                    # Validation utilities
│   └── errors.go                        # Error utilities
│
├── api/                                 # API definitions
│   ├── openapi/
│   │   └── openapi.yaml
│   └── proto/
│       └── ai.proto
│
├── configs/
│   ├── config.yaml
│   └── config.example.yaml
│
├── scripts/
│   ├── build.sh
│   └── migrate.sh
│
├── docs/
│
└── tests/
    ├── e2e/
    └── fixtures/
```

---

## 各层详细结构

### Domain 层 - 核心业务逻辑

按业务实体划分，每个子目录包含该领域的聚合根、实体、值对象和领域服务：

```
internal/domain/
├── registry.go                          # Domain service registry
│
├── auth/                                # Auth domain
│   ├── domain.go                        # Auth domain service
│   ├── domain_test.go                   # Unit tests
│   ├── user.go                          # User aggregate root
│   ├── token.go                         # Token entity
│   ├── apikey.go                        # API Key entity
│   ├── session.go                       # Session value object
│   └── event.go                         # Auth domain events
│
├── user/                                # User domain
│   ├── domain.go                        # User domain service
│   ├── domain_test.go
│   ├── profile.go                       # Profile value object
│   └── preferences.go                   # Preferences value object
│
├── billing/                             # Billing domain
│   ├── domain.go                        # Billing domain service
│   ├── domain_test.go
│   ├── subscription.go                  # Subscription aggregate root
│   ├── plan.go                          # Plan entity
│   ├── balance.go                       # Balance aggregate
│   ├── usage.go                         # Usage entity
│   ├── quota.go                         # Quota value object
│   └── event.go                         # Billing domain events
│
├── order/                               # Order domain
│   ├── domain.go                        # Order domain service
│   ├── domain_test.go
│   ├── order.go                         # Order aggregate root
│   ├── order_item.go                    # Order item entity
│   ├── money.go                         # Money value object
│   ├── status.go                        # Status value object
│   └── event.go                         # Order domain events
│
├── payment/                             # Payment domain
│   ├── domain.go                        # Payment domain service
│   ├── domain_test.go
│   ├── transaction.go                   # Transaction aggregate
│   ├── payment_method.go                # Payment method entity
│   ├── refund.go                        # Refund entity
│   └── event.go                         # Payment domain events
│
├── ai/                                  # AI domain
│   ├── domain.go                        # AI domain service
│   ├── domain_test.go
│   ├── model.go                         # AI Model entity
│   ├── provider.go                      # Provider aggregate
│   ├── request.go                       # Request entity
│   ├── response.go                      # Response value object
│   ├── routing.go                       # Routing strategy
│   └── event.go                         # AI domain events
│
├── git/                                 # Git domain
│   ├── domain.go                        # Git domain service
│   ├── domain_test.go
│   ├── repository.go                    # Repository aggregate
│   ├── branch.go                        # Branch entity
│   ├── commit.go                        # Commit value object
│   ├── lfs_object.go                    # LFS object entity
│   └── permission.go                    # Permission value object
│
├── media/                               # Media domain
│   ├── domain.go                        # Media domain service
│   ├── domain_test.go
│   ├── media.go                         # Media aggregate
│   ├── format.go                        # Format value object
│   └── metadata.go                      # Metadata value object
│
└── collaboration/                       # Collaboration domain
    ├── domain.go                        # Collaboration domain service
    ├── domain_test.go
    ├── workspace.go                     # Workspace aggregate
    ├── member.go                        # Member entity
    ├── role.go                          # Role value object
    ├── permission.go                    # Permission value object
    └── invitation.go                    # Invitation entity
```

---

### Port 层 - 接口定义

#### Inbound Ports（入站端口）

定义应用服务接口，供适配器调用：

```
internal/port/inbound/
├── registry.go                          # Inbound port registry
│
├── auth.go                              # Auth service ports
│   // AuthHttpPort interface {
│   //     Login(ctx) error
│   //     Logout(ctx) error
│   //     Refresh(ctx) error
│   // }
│   // AuthAPIKeyPort interface {
│   //     Create(ctx) error
│   //     Validate(ctx) error
│   //     Revoke(ctx) error
│   // }
│
├── user.go                              # User service ports
│   // UserHttpPort interface {
│   //     GetProfile(ctx) error
│   //     UpdateProfile(ctx) error
│   // }
│   // UserAdminPort interface {
│   //     ListUsers(ctx) error
│   //     SuspendUser(ctx) error
│   // }
│
├── billing.go                           # Billing service ports
│   // BillingHttpPort interface {
│   //     GetSubscription(ctx) error
│   //     GetBalance(ctx) error
│   //     GetUsageStats(ctx) error
│   // }
│   // QuotaPort interface {
│   //     CheckQuota(ctx) error
│   // }
│
├── order.go                             # Order service ports
│   // OrderHttpPort interface {
│   //     CreateOrder(ctx) error
│   //     GetOrder(ctx) error
│   //     CancelOrder(ctx) error
│   // }
│
├── payment.go                           # Payment service ports
│   // PaymentHttpPort interface {
│   //     InitiatePayment(ctx) error
│   //     GetPaymentStatus(ctx) error
│   // }
│   // WebhookPort interface {
│   //     HandleStripeWebhook(ctx) error
│   //     HandleAlipayCallback(ctx) error
│   // }
│
├── ai.go                                # AI service ports
│   // AIHttpPort interface {
│   //     Chat(ctx) error
│   //     ChatStream(ctx) error
│   //     Embedding(ctx) error
│   //     ListModels(ctx) error
│   // }
│   // AITaskPort interface {
│   //     CreateTask(ctx) error
│   //     GetTaskStatus(ctx) error
│   // }
│   // AIAdminPort interface {
│   //     ListProviders(ctx) error
│   //     UpdateProvider(ctx) error
│   // }
│
├── git.go                               # Git service ports
│   // GitHttpPort interface {
│   //     CreateRepository(ctx) error
│   //     GetRepository(ctx) error
│   //     DeleteRepository(ctx) error
│   // }
│   // GitProtocolPort interface {
│   //     ReceivePack(ctx) error
│   //     UploadPack(ctx) error
│   // }
│   // LFSPort interface {
│   //     BatchDownload(ctx) error
│   //     BatchUpload(ctx) error
│   // }
│
├── media.go                             # Media service ports
│   // MediaHttpPort interface {
│   //     Upload(ctx) error
│   //     GetMedia(ctx) error
│   //     DeleteMedia(ctx) error
│   // }
│
└── collaboration.go                     # Collaboration service ports
    // WorkspaceHttpPort interface {
    //     CreateWorkspace(ctx) error
    //     GetWorkspace(ctx) error
    // }
    // MemberPort interface {
    //     AddMember(ctx) error
    //     RemoveMember(ctx) error
    // }
```

#### Outbound Ports（出站端口）

定义对外部系统的依赖接口：

```
internal/port/outbound/
├── registry.go                          # Outbound port registry
│
├── auth.go                              # Auth repository ports
│   // UserDatabasePort interface {
│   //     FindByID(ctx, id) (*model.User, error)
│   //     FindByEmail(ctx, email) (*model.User, error)
│   //     Create(ctx, user) error
│   //     Update(ctx, user) error
│   // }
│   // TokenDatabasePort interface {
│   //     SaveRefreshToken(ctx, token) error
│   //     FindRefreshToken(ctx, token) (*model.RefreshToken, error)
│   //     RevokeRefreshToken(ctx, token) error
│   // }
│   // SessionCachePort interface {
│   //     Set(ctx, sessionID, user, ttl) error
│   //     Get(ctx, sessionID) (*model.User, error)
│   //     Delete(ctx, sessionID) error
│   // }
│   // OAuthProviderPort interface {
│   //     GetAuthURL(state) string
│   //     ExchangeCode(ctx, code) (*OAuthToken, error)
│   //     GetUserInfo(ctx, token) (*OAuthUser, error)
│   // }
│
├── user.go                              # User repository ports
│   // UserDatabasePort interface { ... }
│   // AvatarStoragePort interface {
│   //     Upload(ctx, userID, file) (string, error)
│   //     Delete(ctx, userID) error
│   // }
│
├── billing.go                           # Billing repository ports
│   // SubscriptionDatabasePort interface {
│   //     FindByUserID(ctx, userID) (*model.Subscription, error)
│   //     Create(ctx, sub) error
│   //     Update(ctx, sub) error
│   // }
│   // BalanceDatabasePort interface {
│   //     Get(ctx, userID) (*model.Balance, error)
│   //     IncrementCredits(ctx, userID, amount) error
│   //     DecrementCredits(ctx, userID, amount) error
│   // }
│   // UsageDatabasePort interface {
│   //     Record(ctx, usage) error
│   //     Aggregate(ctx, userID, period) (*model.UsageAggregate, error)
│   // }
│   // QuotaCachePort interface {
│   //     Get(ctx, userID, resource) (*model.Quota, error)
│   //     Set(ctx, userID, resource, quota) error
│   // }
│
├── order.go                             # Order repository ports
│   // OrderDatabasePort interface {
│   //     Save(ctx, order) error
│   //     FindByID(ctx, id) (*model.Order, error)
│   //     FindByUserID(ctx, userID, filter) ([]*model.Order, error)
│   //     Update(ctx, order) error
│   // }
│
├── payment.go                           # Payment gateway ports
│   // TransactionDatabasePort interface {
│   //     Save(ctx, tx) error
│   //     FindByID(ctx, id) (*model.Transaction, error)
│   // }
│   // PaymentGatewayPort interface {
│   //     CreatePaymentIntent(ctx, amount, currency, metadata) (string, error)
│   //     ConfirmPaymentIntent(ctx, id) error
│   //     CreateRefund(ctx, paymentIntentID, amount) (string, error)
│   //     VerifyWebhookSignature(payload, signature) error
│   // }
│
├── ai.go                                # AI adapter ports
│   // ProviderDatabasePort interface {
│   //     FindAll(ctx) ([]*model.Provider, error)
│   //     FindByID(ctx, id) (*model.Provider, error)
│   //     Update(ctx, provider) error
│   // }
│   // ModelDatabasePort interface {
│   //     FindAll(ctx) ([]*model.AIModel, error)
│   //     FindByProvider(ctx, providerID) ([]*model.AIModel, error)
│   // }
│   // EmbeddingCachePort interface {
│   //     Get(ctx, hash) ([]float64, error)
│   //     Set(ctx, hash, embedding, ttl) error
│   // }
│   // AIProviderPort interface {
│   //     Chat(ctx, request) (*model.AIResponse, error)
│   //     ChatStream(ctx, request) (<-chan *model.AIStreamChunk, error)
│   //     Embedding(ctx, request) ([]float64, error)
│   //     HealthCheck(ctx) (bool, error)
│   // }
│
├── git.go                               # Git storage ports
│   // RepositoryDatabasePort interface {
│   //     FindByID(ctx, id) (*model.Repository, error)
│   //     FindByUserID(ctx, userID) ([]*model.Repository, error)
│   //     Create(ctx, repo) error
│   //     Delete(ctx, id) error
│   // }
│   // LFSDatabasePort interface {
│   //     FindByOID(ctx, oid) (*model.LFSObject, error)
│   //     Create(ctx, obj) error
│   // }
│   // ObjectStoragePort interface {
│   //     PutObject(ctx, key, reader, size) error
│   //     GetObject(ctx, key) (io.ReadCloser, error)
│   //     DeleteObject(ctx, key) error
│   //     GetPresignedURL(ctx, key, duration) (string, error)
│   // }
│
├── media.go                             # Media storage ports
│   // MediaDatabasePort interface {
│   //     FindByID(ctx, id) (*model.Media, error)
│   //     Create(ctx, media) error
│   //     Delete(ctx, id) error
│   // }
│   // MediaStoragePort interface {
│   //     Upload(ctx, key, reader) error
│   //     GetURL(ctx, key) (string, error)
│   //     Delete(ctx, key) error
│   // }
│   // MediaProcessorPort interface {
│   //     Transcode(ctx, input, format) (output, error)
│   //     GenerateThumbnail(ctx, input) (output, error)
│   // }
│
├── collaboration.go                     # Collaboration repository ports
│   // WorkspaceDatabasePort interface { ... }
│   // MemberDatabasePort interface { ... }
│   // NotificationPort interface {
│   //     SendInvitation(ctx, email, invitation) error
│   // }
│
├── database.go                          # Generic database port
│   // DatabaseExecutor interface {
│   //     Exec(ctx, query, args...) error
│   //     Query(ctx, query, args...) (*Rows, error)
│   //     Transaction(ctx, fn func(tx Transaction) error) error
│   // }
│
├── cache.go                             # Generic cache port
│   // CachePort interface {
│   //     Get(ctx, key) ([]byte, error)
│   //     Set(ctx, key, value, ttl) error
│   //     Delete(ctx, key) error
│   // }
│
├── storage.go                           # Generic storage port
│   // StoragePort interface {
│   //     Put(ctx, key, reader, size) error
│   //     Get(ctx, key) (io.ReadCloser, error)
│   //     Delete(ctx, key) error
│   //     GetPresignedURL(ctx, key, duration) (string, error)
│   // }
│
└── message.go                           # Generic message port
    // MessagePort interface {
    //     Publish(ctx, topic, message) error
    //     Subscribe(ctx, topic, handler) error
    // }
    // EventPublisherPort interface {
    //     Publish(ctx, event) error
    // }
```

---

### Adapter 层 - 接口实现

#### Inbound Adapters（入站适配器）

按技术分组，实现入站端口：

```
internal/adapter/inbound/
├── gin/                                 # HTTP adapters (Gin framework)
│   ├── router.go                        # Route registration
│   ├── middleware.go                    # HTTP middlewares (auth, cors, logging, ratelimit)
│   │
│   ├── auth.go                          # Auth handlers
│   │   // type authAdapter struct { domain domain.AuthDomain }
│   │   // func (a *authAdapter) Login(c *gin.Context) error
│   │   // func (a *authAdapter) Logout(c *gin.Context) error
│   │   // func (a *authAdapter) Refresh(c *gin.Context) error
│   │
│   ├── user.go                          # User handlers
│   │   // type userAdapter struct { domain domain.UserDomain }
│   │   // func (a *userAdapter) GetProfile(c *gin.Context) error
│   │   // func (a *userAdapter) UpdateProfile(c *gin.Context) error
│   │
│   ├── billing.go                       # Billing handlers
│   │   // type billingAdapter struct { domain domain.BillingDomain }
│   │   // func (a *billingAdapter) GetSubscription(c *gin.Context) error
│   │   // func (a *billingAdapter) GetBalance(c *gin.Context) error
│   │
│   ├── order.go                         # Order handlers
│   │   // type orderAdapter struct { domain domain.OrderDomain }
│   │   // func (a *orderAdapter) CreateOrder(c *gin.Context) error
│   │   // func (a *orderAdapter) GetOrder(c *gin.Context) error
│   │
│   ├── payment.go                       # Payment handlers
│   │   // type paymentAdapter struct { domain domain.PaymentDomain }
│   │   // func (a *paymentAdapter) InitiatePayment(c *gin.Context) error
│   │   // func (a *paymentAdapter) HandleStripeWebhook(c *gin.Context) error
│   │
│   ├── ai.go                            # AI handlers
│   │   // type aiAdapter struct { domain domain.AIDomain }
│   │   // func (a *aiAdapter) Chat(c *gin.Context) error
│   │   // func (a *aiAdapter) ChatStream(c *gin.Context) error
│   │   // func (a *aiAdapter) Embedding(c *gin.Context) error
│   │
│   ├── git.go                           # Git handlers
│   │   // type gitAdapter struct { domain domain.GitDomain }
│   │   // func (a *gitAdapter) CreateRepository(c *gin.Context) error
│   │   // func (a *gitAdapter) ReceivePack(c *gin.Context) error
│   │
│   ├── media.go                         # Media handlers
│   │   // type mediaAdapter struct { domain domain.MediaDomain }
│   │   // func (a *mediaAdapter) Upload(c *gin.Context) error
│   │
│   └── collaboration.go                 # Collaboration handlers
│       // type collaborationAdapter struct { domain domain.CollaborationDomain }
│       // func (a *collaborationAdapter) CreateWorkspace(c *gin.Context) error
│
├── grpc/                                # gRPC adapters (future)
│   └── ai.go                            # AI gRPC service
│
└── command/                             # CLI adapters
    ├── migrate.go                       # Database migration commands
    └── seed.go                          # Data seeding commands
```

#### Outbound Adapters（出站适配器）

按技术分组，实现出站端口：

```
internal/adapter/outbound/
├── postgres/                            # PostgreSQL adapters
│   ├── database.go                      # Connection pool & transaction
│   │   // func NewDatabase(cfg *config.Database) (*gorm.DB, error)
│   │   // func (db *Database) Transaction(ctx, fn) error
│   │
│   ├── auth.go                          # Auth repository impl
│   │   // type authAdapter struct { db *gorm.DB }
│   │   // func (a *authAdapter) FindByID(ctx, id) (*model.User, error)
│   │   // func (a *authAdapter) Create(ctx, user) error
│   │
│   ├── user.go                          # User repository impl
│   ├── billing.go                       # Billing repository impl
│   ├── order.go                         # Order repository impl
│   ├── payment.go                       # Payment repository impl
│   ├── ai.go                            # AI repository impl
│   ├── git.go                           # Git repository impl
│   ├── media.go                         # Media repository impl
│   └── collaboration.go                 # Collaboration repository impl
│
├── redis/                               # Redis adapters
│   ├── cache.go                         # Generic cache impl
│   │   // type cacheAdapter struct { client redis.UniversalClient }
│   │   // func (a *cacheAdapter) Get(ctx, key) ([]byte, error)
│   │   // func (a *cacheAdapter) Set(ctx, key, value, ttl) error
│   │
│   ├── session.go                       # Session cache (auth)
│   ├── quota.go                         # Quota cache (billing)
│   └── embedding.go                     # Embedding cache (ai)
│
├── r2/                                  # Cloudflare R2 adapters
│   ├── storage.go                       # Generic storage impl
│   │   // type storageAdapter struct { client *s3.Client }
│   │   // func (a *storageAdapter) Put(ctx, key, reader, size) error
│   │   // func (a *storageAdapter) Get(ctx, key) (io.ReadCloser, error)
│   │
│   ├── git.go                           # Git LFS storage
│   └── media.go                         # Media storage
│
├── http/                                # HTTP client adapters
│   ├── openai.go                        # OpenAI client
│   │   // type openaiAdapter struct { client *http.Client; apiKey string }
│   │   // func (a *openaiAdapter) Chat(ctx, request) (*model.AIResponse, error)
│   │   // func (a *openaiAdapter) ChatStream(ctx, request) (<-chan *model.AIStreamChunk, error)
│   │   // func (a *openaiAdapter) Embedding(ctx, request) ([]float64, error)
│   │
│   ├── anthropic.go                     # Anthropic client
│   ├── google.go                        # Google AI client
│   ├── azure.go                         # Azure OpenAI client
│   ├── stripe.go                        # Stripe client
│   │   // type stripeAdapter struct { client *stripe.Client }
│   │   // func (a *stripeAdapter) CreatePaymentIntent(ctx, amount, currency, metadata) (string, error)
│   │
│   ├── alipay.go                        # Alipay client
│   └── wechat.go                        # WeChat Pay client
│
├── rabbitmq/                            # Message queue adapters
│   └── publisher.go                     # Event publisher impl
│
└── oauth/                               # OAuth provider adapters
    ├── google.go                        # Google OAuth
    ├── github.go                        # GitHub OAuth
    └── apple.go                         # Apple OAuth
```

---

### Model 层 - 数据结构

```
internal/model/
├── auth.go                              # Auth models
│   // type User struct { ID, Email, Name, CreatedAt, ... }
│   // type RefreshToken struct { Token, UserID, ExpiresAt, ... }
│   // type APIKey struct { ID, UserID, Name, Key, ... }
│   // type OAuthToken struct { AccessToken, RefreshToken, ... }
│
├── user.go                              # User models
│   // type Profile struct { UserID, DisplayName, Avatar, Bio, ... }
│   // type Preferences struct { UserID, Theme, Language, ... }
│
├── billing.go                           # Billing models
│   // type Subscription struct { ID, UserID, PlanID, Status, ... }
│   // type Plan struct { ID, Name, Price, Features, ... }
│   // type Balance struct { UserID, Credits, ... }
│   // type Usage struct { ID, UserID, Model, Tokens, Cost, ... }
│   // type Quota struct { Resource, Limit, Used, ... }
│
├── order.go                             # Order models
│   // type Order struct { ID, UserID, Status, Total, Items, ... }
│   // type OrderItem struct { ID, OrderID, Type, Amount, ... }
│   // type Money struct { Amount, Currency }
│
├── payment.go                           # Payment models
│   // type Transaction struct { ID, OrderID, Status, Amount, ... }
│   // type PaymentIntent struct { ID, ClientSecret, ... }
│   // type Refund struct { ID, TransactionID, Amount, ... }
│
├── ai.go                                # AI models
│   // type Provider struct { ID, Name, Type, Endpoint, ... }
│   // type AIModel struct { ID, ProviderID, Name, Capabilities, ... }
│   // type AIRequest struct { Model, Messages, Temperature, ... }
│   // type AIResponse struct { ID, Model, Content, Usage, ... }
│   // type AIStreamChunk struct { ID, Delta, FinishReason, ... }
│
├── git.go                               # Git models
│   // type Repository struct { ID, UserID, Name, ... }
│   // type Branch struct { Name, SHA, ... }
│   // type LFSObject struct { OID, Size, ... }
│
├── media.go                             # Media models
│   // type Media struct { ID, UserID, Name, URL, Format, ... }
│   // type MediaMetadata struct { Width, Height, Duration, ... }
│
├── collaboration.go                     # Collaboration models
│   // type Workspace struct { ID, Name, OwnerID, ... }
│   // type Member struct { WorkspaceID, UserID, Role, ... }
│   // type Invitation struct { ID, WorkspaceID, Email, Token, ... }
│
├── request.go                           # Common request models
│   // type PaginationRequest struct { Page, PageSize }
│   // type FilterRequest struct { ... }
│
└── response.go                          # Common response models
    // type PaginatedResponse struct { Data, Total, Page, ... }
    // type ErrorResponse struct { Code, Message, Details, ... }
```

---

## 命名规范

### 文件命名

| 层级 | 文件命名 | 示例 |
|------|---------|------|
| Domain | `domain.go`, `{entity}.go` | `domain/auth/domain.go`, `domain/auth/user.go` |
| Domain Test | `domain_test.go` | `domain/auth/domain_test.go` |
| Inbound Port | `{entity}.go` | `port/inbound/auth.go` |
| Outbound Port | `{entity}.go` | `port/outbound/auth.go` |
| Inbound Adapter | `{entity}.go` | `adapter/inbound/gin/auth.go` |
| Outbound Adapter | `{entity}.go` | `adapter/outbound/postgres/auth.go` |
| Model | `{entity}.go` | `model/auth.go` |
| Registry | `registry.go` | `domain/registry.go` |

### 包命名

| 层级 | 包路径 | 包名 |
|------|--------|------|
| Domain | `internal/domain/{entity}` | `auth`, `billing`, `ai` |
| Inbound Port | `internal/port/inbound` | `inbound` |
| Outbound Port | `internal/port/outbound` | `outbound` |
| HTTP Adapter | `internal/adapter/inbound/gin` | `gin` |
| DB Adapter | `internal/adapter/outbound/postgres` | `postgres` |
| Cache Adapter | `internal/adapter/outbound/redis` | `redis` |
| Model | `internal/model` | `model` |

### 接口命名

```go
// Domain Service Interface
type AuthDomain interface {
    Upsert(ctx context.Context, inputs []model.UserInput) ([]model.User, error)
    FindByFilter(ctx context.Context, filter model.UserFilter) ([]model.User, error)
    DeleteByFilter(ctx context.Context, filter model.UserFilter) error
}

// Inbound Port - HttpPort / MessagePort / CommandPort 结尾
type AuthHttpPort interface {
    Login(c any) error
    Logout(c any) error
}

type AuthMessagePort interface {
    HandleUserCreated(msg any) bool
}

// Outbound Port - DatabasePort / CachePort / StoragePort / GatewayPort 结尾
type UserDatabasePort interface {
    Upsert(users []model.UserInput) error
    FindByFilter(filter model.UserFilter, lock bool) ([]model.User, error)
}

type SessionCachePort interface {
    Set(sessionID string, user model.User) error
    Get(sessionID string) (model.User, error)
}

type PaymentGatewayPort interface {
    CreatePaymentIntent(amount int64, currency string) (string, error)
}
```

---

## 依赖流向图

```
                        ┌─────────────────────────────────────────────┐
                        │              External World                  │
                        │  (HTTP, gRPC, CLI, Message Queue, Cron)     │
                        └─────────────────────┬───────────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Inbound Adapters (adapter/inbound/)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐                     │
│  │   gin/   │  │  grpc/   │  │ command/ │  │rabbitmq/ │                     │
│  │ auth.go  │  │  ai.go   │  │migrate.go│  │consumer/ │                     │
│  │ user.go  │  │          │  │          │  │          │                     │
│  │ ai.go    │  │          │  │          │  │          │                     │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘                     │
└───────┼─────────────┼────────────┼─────────────┼────────────────────────────┘
        │             │            │             │
        └─────────────┴────────────┴─────────────┘
                                   │ implements
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Inbound Ports (port/inbound/)                            │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐                 │
│  │ AuthHttpPort   │  │ AIHttpPort     │  │ GitHttpPort    │  ...            │
│  │ AuthMessagePort│  │ AITaskPort     │  │ LFSPort        │                 │
│  └────────────────┘  └────────────────┘  └────────────────┘                 │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │ calls
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Domain Layer (domain/)                                 │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐                 │
│  │  domain/auth/  │  │ domain/billing/│  │   domain/ai/   │  ...            │
│  │   domain.go    │  │   domain.go    │  │   domain.go    │                 │
│  │   user.go      │  │ subscription.go│  │   provider.go  │                 │
│  │   token.go     │  │   balance.go   │  │   routing.go   │                 │
│  └────────────────┘  └────────────────┘  └────────────────┘                 │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │ depends on
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                     Outbound Ports (port/outbound/)                          │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐                 │
│  │UserDatabasePort│  │ SessionCache   │  │PaymentGateway  │  ...            │
│  │TokenDatabasePrt│  │ QuotaCache     │  │AIProviderPort  │                 │
│  └────────────────┘  └────────────────┘  └────────────────┘                 │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │ implemented by
                                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Outbound Adapters (adapter/outbound/)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │postgres/ │  │  redis/  │  │   r2/    │  │  http/   │  │rabbitmq/ │       │
│  │ auth.go  │  │session.go│  │storage.go│  │openai.go │  │publisher │       │
│  │ user.go  │  │quota.go  │  │  git.go  │  │stripe.go │  │          │       │
│  │ ai.go    │  │          │  │          │  │          │  │          │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
└───────┼─────────────┼────────────┼─────────────┼─────────────┼──────────────┘
        │             │            │             │             │
        └─────────────┴────────────┴─────────────┴─────────────┘
                                   │
                                   ▼
                        ┌─────────────────────────────────────────────┐
                        │           External Systems                   │
                        │  (PostgreSQL, Redis, R2, Stripe, OpenAI)    │
                        └─────────────────────────────────────────────┘
```

### 依赖规则

```
┌─────────────────────────────────────────────────────────────────┐
│                        依赖方向（向内）                           │
│                                                                  │
│   adapter/inbound  ───────────────┐                             │
│                                   │                             │
│                                   ▼                             │
│                           port/inbound                          │
│                                   │                             │
│                                   ▼                             │
│                              domain ◄─────── model              │
│                                   │                             │
│                                   ▼                             │
│                          port/outbound                          │
│                                   │                             │
│                                   ▼                             │
│   adapter/outbound ◄──────────────┘                             │
│                                                                  │
│   规则：                                                         │
│   1. Domain 不依赖任何外部包（只依赖 model）                      │
│   2. Port 定义接口，不依赖具体实现                               │
│   3. Adapter 实现 Port 接口，依赖外部库                          │
│   4. 依赖始终指向内层（Domain）                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Application 组装层

```go
// internal/app.go - 应用组装与启动

package internal

import (
    "github.com/uniedit/server/internal/domain/auth"
    "github.com/uniedit/server/internal/domain/billing"
    "github.com/uniedit/server/internal/adapter/inbound/gin"
    "github.com/uniedit/server/internal/adapter/outbound/postgres"
    "github.com/uniedit/server/internal/adapter/outbound/redis"
)

type App struct {
    config *Config
    router *gin.Engine

    // Domain services
    authDomain    auth.AuthDomain
    billingDomain billing.BillingDomain
    // ...
}

func NewApp(cfg *Config) (*App, error) {
    // 1. Initialize outbound adapters (infrastructure)
    db := postgres.NewDatabase(cfg.Database)
    cache := redis.NewCache(cfg.Redis)

    // 2. Create outbound port implementations
    userDBPort := postgres.NewUserAdapter(db)
    sessionCachePort := redis.NewSessionAdapter(cache)

    // 3. Create domain services with outbound ports
    authDomain := auth.NewAuthDomain(userDBPort, sessionCachePort)
    billingDomain := billing.NewBillingDomain(...)

    // 4. Create inbound adapters with domain services
    authAdapter := gin.NewAuthAdapter(authDomain)
    billingAdapter := gin.NewBillingAdapter(billingDomain)

    // 5. Register routes
    router := gin.NewRouter()
    authAdapter.RegisterRoutes(router)
    billingAdapter.RegisterRoutes(router)

    return &App{...}, nil
}
```

---

## Domain Registry

```go
// internal/domain/registry.go - Domain 服务注册表

package domain

import (
    "github.com/uniedit/server/internal/domain/auth"
    "github.com/uniedit/server/internal/domain/billing"
    "github.com/uniedit/server/internal/domain/ai"
    // ...
)

// Domain holds all domain services
type Domain struct {
    Auth          auth.AuthDomain
    Billing       billing.BillingDomain
    AI            ai.AIDomain
    Order         order.OrderDomain
    Payment       payment.PaymentDomain
    Git           git.GitDomain
    Media         media.MediaDomain
    User          user.UserDomain
    Collaboration collaboration.CollaborationDomain
}

// NewDomain creates all domain services with their dependencies
func NewDomain(ports *OutboundPorts) *Domain {
    return &Domain{
        Auth:    auth.NewAuthDomain(ports.UserDB, ports.TokenDB, ports.SessionCache),
        Billing: billing.NewBillingDomain(ports.SubscriptionDB, ports.BalanceDB, ports.QuotaCache),
        AI:      ai.NewAIDomain(ports.ProviderDB, ports.EmbeddingCache, ports.AIProvider),
        // ...
    }
}
```

---

## 检查清单

### 新增业务实体检查

- [ ] 在 `domain/{entity}/` 创建领域服务和实体
- [ ] 在 `port/inbound/{entity}.go` 定义入站端口
- [ ] 在 `port/outbound/{entity}.go` 定义出站端口
- [ ] 在 `adapter/inbound/gin/{entity}.go` 实现 HTTP 适配器
- [ ] 在 `adapter/outbound/postgres/{entity}.go` 实现数据库适配器
- [ ] 在 `model/{entity}.go` 定义数据模型
- [ ] 在 `domain/registry.go` 注册领域服务
- [ ] 在 `app.go` 组装依赖

### 目录结构验证

```bash
# 验证整体目录结构
tree internal/ -L 3

# 预期输出
internal/
├── app.go
├── config.go
├── domain/
│   ├── registry.go
│   ├── auth/
│   ├── billing/
│   ├── ai/
│   └── ...
├── port/
│   ├── inbound/
│   │   ├── registry.go
│   │   ├── auth.go
│   │   ├── billing.go
│   │   └── ...
│   └── outbound/
│       ├── registry.go
│       ├── auth.go
│       ├── billing.go
│       └── ...
├── adapter/
│   ├── inbound/
│   │   ├── gin/
│   │   ├── grpc/
│   │   └── command/
│   └── outbound/
│       ├── postgres/
│       ├── redis/
│       ├── r2/
│       └── http/
├── model/
│   ├── auth.go
│   ├── billing.go
│   └── ...
└── migration/
    └── postgres/
```

### 依赖检查

```bash
# 检查 domain 不依赖 adapter
grep -r "adapter" internal/domain/
# 应该无输出

# 检查 port 不依赖 adapter
grep -r "adapter" internal/port/
# 应该无输出

# 检查循环依赖
go mod graph | grep cycle
# 应该无输出
```
