-- Rollback: Extend Users Table

-- Drop email verifications table
DROP TABLE IF EXISTS email_verifications;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_is_admin;

-- Remove columns from users table
ALTER TABLE users
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS email_verified,
    DROP COLUMN IF EXISTS is_admin,
    DROP COLUMN IF EXISTS suspended_at,
    DROP COLUMN IF EXISTS suspend_reason,
    DROP COLUMN IF EXISTS deleted_at;

-- Restore NOT NULL constraints on OAuth fields
-- Note: This may fail if there are email-only users
ALTER TABLE users
    ALTER COLUMN oauth_provider SET NOT NULL,
    ALTER COLUMN oauth_id SET NOT NULL;
