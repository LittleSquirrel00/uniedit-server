# Change: Add User Management, Billing, Order, and Payment Modules

## Why

The current system only supports OAuth authentication. To enable monetization and user self-service, we need:
1. Email/password registration alongside OAuth
2. Subscription-based billing with usage quotas
3. Order management for purchases and invoices
4. Stripe payment integration for subscriptions and top-ups

## What Changes

### New Capabilities
- **user-management**: Email registration, email verification, password management, user status (active/suspended/deleted), admin operations
- **billing**: Subscription plans, quota management, usage tracking, credits balance
- **order**: Order lifecycle, invoice generation, order items
- **payment**: Stripe integration, webhook handling, payment methods, refunds

### Existing Changes
- **auth module**: Extend User model with status, password_hash, email_verified fields; make OAuth fields nullable

### **BREAKING** Changes
- Database schema changes to `users` table (migration required)
- New middleware for quota checking on AI endpoints

## Impact

- Affected specs: None existing (4 new capabilities)
- Affected code:
  - `internal/module/auth/model.go` - User model extension
  - `internal/module/user/` - New module
  - `internal/module/billing/` - New module
  - `internal/module/order/` - New module
  - `internal/module/payment/` - New module
  - `migrations/` - 5 new migration files
  - `internal/shared/config/` - Stripe and email configuration
