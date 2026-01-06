-- System API Keys table (OpenAI-style API key authentication)
CREATE TABLE IF NOT EXISTS system_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,           -- SHA-256 hash of the full key
    key_prefix VARCHAR(12) NOT NULL,                -- sk-abc12345... (for identification)
    scopes TEXT[] DEFAULT '{chat,image,embedding}', -- Permission scopes

    -- Rate limiting configuration
    rate_limit_rpm INT DEFAULT 60,                  -- Requests per minute
    rate_limit_tpm INT DEFAULT 100000,              -- Tokens per minute

    -- Usage statistics (denormalized for quick access)
    total_requests BIGINT DEFAULT 0,
    total_input_tokens BIGINT DEFAULT 0,
    total_output_tokens BIGINT DEFAULT 0,
    total_cost_usd DECIMAL(12, 6) DEFAULT 0,

    -- Cache statistics
    cache_hits BIGINT DEFAULT 0,
    cache_misses BIGINT DEFAULT 0,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,            -- NULL = never expires

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_system_api_keys_user_id ON system_api_keys(user_id);
CREATE INDEX idx_system_api_keys_key_hash ON system_api_keys(key_hash);
CREATE INDEX idx_system_api_keys_prefix ON system_api_keys(key_prefix);
CREATE INDEX idx_system_api_keys_active ON system_api_keys(is_active) WHERE is_active = TRUE;

-- Extend usage_records to track API key usage
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS api_key_id UUID REFERENCES system_api_keys(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key_id ON usage_records(api_key_id);

-- Add cache statistics columns to usage_records
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS cache_hit BOOLEAN DEFAULT FALSE;
ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS cached_tokens INT DEFAULT 0;

-- Trigger to update usage statistics on system_api_keys
CREATE OR REPLACE FUNCTION update_api_key_usage_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.api_key_id IS NOT NULL THEN
        UPDATE system_api_keys
        SET
            total_requests = total_requests + 1,
            total_input_tokens = total_input_tokens + COALESCE(NEW.input_tokens, 0),
            total_output_tokens = total_output_tokens + COALESCE(NEW.output_tokens, 0),
            total_cost_usd = total_cost_usd + COALESCE(NEW.cost_usd, 0),
            cache_hits = cache_hits + CASE WHEN NEW.cache_hit THEN 1 ELSE 0 END,
            cache_misses = cache_misses + CASE WHEN NOT COALESCE(NEW.cache_hit, FALSE) THEN 1 ELSE 0 END,
            last_used_at = NEW.timestamp,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.api_key_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_api_key_usage
    AFTER INSERT ON usage_records
    FOR EACH ROW
    EXECUTE FUNCTION update_api_key_usage_stats();

-- Comment
COMMENT ON TABLE system_api_keys IS 'System-generated API keys for API access authentication (OpenAI-style sk-xxx keys)';
