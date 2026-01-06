-- Rollback: Payment Module Tables

-- Drop trigger
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;

-- Drop indexes
DROP INDEX IF EXISTS idx_payments_order;
DROP INDEX IF EXISTS idx_payments_user;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_stripe_pi;
DROP INDEX IF EXISTS idx_payments_stripe_charge;
DROP INDEX IF EXISTS idx_webhook_events_event_id;
DROP INDEX IF EXISTS idx_webhook_events_type;
DROP INDEX IF EXISTS idx_webhook_events_processed;

-- Drop tables
DROP TABLE IF EXISTS stripe_webhook_events;
DROP TABLE IF EXISTS payments;
