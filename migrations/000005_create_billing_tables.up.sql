-- Billing Module Tables
-- Plans, subscriptions, and usage tracking

-- Plans table (subscription tiers)
CREATE TABLE IF NOT EXISTS plans (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,  -- free, pro, team, enterprise
    name VARCHAR(255) NOT NULL,
    description TEXT,
    billing_cycle VARCHAR(50),  -- monthly, yearly, NULL for free
    price_usd BIGINT NOT NULL DEFAULT 0,  -- in cents
    stripe_price_id VARCHAR(255),
    monthly_tokens BIGINT NOT NULL DEFAULT 0,  -- -1 for unlimited
    daily_requests INT NOT NULL DEFAULT 0,     -- -1 for unlimited
    max_api_keys INT NOT NULL DEFAULT 3,
    features TEXT[] DEFAULT '{}',
    active BOOLEAN DEFAULT true,
    display_order INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_plans_type ON plans(type);
CREATE INDEX idx_plans_active ON plans(active) WHERE active = true;

-- Subscriptions table (user subscriptions)
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    plan_id VARCHAR(50) NOT NULL REFERENCES plans(id),
    status VARCHAR(50) NOT NULL DEFAULT 'active',  -- trialing, active, past_due, canceled, incomplete
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMP WITH TIME ZONE,
    credits_balance BIGINT DEFAULT 0,  -- in cents
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_stripe_sub ON subscriptions(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;
CREATE INDEX idx_subscriptions_stripe_cust ON subscriptions(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

-- Usage records table (TimescaleDB hypertable for time-series data)
CREATE TABLE IF NOT EXISTS usage_records (
    id BIGSERIAL,
    user_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    request_id VARCHAR(255) NOT NULL,
    task_type VARCHAR(50) NOT NULL,  -- chat, image, video, embedding
    provider_id UUID NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    cost_usd DECIMAL(10, 6) NOT NULL DEFAULT 0,
    latency_ms INT NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (id, timestamp)
);

-- Convert to TimescaleDB hypertable (if TimescaleDB is available)
-- This will fail gracefully if TimescaleDB is not installed
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('usage_records', 'timestamp', if_not_exists => TRUE);

        -- Add compression policy (compress chunks older than 7 days)
        ALTER TABLE usage_records SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'user_id'
        );
        PERFORM add_compression_policy('usage_records', INTERVAL '7 days', if_not_exists => TRUE);

        -- Add retention policy (keep 1 year of data)
        PERFORM add_retention_policy('usage_records', INTERVAL '1 year', if_not_exists => TRUE);
    END IF;
END $$;

-- Create indexes for usage_records
CREATE INDEX idx_usage_user_time ON usage_records(user_id, timestamp DESC);
CREATE INDEX idx_usage_task_type ON usage_records(task_type, timestamp DESC);
CREATE INDEX idx_usage_request ON usage_records(request_id);

-- Triggers for updated_at
CREATE TRIGGER update_plans_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE plans IS 'Subscription plan definitions with quotas and pricing';
COMMENT ON TABLE subscriptions IS 'User subscriptions linking users to plans';
COMMENT ON TABLE usage_records IS 'AI usage tracking for billing and analytics';
