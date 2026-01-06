-- Rollback: Billing Module Tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_plans_updated_at ON plans;
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;

-- Drop indexes
DROP INDEX IF EXISTS idx_usage_user_time;
DROP INDEX IF EXISTS idx_usage_task_type;
DROP INDEX IF EXISTS idx_usage_request;
DROP INDEX IF EXISTS idx_subscriptions_user;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_stripe_sub;
DROP INDEX IF EXISTS idx_subscriptions_stripe_cust;
DROP INDEX IF EXISTS idx_plans_type;
DROP INDEX IF EXISTS idx_plans_active;

-- Drop tables in order (respecting foreign keys)
DROP TABLE IF EXISTS usage_records;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
