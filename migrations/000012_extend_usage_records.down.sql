-- Remove API key tracking and cache statistics from usage_records

DROP INDEX IF EXISTS idx_usage_records_cache_hit;
DROP INDEX IF EXISTS idx_usage_records_api_key_id;

ALTER TABLE usage_records
DROP COLUMN IF EXISTS cache_hit;

ALTER TABLE usage_records
DROP COLUMN IF EXISTS api_key_id;
