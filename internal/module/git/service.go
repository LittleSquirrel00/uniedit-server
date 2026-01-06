package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/module/git/storage"
	"github.com/uniedit/server/internal/shared/config"
)

// StorageQuotaChecker defines the interface for checking storage quota.
type StorageQuotaChecker interface {
	GetStorageQuota(ctx context.Context, userID uuid.UUID) (int64, error)
	GetStorageUsed(ctx context.Context, userID uuid.UUID) (int64, error)
}

// ServiceInterface defines the git service interface.
type ServiceInterface interface {
	// Repository operations
	CreateRepo(ctx context.Context, ownerID uuid.UUID, req *CreateRepoRequest) (*GitRepo, error)
	GetRepo(ctx context.Context, id uuid.UUID) (*GitRepo, error)
	GetRepoByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*GitRepo, error)
	ListRepos(ctx context.Context, ownerID uuid.UUID, filter *RepoFilter) ([]*GitRepo, int64, error)
	ListPublicRepos(ctx context.Context, filter *RepoFilter) ([]*GitRepo, int64, error)
	UpdateRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateRepoRequest) (*GitRepo, error)
	DeleteRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// Access control
	CheckAccess(ctx context.Context, repoID uuid.UUID, userID uuid.UUID, required Permission) error
	CanAccess(ctx context.Context, repoID uuid.UUID, userID *uuid.UUID, required Permission) (bool, error)

	// Collaborator operations
	AddCollaborator(ctx context.Context, repoID uuid.UUID, ownerID uuid.UUID, targetUserID uuid.UUID, permission Permission) error
	ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*RepoCollaborator, error)
	UpdateCollaborator(ctx context.Context, repoID uuid.UUID, ownerID uuid.UUID, targetUserID uuid.UUID, permission Permission) error
	RemoveCollaborator(ctx context.Context, repoID uuid.UUID, ownerID uuid.UUID, targetUserID uuid.UUID) error

	// Pull request operations
	CreatePR(ctx context.Context, repoID uuid.UUID, authorID uuid.UUID, req *CreatePRRequest) (*PullRequest, error)
	GetPR(ctx context.Context, repoID uuid.UUID, number int) (*PullRequest, error)
	ListPRs(ctx context.Context, repoID uuid.UUID, status *PRStatus, limit, offset int) ([]*PullRequest, int64, error)
	UpdatePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID, req *UpdatePRRequest) (*PullRequest, error)
	MergePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID) (*PullRequest, error)

	// Storage operations
	GetStorageStats(ctx context.Context, repoID uuid.UUID) (*StorageStats, error)
	GetUserStorageStats(ctx context.Context, userID uuid.UUID) (*UserStorageStats, error)

	// R2 filesystem access
	GetR2Filesystem(ctx context.Context, repoID uuid.UUID) (*storage.R2Filesystem, error)

	// Git protocol helpers
	UpdatePushedAt(ctx context.Context, repoID uuid.UUID) error
	UpdateRepoSize(ctx context.Context, repoID uuid.UUID, sizeBytes, lfsSizeBytes int64) error
}

// StorageStats represents repository storage statistics.
type StorageStats struct {
	RepoSizeBytes  int64 `json:"repo_size_bytes"`
	LFSSizeBytes   int64 `json:"lfs_size_bytes"`
	TotalSizeBytes int64 `json:"total_size_bytes"`
	LFSObjectCount int   `json:"lfs_object_count"`
}

// UserStorageStats represents user-level storage statistics.
type UserStorageStats struct {
	TotalUsedBytes  int64 `json:"total_used_bytes"`
	QuotaBytes      int64 `json:"quota_bytes"`
	RemainingBytes  int64 `json:"remaining_bytes"`
	RepoCount       int   `json:"repo_count"`
}

// Service implements git operations.
type Service struct {
	repo         Repository
	r2Client     *storage.R2Client
	quotaChecker StorageQuotaChecker
	cfg          *config.GitConfig
	logger       *zap.Logger
}

// NewService creates a new git service.
func NewService(
	repo Repository,
	r2Client *storage.R2Client,
	quotaChecker StorageQuotaChecker,
	cfg *config.GitConfig,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:         repo,
		r2Client:     r2Client,
		quotaChecker: quotaChecker,
		cfg:          cfg,
		logger:       logger,
	}
}

// --- Repository Operations ---

var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// CreateRepo creates a new repository.
func (s *Service) CreateRepo(ctx context.Context, ownerID uuid.UUID, req *CreateRepoRequest) (*GitRepo, error) {
	// Generate slug from name
	slug := generateSlug(req.Name)
	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidRepoName
	}

	// Check if repo already exists
	_, err := s.repo.GetByOwnerAndSlug(ctx, ownerID, slug)
	if err == nil {
		return nil, ErrRepoAlreadyExists
	}
	if err != ErrRepoNotFound {
		return nil, fmt.Errorf("check existing repo: %w", err)
	}

	// Check storage quota if quota checker is available
	if s.quotaChecker != nil {
		quota, err := s.quotaChecker.GetStorageQuota(ctx, ownerID)
		if err != nil {
			s.logger.Warn("failed to get storage quota", zap.Error(err))
		} else if quota > 0 {
			used, err := s.quotaChecker.GetStorageUsed(ctx, ownerID)
			if err != nil {
				s.logger.Warn("failed to get storage used", zap.Error(err))
			} else if used >= quota {
				return nil, ErrStorageQuotaExceeded
			}
		}
	}

	// Create repository record
	repoID := uuid.New()
	storagePath := fmt.Sprintf("%s%s/%s/", s.cfg.RepoPrefix, ownerID.String(), repoID.String())

	repoType := RepoTypeCode
	if req.Type != "" {
		repoType = RepoType(req.Type)
	}

	visibility := VisibilityPrivate
	if req.Visibility != "" {
		visibility = Visibility(req.Visibility)
	}

	gitRepo := &GitRepo{
		ID:            repoID,
		OwnerID:       ownerID,
		Name:          req.Name,
		Slug:          slug,
		RepoType:      repoType,
		Visibility:    visibility,
		Description:   req.Description,
		DefaultBranch: "main",
		LFSEnabled:    req.LFSEnabled,
		StoragePath:   storagePath,
	}

	// Initialize bare Git repository in R2
	if err := s.initBareRepo(ctx, gitRepo); err != nil {
		return nil, fmt.Errorf("init bare repo: %w", err)
	}

	// Save to database
	if err := s.repo.Create(ctx, gitRepo); err != nil {
		// Cleanup R2 on failure
		s.cleanupR2Storage(ctx, storagePath)
		return nil, fmt.Errorf("create repo: %w", err)
	}

	s.logger.Info("repository created",
		zap.String("repo_id", repoID.String()),
		zap.String("owner_id", ownerID.String()),
		zap.String("name", req.Name),
	)

	return gitRepo, nil
}

// initBareRepo initializes a bare Git repository in R2.
func (s *Service) initBareRepo(ctx context.Context, gitRepo *GitRepo) error {
	// Create R2 filesystem for this repo
	fs := storage.NewR2Filesystem(s.r2Client, gitRepo.StoragePath)

	// Create essential bare repo structure
	// HEAD file
	headFile, err := fs.Create("HEAD")
	if err != nil {
		return fmt.Errorf("create HEAD: %w", err)
	}
	_, err = headFile.Write([]byte("ref: refs/heads/" + gitRepo.DefaultBranch + "\n"))
	if err != nil {
		headFile.Close()
		return fmt.Errorf("write HEAD: %w", err)
	}
	if err := headFile.Close(); err != nil {
		return fmt.Errorf("close HEAD: %w", err)
	}

	// config file - write raw git config content
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = true
`
	configFile, err := fs.Create("config")
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	_, err = configFile.Write([]byte(configContent))
	if err != nil {
		configFile.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := configFile.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}

	// Create directories (in R2 these are implied by object keys)
	dirs := []string{"objects", "objects/info", "objects/pack", "refs", "refs/heads", "refs/tags", "info"}
	for _, dir := range dirs {
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Create info/exclude file
	excludeFile, err := fs.Create("info/exclude")
	if err != nil {
		return fmt.Errorf("create exclude: %w", err)
	}
	_, err = excludeFile.Write([]byte("# git ls-files --others --exclude-from=.git/info/exclude\n"))
	if err != nil {
		excludeFile.Close()
		return fmt.Errorf("write exclude: %w", err)
	}
	if err := excludeFile.Close(); err != nil {
		return fmt.Errorf("close exclude: %w", err)
	}

	// Create description file
	descFile, err := fs.Create("description")
	if err != nil {
		return fmt.Errorf("create description: %w", err)
	}
	desc := "Unnamed repository; edit this file 'description' to name the repository.\n"
	if gitRepo.Description != "" {
		desc = gitRepo.Description + "\n"
	}
	_, err = descFile.Write([]byte(desc))
	if err != nil {
		descFile.Close()
		return fmt.Errorf("write description: %w", err)
	}
	if err := descFile.Close(); err != nil {
		return fmt.Errorf("close description: %w", err)
	}

	return nil
}

// cleanupR2Storage removes all objects with the given prefix.
func (s *Service) cleanupR2Storage(ctx context.Context, prefix string) {
	objects, err := s.r2Client.ListObjects(ctx, prefix, 1000)
	if err != nil {
		s.logger.Error("failed to list objects for cleanup", zap.Error(err))
		return
	}

	keys := make([]string, len(objects))
	for i, obj := range objects {
		keys[i] = obj.Key
	}

	if len(keys) > 0 {
		if err := s.r2Client.DeleteObjects(ctx, keys); err != nil {
			s.logger.Error("failed to cleanup R2 storage", zap.Error(err))
		}
	}
}

// GetRepo retrieves a repository by ID.
func (s *Service) GetRepo(ctx context.Context, id uuid.UUID) (*GitRepo, error) {
	return s.repo.GetByID(ctx, id)
}

// GetRepoByOwnerAndSlug retrieves a repository by owner and slug.
func (s *Service) GetRepoByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*GitRepo, error) {
	return s.repo.GetByOwnerAndSlug(ctx, ownerID, slug)
}

// ListRepos lists repositories for a user.
func (s *Service) ListRepos(ctx context.Context, ownerID uuid.UUID, filter *RepoFilter) ([]*GitRepo, int64, error) {
	return s.repo.List(ctx, ownerID, filter)
}

// ListPublicRepos lists public repositories.
func (s *Service) ListPublicRepos(ctx context.Context, filter *RepoFilter) ([]*GitRepo, int64, error) {
	return s.repo.ListPublic(ctx, filter)
}

// UpdateRepo updates a repository.
func (s *Service) UpdateRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateRepoRequest) (*GitRepo, error) {
	gitRepo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership or admin permission
	if gitRepo.OwnerID != userID {
		if err := s.CheckAccess(ctx, id, userID, PermissionAdmin); err != nil {
			return nil, ErrNotOwner
		}
	}

	// Update fields
	if req.Name != "" {
		newSlug := generateSlug(req.Name)
		if !slugRegex.MatchString(newSlug) {
			return nil, ErrInvalidRepoName
		}
		// Check if new slug conflicts
		if newSlug != gitRepo.Slug {
			_, err := s.repo.GetByOwnerAndSlug(ctx, gitRepo.OwnerID, newSlug)
			if err == nil {
				return nil, ErrRepoAlreadyExists
			}
			if err != ErrRepoNotFound {
				return nil, fmt.Errorf("check slug conflict: %w", err)
			}
		}
		gitRepo.Name = req.Name
		gitRepo.Slug = newSlug
	}

	if req.Description != "" {
		gitRepo.Description = req.Description
	}

	if req.Visibility != "" {
		gitRepo.Visibility = Visibility(req.Visibility)
	}

	if req.DefaultBranch != "" {
		gitRepo.DefaultBranch = req.DefaultBranch
	}

	if req.LFSEnabled != nil {
		gitRepo.LFSEnabled = *req.LFSEnabled
	}

	if err := s.repo.Update(ctx, gitRepo); err != nil {
		return nil, fmt.Errorf("update repo: %w", err)
	}

	return gitRepo, nil
}

// DeleteRepo deletes a repository.
func (s *Service) DeleteRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	gitRepo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Only owner can delete
	if gitRepo.OwnerID != userID {
		return ErrNotOwner
	}

	// Delete from R2
	s.cleanupR2Storage(ctx, gitRepo.StoragePath)

	// Delete LFS links (cascade will handle this in DB, but let's be explicit)
	// LFS objects themselves are not deleted (content-addressable, may be shared)

	// Delete from database
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete repo: %w", err)
	}

	s.logger.Info("repository deleted",
		zap.String("repo_id", id.String()),
		zap.String("owner_id", userID.String()),
	)

	return nil
}

// --- Access Control ---

// CheckAccess checks if a user has the required permission level.
func (s *Service) CheckAccess(ctx context.Context, repoID uuid.UUID, userID uuid.UUID, required Permission) error {
	hasAccess, err := s.CanAccess(ctx, repoID, &userID, required)
	if err != nil {
		return err
	}
	if !hasAccess {
		return ErrAccessDenied
	}
	return nil
}

// CanAccess checks if a user can access a repository with the required permission.
// If userID is nil, only public read access is allowed.
func (s *Service) CanAccess(ctx context.Context, repoID uuid.UUID, userID *uuid.UUID, required Permission) (bool, error) {
	gitRepo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return false, err
	}

	// Public repos allow read access to everyone
	if gitRepo.Visibility == VisibilityPublic && required == PermissionRead {
		return true, nil
	}

	// Anonymous users can't do anything else
	if userID == nil {
		return false, nil
	}

	// Owner has full access
	if gitRepo.OwnerID == *userID {
		return true, nil
	}

	// Check collaborator permission
	collab, err := s.repo.GetCollaborator(ctx, repoID, *userID)
	if err != nil {
		if err == ErrNotCollaborator {
			return false, nil
		}
		return false, err
	}

	return hasPermission(collab.Permission, required), nil
}

// hasPermission checks if the granted permission level satisfies the required level.
func hasPermission(granted, required Permission) bool {
	levels := map[Permission]int{
		PermissionRead:  1,
		PermissionWrite: 2,
		PermissionAdmin: 3,
	}

	grantedLevel, ok := levels[granted]
	if !ok {
		return false
	}

	requiredLevel, ok := levels[required]
	if !ok {
		return false
	}

	return grantedLevel >= requiredLevel
}

// --- Collaborator Operations ---

// AddCollaborator adds a collaborator to a repository.
func (s *Service) AddCollaborator(ctx context.Context, repoID uuid.UUID, ownerID uuid.UUID, targetUserID uuid.UUID, permission Permission) error {
	gitRepo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// Only owner can add collaborators
	if gitRepo.OwnerID != ownerID {
		return ErrNotOwner
	}

	// Can't add owner as collaborator
	if targetUserID == ownerID {
		return ErrInvalidPermission
	}

	collab := &RepoCollaborator{
		RepoID:     repoID,
		UserID:     targetUserID,
		Permission: permission,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.AddCollaborator(ctx, collab); err != nil {
		return fmt.Errorf("add collaborator: %w", err)
	}

	return nil
}

// ListCollaborators lists collaborators of a repository.
func (s *Service) ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*RepoCollaborator, error) {
	return s.repo.ListCollaborators(ctx, repoID)
}

// UpdateCollaborator updates a collaborator's permission.
func (s *Service) UpdateCollaborator(ctx context.Context, repoID uuid.UUID, ownerID uuid.UUID, targetUserID uuid.UUID, permission Permission) error {
	gitRepo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// Only owner can update collaborators
	if gitRepo.OwnerID != ownerID {
		return ErrNotOwner
	}

	collab := &RepoCollaborator{
		RepoID:     repoID,
		UserID:     targetUserID,
		Permission: permission,
	}

	if err := s.repo.UpdateCollaborator(ctx, collab); err != nil {
		return fmt.Errorf("update collaborator: %w", err)
	}

	return nil
}

// RemoveCollaborator removes a collaborator from a repository.
func (s *Service) RemoveCollaborator(ctx context.Context, repoID uuid.UUID, ownerID uuid.UUID, targetUserID uuid.UUID) error {
	gitRepo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return err
	}

	// Only owner can remove collaborators
	if gitRepo.OwnerID != ownerID {
		return ErrNotOwner
	}

	if err := s.repo.RemoveCollaborator(ctx, repoID, targetUserID); err != nil {
		return fmt.Errorf("remove collaborator: %w", err)
	}

	return nil
}

// --- Pull Request Operations ---

// CreatePR creates a new pull request.
func (s *Service) CreatePR(ctx context.Context, repoID uuid.UUID, authorID uuid.UUID, req *CreatePRRequest) (*PullRequest, error) {
	// Check access
	if err := s.CheckAccess(ctx, repoID, authorID, PermissionRead); err != nil {
		return nil, err
	}

	// Validate branches
	if req.SourceBranch == req.TargetBranch {
		return nil, ErrSameBranch
	}

	// Get next PR number
	number, err := s.repo.GetNextPRNumber(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get next PR number: %w", err)
	}

	pr := &PullRequest{
		ID:           uuid.New(),
		RepoID:       repoID,
		Number:       number,
		Title:        req.Title,
		Description:  req.Description,
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
		Status:       PRStatusOpen,
		AuthorID:     authorID,
	}

	if err := s.repo.CreatePR(ctx, pr); err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	return pr, nil
}

// GetPR retrieves a pull request by number.
func (s *Service) GetPR(ctx context.Context, repoID uuid.UUID, number int) (*PullRequest, error) {
	return s.repo.GetPRByNumber(ctx, repoID, number)
}

// ListPRs lists pull requests for a repository.
func (s *Service) ListPRs(ctx context.Context, repoID uuid.UUID, status *PRStatus, limit, offset int) ([]*PullRequest, int64, error) {
	return s.repo.ListPRs(ctx, repoID, status, limit, offset)
}

// UpdatePR updates a pull request.
func (s *Service) UpdatePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID, req *UpdatePRRequest) (*PullRequest, error) {
	pr, err := s.repo.GetPRByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}

	// Check if user is author or has write access
	if pr.AuthorID != userID {
		if err := s.CheckAccess(ctx, repoID, userID, PermissionWrite); err != nil {
			return nil, err
		}
	}

	// Can't update merged/closed PR
	if pr.Status == PRStatusMerged {
		return nil, ErrPRAlreadyMerged
	}

	if req.Title != "" {
		pr.Title = req.Title
	}
	if req.Description != "" {
		pr.Description = req.Description
	}
	if req.Status != "" {
		if req.Status == "closed" && pr.Status == PRStatusOpen {
			now := time.Now()
			pr.Status = PRStatusClosed
			pr.ClosedAt = &now
		} else if req.Status == "open" && pr.Status == PRStatusClosed {
			pr.Status = PRStatusOpen
			pr.ClosedAt = nil
		}
	}

	if err := s.repo.UpdatePR(ctx, pr); err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}

	return pr, nil
}

// MergePR merges a pull request.
func (s *Service) MergePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID) (*PullRequest, error) {
	// Check write access
	if err := s.CheckAccess(ctx, repoID, userID, PermissionWrite); err != nil {
		return nil, err
	}

	pr, err := s.repo.GetPRByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}

	if pr.Status == PRStatusMerged {
		return nil, ErrPRAlreadyMerged
	}
	if pr.Status == PRStatusClosed {
		return nil, ErrPRAlreadyClosed
	}

	// TODO: Perform actual Git merge operation using go-git
	// For now, just update the status

	now := time.Now()
	pr.Status = PRStatusMerged
	pr.MergedBy = &userID
	pr.MergedAt = &now

	if err := s.repo.UpdatePR(ctx, pr); err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}

	return pr, nil
}

// --- Storage Operations ---

// GetStorageStats returns storage statistics for a repository.
func (s *Service) GetStorageStats(ctx context.Context, repoID uuid.UUID) (*StorageStats, error) {
	gitRepo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	// Get LFS object count
	lfsObjects, err := s.repo.ListRepoLFSObjects(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list LFS objects: %w", err)
	}

	return &StorageStats{
		RepoSizeBytes:  gitRepo.SizeBytes,
		LFSSizeBytes:   gitRepo.LFSSizeBytes,
		TotalSizeBytes: gitRepo.TotalSize(),
		LFSObjectCount: len(lfsObjects),
	}, nil
}

// GetUserStorageStats returns storage statistics for a user.
func (s *Service) GetUserStorageStats(ctx context.Context, userID uuid.UUID) (*UserStorageStats, error) {
	totalUsed, err := s.repo.GetUserTotalStorage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get total storage: %w", err)
	}

	// Get repo count
	repos, _, err := s.repo.List(ctx, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	var quota int64 = -1 // -1 means unlimited
	if s.quotaChecker != nil {
		q, err := s.quotaChecker.GetStorageQuota(ctx, userID)
		if err == nil {
			quota = q
		}
	}

	remaining := int64(-1)
	if quota > 0 {
		remaining = quota - totalUsed
		if remaining < 0 {
			remaining = 0
		}
	}

	return &UserStorageStats{
		TotalUsedBytes: totalUsed,
		QuotaBytes:     quota,
		RemainingBytes: remaining,
		RepoCount:      len(repos),
	}, nil
}

// GetR2Filesystem returns an R2 filesystem for a repository.
func (s *Service) GetR2Filesystem(ctx context.Context, repoID uuid.UUID) (*storage.R2Filesystem, error) {
	gitRepo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	return storage.NewR2Filesystem(s.r2Client, gitRepo.StoragePath), nil
}

// --- Helpers ---

// generateSlug generates a URL-safe slug from a name.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove invalid characters
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove leading/trailing dashes and collapse multiple dashes
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	return slug
}

// UpdatePushedAt updates the pushed_at timestamp for a repository.
func (s *Service) UpdatePushedAt(ctx context.Context, repoID uuid.UUID) error {
	now := time.Now()
	return s.repo.UpdatePushedAt(ctx, repoID, &now)
}

// UpdateRepoSize updates the size of a repository.
func (s *Service) UpdateRepoSize(ctx context.Context, repoID uuid.UUID, sizeBytes, lfsSizeBytes int64) error {
	return s.repo.UpdateSize(ctx, repoID, sizeBytes, lfsSizeBytes)
}

// GetBranch retrieves a branch reference.
func (s *Service) GetBranch(ctx context.Context, repoID uuid.UUID, branchName string) (*plumbing.Reference, error) {
	fs, err := s.GetR2Filesystem(ctx, repoID)
	if err != nil {
		return nil, err
	}

	// Read the reference file
	refPath := fmt.Sprintf("refs/heads/%s", branchName)
	file, err := fs.Open(refPath)
	if err != nil {
		return nil, ErrInvalidBranch
	}
	defer file.Close()

	// Read the hash
	buf := make([]byte, 40)
	n, err := file.Read(buf)
	if err != nil || n < 40 {
		return nil, ErrInvalidBranch
	}

	hash := plumbing.NewHash(string(buf[:40]))
	return plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), hash), nil
}
