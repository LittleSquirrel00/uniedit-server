package model

import (
	"time"

	"github.com/google/uuid"
)

// ===== Repository Types =====

// GitRepoType represents the type of repository.
type GitRepoType string

const (
	GitRepoTypeCode     GitRepoType = "code"
	GitRepoTypeWorkflow GitRepoType = "workflow"
	GitRepoTypeProject  GitRepoType = "project"
)

// GitVisibility represents repository visibility.
type GitVisibility string

const (
	GitVisibilityPublic  GitVisibility = "public"
	GitVisibilityPrivate GitVisibility = "private"
)

// GitPermission represents collaboration permission level.
type GitPermission string

const (
	GitPermissionRead  GitPermission = "read"
	GitPermissionWrite GitPermission = "write"
	GitPermissionAdmin GitPermission = "admin"
)

// CanWrite returns true if the permission allows write access.
func (p GitPermission) CanWrite() bool {
	return p == GitPermissionWrite || p == GitPermissionAdmin
}

// CanAdmin returns true if the permission allows admin access.
func (p GitPermission) CanAdmin() bool {
	return p == GitPermissionAdmin
}

// ===== Repository Entity =====

// GitRepo represents a Git repository.
type GitRepo struct {
	ID            uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID       uuid.UUID     `json:"owner_id" gorm:"type:uuid;not null;index"`
	Name          string        `json:"name" gorm:"not null"`
	Slug          string        `json:"slug" gorm:"not null;index"`
	RepoType      GitRepoType   `json:"repo_type" gorm:"column:repo_type;not null;default:code"`
	Visibility    GitVisibility `json:"visibility" gorm:"not null;default:private"`
	Description   string        `json:"description,omitempty"`
	DefaultBranch string        `json:"default_branch" gorm:"default:main"`
	SizeBytes     int64         `json:"size_bytes" gorm:"default:0"`
	LFSEnabled    bool          `json:"lfs_enabled" gorm:"default:false"`
	LFSSizeBytes  int64         `json:"lfs_size_bytes" gorm:"default:0"`
	StoragePath   string        `json:"-" gorm:"not null"` // R2/S3 prefix
	StarsCount    int           `json:"stars_count" gorm:"default:0"`
	ForksCount    int           `json:"forks_count" gorm:"default:0"`
	ForkedFrom    *uuid.UUID    `json:"forked_from,omitempty" gorm:"type:uuid"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	PushedAt      *time.Time    `json:"pushed_at,omitempty"`

	// Relations (not loaded by default)
	Collaborators []*GitRepoCollaborator `json:"collaborators,omitempty" gorm:"foreignKey:RepoID"`
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

// IsPublic returns true if the repository is public.
func (r *GitRepo) IsPublic() bool {
	return r.Visibility == GitVisibilityPublic
}

// ===== Collaborator Entity =====

// GitRepoCollaborator represents a repository collaborator.
type GitRepoCollaborator struct {
	RepoID     uuid.UUID     `json:"repo_id" gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID     `json:"user_id" gorm:"type:uuid;primaryKey"`
	Permission GitPermission `json:"permission" gorm:"not null;default:read"`
	CreatedAt  time.Time     `json:"created_at"`
}

// TableName returns the database table name.
func (GitRepoCollaborator) TableName() string {
	return "git_repo_collaborators"
}

// ===== Pull Request Types =====

// GitPRStatus represents pull request status.
type GitPRStatus string

const (
	GitPRStatusOpen   GitPRStatus = "open"
	GitPRStatusMerged GitPRStatus = "merged"
	GitPRStatusClosed GitPRStatus = "closed"
)

// IsOpen returns true if the PR is open.
func (s GitPRStatus) IsOpen() bool {
	return s == GitPRStatusOpen
}

// IsMerged returns true if the PR is merged.
func (s GitPRStatus) IsMerged() bool {
	return s == GitPRStatusMerged
}

// ===== Pull Request Entity =====

// GitPullRequest represents a pull request.
type GitPullRequest struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepoID       uuid.UUID   `json:"repo_id" gorm:"type:uuid;not null;index"`
	Number       int         `json:"number" gorm:"not null"`
	Title        string      `json:"title" gorm:"not null"`
	Description  string      `json:"description,omitempty"`
	SourceBranch string      `json:"source_branch" gorm:"not null"`
	TargetBranch string      `json:"target_branch" gorm:"not null"`
	Status       GitPRStatus `json:"status" gorm:"not null;default:open"`
	AuthorID     uuid.UUID   `json:"author_id" gorm:"type:uuid;not null"`
	MergedBy     *uuid.UUID  `json:"merged_by,omitempty" gorm:"type:uuid"`
	MergedAt     *time.Time  `json:"merged_at,omitempty"`
	ClosedAt     *time.Time  `json:"closed_at,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// TableName returns the database table name.
func (GitPullRequest) TableName() string {
	return "pull_requests"
}

// ===== LFS Types =====

// GitLFSObject represents an LFS object (content-addressable).
type GitLFSObject struct {
	OID         string    `json:"oid" gorm:"primaryKey"`           // SHA-256
	Size        int64     `json:"size" gorm:"not null"`
	StorageKey  string    `json:"-" gorm:"not null"`               // R2/S3 key
	ContentType string    `json:"content_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName returns the database table name.
func (GitLFSObject) TableName() string {
	return "lfs_objects"
}

// GitLFSRepoObject represents the repository-LFS object association.
type GitLFSRepoObject struct {
	RepoID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	OID       string    `gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the database table name.
func (GitLFSRepoObject) TableName() string {
	return "lfs_repo_objects"
}

// GitLFSLock represents a file lock in LFS.
type GitLFSLock struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RepoID   uuid.UUID `json:"repo_id" gorm:"type:uuid;not null;index"`
	Path     string    `json:"path" gorm:"not null"`
	OwnerID  uuid.UUID `json:"owner_id" gorm:"type:uuid;not null"`
	LockedAt time.Time `json:"locked_at"`
}

// TableName returns the database table name.
func (GitLFSLock) TableName() string {
	return "lfs_locks"
}

// ===== Git Service Types =====

// GitService represents git protocol service type.
type GitService string

const (
	GitServiceReceivePack GitService = "git-receive-pack"
	GitServiceUploadPack  GitService = "git-upload-pack"
)

// IsValid validates the git service.
func (s GitService) IsValid() bool {
	return s == GitServiceReceivePack || s == GitServiceUploadPack
}

// IsWrite returns true if this service requires write access.
func (s GitService) IsWrite() bool {
	return s == GitServiceReceivePack
}

// ===== Request/Response Types for LFS =====

// GitLFSBatchRequest represents an LFS batch API request.
type GitLFSBatchRequest struct {
	Operation string             `json:"operation"` // download or upload
	Transfers []string           `json:"transfers,omitempty"`
	Objects   []*GitLFSPointer   `json:"objects"`
	Ref       *GitLFSRef         `json:"ref,omitempty"`
}

// GitLFSBatchResponse represents an LFS batch API response.
type GitLFSBatchResponse struct {
	Transfer string                    `json:"transfer,omitempty"`
	Objects  []*GitLFSObjectResponse   `json:"objects"`
}

// GitLFSPointer represents an LFS pointer.
type GitLFSPointer struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

// GitLFSRef represents a git ref in LFS request.
type GitLFSRef struct {
	Name string `json:"name"`
}

// GitLFSObjectResponse represents an LFS object response.
type GitLFSObjectResponse struct {
	OID           string              `json:"oid"`
	Size          int64               `json:"size"`
	Authenticated bool                `json:"authenticated,omitempty"`
	Actions       map[string]*GitLFSAction `json:"actions,omitempty"`
	Error         *GitLFSError        `json:"error,omitempty"`
}

// GitLFSAction represents an LFS action (upload/download/verify).
type GitLFSAction struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

// GitLFSError represents an LFS error response.
type GitLFSError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GitLFSLockRequest represents an LFS lock request.
type GitLFSLockRequest struct {
	Path string     `json:"path"`
	Ref  *GitLFSRef `json:"ref,omitempty"`
}

// GitLFSLockResponse represents an LFS lock response.
type GitLFSLockResponse struct {
	Lock *GitLFSLockInfo `json:"lock"`
}

// GitLFSLockInfo represents lock information in API response.
type GitLFSLockInfo struct {
	ID       string         `json:"id"`
	Path     string         `json:"path"`
	LockedAt time.Time      `json:"locked_at"`
	Owner    *GitLFSOwner   `json:"owner,omitempty"`
}

// GitLFSOwner represents the lock owner.
type GitLFSOwner struct {
	Name string `json:"name"`
}

// GitLFSLocksResponse represents a list locks response.
type GitLFSLocksResponse struct {
	Locks      []*GitLFSLockInfo `json:"locks"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// GitLFSVerifyLocksResponse represents a verify locks response.
type GitLFSVerifyLocksResponse struct {
	Ours       []*GitLFSLockInfo `json:"ours"`
	Theirs     []*GitLFSLockInfo `json:"theirs"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// ===== Access Check Result =====

// GitAccessResult represents the result of an access check.
type GitAccessResult struct {
	Allowed    bool
	Permission GitPermission
	Reason     string
}

// NewGitAccessAllowed creates a successful access result.
func NewGitAccessAllowed(permission GitPermission) *GitAccessResult {
	return &GitAccessResult{
		Allowed:    true,
		Permission: permission,
	}
}

// NewGitAccessDenied creates a denied access result.
func NewGitAccessDenied(reason string) *GitAccessResult {
	return &GitAccessResult{
		Allowed: false,
		Reason:  reason,
	}
}
