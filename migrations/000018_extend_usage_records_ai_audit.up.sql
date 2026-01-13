-- Extend usage_records for AI audit details (account pool, cache, multiplier)

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS api_key_prefix VARCHAR(32);

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS provider_account_id UUID;

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS provider_account_name VARCHAR(255);

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS provider_account_key_prefix VARCHAR(32);

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS cache_creation_input_tokens INT NOT NULL DEFAULT 0;

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS cache_read_input_tokens INT NOT NULL DEFAULT 0;

ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS cost_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_usage_records_provider_account_id ON usage_records(provider_account_id);

