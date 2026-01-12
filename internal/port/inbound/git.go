package inbound

import (
	"context"
	"io"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/google/uuid"
	commonv1 "github.com/uniedit/server/api/pb/common"
	gitv1 "github.com/uniedit/server/api/pb/git"
	"github.com/uniedit/server/internal/model"
)

// ===== Git Domain Port =====

// GitDomain defines the main Git domain operations.
type GitDomain interface {
	// GitService
	CreateRepo(ctx context.Context, ownerID uuid.UUID, in *gitv1.CreateRepoRequest) (*gitv1.Repo, error)
	ListRepos(ctx context.Context, ownerID uuid.UUID, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error)
	GetRepo(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*gitv1.Repo, error)
	UpdateRepo(ctx context.Context, userID uuid.UUID, in *gitv1.UpdateRepoRequest) (*gitv1.Repo, error)
	DeleteRepo(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*commonv1.Empty, error)
	GetStorageStats(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*gitv1.StorageStats, error)
	AddCollaborator(ctx context.Context, userID uuid.UUID, in *gitv1.AddCollaboratorRequest) (*commonv1.Empty, error)
	ListCollaborators(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*gitv1.ListCollaboratorsResponse, error)
	UpdateCollaborator(ctx context.Context, userID uuid.UUID, in *gitv1.UpdateCollaboratorRequest) (*commonv1.Empty, error)
	RemoveCollaborator(ctx context.Context, userID uuid.UUID, in *gitv1.RemoveCollaboratorRequest) (*commonv1.Empty, error)
	CreatePR(ctx context.Context, userID uuid.UUID, in *gitv1.CreatePRRequest) (*gitv1.PullRequest, error)
	ListPRs(ctx context.Context, userID uuid.UUID, in *gitv1.ListPRsRequest) (*gitv1.ListPRsResponse, error)
	GetPR(ctx context.Context, userID uuid.UUID, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error)
	UpdatePR(ctx context.Context, userID uuid.UUID, in *gitv1.UpdatePRRequest) (*gitv1.PullRequest, error)
	MergePR(ctx context.Context, userID uuid.UUID, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error)

	// GitPublicService
	ListPublicRepos(ctx context.Context, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error)

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
// ===== Output Types =====

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
