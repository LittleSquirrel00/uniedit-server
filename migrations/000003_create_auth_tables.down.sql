-- Rollback Auth Module Tables

DROP TRIGGER IF EXISTS update_user_api_keys_updated_at ON user_api_keys;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS user_api_keys;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
