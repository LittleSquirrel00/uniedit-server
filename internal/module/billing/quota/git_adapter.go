package quota

import (
	"context"

	"github.com/google/uuid"
)

// GitStorageQuotaAdapter adapts StorageQuotaChecker to the git module's StorageQuotaChecker interface.
// This allows the billing quota checker to be used by the git module for storage quota enforcement.
type GitStorageQuotaAdapter struct {
	checker *StorageQuotaChecker
}

// NewGitStorageQuotaAdapter creates a new Git storage quota adapter.
func NewGitStorageQuotaAdapter(checker *StorageQuotaChecker) *GitStorageQuotaAdapter {
	return &GitStorageQuotaAdapter{
		checker: checker,
	}
}

// GetStorageQuota returns the storage quota in bytes for a user.
// Returns -1 for unlimited quota.
func (a *GitStorageQuotaAdapter) GetStorageQuota(ctx context.Context, userID uuid.UUID) (int64, error) {
	sub, err := a.checker.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return -1, nil // Default to unlimited on error
	}

	plan := sub.Plan
	if plan == nil || plan.IsUnlimitedGitStorage() {
		return -1, nil
	}

	// Convert MB to bytes
	return plan.GitStorageMB * 1024 * 1024, nil
}

// GetStorageUsed returns the storage used in bytes for a user.
func (a *GitStorageQuotaAdapter) GetStorageUsed(ctx context.Context, userID uuid.UUID) (int64, error) {
	usage, err := a.checker.GetUserStorageUsage(ctx, userID)
	if err != nil {
		return 0, err
	}
	return usage.GitBytes, nil
}

// LFSStorageQuotaAdapter adapts StorageQuotaChecker for LFS storage quota checks.
type LFSStorageQuotaAdapter struct {
	checker *StorageQuotaChecker
}

// NewLFSStorageQuotaAdapter creates a new LFS storage quota adapter.
func NewLFSStorageQuotaAdapter(checker *StorageQuotaChecker) *LFSStorageQuotaAdapter {
	return &LFSStorageQuotaAdapter{
		checker: checker,
	}
}

// CheckLFSQuota checks if the user has LFS storage quota for the given additional bytes.
// Returns nil if quota is available, error otherwise.
func (a *LFSStorageQuotaAdapter) CheckLFSQuota(ctx context.Context, userID uuid.UUID, additionalBytes int64) error {
	return a.checker.CheckLFSStorageQuota(ctx, userID, additionalBytes)
}

// GetLFSQuota returns the LFS storage quota in bytes for a user.
// Returns -1 for unlimited quota.
func (a *LFSStorageQuotaAdapter) GetLFSQuota(ctx context.Context, userID uuid.UUID) (int64, error) {
	sub, err := a.checker.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return -1, nil // Default to unlimited on error
	}

	plan := sub.Plan
	if plan == nil || plan.IsUnlimitedLFSStorage() {
		return -1, nil
	}

	// Convert MB to bytes
	return plan.LFSStorageMB * 1024 * 1024, nil
}

// GetLFSUsed returns the LFS storage used in bytes for a user.
func (a *LFSStorageQuotaAdapter) GetLFSUsed(ctx context.Context, userID uuid.UUID) (int64, error) {
	usage, err := a.checker.GetUserStorageUsage(ctx, userID)
	if err != nil {
		return 0, err
	}
	return usage.LFSBytes, nil
}

// InvalidateCache invalidates the storage cache for a user.
func (a *LFSStorageQuotaAdapter) InvalidateCache(ctx context.Context, userID uuid.UUID) {
	a.checker.InvalidateStorageCache(ctx, userID)
}
