-- Remove columns from system_api_keys
ALTER TABLE system_api_keys
DROP COLUMN IF EXISTS allowed_ips,
DROP COLUMN IF EXISTS rotate_after_days,
DROP COLUMN IF EXISTS last_rotated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_api_key_audit_time;
DROP INDEX IF EXISTS idx_api_key_audit_key;
DROP INDEX IF EXISTS idx_account_usage_stats_date;
DROP INDEX IF EXISTS idx_account_usage_stats_account;
DROP INDEX IF EXISTS idx_provider_accounts_health;
DROP INDEX IF EXISTS idx_provider_accounts_provider;

-- Drop tables in reverse order
DROP TABLE IF EXISTS api_key_audit_logs;
DROP TABLE IF EXISTS account_usage_stats;
DROP TABLE IF EXISTS provider_accounts;
