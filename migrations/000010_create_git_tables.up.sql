-- Git Module Tables
-- Git repositories, LFS objects, collaborators, and locks

-- Git repositories table
CREATE TABLE IF NOT EXISTS git_repos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    repo_type VARCHAR(50) NOT NULL DEFAULT 'code',  -- code, workflow, project
    visibility VARCHAR(50) NOT NULL DEFAULT 'private',  -- public, private
    description TEXT,
    default_branch VARCHAR(255) DEFAULT 'main',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    lfs_enabled BOOLEAN NOT NULL DEFAULT false,
    lfs_size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_path VARCHAR(512) NOT NULL,  -- R2 prefix: repos/{owner_id}/{repo_id}/
    stars_count INT NOT NULL DEFAULT 0,
    forks_count INT NOT NULL DEFAULT 0,
    forked_from UUID REFERENCES git_repos(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    pushed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT git_repos_owner_slug_unique UNIQUE (owner_id, slug)
);

CREATE INDEX idx_git_repos_owner ON git_repos(owner_id);
CREATE INDEX idx_git_repos_type ON git_repos(repo_type);
CREATE INDEX idx_git_repos_visibility ON git_repos(visibility);
CREATE INDEX idx_git_repos_forked_from ON git_repos(forked_from) WHERE forked_from IS NOT NULL;

-- LFS objects table (content-addressable deduplication)
CREATE TABLE IF NOT EXISTS lfs_objects (
    oid VARCHAR(64) PRIMARY KEY,  -- SHA-256 hash
    size BIGINT NOT NULL,
    storage_key VARCHAR(512) NOT NULL,  -- R2 key: lfs/{oid[0:2]}/{oid[2:4]}/{oid}
    content_type VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_lfs_objects_size ON lfs_objects(size);

-- Repository-LFS object association (many-to-many)
CREATE TABLE IF NOT EXISTS lfs_repo_objects (
    repo_id UUID NOT NULL REFERENCES git_repos(id) ON DELETE CASCADE,
    oid VARCHAR(64) NOT NULL REFERENCES lfs_objects(oid) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (repo_id, oid)
);

CREATE INDEX idx_lfs_repo_objects_oid ON lfs_repo_objects(oid);

-- LFS file locks table
CREATE TABLE IF NOT EXISTS lfs_locks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID NOT NULL REFERENCES git_repos(id) ON DELETE CASCADE,
    path VARCHAR(1024) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id),
    locked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT lfs_locks_repo_path_unique UNIQUE (repo_id, path)
);

CREATE INDEX idx_lfs_locks_repo ON lfs_locks(repo_id);
CREATE INDEX idx_lfs_locks_owner ON lfs_locks(owner_id);

-- Repository collaborators table
CREATE TABLE IF NOT EXISTS git_repo_collaborators (
    repo_id UUID NOT NULL REFERENCES git_repos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(50) NOT NULL DEFAULT 'read',  -- read, write, admin
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (repo_id, user_id)
);

CREATE INDEX idx_git_repo_collaborators_user ON git_repo_collaborators(user_id);

-- Pull requests table
CREATE TABLE IF NOT EXISTS pull_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID NOT NULL REFERENCES git_repos(id) ON DELETE CASCADE,
    number INT NOT NULL,
    title VARCHAR(512) NOT NULL,
    description TEXT,
    source_branch VARCHAR(255) NOT NULL,
    target_branch VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'open',  -- open, merged, closed
    author_id UUID NOT NULL REFERENCES users(id),
    merged_by UUID REFERENCES users(id),
    merged_at TIMESTAMP WITH TIME ZONE,
    closed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT pull_requests_repo_number_unique UNIQUE (repo_id, number)
);

CREATE INDEX idx_pull_requests_repo ON pull_requests(repo_id);
CREATE INDEX idx_pull_requests_status ON pull_requests(status);
CREATE INDEX idx_pull_requests_author ON pull_requests(author_id);

-- Triggers for updated_at
CREATE TRIGGER update_git_repos_updated_at
    BEFORE UPDATE ON git_repos
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pull_requests_updated_at
    BEFORE UPDATE ON pull_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE git_repos IS 'Git repositories with LFS support, stored on R2';
COMMENT ON TABLE lfs_objects IS 'LFS objects with content-addressable deduplication';
COMMENT ON TABLE lfs_repo_objects IS 'Many-to-many relationship between repos and LFS objects';
COMMENT ON TABLE lfs_locks IS 'LFS file locking for concurrent editing';
COMMENT ON TABLE git_repo_collaborators IS 'Repository collaboration permissions';
COMMENT ON TABLE pull_requests IS 'Pull requests for code review';

COMMENT ON COLUMN git_repos.repo_type IS 'Repository type: code, workflow, or project';
COMMENT ON COLUMN git_repos.storage_path IS 'R2 storage prefix for this repository';
COMMENT ON COLUMN git_repos.lfs_size_bytes IS 'Total size of LFS objects in this repository';
COMMENT ON COLUMN lfs_objects.oid IS 'SHA-256 hash of file content, used as unique identifier';
COMMENT ON COLUMN lfs_objects.storage_key IS 'R2 object key with sharded structure';
COMMENT ON COLUMN git_repo_collaborators.permission IS 'Permission level: read, write, or admin';
