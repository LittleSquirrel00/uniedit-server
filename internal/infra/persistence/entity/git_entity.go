package entity

import (
	"time"

	"github.com/google/uuid"
)

// GitRepoEntity represents the database entity for git repositories.
type GitRepoEntity struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	Name          string     `gorm:"not null"`
	Slug          string     `gorm:"not null;index:idx_git_repos_owner_slug,unique,where:deleted_at IS NULL"`
	RepoType      string     `gorm:"column:repo_type;not null;default:code"`
	Visibility    string     `gorm:"not null;default:private"`
	Description   string     `gorm:""`
	DefaultBranch string     `gorm:"default:main"`
	SizeBytes     int64      `gorm:"default:0"`
	LFSEnabled    bool       `gorm:"default:false"`
	LFSSizeBytes  int64      `gorm:"default:0"`
	StoragePath   string     `gorm:"not null"`
	StarsCount    int        `gorm:"default:0"`
	ForksCount    int        `gorm:"default:0"`
	ForkedFrom    *uuid.UUID `gorm:"type:uuid"`
	CreatedAt     time.Time  `gorm:""`
	UpdatedAt     time.Time  `gorm:""`
	PushedAt      *time.Time `gorm:""`
	DeletedAt     *time.Time `gorm:"index"`
}

// TableName returns the table name.
func (GitRepoEntity) TableName() string {
	return "git_repos"
}

// GitCollaboratorEntity represents the database entity for collaborators.
type GitCollaboratorEntity struct {
	RepoID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	Permission string    `gorm:"not null;default:read"`
	CreatedAt  time.Time `gorm:""`
}

// TableName returns the table name.
func (GitCollaboratorEntity) TableName() string {
	return "git_repo_collaborators"
}

// PullRequestEntity represents the database entity for pull requests.
type PullRequestEntity struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepoID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	Number       int        `gorm:"not null;index:idx_pull_requests_repo_number,unique"`
	Title        string     `gorm:"not null"`
	Description  string     `gorm:""`
	SourceBranch string     `gorm:"not null"`
	TargetBranch string     `gorm:"not null"`
	Status       string     `gorm:"not null;default:open"`
	AuthorID     uuid.UUID  `gorm:"type:uuid;not null"`
	MergedBy     *uuid.UUID `gorm:"type:uuid"`
	MergedAt     *time.Time `gorm:""`
	ClosedAt     *time.Time `gorm:""`
	CreatedAt    time.Time  `gorm:""`
	UpdatedAt    time.Time  `gorm:""`
}

// TableName returns the table name.
func (PullRequestEntity) TableName() string {
	return "pull_requests"
}

// LFSObjectEntity represents the database entity for LFS objects.
type LFSObjectEntity struct {
	OID         string    `gorm:"primaryKey"`
	Size        int64     `gorm:"not null"`
	StorageKey  string    `gorm:"not null"`
	ContentType string    `gorm:""`
	CreatedAt   time.Time `gorm:""`
}

// TableName returns the table name.
func (LFSObjectEntity) TableName() string {
	return "lfs_objects"
}

// LFSRepoObjectEntity represents the association between repo and LFS object.
type LFSRepoObjectEntity struct {
	RepoID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	OID       string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:""`
}

// TableName returns the table name.
func (LFSRepoObjectEntity) TableName() string {
	return "lfs_repo_objects"
}

// LFSLockEntity represents the database entity for LFS locks.
type LFSLockEntity struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepoID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Path     string    `gorm:"not null"`
	OwnerID  uuid.UUID `gorm:"type:uuid;not null"`
	LockedAt time.Time `gorm:""`
}

// TableName returns the table name.
func (LFSLockEntity) TableName() string {
	return "lfs_locks"
}
