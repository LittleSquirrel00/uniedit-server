-- Add API key tracking and cache statistics to usage_records

-- Add api_key_id column to track which API key was used
ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS api_key_id UUID REFERENCES system_api_keys(id) ON DELETE SET NULL;

-- Add cache_hit column to track cache hits
ALTER TABLE usage_records
ADD COLUMN IF NOT EXISTS cache_hit BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index for API key usage queries
CREATE INDEX IF NOT EXISTS idx_usage_records_api_key_id ON usage_records(api_key_id);

-- Create index for cache hit analysis
CREATE INDEX IF NOT EXISTS idx_usage_records_cache_hit ON usage_records(cache_hit) WHERE cache_hit = TRUE;
