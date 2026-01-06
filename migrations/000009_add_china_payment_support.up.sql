-- Add China payment support (Alipay/WeChat) to payment tables

-- Add trade_no and payer_id columns to payments table
ALTER TABLE payments
ADD COLUMN IF NOT EXISTS trade_no VARCHAR(255),
ADD COLUMN IF NOT EXISTS payer_id VARCHAR(255);

-- Add index on trade_no for native payment lookups
CREATE INDEX IF NOT EXISTS idx_payments_trade_no ON payments(trade_no) WHERE trade_no IS NOT NULL;

-- Payment webhook events table for native payments (Alipay/WeChat)
-- Separate from stripe_webhook_events for clarity
CREATE TABLE IF NOT EXISTS payment_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,  -- alipay, wechat
    event_id VARCHAR(255) NOT NULL,  -- Unique identifier (provider:trade_no)
    event_type VARCHAR(100) NOT NULL,  -- payment, refund
    trade_no VARCHAR(255),  -- Provider's trade number
    out_trade_no VARCHAR(255),  -- Our payment ID
    data JSONB NOT NULL,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uk_payment_webhook_events_event_id UNIQUE (provider, event_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_provider ON payment_webhook_events(provider);
CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_trade_no ON payment_webhook_events(trade_no) WHERE trade_no IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_out_trade_no ON payment_webhook_events(out_trade_no) WHERE out_trade_no IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_processed ON payment_webhook_events(processed) WHERE processed = false;

-- Comments
COMMENT ON TABLE payment_webhook_events IS 'Native payment (Alipay/WeChat) webhook events for idempotent processing';
COMMENT ON COLUMN payments.trade_no IS 'Provider trade number for native payments (Alipay/WeChat)';
COMMENT ON COLUMN payments.payer_id IS 'Payer ID from provider (openid for WeChat, buyer_id for Alipay)';
COMMENT ON COLUMN payment_webhook_events.event_id IS 'Unique event identifier in format provider:trade_no';
