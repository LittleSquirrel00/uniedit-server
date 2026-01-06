-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_update_api_key_usage ON usage_records;
DROP FUNCTION IF EXISTS update_api_key_usage_stats();

-- Remove columns from usage_records
ALTER TABLE usage_records DROP COLUMN IF EXISTS cached_tokens;
ALTER TABLE usage_records DROP COLUMN IF EXISTS cache_hit;
ALTER TABLE usage_records DROP COLUMN IF EXISTS api_key_id;

-- Drop system_api_keys table
DROP TABLE IF EXISTS system_api_keys;
