package git

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepoFilter represents repository filter options.
type RepoFilter struct {
	RepoType   *RepoType
	Visibility *Visibility
	Search     string
	Limit      int
	Offset     int
}

// RepositoryRepository defines the interface for repository persistence.
type RepositoryRepository interface {
	// Create creates a new repository.
	Create(ctx context.Context, repo *Repository) error

	// GetByID retrieves a repository by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Repository, error)

	// GetByOwnerAndSlug retrieves a repository by owner and slug.
	GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*Repository, error)

	// List lists repositories for a user.
	List(ctx context.Context, ownerID uuid.UUID, filter *RepoFilter) ([]*Repository, int64, error)

	// ListPublic lists public repositories.
	ListPublic(ctx context.Context, filter *RepoFilter) ([]*Repository, int64, error)

	// Update updates a repository.
	Update(ctx context.Context, repo *Repository) error

	// Delete soft-deletes a repository.
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdatePushedAt updates the pushed_at timestamp.
	UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error

	// UpdateSize updates the size of a repository.
	UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error

	// GetUserTotalStorage returns total storage used by a user.
	GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error)
}

// CollaboratorRepository defines the interface for collaborator persistence.
type CollaboratorRepository interface {
	// Add adds a collaborator.
	Add(ctx context.Context, collab *Collaborator) error

	// Get retrieves a collaborator.
	Get(ctx context.Context, repoID, userID uuid.UUID) (*Collaborator, error)

	// List lists collaborators for a repository.
	List(ctx context.Context, repoID uuid.UUID) ([]*Collaborator, error)

	// Update updates a collaborator's permission.
	Update(ctx context.Context, collab *Collaborator) error

	// Remove removes a collaborator.
	Remove(ctx context.Context, repoID, userID uuid.UUID) error
}

// PullRequestRepository defines the interface for pull request persistence.
type PullRequestRepository interface {
	// Create creates a new pull request.
	Create(ctx context.Context, pr *PullRequest) error

	// GetByID retrieves a pull request by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*PullRequest, error)

	// GetByNumber retrieves a pull request by number.
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*PullRequest, error)

	// List lists pull requests for a repository.
	List(ctx context.Context, repoID uuid.UUID, status *PRStatus, limit, offset int) ([]*PullRequest, int64, error)

	// Update updates a pull request.
	Update(ctx context.Context, pr *PullRequest) error

	// GetNextNumber gets the next PR number for a repository.
	GetNextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}

// LFSRepository defines the interface for LFS persistence.
type LFSRepository interface {
	// CreateObject creates a new LFS object.
	CreateObject(ctx context.Context, obj *LFSObject) error

	// GetObject retrieves an LFS object by OID.
	GetObject(ctx context.Context, oid string) (*LFSObject, error)

	// LinkObjectToRepo links an LFS object to a repository.
	LinkObjectToRepo(ctx context.Context, link *LFSRepoObject) error

	// ListRepoObjects lists LFS objects for a repository.
	ListRepoObjects(ctx context.Context, repoID uuid.UUID) ([]*LFSRepoObject, error)

	// CreateLock creates a new LFS lock.
	CreateLock(ctx context.Context, lock *LFSLock) error

	// GetLock retrieves an LFS lock.
	GetLock(ctx context.Context, id uuid.UUID) (*LFSLock, error)

	// GetLockByPath retrieves an LFS lock by path.
	GetLockByPath(ctx context.Context, repoID uuid.UUID, path string) (*LFSLock, error)

	// ListLocks lists LFS locks for a repository.
	ListLocks(ctx context.Context, repoID uuid.UUID, path string, limit, offset int) ([]*LFSLock, error)

	// DeleteLock deletes an LFS lock.
	DeleteLock(ctx context.Context, id uuid.UUID) error
}

// StorageQuotaChecker defines the interface for checking storage quota.
type StorageQuotaChecker interface {
	GetStorageQuota(ctx context.Context, userID uuid.UUID) (int64, error)
	GetStorageUsed(ctx context.Context, userID uuid.UUID) (int64, error)
}
