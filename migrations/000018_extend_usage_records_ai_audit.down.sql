-- Rollback AI audit extensions on usage_records

DROP INDEX IF EXISTS idx_usage_records_provider_account_id;

ALTER TABLE usage_records DROP COLUMN IF EXISTS cost_multiplier;
ALTER TABLE usage_records DROP COLUMN IF EXISTS cache_read_input_tokens;
ALTER TABLE usage_records DROP COLUMN IF EXISTS cache_creation_input_tokens;
ALTER TABLE usage_records DROP COLUMN IF EXISTS provider_account_key_prefix;
ALTER TABLE usage_records DROP COLUMN IF EXISTS provider_account_name;
ALTER TABLE usage_records DROP COLUMN IF EXISTS provider_account_id;
ALTER TABLE usage_records DROP COLUMN IF EXISTS api_key_prefix;

