# Implementation Tasks

## Phase 0: Database Schema

- [x] 0.1 Create migration 000004: Extend users table
  - Add status, password_hash, email_verified, is_admin columns
  - Make oauth_provider, oauth_id nullable
  - Add soft delete (deleted_at) and suspend fields
  - Create email_verifications table

- [x] 0.2 Create migration 000005: Create billing tables
  - Create plans table with quotas and Stripe price IDs
  - Create subscriptions table with Stripe fields
  - Create usage_records hypertable (TimescaleDB)
  - Add compression and retention policies

- [x] 0.3 Create migration 000006: Create order tables
  - Create orders table with state machine columns
  - Create order_items table
  - Create invoices table

- [x] 0.4 Create migration 000007: Create payment tables
  - Create payments table
  - Create stripe_webhook_events table (idempotency)

- [x] 0.5 Create migration 000008: Seed initial plans
  - Insert free, pro_monthly, pro_yearly, team_monthly, team_yearly, enterprise plans

- [x] 0.6 Create migration 000009: Add China payment support
  - Add trade_no, payer_id columns to payments
  - Create payment_webhook_events table for Alipay/WeChat

## Phase 1: User Module (P0)

- [x] 1.1 Create user module structure
  - `internal/module/user/model.go` - Extended User model, EmailVerification
  - `internal/module/user/dto.go` - Request/response types
  - `internal/module/user/errors.go` - Module errors

- [x] 1.2 Implement user repository
  - `internal/module/user/repository.go` - CRUD operations
  - Methods: Create, GetByID, GetByEmail, Update, SoftDelete
  - Email verification token methods

- [x] 1.3 Implement user service
  - `internal/module/user/service.go` - Business logic
  - Register, VerifyEmail, ResendVerification
  - Login (password), RequestPasswordReset, ResetPassword, ChangePassword
  - UpdateProfile, DeleteAccount

- [x] 1.4 Implement email verification service
  - `internal/module/user/email.go` - Token generation, email sending
  - Token generation with expiration
  - Email template rendering
  - SMTP integration (configurable)

- [x] 1.5 Implement user handlers
  - `internal/module/user/handler.go` - HTTP endpoints
  - POST /auth/register, POST /auth/verify-email
  - POST /auth/login/password
  - POST /auth/password/reset-request, POST /auth/password/reset
  - PUT /users/me/password, DELETE /users/me

- [x] 1.6 Implement admin handlers
  - `internal/module/user/admin_handler.go` - Admin endpoints
  - GET /admin/users (with pagination, filters)
  - POST /admin/users/:id/suspend, POST /admin/users/:id/reactivate
  - PUT /admin/users/:id/admin-status

- [ ] 1.7 Write user module tests
  - Repository tests
  - Service tests (with mocks)
  - Handler tests

## Phase 2: Billing Module (P0)

- [x] 2.1 Create billing module structure
  - `internal/module/billing/model.go` - Plan, Subscription, UsageRecord
  - `internal/module/billing/dto.go` - Request/response types
  - `internal/module/billing/errors.go` - Module errors

- [x] 2.2 Implement billing repository
  - `internal/module/billing/repository.go` - CRUD operations
  - Plan methods: ListActive, GetByID
  - Subscription methods: Create, GetByUserID, Update
  - Usage record methods: Create, Aggregate

- [x] 2.3 Implement quota manager
  - `internal/module/billing/quota/manager.go` - Quota logic
  - Redis counters for tokens/requests
  - Check, Consume, Reset methods
  - Graceful degradation when Redis unavailable

- [x] 2.4 Implement quota checker middleware
  - `internal/module/billing/quota/checker.go` - Gin middleware
  - Check quota before AI requests
  - Return 402 when exceeded

- [x] 2.5 Implement usage recorder
  - `internal/module/billing/usage/recorder.go` - Usage recording
  - Async recording to avoid latency
  - Batch insert optimization

- [ ] 2.6 Implement usage aggregator
  - `internal/module/billing/usage/aggregator.go` - Statistics
  - Aggregate by time period, model, task type
  - TimescaleDB continuous aggregates (optional)

- [x] 2.7 Implement billing service
  - `internal/module/billing/service.go` - Business logic
  - GetSubscription, CreateSubscription, UpgradePlan, DowngradePlan, CancelSubscription
  - GetQuotaStatus, CheckQuota, ConsumeQuota
  - GetUsageStats, RecordUsage
  - GetBalance, AddCredits, DeductCredits

- [x] 2.8 Implement billing handlers
  - `internal/module/billing/handler.go` - HTTP endpoints
  - GET /billing/plans
  - GET /billing/subscription, POST /billing/subscription/cancel
  - GET /billing/quota
  - GET /billing/usage

- [ ] 2.9 Write billing module tests
  - Repository tests
  - Service tests
  - Quota manager tests
  - Integration tests with Redis

## Phase 3: Order Module (P1)

- [x] 3.1 Create order module structure
  - `internal/module/order/model.go` - Order, OrderItem, Invoice
  - `internal/module/order/dto.go` - Request/response types
  - `internal/module/order/errors.go` - Module errors

- [x] 3.2 Implement order state machine
  - `internal/module/order/state_machine.go` - State transitions
  - Validate transitions, apply transitions

- [x] 3.3 Implement order repository
  - `internal/module/order/repository.go` - CRUD operations
  - Order methods: Create, GetByID, GetByOrderNo, ListByUser, Update
  - OrderItem methods: CreateBatch
  - Invoice methods: Create, GetByID, ListByUser

- [x] 3.4 Implement order service
  - `internal/module/order/service.go` - Business logic
  - CreateSubscriptionOrder, CreateTopupOrder, CreateUpgradeOrder
  - GetOrder, ListOrders
  - MarkAsPaid, CancelOrder, RefundOrder
  - GenerateInvoice, GetInvoice, ListInvoices

- [ ] 3.5 Implement invoice generator
  - `internal/module/order/invoice/generator.go` - Invoice creation
  - Number generation, PDF generation (optional)

- [x] 3.6 Implement order handlers
  - `internal/module/order/handler.go` - HTTP endpoints
  - POST /orders/subscription, POST /orders/topup
  - GET /orders, GET /orders/:id
  - POST /orders/:id/cancel
  - GET /invoices, GET /invoices/:id

- [ ] 3.7 Implement order expiration job
  - Background job to expire pending orders
  - Cron or ticker-based

- [ ] 3.8 Write order module tests
  - State machine tests
  - Repository tests
  - Service tests

## Phase 4: Payment Module (P1)

- [x] 4.1 Create payment module structure
  - `internal/module/payment/model.go` - Payment, StripeWebhookEvent, PaymentWebhookEvent
  - `internal/module/payment/dto.go` - Request/response types
  - `internal/module/payment/errors.go` - Module errors

- [x] 4.2 Implement payment provider interface
  - `internal/module/payment/provider/provider.go` - Interface definition
  - Methods: CreateCustomer, CreatePaymentIntent, CreateSubscription, etc.
  - NativePaymentProvider interface for Alipay/WeChat

- [x] 4.3 Implement Stripe provider
  - `internal/module/payment/provider/stripe.go` - Stripe implementation
  - Use stripe-go SDK
  - Map Stripe errors to internal errors

- [x] 4.4 Implement Alipay provider
  - `internal/module/payment/provider/alipay.go` - Alipay implementation
  - Use gopay SDK
  - Support web, h5, app, native (QR) scenes

- [x] 4.5 Implement WeChat Pay provider
  - `internal/module/payment/provider/wechat.go` - WeChat V3 implementation
  - Use gopay SDK
  - Support native, h5, app, mini, jsapi scenes

- [x] 4.6 Implement provider registry
  - `internal/module/payment/registry.go` - Provider registry
  - Register, Get, GetNative, GetByMethod methods

- [x] 4.7 Implement payment repository
  - `internal/module/payment/repository.go` - CRUD operations
  - Payment methods: Create, GetByID, Update
  - WebhookEvent methods: Create, Exists, MarkProcessed
  - PaymentWebhookEvent methods for native payments

- [x] 4.8 Implement payment service
  - `internal/module/payment/service.go` - Business logic
  - CreatePaymentIntent, ConfirmPayment
  - CreateNativePayment, HandleNativePaymentNotify
  - CreateRefund
  - AttachPaymentMethod, ListPaymentMethods, DetachPaymentMethod

- [x] 4.9 Implement webhook handler
  - `internal/module/payment/webhook_handler.go` - Webhook processing
  - Stripe signature verification
  - Alipay/WeChat notification handling
  - Idempotency via event_id

- [x] 4.10 Implement payment handlers
  - `internal/module/payment/handler.go` - HTTP endpoints
  - POST /payments/intent, POST /payments/native
  - GET /payments/methods, POST /payments/methods, DELETE /payments/methods/:id
  - POST /webhooks/stripe, POST /webhooks/alipay, POST /webhooks/wechat

- [ ] 4.11 Write payment module tests
  - Provider tests (with Stripe mocks)
  - Service tests
  - Webhook handler tests

## Phase 5: Integration (P1)

- [x] 5.1 Add configuration
  - `internal/shared/config/` - Stripe, Alipay, WeChat config
  - Environment variables for secrets

- [x] 5.2 Wire modules in app
  - `internal/app/app.go` - Module initialization
  - Dependency injection setup
  - Route registration

- [x] 5.3 Add quota middleware to AI routes
  - Apply QuotaChecker to /ai/* endpoints
  - Pass user context for quota lookup

- [ ] 5.4 Add billing hooks to AI service
  - Record usage after AI requests
  - Update quota counters

- [ ] 5.5 Integration tests
  - End-to-end registration flow
  - End-to-end subscription flow
  - Webhook processing tests

## Phase 6: Polish (P2)

- [ ] 6.1 Admin user management UI APIs
  - Enhance admin endpoints with more filters
  - Add bulk operations

- [ ] 6.2 Invoice PDF generation
  - PDF template
  - R2 storage for PDFs

- [ ] 6.3 Usage dashboard APIs
  - Detailed usage analytics
  - Cost breakdown

- [ ] 6.4 Documentation
  - API documentation (OpenAPI)
  - Integration guide for Stripe/Alipay/WeChat
