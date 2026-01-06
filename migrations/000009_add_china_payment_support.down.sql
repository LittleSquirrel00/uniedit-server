-- Rollback China payment support

-- Drop payment_webhook_events table
DROP TABLE IF EXISTS payment_webhook_events;

-- Drop indexes on payments table
DROP INDEX IF EXISTS idx_payments_trade_no;

-- Remove trade_no and payer_id columns from payments table
ALTER TABLE payments
DROP COLUMN IF EXISTS trade_no,
DROP COLUMN IF EXISTS payer_id;
