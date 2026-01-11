package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/inbound"
	"github.com/uniedit/server/internal/port/outbound"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Domain implements the Git domain logic.
type Domain struct {
	repoDB       outbound.GitRepoDatabasePort
	collabDB     outbound.GitCollaboratorDatabasePort
	prDB         outbound.GitPullRequestDatabasePort
	lfsObjDB     outbound.GitLFSObjectDatabasePort
	lfsLockDB    outbound.GitLFSLockDatabasePort
	storage      outbound.GitStoragePort
	lfsStorage   outbound.GitLFSStoragePort
	quotaChecker inbound.GitStorageQuotaChecker
	cfg          *Config
	logger       *zap.Logger
}

// NewDomain creates a new Git domain.
func NewDomain(
	repoDB outbound.GitRepoDatabasePort,
	collabDB outbound.GitCollaboratorDatabasePort,
	prDB outbound.GitPullRequestDatabasePort,
	lfsObjDB outbound.GitLFSObjectDatabasePort,
	lfsLockDB outbound.GitLFSLockDatabasePort,
	storage outbound.GitStoragePort,
	lfsStorage outbound.GitLFSStoragePort,
	quotaChecker inbound.GitStorageQuotaChecker,
	cfg *Config,
	logger *zap.Logger,
) *Domain {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	_ = cfg.Validate()

	return &Domain{
		repoDB:       repoDB,
		collabDB:     collabDB,
		prDB:         prDB,
		lfsObjDB:     lfsObjDB,
		lfsLockDB:    lfsLockDB,
		storage:      storage,
		lfsStorage:   lfsStorage,
		quotaChecker: quotaChecker,
		cfg:          cfg,
		logger:       logger,
	}
}

// ===== Repository Operations =====

// CreateRepo creates a new repository.
func (d *Domain) CreateRepo(ctx context.Context, ownerID uuid.UUID, input *inbound.GitCreateRepoInput) (*model.GitRepo, error) {
	// Generate slug from name
	slug := generateSlug(input.Name)
	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidRepoName
	}

	// Check if repo already exists
	_, err := d.repoDB.FindByOwnerAndSlug(ctx, ownerID, slug)
	if err == nil {
		return nil, ErrRepoAlreadyExists
	}

	// Check storage quota
	if d.quotaChecker != nil {
		quota, err := d.quotaChecker.GetStorageQuota(ctx, ownerID)
		if err != nil {
			d.logger.Warn("failed to get storage quota", zap.Error(err))
		} else if quota > 0 {
			used, err := d.quotaChecker.GetStorageUsed(ctx, ownerID)
			if err != nil {
				d.logger.Warn("failed to get storage used", zap.Error(err))
			} else if used >= quota {
				return nil, ErrStorageQuotaExceeded
			}
		}
	}

	// Create repository record
	repoID := uuid.New()
	storagePath := fmt.Sprintf("%s%s/%s/", d.cfg.RepoPrefix, ownerID.String(), repoID.String())

	repoType := model.GitRepoTypeCode
	if input.Type != "" {
		repoType = input.Type
	}

	visibility := model.GitVisibilityPrivate
	if input.Visibility != "" {
		visibility = input.Visibility
	}

	repo := &model.GitRepo{
		ID:            repoID,
		OwnerID:       ownerID,
		Name:          input.Name,
		Slug:          slug,
		RepoType:      repoType,
		Visibility:    visibility,
		Description:   input.Description,
		DefaultBranch: d.cfg.DefaultBranch,
		LFSEnabled:    input.LFSEnabled,
		StoragePath:   storagePath,
	}

	// Initialize bare Git repository in storage
	if err := d.initBareRepo(ctx, repo); err != nil {
		return nil, fmt.Errorf("init bare repo: %w", err)
	}

	// Save to database
	if err := d.repoDB.Create(ctx, repo); err != nil {
		// Cleanup storage on failure
		_ = d.storage.DeleteRepository(ctx, storagePath)
		return nil, fmt.Errorf("create repo: %w", err)
	}

	d.logger.Info("repository created",
		zap.String("repo_id", repoID.String()),
		zap.String("owner_id", ownerID.String()),
		zap.String("name", input.Name),
	)

	return repo, nil
}

// initBareRepo initializes a bare Git repository in storage.
func (d *Domain) initBareRepo(ctx context.Context, repo *model.GitRepo) error {
	fs, err := d.storage.GetFilesystem(ctx, repo.StoragePath)
	if err != nil {
		return fmt.Errorf("get filesystem: %w", err)
	}

	// Create HEAD file
	headFile, err := fs.Create("HEAD")
	if err != nil {
		return fmt.Errorf("create HEAD: %w", err)
	}
	_, err = headFile.Write([]byte("ref: refs/heads/" + repo.DefaultBranch + "\n"))
	if err != nil {
		headFile.Close()
		return fmt.Errorf("write HEAD: %w", err)
	}
	if err := headFile.Close(); err != nil {
		return fmt.Errorf("close HEAD: %w", err)
	}

	// Create config file
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

	// Create directories
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
	if repo.Description != "" {
		desc = repo.Description + "\n"
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

// GetRepo retrieves a repository by ID.
func (d *Domain) GetRepo(ctx context.Context, id uuid.UUID) (*model.GitRepo, error) {
	repo, err := d.repoDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	return repo, nil
}

// GetRepoByOwnerAndSlug retrieves a repository by owner and slug.
func (d *Domain) GetRepoByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error) {
	repo, err := d.repoDB.FindByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	return repo, nil
}

// ListRepos lists repositories for a user.
func (d *Domain) ListRepos(ctx context.Context, ownerID uuid.UUID, filter *inbound.GitRepoFilter) ([]*model.GitRepo, int64, error) {
	dbFilter := convertFilter(filter)
	return d.repoDB.FindByOwner(ctx, ownerID, dbFilter)
}

// ListPublicRepos lists public repositories.
func (d *Domain) ListPublicRepos(ctx context.Context, filter *inbound.GitRepoFilter) ([]*model.GitRepo, int64, error) {
	dbFilter := convertFilter(filter)
	return d.repoDB.FindPublic(ctx, dbFilter)
}

// UpdateRepo updates a repository.
func (d *Domain) UpdateRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID, input *inbound.GitUpdateRepoInput) (*model.GitRepo, error) {
	repo, err := d.repoDB.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Check ownership or admin permission
	if repo.OwnerID != userID {
		if err := d.CheckAccess(ctx, id, userID, model.GitPermissionAdmin); err != nil {
			return nil, ErrNotOwner
		}
	}

	// Update fields
	if input.Name != "" {
		newSlug := generateSlug(input.Name)
		if !slugRegex.MatchString(newSlug) {
			return nil, ErrInvalidRepoName
		}
		if newSlug != repo.Slug {
			_, err := d.repoDB.FindByOwnerAndSlug(ctx, repo.OwnerID, newSlug)
			if err == nil {
				return nil, ErrRepoAlreadyExists
			}
		}
		repo.Name = input.Name
		repo.Slug = newSlug
	}

	if input.Description != "" {
		repo.Description = input.Description
	}

	if input.Visibility != "" {
		repo.Visibility = input.Visibility
	}

	if input.DefaultBranch != "" {
		repo.DefaultBranch = input.DefaultBranch
	}

	if input.LFSEnabled != nil {
		repo.LFSEnabled = *input.LFSEnabled
	}

	if err := d.repoDB.Update(ctx, repo); err != nil {
		return nil, fmt.Errorf("update repo: %w", err)
	}

	return repo, nil
}

// DeleteRepo deletes a repository.
func (d *Domain) DeleteRepo(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	repo, err := d.repoDB.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrRepoNotFound
	}

	// Only owner can delete
	if repo.OwnerID != userID {
		return ErrNotOwner
	}

	// Delete from storage
	if err := d.storage.DeleteRepository(ctx, repo.StoragePath); err != nil {
		d.logger.Error("failed to delete repository storage", zap.Error(err))
	}

	// Delete from database
	if err := d.repoDB.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete repo: %w", err)
	}

	d.logger.Info("repository deleted",
		zap.String("repo_id", id.String()),
		zap.String("owner_id", userID.String()),
	)

	return nil
}

// ===== Access Control =====

// CheckAccess checks if a user has the required permission level.
func (d *Domain) CheckAccess(ctx context.Context, repoID, userID uuid.UUID, required model.GitPermission) error {
	hasAccess, err := d.CanAccess(ctx, repoID, &userID, required)
	if err != nil {
		return err
	}
	if !hasAccess {
		return ErrAccessDenied
	}
	return nil
}

// CanAccess checks if a user can access a repository with the required permission.
func (d *Domain) CanAccess(ctx context.Context, repoID uuid.UUID, userID *uuid.UUID, required model.GitPermission) (bool, error) {
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return false, err
	}
	if repo == nil {
		return false, ErrRepoNotFound
	}

	// Public repos allow read access to everyone
	if repo.IsPublic() && required == model.GitPermissionRead {
		return true, nil
	}

	// Anonymous users can't do anything else
	if userID == nil {
		return false, nil
	}

	// Owner has full access
	if repo.OwnerID == *userID {
		return true, nil
	}

	// Check collaborator permission
	collab, err := d.collabDB.FindByRepoAndUser(ctx, repoID, *userID)
	if err != nil {
		return false, nil
	}
	if collab == nil {
		return false, nil
	}

	return hasPermission(collab.Permission, required), nil
}

// hasPermission checks if the granted permission level satisfies the required level.
func hasPermission(granted, required model.GitPermission) bool {
	levels := map[model.GitPermission]int{
		model.GitPermissionRead:  1,
		model.GitPermissionWrite: 2,
		model.GitPermissionAdmin: 3,
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

// ===== Collaborator Operations =====

// AddCollaborator adds a collaborator to a repository.
func (d *Domain) AddCollaborator(ctx context.Context, repoID, ownerID, targetUserID uuid.UUID, permission model.GitPermission) error {
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrRepoNotFound
	}

	// Only owner can add collaborators
	if repo.OwnerID != ownerID {
		return ErrNotOwner
	}

	// Can't add owner as collaborator
	if targetUserID == ownerID {
		return ErrInvalidPermission
	}

	collab := &model.GitRepoCollaborator{
		RepoID:     repoID,
		UserID:     targetUserID,
		Permission: permission,
		CreatedAt:  time.Now(),
	}

	if err := d.collabDB.Add(ctx, collab); err != nil {
		return fmt.Errorf("add collaborator: %w", err)
	}

	return nil
}

// ListCollaborators lists collaborators of a repository.
func (d *Domain) ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*model.GitRepoCollaborator, error) {
	return d.collabDB.FindByRepo(ctx, repoID)
}

// UpdateCollaborator updates a collaborator's permission.
func (d *Domain) UpdateCollaborator(ctx context.Context, repoID, ownerID, targetUserID uuid.UUID, permission model.GitPermission) error {
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrRepoNotFound
	}

	// Only owner can update collaborators
	if repo.OwnerID != ownerID {
		return ErrNotOwner
	}

	collab := &model.GitRepoCollaborator{
		RepoID:     repoID,
		UserID:     targetUserID,
		Permission: permission,
	}

	if err := d.collabDB.Update(ctx, collab); err != nil {
		return fmt.Errorf("update collaborator: %w", err)
	}

	return nil
}

// RemoveCollaborator removes a collaborator from a repository.
func (d *Domain) RemoveCollaborator(ctx context.Context, repoID, ownerID, targetUserID uuid.UUID) error {
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrRepoNotFound
	}

	// Only owner can remove collaborators
	if repo.OwnerID != ownerID {
		return ErrNotOwner
	}

	if err := d.collabDB.Remove(ctx, repoID, targetUserID); err != nil {
		return fmt.Errorf("remove collaborator: %w", err)
	}

	return nil
}

// ===== Pull Request Operations =====

// CreatePR creates a new pull request.
func (d *Domain) CreatePR(ctx context.Context, repoID, authorID uuid.UUID, input *inbound.GitCreatePRInput) (*model.GitPullRequest, error) {
	// Check access
	if err := d.CheckAccess(ctx, repoID, authorID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	// Validate branches
	if input.SourceBranch == input.TargetBranch {
		return nil, ErrSameBranch
	}

	// Get next PR number
	number, err := d.prDB.GetNextNumber(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("get next PR number: %w", err)
	}

	pr := &model.GitPullRequest{
		ID:           uuid.New(),
		RepoID:       repoID,
		Number:       number,
		Title:        input.Title,
		Description:  input.Description,
		SourceBranch: input.SourceBranch,
		TargetBranch: input.TargetBranch,
		Status:       model.GitPRStatusOpen,
		AuthorID:     authorID,
	}

	if err := d.prDB.Create(ctx, pr); err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	return pr, nil
}

// GetPR retrieves a pull request by number.
func (d *Domain) GetPR(ctx context.Context, repoID uuid.UUID, number int) (*model.GitPullRequest, error) {
	pr, err := d.prDB.FindByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrPRNotFound
	}
	return pr, nil
}

// ListPRs lists pull requests for a repository.
func (d *Domain) ListPRs(ctx context.Context, repoID uuid.UUID, status *model.GitPRStatus, limit, offset int) ([]*model.GitPullRequest, int64, error) {
	return d.prDB.FindByRepo(ctx, repoID, status, limit, offset)
}

// UpdatePR updates a pull request.
func (d *Domain) UpdatePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID, input *inbound.GitUpdatePRInput) (*model.GitPullRequest, error) {
	pr, err := d.prDB.FindByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrPRNotFound
	}

	// Check if user is author or has write access
	if pr.AuthorID != userID {
		if err := d.CheckAccess(ctx, repoID, userID, model.GitPermissionWrite); err != nil {
			return nil, err
		}
	}

	// Can't update merged PR
	if pr.Status.IsMerged() {
		return nil, ErrPRAlreadyMerged
	}

	if input.Title != "" {
		pr.Title = input.Title
	}
	if input.Description != "" {
		pr.Description = input.Description
	}
	if input.Status != "" {
		if input.Status == "closed" && pr.Status.IsOpen() {
			now := time.Now()
			pr.Status = model.GitPRStatusClosed
			pr.ClosedAt = &now
		} else if input.Status == "open" && pr.Status == model.GitPRStatusClosed {
			pr.Status = model.GitPRStatusOpen
			pr.ClosedAt = nil
		}
	}

	if err := d.prDB.Update(ctx, pr); err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}

	return pr, nil
}

// MergePR merges a pull request.
func (d *Domain) MergePR(ctx context.Context, repoID uuid.UUID, number int, userID uuid.UUID) (*model.GitPullRequest, error) {
	// Check write access
	if err := d.CheckAccess(ctx, repoID, userID, model.GitPermissionWrite); err != nil {
		return nil, err
	}

	pr, err := d.prDB.FindByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrPRNotFound
	}

	if pr.Status.IsMerged() {
		return nil, ErrPRAlreadyMerged
	}
	if pr.Status == model.GitPRStatusClosed {
		return nil, ErrPRAlreadyClosed
	}

	// TODO: Perform actual Git merge operation using go-git
	now := time.Now()
	pr.Status = model.GitPRStatusMerged
	pr.MergedBy = &userID
	pr.MergedAt = &now

	if err := d.prDB.Update(ctx, pr); err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}

	return pr, nil
}

// ===== Storage Operations =====

// GetStorageStats returns storage statistics for a repository.
func (d *Domain) GetStorageStats(ctx context.Context, repoID uuid.UUID) (*inbound.GitStorageStats, error) {
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Get LFS object count
	lfsObjects, err := d.lfsObjDB.FindByRepo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list LFS objects: %w", err)
	}

	return &inbound.GitStorageStats{
		RepoSizeBytes:  repo.SizeBytes,
		LFSSizeBytes:   repo.LFSSizeBytes,
		TotalSizeBytes: repo.TotalSize(),
		LFSObjectCount: len(lfsObjects),
	}, nil
}

// GetUserStorageStats returns storage statistics for a user.
func (d *Domain) GetUserStorageStats(ctx context.Context, userID uuid.UUID) (*inbound.GitUserStorageStats, error) {
	totalUsed, err := d.lfsObjDB.GetUserTotalStorage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get total storage: %w", err)
	}

	// Get repo count
	repos, _, err := d.repoDB.FindByOwner(ctx, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	var quota int64 = -1
	if d.quotaChecker != nil {
		q, err := d.quotaChecker.GetStorageQuota(ctx, userID)
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

	return &inbound.GitUserStorageStats{
		TotalUsedBytes: totalUsed,
		QuotaBytes:     quota,
		RemainingBytes: remaining,
		RepoCount:      len(repos),
	}, nil
}

// GetFilesystem returns a billy.Filesystem for a repository.
func (d *Domain) GetFilesystem(ctx context.Context, repoID uuid.UUID) (billy.Filesystem, error) {
	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	return d.storage.GetFilesystem(ctx, repo.StoragePath)
}

// UpdatePushedAt updates the pushed_at timestamp for a repository.
func (d *Domain) UpdatePushedAt(ctx context.Context, repoID uuid.UUID) error {
	now := time.Now()
	return d.repoDB.UpdatePushedAt(ctx, repoID, &now)
}

// UpdateRepoSize updates the size of a repository.
func (d *Domain) UpdateRepoSize(ctx context.Context, repoID uuid.UUID, sizeBytes, lfsSizeBytes int64) error {
	return d.repoDB.UpdateSize(ctx, repoID, sizeBytes, lfsSizeBytes)
}

// ===== Helpers =====

// generateSlug generates a URL-safe slug from a name.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")

	return slug
}

func convertFilter(filter *inbound.GitRepoFilter) *outbound.GitRepoFilter {
	if filter == nil {
		return nil
	}
	return &outbound.GitRepoFilter{
		Type:       filter.Type,
		Visibility: filter.Visibility,
		Search:     filter.Search,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}
}

// Compile-time interface check
var _ inbound.GitDomain = (*Domain)(nil)
