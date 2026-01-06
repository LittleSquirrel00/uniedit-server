-- Rollback Git Module Tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_pull_requests_updated_at ON pull_requests;
DROP TRIGGER IF EXISTS update_git_repos_updated_at ON git_repos;

-- Drop tables in reverse order (respect foreign key constraints)
DROP TABLE IF EXISTS pull_requests;
DROP TABLE IF EXISTS git_repo_collaborators;
DROP TABLE IF EXISTS lfs_locks;
DROP TABLE IF EXISTS lfs_repo_objects;
DROP TABLE IF EXISTS lfs_objects;
DROP TABLE IF EXISTS git_repos;
