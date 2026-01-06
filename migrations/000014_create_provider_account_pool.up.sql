-- Provider account pool for managing multiple API keys per provider
CREATE TABLE IF NOT EXISTS provider_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    encrypted_api_key TEXT NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,

    -- Scheduling
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,

    -- Health monitoring
    health_status VARCHAR(20) DEFAULT 'healthy',
    last_health_check TIMESTAMP,
    consecutive_failures INT DEFAULT 0,
    last_failure_at TIMESTAMP,

    -- Rate limits (0 = use provider default)
    rate_limit_rpm INT DEFAULT 0,
    rate_limit_tpm INT DEFAULT 0,
    daily_limit INT DEFAULT 0,

    -- Usage statistics (denormalized)
    total_requests BIGINT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    total_cost_usd DECIMAL(12,6) DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_provider_account_name UNIQUE (provider_id, name)
);

CREATE INDEX idx_provider_accounts_provider ON provider_accounts(provider_id);
CREATE INDEX idx_provider_accounts_health ON provider_accounts(health_status, is_active);

-- Daily usage statistics per account
CREATE TABLE IF NOT EXISTS account_usage_stats (
    id BIGSERIAL PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES provider_accounts(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    requests_count BIGINT DEFAULT 0,
    tokens_count BIGINT DEFAULT 0,
    cost_usd DECIMAL(12,6) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_account_date UNIQUE (account_id, date)
);

CREATE INDEX idx_account_usage_stats_account ON account_usage_stats(account_id);
CREATE INDEX idx_account_usage_stats_date ON account_usage_stats(date);

-- API key audit logs
CREATE TABLE IF NOT EXISTS api_key_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_key_audit_key ON api_key_audit_logs(api_key_id);
CREATE INDEX idx_api_key_audit_time ON api_key_audit_logs(created_at);

-- Extend system_api_keys with IP whitelist and auto-rotation
ALTER TABLE system_api_keys
ADD COLUMN IF NOT EXISTS allowed_ips TEXT[] DEFAULT '{}',
ADD COLUMN IF NOT EXISTS rotate_after_days INT,
ADD COLUMN IF NOT EXISTS last_rotated_at TIMESTAMP;
