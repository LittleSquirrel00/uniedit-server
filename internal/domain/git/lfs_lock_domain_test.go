package git

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
)

// --- LFS Lock Domain Tests ---
// Note: MockGitAccessControl is defined in lfs_domain_test.go

func TestLFSLockDomain_CreateLock(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSLockDB := new(MockGitLFSLockDB)
		mockAccessCtrl := new(MockGitAccessControl)

		domain := NewLFSLockDomain(mockRepoDB, mockLFSLockDB, mockAccessCtrl, logger)

		repoID := uuid.New()
		userID := uuid.New()
		path := "path/to/file.bin"
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionWrite).
			Return(&model.GitAccessResult{Allowed: true}, nil)
		mockLFSLockDB.On("FindByPath", mock.Anything, repoID, path).Return(nil, nil)
		mockLFSLockDB.On("Create", mock.Anything, mock.AnythingOfType("*model.GitLFSLock")).Return(nil)

		lock, err := domain.CreateLock(context.Background(), repoID, userID, path)

		require.NoError(t, err)
		assert.NotNil(t, lock)
		assert.Equal(t, path, lock.Path)
		assert.Equal(t, userID, lock.OwnerID)
		assert.Equal(t, repoID, lock.RepoID)
	})

	t.Run("repo_not_found", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewLFSLockDomain(mockRepoDB, nil, nil, logger)

		mockRepoDB.On("FindByID", mock.Anything, mock.Anything).Return(nil, nil)

		lock, err := domain.CreateLock(context.Background(), uuid.New(), uuid.New(), "path")

		assert.ErrorIs(t, err, ErrRepoNotFound)
		assert.Nil(t, lock)
	})

	t.Run("lfs_not_enabled", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewLFSLockDomain(mockRepoDB, nil, nil, logger)

		repoID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: false,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		lock, err := domain.CreateLock(context.Background(), repoID, uuid.New(), "path")

		assert.ErrorIs(t, err, ErrLFSNotEnabled)
		assert.Nil(t, lock)
	})

	t.Run("access_denied", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockAccessCtrl := new(MockGitAccessControl)

		domain := NewLFSLockDomain(mockRepoDB, nil, mockAccessCtrl, logger)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionWrite).
			Return(&model.GitAccessResult{Allowed: false}, nil)

		lock, err := domain.CreateLock(context.Background(), repoID, userID, "path")

		assert.ErrorIs(t, err, ErrAccessDenied)
		assert.Nil(t, lock)
	})

	t.Run("lock_already_exists", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSLockDB := new(MockGitLFSLockDB)
		mockAccessCtrl := new(MockGitAccessControl)

		domain := NewLFSLockDomain(mockRepoDB, mockLFSLockDB, mockAccessCtrl, logger)

		repoID := uuid.New()
		userID := uuid.New()
		path := "already/locked.bin"
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}
		existingLock := &model.GitLFSLock{
			ID:      uuid.New(),
			RepoID:  repoID,
			Path:    path,
			OwnerID: uuid.New(), // Different user
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionWrite).
			Return(&model.GitAccessResult{Allowed: true}, nil)
		mockLFSLockDB.On("FindByPath", mock.Anything, repoID, path).Return(existingLock, nil)

		lock, err := domain.CreateLock(context.Background(), repoID, userID, path)

		assert.ErrorIs(t, err, ErrLockAlreadyExists)
		assert.Nil(t, lock)
	})
}

func TestLFSLockDomain_GetLock(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		lockID := uuid.New()
		expectedLock := &model.GitLFSLock{
			ID:       lockID,
			Path:     "test/file.bin",
			OwnerID:  uuid.New(),
			LockedAt: time.Now(),
		}

		mockLFSLockDB.On("FindByID", mock.Anything, lockID).Return(expectedLock, nil)

		lock, err := domain.GetLock(context.Background(), lockID)

		require.NoError(t, err)
		assert.Equal(t, expectedLock, lock)
	})

	t.Run("not_found", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		mockLFSLockDB.On("FindByID", mock.Anything, mock.Anything).Return(nil, nil)

		lock, err := domain.GetLock(context.Background(), uuid.New())

		assert.ErrorIs(t, err, ErrLockNotFound)
		assert.Nil(t, lock)
	})
}

func TestLFSLockDomain_DeleteLock(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success_by_owner", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		lockID := uuid.New()
		userID := uuid.New()
		lock := &model.GitLFSLock{
			ID:      lockID,
			Path:    "test/file.bin",
			OwnerID: userID,
			RepoID:  uuid.New(),
		}

		mockLFSLockDB.On("FindByID", mock.Anything, lockID).Return(lock, nil)
		mockLFSLockDB.On("Delete", mock.Anything, lockID).Return(nil)

		err := domain.DeleteLock(context.Background(), lockID, userID, false)

		require.NoError(t, err)
		mockLFSLockDB.AssertExpectations(t)
	})

	t.Run("not_owner_denied", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)
		mockAccessCtrl := new(MockGitAccessControl)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, mockAccessCtrl, logger)

		lockID := uuid.New()
		ownerID := uuid.New()
		requesterID := uuid.New()
		repoID := uuid.New()
		lock := &model.GitLFSLock{
			ID:      lockID,
			Path:    "test/file.bin",
			OwnerID: ownerID, // Different from requester
			RepoID:  repoID,
		}

		mockLFSLockDB.On("FindByID", mock.Anything, lockID).Return(lock, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, requesterID, repoID, model.GitPermissionAdmin).
			Return(&model.GitAccessResult{Allowed: false}, nil)

		err := domain.DeleteLock(context.Background(), lockID, requesterID, false)

		assert.ErrorIs(t, err, ErrLockNotOwned)
	})

	t.Run("force_delete_by_admin", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		lockID := uuid.New()
		ownerID := uuid.New()
		adminID := uuid.New()
		lock := &model.GitLFSLock{
			ID:      lockID,
			Path:    "test/file.bin",
			OwnerID: ownerID,
			RepoID:  uuid.New(),
		}

		mockLFSLockDB.On("FindByID", mock.Anything, lockID).Return(lock, nil)
		mockLFSLockDB.On("Delete", mock.Anything, lockID).Return(nil)

		// Force delete doesn't check ownership
		err := domain.DeleteLock(context.Background(), lockID, adminID, true)

		require.NoError(t, err)
		mockLFSLockDB.AssertExpectations(t)
	})

	t.Run("lock_not_found", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		mockLFSLockDB.On("FindByID", mock.Anything, mock.Anything).Return(nil, nil)

		err := domain.DeleteLock(context.Background(), uuid.New(), uuid.New(), false)

		assert.ErrorIs(t, err, ErrLockNotFound)
	})
}

func TestLFSLockDomain_ListLocks(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		repoID := uuid.New()
		locks := []*model.GitLFSLock{
			{ID: uuid.New(), Path: "file1.bin", OwnerID: uuid.New()},
			{ID: uuid.New(), Path: "file2.bin", OwnerID: uuid.New()},
		}

		mockLFSLockDB.On("FindByRepo", mock.Anything, repoID, "", 100).Return(locks, nil)

		result, err := domain.ListLocks(context.Background(), repoID, "", 100)

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("with_path_filter", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		repoID := uuid.New()
		path := "specific/path"
		locks := []*model.GitLFSLock{
			{ID: uuid.New(), Path: "specific/path/file.bin", OwnerID: uuid.New()},
		}

		mockLFSLockDB.On("FindByRepo", mock.Anything, repoID, path, 50).Return(locks, nil)

		result, err := domain.ListLocks(context.Background(), repoID, path, 50)

		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestLFSLockDomain_VerifyLocks(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		repoID := uuid.New()
		userID := uuid.New()
		otherUserID := uuid.New()

		locks := []*model.GitLFSLock{
			{ID: uuid.New(), Path: "my-file.bin", OwnerID: userID},
			{ID: uuid.New(), Path: "other-file.bin", OwnerID: otherUserID},
			{ID: uuid.New(), Path: "another-file.bin", OwnerID: userID},
		}

		mockLFSLockDB.On("FindByRepo", mock.Anything, repoID, "", 0).Return(locks, nil)

		result, err := domain.VerifyLocks(context.Background(), repoID, userID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Ours, 2)   // Two locks owned by userID
		assert.Len(t, result.Theirs, 1) // One lock owned by otherUserID
	})

	t.Run("empty_result", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		repoID := uuid.New()
		userID := uuid.New()

		mockLFSLockDB.On("FindByRepo", mock.Anything, repoID, "", 0).Return([]*model.GitLFSLock{}, nil)

		result, err := domain.VerifyLocks(context.Background(), repoID, userID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Ours)
		assert.Empty(t, result.Theirs)
	})

	t.Run("all_ours", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		repoID := uuid.New()
		userID := uuid.New()

		locks := []*model.GitLFSLock{
			{ID: uuid.New(), Path: "file1.bin", OwnerID: userID},
			{ID: uuid.New(), Path: "file2.bin", OwnerID: userID},
		}

		mockLFSLockDB.On("FindByRepo", mock.Anything, repoID, "", 0).Return(locks, nil)

		result, err := domain.VerifyLocks(context.Background(), repoID, userID)

		require.NoError(t, err)
		assert.Len(t, result.Ours, 2)
		assert.Empty(t, result.Theirs)
	})

	t.Run("all_theirs", func(t *testing.T) {
		mockLFSLockDB := new(MockGitLFSLockDB)

		domain := NewLFSLockDomain(nil, mockLFSLockDB, nil, logger)

		repoID := uuid.New()
		userID := uuid.New()
		otherUserID := uuid.New()

		locks := []*model.GitLFSLock{
			{ID: uuid.New(), Path: "file1.bin", OwnerID: otherUserID},
			{ID: uuid.New(), Path: "file2.bin", OwnerID: otherUserID},
		}

		mockLFSLockDB.On("FindByRepo", mock.Anything, repoID, "", 0).Return(locks, nil)

		result, err := domain.VerifyLocks(context.Background(), repoID, userID)

		require.NoError(t, err)
		assert.Empty(t, result.Ours)
		assert.Len(t, result.Theirs, 2)
	})
}
