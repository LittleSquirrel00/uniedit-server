-- Revert tasks table back to ai_tasks

-- Drop new indexes
DROP INDEX IF EXISTS idx_tasks_owner;
DROP INDEX IF EXISTS idx_tasks_type;
DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_external;

-- Remove metadata column
ALTER TABLE tasks DROP COLUMN IF EXISTS metadata;

-- Rename owner_id back to user_id
ALTER TABLE tasks RENAME COLUMN owner_id TO user_id;

-- Rename table back
ALTER TABLE tasks RENAME TO ai_tasks;

-- Recreate original indexes
CREATE INDEX idx_ai_tasks_user ON ai_tasks(user_id, created_at DESC);
CREATE INDEX idx_ai_tasks_status ON ai_tasks(status) WHERE status IN ('pending', 'running');
CREATE INDEX idx_ai_tasks_external ON ai_tasks(external_task_id) WHERE external_task_id IS NOT NULL;
