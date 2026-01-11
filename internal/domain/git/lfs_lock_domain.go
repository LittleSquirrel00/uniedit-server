package git

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
)

// LFSLockDomain implements the Git LFS locking domain logic.
type LFSLockDomain struct {
	repoDB     outbound.GitRepoDatabasePort
	lfsLockDB  outbound.GitLFSLockDatabasePort
	accessCtrl outbound.GitAccessControlPort
	logger     *zap.Logger
}

// NewLFSLockDomain creates a new LFS lock domain.
func NewLFSLockDomain(
	repoDB outbound.GitRepoDatabasePort,
	lfsLockDB outbound.GitLFSLockDatabasePort,
	accessCtrl outbound.GitAccessControlPort,
	logger *zap.Logger,
) *LFSLockDomain {
	return &LFSLockDomain{
		repoDB:     repoDB,
		lfsLockDB:  lfsLockDB,
		accessCtrl: accessCtrl,
		logger:     logger,
	}
}

// CreateLock creates a file lock.
func (d *LFSLockDomain) CreateLock(ctx context.Context, repoID, userID uuid.UUID, path string) (*model.GitLFSLock, error) {
	// Check repository exists and LFS is enabled
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	if !repo.LFSEnabled {
		return nil, ErrLFSNotEnabled
	}

	// Check write access
	result, err := d.accessCtrl.CheckAccess(ctx, userID, repoID, model.GitPermissionWrite)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, ErrAccessDenied
	}

	// Check if path is already locked
	existing, err := d.lfsLockDB.FindByPath(ctx, repoID, path)
	if err == nil && existing != nil {
		return nil, ErrLockAlreadyExists
	}

	// Create lock
	lock := &model.GitLFSLock{
		ID:       uuid.New(),
		RepoID:   repoID,
		Path:     path,
		OwnerID:  userID,
		LockedAt: time.Now(),
	}

	if err := d.lfsLockDB.Create(ctx, lock); err != nil {
		return nil, fmt.Errorf("create lock: %w", err)
	}

	d.logger.Info("LFS lock created",
		zap.String("lock_id", lock.ID.String()),
		zap.String("repo_id", repoID.String()),
		zap.String("path", path),
		zap.String("owner_id", userID.String()),
	)

	return lock, nil
}

// GetLock gets a lock by ID.
func (d *LFSLockDomain) GetLock(ctx context.Context, id uuid.UUID) (*model.GitLFSLock, error) {
	lock, err := d.lfsLockDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lock == nil {
		return nil, ErrLockNotFound
	}
	return lock, nil
}

// DeleteLock deletes a lock.
func (d *LFSLockDomain) DeleteLock(ctx context.Context, id, userID uuid.UUID, force bool) error {
	lock, err := d.lfsLockDB.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if lock == nil {
		return ErrLockNotFound
	}

	// Check ownership unless force is used
	if !force && lock.OwnerID != userID {
		// Check admin access for force unlock
		result, err := d.accessCtrl.CheckAccess(ctx, userID, lock.RepoID, model.GitPermissionAdmin)
		if err != nil {
			return err
		}
		if !result.Allowed {
			return ErrLockNotOwned
		}
	}

	if err := d.lfsLockDB.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete lock: %w", err)
	}

	d.logger.Info("LFS lock deleted",
		zap.String("lock_id", id.String()),
		zap.String("deleted_by", userID.String()),
		zap.Bool("force", force),
	)

	return nil
}

// ListLocks lists locks for a repository.
func (d *LFSLockDomain) ListLocks(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*model.GitLFSLock, error) {
	return d.lfsLockDB.FindByRepo(ctx, repoID, path, limit)
}

// VerifyLocks verifies locks for a user.
func (d *LFSLockDomain) VerifyLocks(ctx context.Context, repoID, userID uuid.UUID) (*inbound.GitVerifyLocksResult, error) {
	// Get all locks for the repository
	locks, err := d.lfsLockDB.FindByRepo(ctx, repoID, "", 0)
	if err != nil {
		return nil, fmt.Errorf("list locks: %w", err)
	}

	result := &inbound.GitVerifyLocksResult{
		Ours:   make([]*model.GitLFSLock, 0),
		Theirs: make([]*model.GitLFSLock, 0),
	}

	for _, lock := range locks {
		if lock.OwnerID == userID {
			result.Ours = append(result.Ours, lock)
		} else {
			result.Theirs = append(result.Theirs, lock)
		}
	}

	return result, nil
}

// Compile-time interface check
var _ inbound.GitLFSLockDomain = (*LFSLockDomain)(nil)
