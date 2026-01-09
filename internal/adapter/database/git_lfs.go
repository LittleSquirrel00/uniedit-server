package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// GitLFSObjectNotFound indicates the LFS object was not found.
var GitLFSObjectNotFound = errors.New("LFS object not found")

// GitLFSObjectDatabaseAdapter implements GitLFSObjectDatabasePort using GORM.
type GitLFSObjectDatabaseAdapter struct {
	db *gorm.DB
}

// NewGitLFSObjectDatabaseAdapter creates a new LFS object database adapter.
func NewGitLFSObjectDatabaseAdapter(db *gorm.DB) *GitLFSObjectDatabaseAdapter {
	return &GitLFSObjectDatabaseAdapter{db: db}
}

// Create creates an LFS object (idempotent, uses OID as key).
func (a *GitLFSObjectDatabaseAdapter) Create(ctx context.Context, obj *model.GitLFSObject) error {
	return a.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(obj).Error
}

// FindByOID finds an LFS object by its OID (SHA-256).
func (a *GitLFSObjectDatabaseAdapter) FindByOID(ctx context.Context, oid string) (*model.GitLFSObject, error) {
	var obj model.GitLFSObject
	err := a.db.WithContext(ctx).First(&obj, "oid = ?", oid).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &obj, nil
}

// Link links an LFS object to a repository.
func (a *GitLFSObjectDatabaseAdapter) Link(ctx context.Context, repoID uuid.UUID, oid string) error {
	link := model.GitLFSRepoObject{
		RepoID:    repoID,
		OID:       oid,
		CreatedAt: time.Now(),
	}
	return a.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&link).Error
}

// Unlink unlinks an LFS object from a repository.
func (a *GitLFSObjectDatabaseAdapter) Unlink(ctx context.Context, repoID uuid.UUID, oid string) error {
	return a.db.WithContext(ctx).
		Delete(&model.GitLFSRepoObject{}, "repo_id = ? AND oid = ?", repoID, oid).Error
}

// FindByRepo lists all LFS objects for a repository.
func (a *GitLFSObjectDatabaseAdapter) FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.GitLFSObject, error) {
	var objects []*model.GitLFSObject
	err := a.db.WithContext(ctx).
		Model(&model.GitLFSObject{}).
		Joins("JOIN lfs_repo_objects ON lfs_repo_objects.oid = lfs_objects.oid").
		Where("lfs_repo_objects.repo_id = ?", repoID).
		Find(&objects).Error
	return objects, err
}

// GetRepoLFSSize calculates total LFS size for a repository.
func (a *GitLFSObjectDatabaseAdapter) GetRepoLFSSize(ctx context.Context, repoID uuid.UUID) (int64, error) {
	var total int64
	err := a.db.WithContext(ctx).
		Model(&model.GitLFSRepoObject{}).
		Select("COALESCE(SUM(lfs_objects.size), 0)").
		Joins("JOIN lfs_objects ON lfs_objects.oid = lfs_repo_objects.oid").
		Where("lfs_repo_objects.repo_id = ?", repoID).
		Scan(&total).Error
	return total, err
}

// GetUserTotalStorage calculates total storage for a user.
func (a *GitLFSObjectDatabaseAdapter) GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := a.db.WithContext(ctx).
		Model(&model.GitRepo{}).
		Select("COALESCE(SUM(size_bytes + lfs_size_bytes), 0)").
		Where("owner_id = ?", userID).
		Scan(&total).Error
	return total, err
}

// Compile-time check
var _ outbound.GitLFSObjectDatabasePort = (*GitLFSObjectDatabaseAdapter)(nil)

// ===== LFS Lock Adapter =====

// GitLFSLockNotFound indicates the lock was not found.
var GitLFSLockNotFound = errors.New("LFS lock not found")

// GitLFSLockDatabaseAdapter implements GitLFSLockDatabasePort using GORM.
type GitLFSLockDatabaseAdapter struct {
	db *gorm.DB
}

// NewGitLFSLockDatabaseAdapter creates a new LFS lock database adapter.
func NewGitLFSLockDatabaseAdapter(db *gorm.DB) *GitLFSLockDatabaseAdapter {
	return &GitLFSLockDatabaseAdapter{db: db}
}

// Create creates a file lock.
func (a *GitLFSLockDatabaseAdapter) Create(ctx context.Context, lock *model.GitLFSLock) error {
	err := a.db.WithContext(ctx).Create(lock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("file is already locked")
		}
		return err
	}
	return nil
}

// FindByID finds a lock by ID.
func (a *GitLFSLockDatabaseAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.GitLFSLock, error) {
	var lock model.GitLFSLock
	err := a.db.WithContext(ctx).First(&lock, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lock, nil
}

// FindByPath finds a lock by repository and path.
func (a *GitLFSLockDatabaseAdapter) FindByPath(ctx context.Context, repoID uuid.UUID, path string) (*model.GitLFSLock, error) {
	var lock model.GitLFSLock
	err := a.db.WithContext(ctx).
		Where("repo_id = ? AND path = ?", repoID, path).
		First(&lock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lock, nil
}

// FindByRepo lists locks for a repository.
func (a *GitLFSLockDatabaseAdapter) FindByRepo(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*model.GitLFSLock, error) {
	query := a.db.WithContext(ctx).Where("repo_id = ?", repoID)

	if path != "" {
		query = query.Where("path = ?", path)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var locks []*model.GitLFSLock
	err := query.Order("locked_at ASC").Find(&locks).Error
	return locks, err
}

// Delete deletes a lock.
func (a *GitLFSLockDatabaseAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	result := a.db.WithContext(ctx).Delete(&model.GitLFSLock{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return GitLFSLockNotFound
	}
	return nil
}

// Compile-time check
var _ outbound.GitLFSLockDatabasePort = (*GitLFSLockDatabaseAdapter)(nil)
