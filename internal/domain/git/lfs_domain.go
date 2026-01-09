package git

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
)

// LFSDomain implements the Git LFS domain logic.
type LFSDomain struct {
	repoDB     outbound.GitRepoDatabasePort
	lfsObjDB   outbound.GitLFSObjectDatabasePort
	lfsStorage outbound.GitLFSStoragePort
	accessCtrl outbound.GitAccessControlPort
	cfg        *Config
	logger     *zap.Logger
}

// NewLFSDomain creates a new LFS domain.
func NewLFSDomain(
	repoDB outbound.GitRepoDatabasePort,
	lfsObjDB outbound.GitLFSObjectDatabasePort,
	lfsStorage outbound.GitLFSStoragePort,
	accessCtrl outbound.GitAccessControlPort,
	cfg *Config,
	logger *zap.Logger,
) *LFSDomain {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	_ = cfg.Validate()

	return &LFSDomain{
		repoDB:     repoDB,
		lfsObjDB:   lfsObjDB,
		lfsStorage: lfsStorage,
		accessCtrl: accessCtrl,
		cfg:        cfg,
		logger:     logger,
	}
}

// ProcessBatch processes an LFS batch request.
func (d *LFSDomain) ProcessBatch(ctx context.Context, repoID, userID uuid.UUID, request *model.GitLFSBatchRequest) (*model.GitLFSBatchResponse, error) {
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

	// Check access
	requiredPerm := model.GitPermissionRead
	if request.Operation == "upload" {
		requiredPerm = model.GitPermissionWrite
	}

	result, err := d.accessCtrl.CheckAccess(ctx, userID, repoID, requiredPerm)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, ErrAccessDenied
	}

	// Process objects
	response := &model.GitLFSBatchResponse{
		Transfer: "basic",
		Objects:  make([]*model.GitLFSObjectResponse, 0, len(request.Objects)),
	}

	for _, obj := range request.Objects {
		objResp := d.processBatchObject(ctx, repoID, obj, request.Operation)
		response.Objects = append(response.Objects, objResp)
	}

	return response, nil
}

func (d *LFSDomain) processBatchObject(ctx context.Context, repoID uuid.UUID, obj *model.GitLFSPointer, operation string) *model.GitLFSObjectResponse {
	resp := &model.GitLFSObjectResponse{
		OID:  obj.OID,
		Size: obj.Size,
	}

	// Check file size limit
	if obj.Size > d.cfg.MaxLFSFileSize {
		resp.Error = &model.GitLFSError{
			Code:    422,
			Message: fmt.Sprintf("file size exceeds maximum allowed (%d bytes)", d.cfg.MaxLFSFileSize),
		}
		return resp
	}

	if operation == "download" {
		// Check if object exists
		exists, err := d.lfsStorage.Exists(ctx, obj.OID)
		if err != nil {
			resp.Error = &model.GitLFSError{
				Code:    500,
				Message: "internal error",
			}
			return resp
		}
		if !exists {
			resp.Error = &model.GitLFSError{
				Code:    404,
				Message: "object not found",
			}
			return resp
		}

		// Generate download URL
		presigned, err := d.lfsStorage.GenerateDownloadURL(ctx, obj.OID, d.cfg.PresignedURLExpiry)
		if err != nil {
			resp.Error = &model.GitLFSError{
				Code:    500,
				Message: "failed to generate download URL",
			}
			return resp
		}

		expiresIn := int(d.cfg.PresignedURLExpiry.Seconds())
		resp.Actions = map[string]*model.GitLFSAction{
			"download": {
				Href:      presigned.URL,
				ExpiresIn: expiresIn,
			},
		}
	} else if operation == "upload" {
		// Check if object already exists
		exists, err := d.lfsStorage.Exists(ctx, obj.OID)
		if err != nil {
			resp.Error = &model.GitLFSError{
				Code:    500,
				Message: "internal error",
			}
			return resp
		}

		if exists {
			// Object exists, just link it
			if err := d.lfsObjDB.Link(ctx, repoID, obj.OID); err != nil {
				d.logger.Warn("failed to link existing LFS object", zap.Error(err))
			}
			resp.Authenticated = true
			return resp
		}

		// Generate upload URL
		presigned, err := d.lfsStorage.GenerateUploadURL(ctx, obj.OID, obj.Size, d.cfg.PresignedURLExpiry)
		if err != nil {
			resp.Error = &model.GitLFSError{
				Code:    500,
				Message: "failed to generate upload URL",
			}
			return resp
		}

		expiresIn := int(d.cfg.PresignedURLExpiry.Seconds())
		resp.Actions = map[string]*model.GitLFSAction{
			"upload": {
				Href:      presigned.URL,
				ExpiresIn: expiresIn,
			},
			"verify": {
				Href:      d.cfg.BaseURL + "/lfs/objects/" + obj.OID + "/verify",
				ExpiresIn: expiresIn,
			},
		}
	}

	return resp
}

// GetObject gets an LFS object.
func (d *LFSDomain) GetObject(ctx context.Context, oid string) (*model.GitLFSObject, error) {
	obj, err := d.lfsObjDB.FindByOID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, ErrLFSObjectNotFound
	}
	return obj, nil
}

// CreateObject creates an LFS object record.
func (d *LFSDomain) CreateObject(ctx context.Context, repoID uuid.UUID, oid string, size int64) error {
	storageKey := fmt.Sprintf("%s%s", d.cfg.LFSPrefix, oid)

	obj := &model.GitLFSObject{
		OID:        oid,
		Size:       size,
		StorageKey: storageKey,
		CreatedAt:  time.Now(),
	}

	if err := d.lfsObjDB.Create(ctx, obj); err != nil {
		return fmt.Errorf("create LFS object: %w", err)
	}

	// Link to repository
	if err := d.lfsObjDB.Link(ctx, repoID, oid); err != nil {
		return fmt.Errorf("link LFS object: %w", err)
	}

	return nil
}

// LinkObject links an LFS object to a repository.
func (d *LFSDomain) LinkObject(ctx context.Context, repoID uuid.UUID, oid string) error {
	return d.lfsObjDB.Link(ctx, repoID, oid)
}

// Upload uploads an LFS object.
func (d *LFSDomain) Upload(ctx context.Context, oid string, reader io.Reader, size int64) error {
	return d.lfsStorage.Upload(ctx, oid, reader, size)
}

// Download downloads an LFS object.
func (d *LFSDomain) Download(ctx context.Context, oid string) (io.ReadCloser, int64, error) {
	return d.lfsStorage.Download(ctx, oid)
}

// VerifyObject verifies an LFS object.
func (d *LFSDomain) VerifyObject(ctx context.Context, oid string, expectedSize int64) error {
	exists, err := d.lfsStorage.Exists(ctx, oid)
	if err != nil {
		return fmt.Errorf("check object existence: %w", err)
	}
	if !exists {
		return ErrLFSObjectNotFound
	}

	obj, err := d.lfsObjDB.FindByOID(ctx, oid)
	if err != nil {
		return err
	}
	if obj == nil {
		return ErrLFSObjectNotFound
	}

	if obj.Size != expectedSize {
		return fmt.Errorf("size mismatch: expected %d, got %d", expectedSize, obj.Size)
	}

	return nil
}

// GenerateUploadURL generates a presigned upload URL.
func (d *LFSDomain) GenerateUploadURL(ctx context.Context, oid string, size int64) (*inbound.GitPresignedURLResult, error) {
	presigned, err := d.lfsStorage.GenerateUploadURL(ctx, oid, size, d.cfg.PresignedURLExpiry)
	if err != nil {
		return nil, err
	}

	return &inbound.GitPresignedURLResult{
		URL:       presigned.URL,
		Method:    presigned.Method,
		ExpiresAt: presigned.ExpiresAt,
	}, nil
}

// GenerateDownloadURL generates a presigned download URL.
func (d *LFSDomain) GenerateDownloadURL(ctx context.Context, oid string) (*inbound.GitPresignedURLResult, error) {
	presigned, err := d.lfsStorage.GenerateDownloadURL(ctx, oid, d.cfg.PresignedURLExpiry)
	if err != nil {
		return nil, err
	}

	return &inbound.GitPresignedURLResult{
		URL:       presigned.URL,
		Method:    presigned.Method,
		ExpiresAt: presigned.ExpiresAt,
	}, nil
}

// Compile-time interface check
var _ inbound.GitLFSDomain = (*LFSDomain)(nil)
