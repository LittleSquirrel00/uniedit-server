package lfs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/uniedit/server/internal/module/git/storage"
	"github.com/uniedit/server/internal/shared/config"
)

// StorageManager handles LFS object storage operations.
type StorageManager struct {
	r2Client *storage.R2Client
	cfg      *config.GitConfig
}

// NewStorageManager creates a new LFS storage manager.
func NewStorageManager(r2Client *storage.R2Client, cfg *config.GitConfig) *StorageManager {
	return &StorageManager{
		r2Client: r2Client,
		cfg:      cfg,
	}
}

// UploadAction represents an LFS upload action.
type UploadAction struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in"`
	ExpiresAt string            `json:"expires_at"`
}

// DownloadAction represents an LFS download action.
type DownloadAction struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in"`
	ExpiresAt string            `json:"expires_at"`
}

// VerifyAction represents an LFS verify action.
type VerifyAction struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header,omitempty"`
}

// GenerateUploadURL generates a presigned URL for uploading an LFS object.
func (m *StorageManager) GenerateUploadURL(ctx context.Context, oid string, size int64) (*UploadAction, error) {
	// Check file size limit
	if m.cfg.LFSMaxFileSize > 0 && size > m.cfg.LFSMaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", size, m.cfg.LFSMaxFileSize)
	}

	key := m.getObjectKey(oid)
	expiry := m.cfg.LFSURLExpiry
	if expiry == 0 {
		expiry = time.Hour
	}

	presigned, err := m.r2Client.PresignUpload(ctx, key, size, expiry)
	if err != nil {
		return nil, fmt.Errorf("generate upload URL: %w", err)
	}

	expiresAt := presigned.ExpiresAt.UTC().Format(time.RFC3339)
	expiresIn := int(expiry.Seconds())

	return &UploadAction{
		Href:      presigned.URL,
		ExpiresIn: expiresIn,
		ExpiresAt: expiresAt,
	}, nil
}

// GenerateDownloadURL generates a presigned URL for downloading an LFS object.
func (m *StorageManager) GenerateDownloadURL(ctx context.Context, oid string) (*DownloadAction, error) {
	key := m.getObjectKey(oid)
	expiry := m.cfg.LFSURLExpiry
	if expiry == 0 {
		expiry = time.Hour
	}

	presigned, err := m.r2Client.PresignDownload(ctx, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("generate download URL: %w", err)
	}

	expiresAt := presigned.ExpiresAt.UTC().Format(time.RFC3339)
	expiresIn := int(expiry.Seconds())

	return &DownloadAction{
		Href:      presigned.URL,
		ExpiresIn: expiresIn,
		ExpiresAt: expiresAt,
	}, nil
}

// ObjectExists checks if an LFS object exists in storage.
func (m *StorageManager) ObjectExists(ctx context.Context, oid string) (bool, error) {
	key := m.getObjectKey(oid)
	return m.r2Client.ObjectExists(ctx, key)
}

// GetObjectInfo retrieves metadata for an LFS object.
func (m *StorageManager) GetObjectInfo(ctx context.Context, oid string) (*storage.ObjectInfo, error) {
	key := m.getObjectKey(oid)
	return m.r2Client.HeadObject(ctx, key)
}

// DeleteObject deletes an LFS object from storage.
func (m *StorageManager) DeleteObject(ctx context.Context, oid string) error {
	key := m.getObjectKey(oid)
	return m.r2Client.DeleteObject(ctx, key)
}

// getObjectKey generates the R2 key for an LFS object.
// Uses a sharded structure: lfs/{oid[0:2]}/{oid[2:4]}/{oid}
func (m *StorageManager) getObjectKey(oid string) string {
	prefix := m.cfg.LFSPrefix
	if prefix == "" {
		prefix = "lfs/"
	}

	// Shard by first 4 characters of OID for better distribution
	if len(oid) >= 4 {
		return fmt.Sprintf("%s%s/%s/%s", prefix, oid[0:2], oid[2:4], oid)
	}
	return prefix + oid
}

// GetStorageKey returns the full storage key for an LFS object.
func (m *StorageManager) GetStorageKey(oid string) string {
	return m.getObjectKey(oid)
}

// LFSObjectRecord represents an LFS object record for database storage.
type LFSObjectRecord struct {
	OID         string
	Size        int64
	StorageKey  string
	ContentType string
}

// CreateObjectRecord creates a new LFS object record.
func (m *StorageManager) CreateObjectRecord(oid string, size int64, contentType string) *LFSObjectRecord {
	return &LFSObjectRecord{
		OID:         oid,
		Size:        size,
		StorageKey:  m.getObjectKey(oid),
		ContentType: contentType,
	}
}

// QuotaChecker defines the interface for checking LFS quota.
type QuotaChecker interface {
	GetLFSQuota(ctx context.Context, userID uuid.UUID) (int64, error)
	GetLFSUsed(ctx context.Context, userID uuid.UUID) (int64, error)
}

// CheckQuota checks if adding size bytes would exceed the user's LFS quota.
func (m *StorageManager) CheckQuota(ctx context.Context, checker QuotaChecker, userID uuid.UUID, additionalSize int64) error {
	if checker == nil {
		return nil // No quota checking
	}

	quota, err := checker.GetLFSQuota(ctx, userID)
	if err != nil {
		return fmt.Errorf("get quota: %w", err)
	}
	if quota <= 0 {
		return nil // Unlimited
	}

	used, err := checker.GetLFSUsed(ctx, userID)
	if err != nil {
		return fmt.Errorf("get used: %w", err)
	}

	if used+additionalSize > quota {
		return fmt.Errorf("LFS quota exceeded: used %d + %d > quota %d", used, additionalSize, quota)
	}

	return nil
}
