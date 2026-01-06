-- Extend Users Table for Email Registration and User Management
-- Adds support for email/password registration alongside OAuth

-- Add new columns to users table
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS password_hash TEXT,
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS suspend_reason TEXT,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Make OAuth fields nullable for email registration users
ALTER TABLE users
    ALTER COLUMN oauth_provider DROP NOT NULL,
    ALTER COLUMN oauth_id DROP NOT NULL;

-- Add indexes for new columns
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin) WHERE is_admin = true;

-- Email verification tokens table
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    purpose VARCHAR(50) NOT NULL,  -- registration, password_reset
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token);
CREATE INDEX idx_email_verifications_user ON email_verifications(user_id);
CREATE INDEX idx_email_verifications_expires ON email_verifications(expires_at) WHERE used_at IS NULL;

-- Comment on columns
COMMENT ON COLUMN users.status IS 'User status: active, pending, suspended, deleted';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hash for email/password users';
COMMENT ON COLUMN users.email_verified IS 'Whether email has been verified';
COMMENT ON COLUMN users.is_admin IS 'Whether user has admin privileges';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp';
COMMENT ON TABLE email_verifications IS 'Email verification and password reset tokens';
