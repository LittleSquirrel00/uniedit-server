package git

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitv1 "github.com/uniedit/server/api/pb/git"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type MockGitRepoDB struct {
	mock.Mock
}

func (m *MockGitRepoDB) Create(ctx context.Context, repo *model.GitRepo) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *MockGitRepoDB) FindByID(ctx context.Context, id uuid.UUID) (*model.GitRepo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitRepo), args.Error(1)
}

func (m *MockGitRepoDB) FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error) {
	args := m.Called(ctx, ownerID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitRepo), args.Error(1)
}

func (m *MockGitRepoDB) FindByOwner(ctx context.Context, ownerID uuid.UUID, filter *outbound.GitRepoFilter) ([]*model.GitRepo, int64, error) {
	args := m.Called(ctx, ownerID, filter)
	return args.Get(0).([]*model.GitRepo), args.Get(1).(int64), args.Error(2)
}

func (m *MockGitRepoDB) FindPublic(ctx context.Context, filter *outbound.GitRepoFilter) ([]*model.GitRepo, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*model.GitRepo), args.Get(1).(int64), args.Error(2)
}

func (m *MockGitRepoDB) Update(ctx context.Context, repo *model.GitRepo) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *MockGitRepoDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockGitRepoDB) UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error {
	args := m.Called(ctx, id, pushedAt)
	return args.Error(0)
}

func (m *MockGitRepoDB) UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error {
	args := m.Called(ctx, id, sizeBytes, lfsSizeBytes)
	return args.Error(0)
}

func (m *MockGitRepoDB) IncrementStars(ctx context.Context, id uuid.UUID, delta int) error {
	args := m.Called(ctx, id, delta)
	return args.Error(0)
}

func (m *MockGitRepoDB) IncrementForks(ctx context.Context, id uuid.UUID, delta int) error {
	args := m.Called(ctx, id, delta)
	return args.Error(0)
}

type MockGitCollabDB struct {
	mock.Mock
}

func (m *MockGitCollabDB) Add(ctx context.Context, collab *model.GitRepoCollaborator) error {
	args := m.Called(ctx, collab)
	return args.Error(0)
}

func (m *MockGitCollabDB) FindByRepoAndUser(ctx context.Context, repoID, userID uuid.UUID) (*model.GitRepoCollaborator, error) {
	args := m.Called(ctx, repoID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitRepoCollaborator), args.Error(1)
}

func (m *MockGitCollabDB) FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.GitRepoCollaborator, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.GitRepoCollaborator), args.Error(1)
}

func (m *MockGitCollabDB) FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.GitRepoCollaborator, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.GitRepoCollaborator), args.Error(1)
}

func (m *MockGitCollabDB) Update(ctx context.Context, collab *model.GitRepoCollaborator) error {
	args := m.Called(ctx, collab)
	return args.Error(0)
}

func (m *MockGitCollabDB) Remove(ctx context.Context, repoID, userID uuid.UUID) error {
	args := m.Called(ctx, repoID, userID)
	return args.Error(0)
}

type MockGitPRDB struct {
	mock.Mock
}

func (m *MockGitPRDB) Create(ctx context.Context, pr *model.GitPullRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

func (m *MockGitPRDB) FindByID(ctx context.Context, id uuid.UUID) (*model.GitPullRequest, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitPullRequest), args.Error(1)
}

func (m *MockGitPRDB) FindByNumber(ctx context.Context, repoID uuid.UUID, number int) (*model.GitPullRequest, error) {
	args := m.Called(ctx, repoID, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitPullRequest), args.Error(1)
}

func (m *MockGitPRDB) FindByRepo(ctx context.Context, repoID uuid.UUID, status *model.GitPRStatus, limit, offset int) ([]*model.GitPullRequest, int64, error) {
	args := m.Called(ctx, repoID, status, limit, offset)
	return args.Get(0).([]*model.GitPullRequest), args.Get(1).(int64), args.Error(2)
}

func (m *MockGitPRDB) Update(ctx context.Context, pr *model.GitPullRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

func (m *MockGitPRDB) GetNextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	args := m.Called(ctx, repoID)
	return args.Int(0), args.Error(1)
}

type MockGitLFSObjDB struct {
	mock.Mock
}

func (m *MockGitLFSObjDB) Create(ctx context.Context, obj *model.GitLFSObject) error {
	args := m.Called(ctx, obj)
	return args.Error(0)
}

func (m *MockGitLFSObjDB) FindByOID(ctx context.Context, oid string) (*model.GitLFSObject, error) {
	args := m.Called(ctx, oid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitLFSObject), args.Error(1)
}

func (m *MockGitLFSObjDB) Link(ctx context.Context, repoID uuid.UUID, oid string) error {
	args := m.Called(ctx, repoID, oid)
	return args.Error(0)
}

func (m *MockGitLFSObjDB) Unlink(ctx context.Context, repoID uuid.UUID, oid string) error {
	args := m.Called(ctx, repoID, oid)
	return args.Error(0)
}

func (m *MockGitLFSObjDB) FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.GitLFSObject, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.GitLFSObject), args.Error(1)
}

func (m *MockGitLFSObjDB) GetRepoLFSSize(ctx context.Context, repoID uuid.UUID) (int64, error) {
	args := m.Called(ctx, repoID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockGitLFSObjDB) GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

type MockGitLFSLockDB struct {
	mock.Mock
}

func (m *MockGitLFSLockDB) Create(ctx context.Context, lock *model.GitLFSLock) error {
	args := m.Called(ctx, lock)
	return args.Error(0)
}

func (m *MockGitLFSLockDB) FindByID(ctx context.Context, id uuid.UUID) (*model.GitLFSLock, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitLFSLock), args.Error(1)
}

func (m *MockGitLFSLockDB) FindByPath(ctx context.Context, repoID uuid.UUID, path string) (*model.GitLFSLock, error) {
	args := m.Called(ctx, repoID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitLFSLock), args.Error(1)
}

func (m *MockGitLFSLockDB) FindByRepo(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*model.GitLFSLock, error) {
	args := m.Called(ctx, repoID, path, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.GitLFSLock), args.Error(1)
}

func (m *MockGitLFSLockDB) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type MockGitStorage struct {
	mock.Mock
	fs billy.Filesystem
}

func (m *MockGitStorage) GetFilesystem(ctx context.Context, path string) (billy.Filesystem, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(billy.Filesystem), args.Error(1)
}

func (m *MockGitStorage) DeleteRepository(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockGitStorage) GetRepositorySize(ctx context.Context, path string) (int64, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(int64), args.Error(1)
}

type MockGitLFSStorage struct {
	mock.Mock
}

func (m *MockGitLFSStorage) Upload(ctx context.Context, oid string, reader io.Reader, size int64) error {
	args := m.Called(ctx, oid, reader, size)
	return args.Error(0)
}

func (m *MockGitLFSStorage) Download(ctx context.Context, oid string) (io.ReadCloser, int64, error) {
	args := m.Called(ctx, oid)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).(io.ReadCloser), args.Get(1).(int64), args.Error(2)
}

func (m *MockGitLFSStorage) Exists(ctx context.Context, oid string) (bool, error) {
	args := m.Called(ctx, oid)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitLFSStorage) Delete(ctx context.Context, oid string) error {
	args := m.Called(ctx, oid)
	return args.Error(0)
}

func (m *MockGitLFSStorage) GenerateUploadURL(ctx context.Context, oid string, size int64, expiry time.Duration) (*outbound.GitPresignedURL, error) {
	args := m.Called(ctx, oid, size, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.GitPresignedURL), args.Error(1)
}

func (m *MockGitLFSStorage) GenerateDownloadURL(ctx context.Context, oid string, expiry time.Duration) (*outbound.GitPresignedURL, error) {
	args := m.Called(ctx, oid, expiry)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*outbound.GitPresignedURL), args.Error(1)
}

// --- Tests ---

func TestDomain_CreateRepo(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockStorage := new(MockGitStorage)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			mockStorage,
			nil,
			nil,
			nil,
			logger,
		)

		ownerID := uuid.New()
		fs := memfs.New()

		mockRepoDB.On("FindByOwnerAndSlug", mock.Anything, ownerID, "my-repo").Return(nil, ErrRepoNotFound)
		mockStorage.On("GetFilesystem", mock.Anything, mock.AnythingOfType("string")).Return(fs, nil)
		mockRepoDB.On("Create", mock.Anything, mock.AnythingOfType("*model.GitRepo")).Return(nil)

		repo, err := domain.CreateRepo(context.Background(), ownerID, &gitv1.CreateRepoRequest{
			Name:        "My Repo",
			Description: "Test repository",
		})

		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.Equal(t, "My Repo", repo.GetName())
		assert.Equal(t, "my-repo", repo.GetSlug())
		assert.Equal(t, ownerID.String(), repo.GetOwnerId())
		mockRepoDB.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("repo already exists", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		ownerID := uuid.New()
		existingRepo := &model.GitRepo{
			ID:      uuid.New(),
			OwnerID: ownerID,
			Name:    "My Repo",
			Slug:    "my-repo",
		}

		mockRepoDB.On("FindByOwnerAndSlug", mock.Anything, ownerID, "my-repo").Return(existingRepo, nil)

		repo, err := domain.CreateRepo(context.Background(), ownerID, &gitv1.CreateRepoRequest{Name: "My Repo"})

		assert.ErrorIs(t, err, ErrRepoAlreadyExists)
		assert.Nil(t, repo)
	})

	t.Run("invalid repo name", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		ownerID := uuid.New()
		repo, err := domain.CreateRepo(context.Background(), ownerID, &gitv1.CreateRepoRequest{Name: "@#$%^&*()"})

		assert.ErrorIs(t, err, ErrInvalidRepoName)
		assert.Nil(t, repo)
	})
}

func TestDomain_GetRepo(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		expectedRepo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    userID,
			Name:       "Test Repo",
			Visibility: model.GitVisibilityPrivate,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(expectedRepo, nil)

		repo, err := domain.GetRepo(context.Background(), userID, &gitv1.GetByIDRequest{Id: repoID.String()})

		assert.NoError(t, err)
		assert.Equal(t, repoID.String(), repo.GetId())
		assert.Equal(t, "Test Repo", repo.GetName())
	})

	t.Run("not found", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(nil, nil)

		repo, err := domain.GetRepo(context.Background(), userID, &gitv1.GetByIDRequest{Id: repoID.String()})

		assert.ErrorIs(t, err, ErrRepoNotFound)
		assert.Nil(t, repo)
	})
}

func TestDomain_DeleteRepo(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockStorage := new(MockGitStorage)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			mockStorage,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		ownerID := uuid.New()
		repo := &model.GitRepo{
			ID:          repoID,
			OwnerID:     ownerID,
			StoragePath: "repos/test/",
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockStorage.On("DeleteRepository", mock.Anything, "repos/test/").Return(nil)
		mockRepoDB.On("Delete", mock.Anything, repoID).Return(nil)

		out, err := domain.DeleteRepo(context.Background(), ownerID, &gitv1.GetByIDRequest{Id: repoID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockRepoDB.AssertExpectations(t)
		mockStorage.AssertExpectations(t)
	})

	t.Run("not owner", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		repo := &model.GitRepo{
			ID:      repoID,
			OwnerID: ownerID,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		out, err := domain.DeleteRepo(context.Background(), otherUserID, &gitv1.GetByIDRequest{Id: repoID.String()})

		assert.ErrorIs(t, err, ErrNotOwner)
		assert.Nil(t, out)
	})
}

func TestDomain_CanAccess(t *testing.T) {
	logger := zap.NewNop()

	t.Run("owner has full access", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		ownerID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    ownerID,
			Visibility: model.GitVisibilityPrivate,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		canAccess, err := domain.CanAccess(context.Background(), repoID, &ownerID, model.GitPermissionAdmin)

		assert.NoError(t, err)
		assert.True(t, canAccess)
	})

	t.Run("public repo read access", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    uuid.New(),
			Visibility: model.GitVisibilityPublic,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		// Anonymous user can read public repo
		canAccess, err := domain.CanAccess(context.Background(), repoID, nil, model.GitPermissionRead)

		assert.NoError(t, err)
		assert.True(t, canAccess)
	})

	t.Run("collaborator access", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockCollabDB := new(MockGitCollabDB)

		domain := NewDomain(
			mockRepoDB,
			mockCollabDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    uuid.New(),
			Visibility: model.GitVisibilityPrivate,
		}
		collab := &model.GitRepoCollaborator{
			RepoID:     repoID,
			UserID:     userID,
			Permission: model.GitPermissionWrite,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockCollabDB.On("FindByRepoAndUser", mock.Anything, repoID, userID).Return(collab, nil)

		canAccess, err := domain.CanAccess(context.Background(), repoID, &userID, model.GitPermissionWrite)

		assert.NoError(t, err)
		assert.True(t, canAccess)
	})

	t.Run("insufficient permission", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockCollabDB := new(MockGitCollabDB)

		domain := NewDomain(
			mockRepoDB,
			mockCollabDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    uuid.New(),
			Visibility: model.GitVisibilityPrivate,
		}
		collab := &model.GitRepoCollaborator{
			RepoID:     repoID,
			UserID:     userID,
			Permission: model.GitPermissionRead, // Only read permission
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockCollabDB.On("FindByRepoAndUser", mock.Anything, repoID, userID).Return(collab, nil)

		canAccess, err := domain.CanAccess(context.Background(), repoID, &userID, model.GitPermissionAdmin)

		assert.NoError(t, err)
		assert.False(t, canAccess)
	})
}

func TestDomain_AddCollaborator(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockCollabDB := new(MockGitCollabDB)

		domain := NewDomain(
			mockRepoDB,
			mockCollabDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		ownerID := uuid.New()
		targetUserID := uuid.New()
		repo := &model.GitRepo{
			ID:      repoID,
			OwnerID: ownerID,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockCollabDB.On("Add", mock.Anything, mock.AnythingOfType("*model.GitRepoCollaborator")).Return(nil)

		out, err := domain.AddCollaborator(context.Background(), ownerID, &gitv1.AddCollaboratorRequest{
			Id:         repoID.String(),
			UserId:     targetUserID.String(),
			Permission: string(model.GitPermissionWrite),
		})

		assert.NoError(t, err)
		assert.NotNil(t, out)
		mockCollabDB.AssertExpectations(t)
	})

	t.Run("not owner", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		repo := &model.GitRepo{
			ID:      repoID,
			OwnerID: ownerID,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		out, err := domain.AddCollaborator(context.Background(), otherUserID, &gitv1.AddCollaboratorRequest{
			Id:         repoID.String(),
			UserId:     uuid.New().String(),
			Permission: string(model.GitPermissionWrite),
		})

		assert.ErrorIs(t, err, ErrNotOwner)
		assert.Nil(t, out)
	})

	t.Run("cannot add owner as collaborator", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		ownerID := uuid.New()
		repo := &model.GitRepo{
			ID:      repoID,
			OwnerID: ownerID,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		out, err := domain.AddCollaborator(context.Background(), ownerID, &gitv1.AddCollaboratorRequest{
			Id:         repoID.String(),
			UserId:     ownerID.String(),
			Permission: string(model.GitPermissionWrite),
		})

		assert.ErrorIs(t, err, ErrInvalidPermission)
		assert.Nil(t, out)
	})
}

func TestDomain_CreatePR(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockCollabDB := new(MockGitCollabDB)
		mockPRDB := new(MockGitPRDB)

		domain := NewDomain(
			mockRepoDB,
			mockCollabDB,
			mockPRDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		authorID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    authorID,
			Visibility: model.GitVisibilityPrivate,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockPRDB.On("GetNextNumber", mock.Anything, repoID).Return(1, nil)
		mockPRDB.On("Create", mock.Anything, mock.AnythingOfType("*model.GitPullRequest")).Return(nil)

		pr, err := domain.CreatePR(context.Background(), authorID, &gitv1.CreatePRRequest{
			Id:           repoID.String(),
			Title:        "Add new feature",
			Description:  "This PR adds a new feature",
			SourceBranch: "feature/new",
			TargetBranch: "main",
		})

		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, int32(1), pr.GetNumber())
		assert.Equal(t, "Add new feature", pr.GetTitle())
		assert.Equal(t, string(model.GitPRStatusOpen), pr.GetStatus())
	})

	t.Run("same branch error", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		authorID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    authorID,
			Visibility: model.GitVisibilityPrivate,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		pr, err := domain.CreatePR(context.Background(), authorID, &gitv1.CreatePRRequest{
			Id:           repoID.String(),
			Title:        "Test PR",
			SourceBranch: "main",
			TargetBranch: "main",
		})

		assert.ErrorIs(t, err, ErrSameBranch)
		assert.Nil(t, pr)
	})
}

func TestDomain_MergePR(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockCollabDB := new(MockGitCollabDB)
		mockPRDB := new(MockGitPRDB)

		domain := NewDomain(
			mockRepoDB,
			mockCollabDB,
			mockPRDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    userID,
			Visibility: model.GitVisibilityPrivate,
		}
		pr := &model.GitPullRequest{
			ID:           uuid.New(),
			RepoID:       repoID,
			Number:       1,
			Status:       model.GitPRStatusOpen,
			SourceBranch: "feature",
			TargetBranch: "main",
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockPRDB.On("FindByNumber", mock.Anything, repoID, 1).Return(pr, nil)
		mockPRDB.On("Update", mock.Anything, mock.AnythingOfType("*model.GitPullRequest")).Return(nil)

		mergedPR, err := domain.MergePR(context.Background(), userID, &gitv1.GetPRRequest{
			Id:     repoID.String(),
			Number: 1,
		})

		assert.NoError(t, err)
		assert.Equal(t, string(model.GitPRStatusMerged), mergedPR.GetStatus())
		assert.NotEmpty(t, mergedPR.GetMergedAt())
		assert.Equal(t, userID.String(), mergedPR.GetMergedBy())
	})

	t.Run("already merged", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockPRDB := new(MockGitPRDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			mockPRDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			OwnerID:    userID,
			Visibility: model.GitVisibilityPrivate,
		}
		pr := &model.GitPullRequest{
			ID:     uuid.New(),
			RepoID: repoID,
			Number: 1,
			Status: model.GitPRStatusMerged,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockPRDB.On("FindByNumber", mock.Anything, repoID, 1).Return(pr, nil)

		mergedPR, err := domain.MergePR(context.Background(), userID, &gitv1.GetPRRequest{
			Id:     repoID.String(),
			Number: 1,
		})

		assert.ErrorIs(t, err, ErrPRAlreadyMerged)
		assert.Nil(t, mergedPR)
	})
}

func TestDomain_GetStorageStats(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSObjDB := new(MockGitLFSObjDB)

		domain := NewDomain(
			mockRepoDB,
			nil,
			nil,
			mockLFSObjDB,
			nil,
			nil,
			nil,
			nil,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:           repoID,
			OwnerID:      userID,
			Visibility:   model.GitVisibilityPrivate,
			SizeBytes:    1024,
			LFSSizeBytes: 2048,
		}
		lfsObjects := []*model.GitLFSObject{
			{OID: "abc123", Size: 1024},
			{OID: "def456", Size: 1024},
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockLFSObjDB.On("FindByRepo", mock.Anything, repoID).Return(lfsObjects, nil)

		stats, err := domain.GetStorageStats(context.Background(), userID, &gitv1.GetByIDRequest{Id: repoID.String()})

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(1024), stats.GetRepoSizeBytes())
		assert.Equal(t, int64(2048), stats.GetLfsSizeBytes())
		assert.Equal(t, int64(3072), stats.GetTotalSizeBytes())
		assert.Equal(t, int32(2), stats.GetLfsObjectCount())
	})
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "My Repo", "my-repo"},
		{"with spaces", "My Great Repo", "my-great-repo"},
		{"with underscores", "my_repo_name", "my-repo-name"},
		{"with special chars", "My@Repo#Name!", "myreponame"},
		{"multiple hyphens", "my--repo---name", "my-repo-name"},
		{"leading hyphen", "-my-repo", "my-repo"},
		{"trailing hyphen", "my-repo-", "my-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		granted  model.GitPermission
		required model.GitPermission
		expected bool
	}{
		{"read grants read", model.GitPermissionRead, model.GitPermissionRead, true},
		{"write grants read", model.GitPermissionWrite, model.GitPermissionRead, true},
		{"admin grants read", model.GitPermissionAdmin, model.GitPermissionRead, true},
		{"write grants write", model.GitPermissionWrite, model.GitPermissionWrite, true},
		{"admin grants write", model.GitPermissionAdmin, model.GitPermissionWrite, true},
		{"admin grants admin", model.GitPermissionAdmin, model.GitPermissionAdmin, true},
		{"read denies write", model.GitPermissionRead, model.GitPermissionWrite, false},
		{"read denies admin", model.GitPermissionRead, model.GitPermissionAdmin, false},
		{"write denies admin", model.GitPermissionWrite, model.GitPermissionAdmin, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPermission(tt.granted, tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}
