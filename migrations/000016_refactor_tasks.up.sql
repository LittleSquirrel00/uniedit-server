-- Rename ai_tasks to tasks and add metadata column for generic task support
-- This migration makes the task system a shared infrastructure

-- Rename table
ALTER TABLE ai_tasks RENAME TO tasks;

-- Rename user_id to owner_id for generic ownership
ALTER TABLE tasks RENAME COLUMN user_id TO owner_id;

-- Add metadata column for extensibility
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Update type column to VARCHAR(100) for more flexibility
ALTER TABLE tasks ALTER COLUMN type TYPE VARCHAR(100);

-- Update indexes
DROP INDEX IF EXISTS idx_ai_tasks_user;
DROP INDEX IF EXISTS idx_ai_tasks_status;
DROP INDEX IF EXISTS idx_ai_tasks_external;

CREATE INDEX idx_tasks_owner ON tasks(owner_id, created_at DESC);
CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_status ON tasks(status) WHERE status IN ('pending', 'running');
CREATE INDEX idx_tasks_external ON tasks(external_task_id) WHERE external_task_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON TABLE tasks IS 'Generic async task management table for all modules (AI, media, workflow, etc.)';
COMMENT ON COLUMN tasks.owner_id IS 'Owner of the task (user_id, team_id, or system_id)';
COMMENT ON COLUMN tasks.type IS 'Task type identifier, e.g., image_generation, video_generation, chat';
COMMENT ON COLUMN tasks.metadata IS 'Additional task metadata (provider_id, model_id, etc.)';
