-- Revert enhanced billing quotas

-- Remove comments first
COMMENT ON COLUMN plans.monthly_chat_tokens IS NULL;
COMMENT ON COLUMN plans.monthly_image_credits IS NULL;
COMMENT ON COLUMN plans.monthly_video_minutes IS NULL;
COMMENT ON COLUMN plans.monthly_embedding_tokens IS NULL;
COMMENT ON COLUMN plans.git_storage_mb IS NULL;
COMMENT ON COLUMN plans.lfs_storage_mb IS NULL;
COMMENT ON COLUMN plans.max_team_members IS NULL;

-- Remove task-specific quota columns
ALTER TABLE plans DROP COLUMN IF EXISTS monthly_chat_tokens;
ALTER TABLE plans DROP COLUMN IF EXISTS monthly_image_credits;
ALTER TABLE plans DROP COLUMN IF EXISTS monthly_video_minutes;
ALTER TABLE plans DROP COLUMN IF EXISTS monthly_embedding_tokens;

-- Remove storage quota columns
ALTER TABLE plans DROP COLUMN IF EXISTS git_storage_mb;
ALTER TABLE plans DROP COLUMN IF EXISTS lfs_storage_mb;

-- Remove team member quota column
ALTER TABLE plans DROP COLUMN IF EXISTS max_team_members;
