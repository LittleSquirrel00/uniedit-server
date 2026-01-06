-- Enhance billing quotas with task-specific limits, storage limits, and team member limits

-- Add task-specific quota columns to plans table
ALTER TABLE plans ADD COLUMN IF NOT EXISTS monthly_chat_tokens BIGINT DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS monthly_image_credits INTEGER DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS monthly_video_minutes INTEGER DEFAULT 0;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS monthly_embedding_tokens BIGINT DEFAULT 0;

-- Add storage quota columns to plans table
ALTER TABLE plans ADD COLUMN IF NOT EXISTS git_storage_mb BIGINT DEFAULT -1;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS lfs_storage_mb BIGINT DEFAULT -1;

-- Add team member quota column to plans table
ALTER TABLE plans ADD COLUMN IF NOT EXISTS max_team_members INTEGER DEFAULT 5;

-- Update existing plans with appropriate quotas
-- Free plan: minimal quotas
UPDATE plans SET
    monthly_chat_tokens = 10000,
    monthly_image_credits = 10,
    monthly_video_minutes = 5,
    monthly_embedding_tokens = 10000,
    git_storage_mb = 100,
    lfs_storage_mb = 500,
    max_team_members = 1
WHERE type = 'free';

-- Pro plan: moderate quotas
UPDATE plans SET
    monthly_chat_tokens = 500000,
    monthly_image_credits = 200,
    monthly_video_minutes = 60,
    monthly_embedding_tokens = 500000,
    git_storage_mb = 5000,
    lfs_storage_mb = 10000,
    max_team_members = 5
WHERE type = 'pro';

-- Team plan: higher quotas
UPDATE plans SET
    monthly_chat_tokens = 2000000,
    monthly_image_credits = 1000,
    monthly_video_minutes = 300,
    monthly_embedding_tokens = 2000000,
    git_storage_mb = 50000,
    lfs_storage_mb = 100000,
    max_team_members = 50
WHERE type = 'team';

-- Enterprise plan: unlimited (-1)
UPDATE plans SET
    monthly_chat_tokens = -1,
    monthly_image_credits = -1,
    monthly_video_minutes = -1,
    monthly_embedding_tokens = -1,
    git_storage_mb = -1,
    lfs_storage_mb = -1,
    max_team_members = -1
WHERE type = 'enterprise';

-- Add comments for documentation
COMMENT ON COLUMN plans.monthly_chat_tokens IS 'Monthly chat/completion token limit (-1=unlimited, 0=use monthly_tokens)';
COMMENT ON COLUMN plans.monthly_image_credits IS 'Monthly image generation limit (-1=unlimited)';
COMMENT ON COLUMN plans.monthly_video_minutes IS 'Monthly video generation minutes limit (-1=unlimited)';
COMMENT ON COLUMN plans.monthly_embedding_tokens IS 'Monthly embedding token limit (-1=unlimited)';
COMMENT ON COLUMN plans.git_storage_mb IS 'Git repository storage limit in MB (-1=unlimited)';
COMMENT ON COLUMN plans.lfs_storage_mb IS 'LFS object storage limit in MB (-1=unlimited)';
COMMENT ON COLUMN plans.max_team_members IS 'Maximum team members allowed (-1=unlimited)';
