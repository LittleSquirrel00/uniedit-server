package git

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// --- Mock implementations for LFS ---

type MockGitAccessControl struct {
	mock.Mock
}

func (m *MockGitAccessControl) CheckAccess(ctx context.Context, userID, repoID uuid.UUID, permission model.GitPermission) (*model.GitAccessResult, error) {
	args := m.Called(ctx, userID, repoID, permission)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitAccessResult), args.Error(1)
}

func (m *MockGitAccessControl) CheckPublicAccess(ctx context.Context, repoID uuid.UUID) (bool, error) {
	args := m.Called(ctx, repoID)
	return args.Bool(0), args.Error(1)
}

// --- LFS Domain Tests ---

func TestLFSDomain_ProcessBatch(t *testing.T) {
	logger := zap.NewNop()

	t.Run("download_success", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSObjDB := new(MockGitLFSObjDB)
		mockLFSStorage := new(MockGitLFSStorage)
		mockAccessCtrl := new(MockGitAccessControl)

		cfg := DefaultConfig()
		domain := NewLFSDomain(
			mockRepoDB,
			mockLFSObjDB,
			mockLFSStorage,
			mockAccessCtrl,
			cfg,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}
		oid := "abc123def456"

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionRead).
			Return(&model.GitAccessResult{Allowed: true}, nil)
		mockLFSStorage.On("Exists", mock.Anything, oid).Return(true, nil)
		mockLFSStorage.On("GenerateDownloadURL", mock.Anything, oid, cfg.PresignedURLExpiry).
			Return(&outbound.GitPresignedURL{URL: "https://example.com/download", Method: "GET"}, nil)

		request := &model.GitLFSBatchRequest{
			Operation: "download",
			Objects: []*model.GitLFSPointer{
				{OID: oid, Size: 1024},
			},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "basic", response.Transfer)
		assert.Len(t, response.Objects, 1)
		assert.Nil(t, response.Objects[0].Error)
		assert.NotNil(t, response.Objects[0].Actions["download"])
	})

	t.Run("upload_new_object", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSObjDB := new(MockGitLFSObjDB)
		mockLFSStorage := new(MockGitLFSStorage)
		mockAccessCtrl := new(MockGitAccessControl)

		cfg := DefaultConfig()
		domain := NewLFSDomain(
			mockRepoDB,
			mockLFSObjDB,
			mockLFSStorage,
			mockAccessCtrl,
			cfg,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}
		oid := "newobject123"

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionWrite).
			Return(&model.GitAccessResult{Allowed: true}, nil)
		mockLFSStorage.On("Exists", mock.Anything, oid).Return(false, nil)
		mockLFSStorage.On("GenerateUploadURL", mock.Anything, oid, int64(1024), cfg.PresignedURLExpiry).
			Return(&outbound.GitPresignedURL{URL: "https://example.com/upload", Method: "PUT"}, nil)

		request := &model.GitLFSBatchRequest{
			Operation: "upload",
			Objects: []*model.GitLFSPointer{
				{OID: oid, Size: 1024},
			},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Objects, 1)
		assert.Nil(t, response.Objects[0].Error)
		assert.NotNil(t, response.Objects[0].Actions["upload"])
		assert.NotNil(t, response.Objects[0].Actions["verify"])
	})

	t.Run("upload_existing_object", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSObjDB := new(MockGitLFSObjDB)
		mockLFSStorage := new(MockGitLFSStorage)
		mockAccessCtrl := new(MockGitAccessControl)

		cfg := DefaultConfig()
		domain := NewLFSDomain(
			mockRepoDB,
			mockLFSObjDB,
			mockLFSStorage,
			mockAccessCtrl,
			cfg,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}
		oid := "existingobject"

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionWrite).
			Return(&model.GitAccessResult{Allowed: true}, nil)
		mockLFSStorage.On("Exists", mock.Anything, oid).Return(true, nil)
		mockLFSObjDB.On("Link", mock.Anything, repoID, oid).Return(nil)

		request := &model.GitLFSBatchRequest{
			Operation: "upload",
			Objects: []*model.GitLFSPointer{
				{OID: oid, Size: 1024},
			},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Objects, 1)
		assert.True(t, response.Objects[0].Authenticated)
		assert.Nil(t, response.Objects[0].Actions) // No actions needed
	})

	t.Run("lfs_not_enabled", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)

		domain := NewLFSDomain(
			mockRepoDB,
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
			LFSEnabled: false,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)

		request := &model.GitLFSBatchRequest{
			Operation: "download",
			Objects:   []*model.GitLFSPointer{{OID: "test", Size: 100}},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		assert.ErrorIs(t, err, ErrLFSNotEnabled)
		assert.Nil(t, response)
	})

	t.Run("access_denied", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockAccessCtrl := new(MockGitAccessControl)

		domain := NewLFSDomain(
			mockRepoDB,
			nil,
			nil,
			mockAccessCtrl,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionRead).
			Return(&model.GitAccessResult{Allowed: false}, nil)

		request := &model.GitLFSBatchRequest{
			Operation: "download",
			Objects:   []*model.GitLFSPointer{{OID: "test", Size: 100}},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		assert.ErrorIs(t, err, ErrAccessDenied)
		assert.Nil(t, response)
	})

	t.Run("file_too_large", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockAccessCtrl := new(MockGitAccessControl)

		cfg := DefaultConfig()
		domain := NewLFSDomain(
			mockRepoDB,
			nil,
			nil,
			mockAccessCtrl,
			cfg,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionWrite).
			Return(&model.GitAccessResult{Allowed: true}, nil)

		request := &model.GitLFSBatchRequest{
			Operation: "upload",
			Objects: []*model.GitLFSPointer{
				{OID: "test", Size: cfg.MaxLFSFileSize + 1},
			},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Objects, 1)
		assert.NotNil(t, response.Objects[0].Error)
		assert.Equal(t, 422, response.Objects[0].Error.Code)
	})

	t.Run("download_object_not_found", func(t *testing.T) {
		mockRepoDB := new(MockGitRepoDB)
		mockLFSStorage := new(MockGitLFSStorage)
		mockAccessCtrl := new(MockGitAccessControl)

		domain := NewLFSDomain(
			mockRepoDB,
			nil,
			mockLFSStorage,
			mockAccessCtrl,
			nil,
			logger,
		)

		repoID := uuid.New()
		userID := uuid.New()
		repo := &model.GitRepo{
			ID:         repoID,
			LFSEnabled: true,
		}

		mockRepoDB.On("FindByID", mock.Anything, repoID).Return(repo, nil)
		mockAccessCtrl.On("CheckAccess", mock.Anything, userID, repoID, model.GitPermissionRead).
			Return(&model.GitAccessResult{Allowed: true}, nil)
		mockLFSStorage.On("Exists", mock.Anything, "missing").Return(false, nil)

		request := &model.GitLFSBatchRequest{
			Operation: "download",
			Objects: []*model.GitLFSPointer{
				{OID: "missing", Size: 1024},
			},
		}

		response, err := domain.ProcessBatch(context.Background(), repoID, userID, request)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Objects, 1)
		assert.NotNil(t, response.Objects[0].Error)
		assert.Equal(t, 404, response.Objects[0].Error.Code)
	})
}

func TestLFSDomain_GetObject(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSObjDB := new(MockGitLFSObjDB)

		domain := NewLFSDomain(nil, mockLFSObjDB, nil, nil, nil, logger)

		oid := "abc123"
		obj := &model.GitLFSObject{
			OID:  oid,
			Size: 1024,
		}

		mockLFSObjDB.On("FindByOID", mock.Anything, oid).Return(obj, nil)

		result, err := domain.GetObject(context.Background(), oid)

		require.NoError(t, err)
		assert.Equal(t, obj, result)
	})

	t.Run("not_found", func(t *testing.T) {
		mockLFSObjDB := new(MockGitLFSObjDB)

		domain := NewLFSDomain(nil, mockLFSObjDB, nil, nil, nil, logger)

		mockLFSObjDB.On("FindByOID", mock.Anything, "missing").Return(nil, nil)

		result, err := domain.GetObject(context.Background(), "missing")

		assert.ErrorIs(t, err, ErrLFSObjectNotFound)
		assert.Nil(t, result)
	})
}

func TestLFSDomain_CreateObject(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSObjDB := new(MockGitLFSObjDB)

		cfg := DefaultConfig()
		domain := NewLFSDomain(nil, mockLFSObjDB, nil, nil, cfg, logger)

		repoID := uuid.New()
		oid := "newobject"
		size := int64(2048)

		mockLFSObjDB.On("Create", mock.Anything, mock.AnythingOfType("*model.GitLFSObject")).Return(nil)
		mockLFSObjDB.On("Link", mock.Anything, repoID, oid).Return(nil)

		err := domain.CreateObject(context.Background(), repoID, oid, size)

		require.NoError(t, err)
		mockLFSObjDB.AssertExpectations(t)
	})
}

func TestLFSDomain_VerifyObject(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSObjDB := new(MockGitLFSObjDB)
		mockLFSStorage := new(MockGitLFSStorage)

		domain := NewLFSDomain(nil, mockLFSObjDB, mockLFSStorage, nil, nil, logger)

		oid := "verified"
		size := int64(1024)
		obj := &model.GitLFSObject{
			OID:  oid,
			Size: size,
		}

		mockLFSStorage.On("Exists", mock.Anything, oid).Return(true, nil)
		mockLFSObjDB.On("FindByOID", mock.Anything, oid).Return(obj, nil)

		err := domain.VerifyObject(context.Background(), oid, size)

		require.NoError(t, err)
	})

	t.Run("object_not_found_in_storage", func(t *testing.T) {
		mockLFSStorage := new(MockGitLFSStorage)

		domain := NewLFSDomain(nil, nil, mockLFSStorage, nil, nil, logger)

		mockLFSStorage.On("Exists", mock.Anything, "missing").Return(false, nil)

		err := domain.VerifyObject(context.Background(), "missing", 1024)

		assert.ErrorIs(t, err, ErrLFSObjectNotFound)
	})

	t.Run("size_mismatch", func(t *testing.T) {
		mockLFSObjDB := new(MockGitLFSObjDB)
		mockLFSStorage := new(MockGitLFSStorage)

		domain := NewLFSDomain(nil, mockLFSObjDB, mockLFSStorage, nil, nil, logger)

		oid := "mismatch"
		obj := &model.GitLFSObject{
			OID:  oid,
			Size: 1024, // Actual size
		}

		mockLFSStorage.On("Exists", mock.Anything, oid).Return(true, nil)
		mockLFSObjDB.On("FindByOID", mock.Anything, oid).Return(obj, nil)

		err := domain.VerifyObject(context.Background(), oid, 2048) // Expected size differs

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "size mismatch")
	})
}

func TestLFSDomain_Upload(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSStorage := new(MockGitLFSStorage)

		domain := NewLFSDomain(nil, nil, mockLFSStorage, nil, nil, logger)

		oid := "uploadtest"
		content := "test content"
		reader := strings.NewReader(content)
		size := int64(len(content))

		mockLFSStorage.On("Upload", mock.Anything, oid, mock.Anything, size).Return(nil)

		err := domain.Upload(context.Background(), oid, reader, size)

		require.NoError(t, err)
	})
}

func TestLFSDomain_Download(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSStorage := new(MockGitLFSStorage)

		domain := NewLFSDomain(nil, nil, mockLFSStorage, nil, nil, logger)

		oid := "downloadtest"
		content := "test content"
		size := int64(len(content))

		mockLFSStorage.On("Download", mock.Anything, oid).Return(io.NopCloser(strings.NewReader(content)), size, nil)

		reader, gotSize, err := domain.Download(context.Background(), oid)

		require.NoError(t, err)
		assert.NotNil(t, reader)
		assert.Equal(t, size, gotSize)
		reader.Close()
	})
}

func TestLFSDomain_GenerateUploadURL(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSStorage := new(MockGitLFSStorage)

		cfg := DefaultConfig()
		domain := NewLFSDomain(nil, nil, mockLFSStorage, nil, cfg, logger)

		oid := "urltest"
		size := int64(1024)
		expiresAt := time.Now().Add(cfg.PresignedURLExpiry)

		mockLFSStorage.On("GenerateUploadURL", mock.Anything, oid, size, cfg.PresignedURLExpiry).
			Return(&outbound.GitPresignedURL{
				URL:       "https://example.com/upload",
				Method:    "PUT",
				ExpiresAt: expiresAt,
			}, nil)

		result, err := domain.GenerateUploadURL(context.Background(), oid, size)

		require.NoError(t, err)
		assert.Equal(t, "https://example.com/upload", result.URL)
		assert.Equal(t, "PUT", result.Method)
	})
}

func TestLFSDomain_GenerateDownloadURL(t *testing.T) {
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		mockLFSStorage := new(MockGitLFSStorage)

		cfg := DefaultConfig()
		domain := NewLFSDomain(nil, nil, mockLFSStorage, nil, cfg, logger)

		oid := "downloadurltest"
		expiresAt := time.Now().Add(cfg.PresignedURLExpiry)

		mockLFSStorage.On("GenerateDownloadURL", mock.Anything, oid, cfg.PresignedURLExpiry).
			Return(&outbound.GitPresignedURL{
				URL:       "https://example.com/download",
				Method:    "GET",
				ExpiresAt: expiresAt,
			}, nil)

		result, err := domain.GenerateDownloadURL(context.Background(), oid)

		require.NoError(t, err)
		assert.Equal(t, "https://example.com/download", result.URL)
		assert.Equal(t, "GET", result.Method)
	})
}
