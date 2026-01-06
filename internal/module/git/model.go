package git

import (
	"time"

	"github.com/google/uuid"
)

// RepoType represents the type of repository.
type RepoType string

const (
	RepoTypeCode     RepoType = "code"
	RepoTypeWorkflow RepoType = "workflow"
	RepoTypeProject  RepoType = "project"
)

// Visibility represents repository visibility.
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// Permission represents collaboration permission level.
type Permission string

const (
	PermissionRead  Permission = "read"
	PermissionWrite Permission = "write"
	PermissionAdmin Permission = "admin"
)

// GitRepo represents a Git repository.
type GitRepo struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID       uuid.UUID  `json:"owner_id" gorm:"type:uuid;not null"`
	Name          string     `json:"name" gorm:"not null"`
	Slug          string     `json:"slug" gorm:"not null"`
	RepoType      RepoType   `json:"repo_type" gorm:"column:repo_type;not null;default:code"`
	Visibility    Visibility `json:"visibility" gorm:"not null;default:private"`
	Description   string     `json:"description,omitempty"`
	DefaultBranch string     `json:"default_branch" gorm:"default:main"`
	SizeBytes     int64      `json:"size_bytes" gorm:"default:0"`
	LFSEnabled    bool       `json:"lfs_enabled" gorm:"default:false"`
	LFSSizeBytes  int64      `json:"lfs_size_bytes" gorm:"default:0"`
	StoragePath   string     `json:"-" gorm:"not null"` // R2 prefix
	StarsCount    int        `json:"stars_count" gorm:"default:0"`
	ForksCount    int        `json:"forks_count" gorm:"default:0"`
	ForkedFrom    *uuid.UUID `json:"forked_from,omitempty" gorm:"type:uuid"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	PushedAt      *time.Time `json:"pushed_at,omitempty"`

	// Relations (not loaded by default)
	Collaborators []RepoCollaborator `json:"collaborators,omitempty" gorm:"foreignKey:RepoID"`
}

// TableName returns the database table name.
func (GitRepo) TableName() string {
	return "git_repos"
}

// TotalSize returns total storage (Git + LFS).
func (r *GitRepo) TotalSize() int64 {
	return r.SizeBytes + r.LFSSizeBytes
}

// FullName returns owner/repo format.
func (r *GitRepo) FullName(ownerName string) string {
	return ownerName + "/" + r.Slug
}

// LFSObject represents an LFS object (content-addressable).
type LFSObject struct {
	OID         string    `json:"oid" gorm:"primaryKey"`          // SHA-256
	Size        int64     `json:"size" gorm:"not null"`
	StorageKey  string    `json:"-" gorm:"not null"`              // R2 key
	ContentType string    `json:"content_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName returns the database table name.
func (LFSObject) TableName() string {
	return "lfs_objects"
}

// LFSRepoObject represents the repo-object association.
type LFSRepoObject struct {
	RepoID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	OID       string    `gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the database table name.
func (LFSRepoObject) TableName() string {
	return "lfs_repo_objects"
}

// LFSLock represents a file lock.
type LFSLock struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepoID   uuid.UUID `json:"repo_id" gorm:"type:uuid;not null"`
	Path     string    `json:"path" gorm:"not null"`
	OwnerID  uuid.UUID `json:"owner_id" gorm:"type:uuid;not null"`
	LockedAt time.Time `json:"locked_at"`
}

// TableName returns the database table name.
func (LFSLock) TableName() string {
	return "lfs_locks"
}

// RepoCollaborator represents a collaborator.
type RepoCollaborator struct {
	RepoID     uuid.UUID  `json:"repo_id" gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;primaryKey"`
	Permission Permission `json:"permission" gorm:"not null;default:read"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TableName returns the database table name.
func (RepoCollaborator) TableName() string {
	return "git_repo_collaborators"
}

// PRStatus represents pull request status.
type PRStatus string

const (
	PRStatusOpen   PRStatus = "open"
	PRStatusMerged PRStatus = "merged"
	PRStatusClosed PRStatus = "closed"
)

// PullRequest represents a pull request.
type PullRequest struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepoID       uuid.UUID  `json:"repo_id" gorm:"type:uuid;not null"`
	Number       int        `json:"number" gorm:"not null"`
	Title        string     `json:"title" gorm:"not null"`
	Description  string     `json:"description,omitempty"`
	SourceBranch string     `json:"source_branch" gorm:"not null"`
	TargetBranch string     `json:"target_branch" gorm:"not null"`
	Status       PRStatus   `json:"status" gorm:"not null;default:open"`
	AuthorID     uuid.UUID  `json:"author_id" gorm:"type:uuid;not null"`
	MergedBy     *uuid.UUID `json:"merged_by,omitempty" gorm:"type:uuid"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName returns the database table name.
func (PullRequest) TableName() string {
	return "pull_requests"
}
