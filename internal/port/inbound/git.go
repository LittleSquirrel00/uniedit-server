package inbound

import (
	"context"
	"io"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// ===== Git Domain Port =====

// GitDomain defines the main Git domain operations.
type GitDomain interface {
	// Repository operations
	CreateRepo(ctx context.Context, ownerID uuid.UUID, input *GitCreateRepoInput) (*model.GitRepo, error)
	GetRepo(ctx context.Context, id uuid.UUID) (*model.GitRepo, error)
	GetRepoByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error)
	ListRepos(ctx context.Context, ownerID uuid.UUID, filter *GitRepoFilter) ([]*model.GitRepo, int64, error)
	ListPublicRepos(ctx context.Context, filter *GitRepoFilter) ([]*model.GitRepo, int64, error)
	UpdateRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID, input *GitUpdateRepoInput) (*model.GitRepo, error)
	DeleteRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// Access control
	CheckAccess(ctx context.Context, repoID, userID uuid.UUID, required model.GitPermission) error
	CanAccess(ctx context.Context, repoID uuid.UUID, userID *uuid.UUID, required model.GitPermission) (bool, error)

	// Collaborator operations
	AddCollaborator(ctx context.Context, repoID, ownerID, targetUserID uuid.UUID, permission model.GitPermission) error
	ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*model.GitRepoCollaborator, error)
	UpdateCollaborator(ctx context.Context, repoID, ownerID, targetUserID uuid.UUID, permission model.GitPermission) error
	RemoveCollaborator(ctx context.Context, repoID, ownerID, targetUserID uuid.UUID) error

	// Pull request operations
	CreatePR(ctx context.Context, repoID, authorID uuid.UUID, input *GitCreatePRInput) (*model.GitPullRequest, error)
	GetPR(ctx context.Context, repoID uuid.UUID, number int) (*model.GitPullRequest, error)
	ListPRs(ctx context.Context, repoID uuid.UUID, status *model.GitPRStatus, limit, offset int) ([]*model.GitPullRequest, int64, error)
	UpdatePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID, input *GitUpdatePRInput) (*model.GitPullRequest, error)
	MergePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID) (*model.GitPullRequest, error)

	// Storage operations
	GetStorageStats(ctx context.Context, repoID uuid.UUID) (*GitStorageStats, error)
	GetUserStorageStats(ctx context.Context, userID uuid.UUID) (*GitUserStorageStats, error)

	// Filesystem access
	GetFilesystem(ctx context.Context, repoID uuid.UUID) (billy.Filesystem, error)

	// Git protocol helpers
	UpdatePushedAt(ctx context.Context, repoID uuid.UUID) error
	UpdateRepoSize(ctx context.Context, repoID uuid.UUID, sizeBytes, lfsSizeBytes int64) error
}

// ===== LFS Domain Port =====

// GitLFSDomain defines Git LFS operations.
type GitLFSDomain interface {
	// Batch operations
	ProcessBatch(ctx context.Context, repoID, userID uuid.UUID, request *model.GitLFSBatchRequest) (*model.GitLFSBatchResponse, error)

	// Object operations
	GetObject(ctx context.Context, oid string) (*model.GitLFSObject, error)
	CreateObject(ctx context.Context, repoID uuid.UUID, oid string, size int64) error
	LinkObject(ctx context.Context, repoID uuid.UUID, oid string) error

	// Storage operations
	Upload(ctx context.Context, oid string, reader io.Reader, size int64) error
	Download(ctx context.Context, oid string) (io.ReadCloser, int64, error)
	VerifyObject(ctx context.Context, oid string, expectedSize int64) error

	// Presigned URL generation
	GenerateUploadURL(ctx context.Context, oid string, size int64) (*GitPresignedURLResult, error)
	GenerateDownloadURL(ctx context.Context, oid string) (*GitPresignedURLResult, error)
}

// ===== LFS Lock Domain Port =====

// GitLFSLockDomain defines Git LFS locking operations.
type GitLFSLockDomain interface {
	// Lock operations
	CreateLock(ctx context.Context, repoID, userID uuid.UUID, path string) (*model.GitLFSLock, error)
	GetLock(ctx context.Context, id uuid.UUID) (*model.GitLFSLock, error)
	DeleteLock(ctx context.Context, id, userID uuid.UUID, force bool) error
	ListLocks(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*model.GitLFSLock, error)
	VerifyLocks(ctx context.Context, repoID, userID uuid.UUID) (*GitVerifyLocksResult, error)
}

// ===== Input Types =====

// GitCreateRepoInput represents input for creating a repository.
type GitCreateRepoInput struct {
	Name        string
	Type        model.GitRepoType
	Visibility  model.GitVisibility
	Description string
	LFSEnabled  bool
}

// GitUpdateRepoInput represents input for updating a repository.
type GitUpdateRepoInput struct {
	Name          string
	Description   string
	Visibility    model.GitVisibility
	DefaultBranch string
	LFSEnabled    *bool
}

// GitRepoFilter represents repository query filters.
type GitRepoFilter struct {
	Type       *model.GitRepoType
	Visibility *model.GitVisibility
	Search     string
	Page       int
	PageSize   int
}

// GitCreatePRInput represents input for creating a pull request.
type GitCreatePRInput struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
}

// GitUpdatePRInput represents input for updating a pull request.
type GitUpdatePRInput struct {
	Title       string
	Description string
	Status      string // "open" or "closed"
}

// ===== Output Types =====

// GitStorageStats represents repository storage statistics.
type GitStorageStats struct {
	RepoSizeBytes  int64
	LFSSizeBytes   int64
	TotalSizeBytes int64
	LFSObjectCount int
}

// GitUserStorageStats represents user-level storage statistics.
type GitUserStorageStats struct {
	TotalUsedBytes int64
	QuotaBytes     int64
	RemainingBytes int64
	RepoCount      int
}

// GitPresignedURLResult represents a presigned URL result.
type GitPresignedURLResult struct {
	URL       string
	Method    string
	ExpiresAt time.Time
	Headers   map[string]string
}

// GitVerifyLocksResult represents the result of verifying locks.
type GitVerifyLocksResult struct {
	Ours       []*model.GitLFSLock
	Theirs     []*model.GitLFSLock
	NextCursor string
}

// ===== Quota Checker Port =====

// GitStorageQuotaChecker defines storage quota checking operations.
type GitStorageQuotaChecker interface {
	// GetStorageQuota returns the storage quota for a user (-1 for unlimited).
	GetStorageQuota(ctx context.Context, userID uuid.UUID) (int64, error)

	// GetStorageUsed returns the storage used by a user.
	GetStorageUsed(ctx context.Context, userID uuid.UUID) (int64, error)
}
