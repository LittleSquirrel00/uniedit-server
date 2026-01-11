package outbound

import (
	"context"
	"io"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// ===== Repository Database Port =====

// GitRepoDatabasePort defines repository persistence operations.
type GitRepoDatabasePort interface {
	// Create creates a new repository.
	Create(ctx context.Context, repo *model.GitRepo) error

	// FindByID finds a repository by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.GitRepo, error)

	// FindByOwnerAndSlug finds a repository by owner and slug.
	FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error)

	// FindByOwner lists repositories owned by a user.
	FindByOwner(ctx context.Context, ownerID uuid.UUID, filter *GitRepoFilter) ([]*model.GitRepo, int64, error)

	// FindPublic lists public repositories.
	FindPublic(ctx context.Context, filter *GitRepoFilter) ([]*model.GitRepo, int64, error)

	// Update updates a repository.
	Update(ctx context.Context, repo *model.GitRepo) error

	// Delete deletes a repository.
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdatePushedAt updates the last push timestamp.
	UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error

	// UpdateSize updates repository size statistics.
	UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error

	// IncrementStars increments the star count.
	IncrementStars(ctx context.Context, id uuid.UUID, delta int) error

	// IncrementForks increments the fork count.
	IncrementForks(ctx context.Context, id uuid.UUID, delta int) error
}

// GitRepoFilter defines repository query filters.
type GitRepoFilter struct {
	Type       *model.GitRepoType
	Visibility *model.GitVisibility
	Search     string
	Page       int
	PageSize   int
}

// ===== Collaborator Database Port =====

// GitCollaboratorDatabasePort defines collaborator persistence operations.
type GitCollaboratorDatabasePort interface {
	// Add adds a collaborator to a repository.
	Add(ctx context.Context, collab *model.GitRepoCollaborator) error

	// FindByRepoAndUser finds a specific collaborator.
	FindByRepoAndUser(ctx context.Context, repoID, userID uuid.UUID) (*model.GitRepoCollaborator, error)

	// FindByRepo lists all collaborators for a repository.
	FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.GitRepoCollaborator, error)

	// FindByUser lists all repositories a user collaborates on.
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.GitRepoCollaborator, error)

	// Update updates a collaborator's permission.
	Update(ctx context.Context, collab *model.GitRepoCollaborator) error

	// Remove removes a collaborator.
	Remove(ctx context.Context, repoID, userID uuid.UUID) error
}

// ===== Pull Request Database Port =====

// GitPullRequestDatabasePort defines pull request persistence operations.
type GitPullRequestDatabasePort interface {
	// Create creates a new pull request.
	Create(ctx context.Context, pr *model.GitPullRequest) error

	// FindByID finds a pull request by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.GitPullRequest, error)

	// FindByNumber finds a pull request by repository and number.
	FindByNumber(ctx context.Context, repoID uuid.UUID, number int) (*model.GitPullRequest, error)

	// FindByRepo lists pull requests for a repository.
	FindByRepo(ctx context.Context, repoID uuid.UUID, status *model.GitPRStatus, limit, offset int) ([]*model.GitPullRequest, int64, error)

	// Update updates a pull request.
	Update(ctx context.Context, pr *model.GitPullRequest) error

	// GetNextNumber gets the next PR number for a repository.
	GetNextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}

// ===== LFS Object Database Port =====

// GitLFSObjectDatabasePort defines LFS object persistence operations.
type GitLFSObjectDatabasePort interface {
	// Create creates an LFS object (idempotent, uses OID as key).
	Create(ctx context.Context, obj *model.GitLFSObject) error

	// FindByOID finds an LFS object by its OID (SHA-256).
	FindByOID(ctx context.Context, oid string) (*model.GitLFSObject, error)

	// Link links an LFS object to a repository.
	Link(ctx context.Context, repoID uuid.UUID, oid string) error

	// Unlink unlinks an LFS object from a repository.
	Unlink(ctx context.Context, repoID uuid.UUID, oid string) error

	// FindByRepo lists all LFS objects for a repository.
	FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.GitLFSObject, error)

	// GetRepoLFSSize calculates total LFS size for a repository.
	GetRepoLFSSize(ctx context.Context, repoID uuid.UUID) (int64, error)

	// GetUserTotalStorage calculates total storage for a user.
	GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error)
}

// ===== LFS Lock Database Port =====

// GitLFSLockDatabasePort defines LFS lock persistence operations.
type GitLFSLockDatabasePort interface {
	// Create creates a file lock.
	Create(ctx context.Context, lock *model.GitLFSLock) error

	// FindByID finds a lock by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.GitLFSLock, error)

	// FindByPath finds a lock by repository and path.
	FindByPath(ctx context.Context, repoID uuid.UUID, path string) (*model.GitLFSLock, error)

	// FindByRepo lists locks for a repository.
	FindByRepo(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*model.GitLFSLock, error)

	// Delete deletes a lock.
	Delete(ctx context.Context, id uuid.UUID) error
}

// ===== Storage Port =====

// GitStoragePort defines Git repository storage operations.
type GitStoragePort interface {
	// GetFilesystem returns a billy.Filesystem for a repository.
	GetFilesystem(ctx context.Context, storagePath string) (billy.Filesystem, error)

	// DeleteRepository deletes all storage for a repository.
	DeleteRepository(ctx context.Context, storagePath string) error

	// GetRepositorySize calculates the size of a repository's storage.
	GetRepositorySize(ctx context.Context, storagePath string) (int64, error)
}

// ===== LFS Storage Port =====

// GitLFSStoragePort defines LFS object storage operations.
type GitLFSStoragePort interface {
	// Upload uploads an LFS object.
	Upload(ctx context.Context, oid string, reader io.Reader, size int64) error

	// Download downloads an LFS object.
	Download(ctx context.Context, oid string) (io.ReadCloser, int64, error)

	// Exists checks if an LFS object exists.
	Exists(ctx context.Context, oid string) (bool, error)

	// Delete deletes an LFS object.
	Delete(ctx context.Context, oid string) error

	// GenerateUploadURL generates a presigned upload URL.
	GenerateUploadURL(ctx context.Context, oid string, size int64, expiry time.Duration) (*GitPresignedURL, error)

	// GenerateDownloadURL generates a presigned download URL.
	GenerateDownloadURL(ctx context.Context, oid string, expiry time.Duration) (*GitPresignedURL, error)
}

// GitPresignedURL represents a presigned URL for LFS operations.
type GitPresignedURL struct {
	URL       string
	Method    string
	ExpiresAt time.Time
}

// ===== Access Control Port =====

// GitAccessControlPort defines repository access control operations.
type GitAccessControlPort interface {
	// CheckAccess checks if a user has access to a repository.
	CheckAccess(ctx context.Context, userID, repoID uuid.UUID, requiredPermission model.GitPermission) (*model.GitAccessResult, error)

	// CheckPublicAccess checks if a repository is publicly accessible.
	CheckPublicAccess(ctx context.Context, repoID uuid.UUID) (bool, error)
}
