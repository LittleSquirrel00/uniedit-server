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

	commonv1 "github.com/uniedit/server/api/pb/common"
	gitv1 "github.com/uniedit/server/api/pb/git"
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
func (d *Domain) CreateRepo(ctx context.Context, ownerID uuid.UUID, in *gitv1.CreateRepoRequest) (*gitv1.Repo, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}
	// Generate slug from name
	slug := generateSlug(in.GetName())
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
	if t := in.GetType(); t != "" {
		repoType = model.GitRepoType(t)
	}

	visibility := model.GitVisibilityPrivate
	if v := in.GetVisibility(); v != "" {
		visibility = model.GitVisibility(v)
	}

	repo := &model.GitRepo{
		ID:            repoID,
		OwnerID:       ownerID,
		Name:          in.GetName(),
		Slug:          slug,
		RepoType:      repoType,
		Visibility:    visibility,
		Description:   in.GetDescription(),
		DefaultBranch: d.cfg.DefaultBranch,
		LFSEnabled:    in.GetLfsEnabled(),
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
		zap.String("name", in.GetName()),
	)

	return toRepoPB(repo, d.cfg.BaseURL), nil
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

// ===== Inbound.GitDomain (pb pass-through) =====

func (d *Domain) ListRepos(ctx context.Context, ownerID uuid.UUID, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	filter, page, pageSize := repoFilterFromPB(in)
	repos, total, err := d.repoDB.FindByOwner(ctx, ownerID, filter)
	if err != nil {
		return nil, err
	}

	out := make([]*gitv1.Repo, 0, len(repos))
	for _, r := range repos {
		out = append(out, toRepoPB(r, d.cfg.BaseURL))
	}

	return &gitv1.ListReposResponse{
		Repos:      out,
		TotalCount: total,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}, nil
}

func (d *Domain) ListPublicRepos(ctx context.Context, in *gitv1.ListReposRequest) (*gitv1.ListReposResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	filter, page, pageSize := repoFilterFromPB(in)
	repos, total, err := d.repoDB.FindPublic(ctx, filter)
	if err != nil {
		return nil, err
	}

	out := make([]*gitv1.Repo, 0, len(repos))
	for _, r := range repos {
		out = append(out, toRepoPB(r, d.cfg.BaseURL))
	}

	return &gitv1.ListReposResponse{
		Repos:      out,
		TotalCount: total,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}, nil
}

func (d *Domain) GetRepo(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*gitv1.Repo, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	if err := d.checkAccessToRepo(ctx, repo, userID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	return toRepoPB(repo, d.cfg.BaseURL), nil
}

func (d *Domain) UpdateRepo(ctx context.Context, userID uuid.UUID, in *gitv1.UpdateRepoRequest) (*gitv1.Repo, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Check ownership or admin permission
	if repo.OwnerID != userID {
		if err := d.checkAccessToRepo(ctx, repo, userID, model.GitPermissionAdmin); err != nil {
			if err == ErrAccessDenied {
				return nil, ErrNotOwner
			}
			return nil, err
		}
	}

	// Update fields (keep behavior compatible with previous handler: empty string means "ignore")
	if v := in.GetName(); v != nil && v.GetValue() != "" {
		newSlug := generateSlug(v.GetValue())
		if !slugRegex.MatchString(newSlug) {
			return nil, ErrInvalidRepoName
		}
		if newSlug != repo.Slug {
			_, err := d.repoDB.FindByOwnerAndSlug(ctx, repo.OwnerID, newSlug)
			if err == nil {
				return nil, ErrRepoAlreadyExists
			}
		}
		repo.Name = v.GetValue()
		repo.Slug = newSlug
	}

	if v := in.GetDescription(); v != nil && v.GetValue() != "" {
		repo.Description = v.GetValue()
	}
	if v := in.GetVisibility(); v != nil && v.GetValue() != "" {
		repo.Visibility = model.GitVisibility(v.GetValue())
	}
	if v := in.GetDefaultBranch(); v != nil && v.GetValue() != "" {
		repo.DefaultBranch = v.GetValue()
	}
	if v := in.GetLfsEnabled(); v != nil {
		repo.LFSEnabled = v.GetValue()
	}

	if err := d.repoDB.Update(ctx, repo); err != nil {
		return nil, fmt.Errorf("update repo: %w", err)
	}

	return toRepoPB(repo, d.cfg.BaseURL), nil
}

func (d *Domain) DeleteRepo(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*commonv1.Empty, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Only owner can delete
	if repo.OwnerID != userID {
		return nil, ErrNotOwner
	}

	// Delete from storage
	if err := d.storage.DeleteRepository(ctx, repo.StoragePath); err != nil {
		d.logger.Error("failed to delete repository storage", zap.Error(err))
	}

	// Delete from database
	if err := d.repoDB.Delete(ctx, repoID); err != nil {
		return nil, fmt.Errorf("delete repo: %w", err)
	}

	d.logger.Info("repository deleted",
		zap.String("repo_id", repoID.String()),
		zap.String("owner_id", userID.String()),
	)

	return empty(), nil
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

func (d *Domain) checkAccessToRepo(ctx context.Context, repo *model.GitRepo, userID uuid.UUID, required model.GitPermission) error {
	if repo == nil {
		return ErrRepoNotFound
	}

	// Public repos allow read access to everyone (protected endpoints still require auth).
	if repo.IsPublic() && required == model.GitPermissionRead {
		return nil
	}

	// Owner has full access.
	if repo.OwnerID == userID {
		return nil
	}

	collab, err := d.collabDB.FindByRepoAndUser(ctx, repo.ID, userID)
	if err != nil {
		return err
	}
	if collab == nil {
		return ErrAccessDenied
	}
	if !hasPermission(collab.Permission, required) {
		return ErrAccessDenied
	}

	return nil
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

func (d *Domain) AddCollaborator(ctx context.Context, userID uuid.UUID, in *gitv1.AddCollaboratorRequest) (*commonv1.Empty, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}
	targetUserID, err := parseUUID(in.GetUserId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	permission := model.GitPermission(in.GetPermission())
	if !isValidGitPermission(permission) {
		return nil, ErrInvalidPermission
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Only owner can add collaborators
	if repo.OwnerID != userID {
		return nil, ErrNotOwner
	}

	// Can't add owner as collaborator
	if targetUserID == userID {
		return nil, ErrInvalidPermission
	}

	collab := &model.GitRepoCollaborator{
		RepoID:     repoID,
		UserID:     targetUserID,
		Permission: permission,
		CreatedAt:  time.Now(),
	}

	if err := d.collabDB.Add(ctx, collab); err != nil {
		return nil, fmt.Errorf("add collaborator: %w", err)
	}

	return empty(), nil
}

func (d *Domain) ListCollaborators(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*gitv1.ListCollaboratorsResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	if err := d.checkAccessToRepo(ctx, repo, userID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	items, err := d.collabDB.FindByRepo(ctx, repoID)
	if err != nil {
		return nil, err
	}

	out := make([]*gitv1.Collaborator, 0, len(items))
	for _, it := range items {
		out = append(out, toCollaboratorPB(it))
	}
	return &gitv1.ListCollaboratorsResponse{Collaborators: out}, nil
}

func (d *Domain) UpdateCollaborator(ctx context.Context, userID uuid.UUID, in *gitv1.UpdateCollaboratorRequest) (*commonv1.Empty, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}
	targetUserID, err := parseUUID(in.GetUserId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	permission := model.GitPermission(in.GetPermission())
	if !isValidGitPermission(permission) {
		return nil, ErrInvalidPermission
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Only owner can update collaborators
	if repo.OwnerID != userID {
		return nil, ErrNotOwner
	}

	collab := &model.GitRepoCollaborator{
		RepoID:     repoID,
		UserID:     targetUserID,
		Permission: permission,
	}

	if err := d.collabDB.Update(ctx, collab); err != nil {
		return nil, fmt.Errorf("update collaborator: %w", err)
	}

	return empty(), nil
}

func (d *Domain) RemoveCollaborator(ctx context.Context, userID uuid.UUID, in *gitv1.RemoveCollaboratorRequest) (*commonv1.Empty, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}
	targetUserID, err := parseUUID(in.GetUserId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Only owner can remove collaborators
	if repo.OwnerID != userID {
		return nil, ErrNotOwner
	}

	if err := d.collabDB.Remove(ctx, repoID, targetUserID); err != nil {
		return nil, fmt.Errorf("remove collaborator: %w", err)
	}

	return empty(), nil
}

// ===== Pull Request Operations =====

func (d *Domain) CreatePR(ctx context.Context, userID uuid.UUID, in *gitv1.CreatePRRequest) (*gitv1.PullRequest, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	// Check access
	if err := d.CheckAccess(ctx, repoID, userID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	// Validate branches
	if in.GetSourceBranch() == in.GetTargetBranch() {
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
		Title:        in.GetTitle(),
		Description:  in.GetDescription(),
		SourceBranch: in.GetSourceBranch(),
		TargetBranch: in.GetTargetBranch(),
		Status:       model.GitPRStatusOpen,
		AuthorID:     userID,
	}

	if err := d.prDB.Create(ctx, pr); err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	return toPullRequestPB(pr), nil
}

func (d *Domain) GetPR(ctx context.Context, userID uuid.UUID, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	if err := d.CheckAccess(ctx, repoID, userID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	number := int(in.GetNumber())
	pr, err := d.prDB.FindByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrPRNotFound
	}
	return toPullRequestPB(pr), nil
}

func (d *Domain) ListPRs(ctx context.Context, userID uuid.UUID, in *gitv1.ListPRsRequest) (*gitv1.ListPRsResponse, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	if err := d.CheckAccess(ctx, repoID, userID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	var status *model.GitPRStatus
	if s := in.GetStatus(); s != "" {
		v := model.GitPRStatus(s)
		status = &v
	}

	limit := int(in.GetLimit())
	offset := int(in.GetOffset())
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	prs, total, err := d.prDB.FindByRepo(ctx, repoID, status, limit, offset)
	if err != nil {
		return nil, err
	}

	out := make([]*gitv1.PullRequest, 0, len(prs))
	for _, pr := range prs {
		out = append(out, toPullRequestPB(pr))
	}
	return &gitv1.ListPRsResponse{PullRequests: out, TotalCount: total}, nil
}

func (d *Domain) UpdatePR(ctx context.Context, userID uuid.UUID, in *gitv1.UpdatePRRequest) (*gitv1.PullRequest, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	number := int(in.GetNumber())

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

	if v := in.GetTitle(); v != nil && v.GetValue() != "" {
		pr.Title = v.GetValue()
	}
	if v := in.GetDescription(); v != nil && v.GetValue() != "" {
		pr.Description = v.GetValue()
	}
	if v := in.GetStatus(); v != nil && v.GetValue() != "" {
		if v.GetValue() == "closed" && pr.Status.IsOpen() {
			now := time.Now()
			pr.Status = model.GitPRStatusClosed
			pr.ClosedAt = &now
		} else if v.GetValue() == "open" && pr.Status == model.GitPRStatusClosed {
			pr.Status = model.GitPRStatusOpen
			pr.ClosedAt = nil
		}
	}

	if err := d.prDB.Update(ctx, pr); err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}

	return toPullRequestPB(pr), nil
}

func (d *Domain) MergePR(ctx context.Context, userID uuid.UUID, in *gitv1.GetPRRequest) (*gitv1.PullRequest, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	number := int(in.GetNumber())

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

	return toPullRequestPB(pr), nil
}

// ===== Storage Operations =====

func (d *Domain) GetStorageStats(ctx context.Context, userID uuid.UUID, in *gitv1.GetByIDRequest) (*gitv1.StorageStats, error) {
	if in == nil {
		return nil, ErrInvalidRequest
	}

	repoID, err := parseUUID(in.GetId())
	if err != nil {
		return nil, ErrInvalidRequest
	}

	repo, err := d.repoDB.FindByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	if err := d.checkAccessToRepo(ctx, repo, userID, model.GitPermissionRead); err != nil {
		return nil, err
	}

	// Get LFS object count
	lfsObjects, err := d.lfsObjDB.FindByRepo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list LFS objects: %w", err)
	}

	return &gitv1.StorageStats{
		RepoSizeBytes:  repo.SizeBytes,
		LfsSizeBytes:   repo.LFSSizeBytes,
		TotalSizeBytes: repo.TotalSize(),
		LfsObjectCount: int32(len(lfsObjects)),
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

func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, fmt.Errorf("empty uuid")
	}
	return uuid.Parse(s)
}

func isValidGitPermission(p model.GitPermission) bool {
	switch p {
	case model.GitPermissionRead, model.GitPermissionWrite, model.GitPermissionAdmin:
		return true
	default:
		return false
	}
}

func repoFilterFromPB(in *gitv1.ListReposRequest) (*outbound.GitRepoFilter, int, int) {
	page := int(in.GetPage())
	pageSize := int(in.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := &outbound.GitRepoFilter{
		Search:   in.GetSearch(),
		Page:     page,
		PageSize: pageSize,
	}

	if t := in.GetType(); t != "" {
		rt := model.GitRepoType(t)
		filter.Type = &rt
	}
	if v := in.GetVisibility(); v != "" {
		vis := model.GitVisibility(v)
		filter.Visibility = &vis
	}

	return filter, page, pageSize
}

// Compile-time interface check
var _ inbound.GitDomain = (*Domain)(nil)
