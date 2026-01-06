-- Payment Module Tables
-- Payments and Stripe webhook events

-- Payments table
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL,  -- in cents
    currency VARCHAR(10) DEFAULT 'usd',
    method VARCHAR(50) NOT NULL,  -- card, alipay, wechat
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending, processing, succeeded, failed, canceled, refunded
    provider VARCHAR(50) NOT NULL DEFAULT 'stripe',
    stripe_payment_intent_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),
    failure_code VARCHAR(100),
    failure_message TEXT,
    refunded_amount BIGINT DEFAULT 0,
    succeeded_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_user ON payments(user_id, created_at DESC);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_stripe_pi ON payments(stripe_payment_intent_id) WHERE stripe_payment_intent_id IS NOT NULL;
CREATE INDEX idx_payments_stripe_charge ON payments(stripe_charge_id) WHERE stripe_charge_id IS NOT NULL;

-- Stripe webhook events table (for idempotency)
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL UNIQUE,  -- Stripe event ID
    type VARCHAR(255) NOT NULL,
    data JSONB NOT NULL,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhook_events_event_id ON stripe_webhook_events(event_id);
CREATE INDEX idx_webhook_events_type ON stripe_webhook_events(type);
CREATE INDEX idx_webhook_events_processed ON stripe_webhook_events(processed) WHERE processed = false;

-- Trigger for payments updated_at
CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE payments IS 'Payment records for orders';
COMMENT ON TABLE stripe_webhook_events IS 'Stripe webhook events for idempotent processing';
COMMENT ON COLUMN payments.method IS 'Payment method: card, alipay, wechat';
COMMENT ON COLUMN payments.status IS 'Payment status: pending, processing, succeeded, failed, canceled, refunded';
COMMENT ON COLUMN stripe_webhook_events.event_id IS 'Stripe event ID for deduplication';
