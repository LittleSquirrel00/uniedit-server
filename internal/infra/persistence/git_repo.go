package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/git"
	"github.com/uniedit/server/internal/infra/persistence/entity"
)

// GitRepositoryRepository implements git.RepositoryRepository.
type GitRepositoryRepository struct {
	db *gorm.DB
}

// NewGitRepositoryRepository creates a new repository repository.
func NewGitRepositoryRepository(db *gorm.DB) *GitRepositoryRepository {
	return &GitRepositoryRepository{db: db}
}

// Create creates a new repository.
func (r *GitRepositoryRepository) Create(ctx context.Context, repo *git.Repository) error {
	e := repoToEntity(repo)
	return r.db.WithContext(ctx).Create(e).Error
}

// GetByID retrieves a repository by ID.
func (r *GitRepositoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*git.Repository, error) {
	var e entity.GitRepoEntity
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrRepoNotFound
		}
		return nil, err
	}
	return entityToRepo(&e), nil
}

// GetByOwnerAndSlug retrieves a repository by owner and slug.
func (r *GitRepositoryRepository) GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*git.Repository, error) {
	var e entity.GitRepoEntity
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? AND slug = ? AND deleted_at IS NULL", ownerID, slug).
		First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrRepoNotFound
		}
		return nil, err
	}
	return entityToRepo(&e), nil
}

// List lists repositories for a user.
func (r *GitRepositoryRepository) List(ctx context.Context, ownerID uuid.UUID, filter *git.RepoFilter) ([]*git.Repository, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.GitRepoEntity{}).Where("owner_id = ? AND deleted_at IS NULL", ownerID)

	if filter != nil {
		if filter.RepoType != nil {
			query = query.Where("repo_type = ?", string(*filter.RepoType))
		}
		if filter.Visibility != nil {
			query = query.Where("visibility = ?", string(*filter.Visibility))
		}
		if filter.Search != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := 20
	offset := 0
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
		}
		offset = filter.Offset
	}

	var entities []entity.GitRepoEntity
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	repos := make([]*git.Repository, len(entities))
	for i, e := range entities {
		repos[i] = entityToRepo(&e)
	}

	return repos, total, nil
}

// ListPublic lists public repositories.
func (r *GitRepositoryRepository) ListPublic(ctx context.Context, filter *git.RepoFilter) ([]*git.Repository, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.GitRepoEntity{}).Where("visibility = ? AND deleted_at IS NULL", "public")

	if filter != nil {
		if filter.RepoType != nil {
			query = query.Where("repo_type = ?", string(*filter.RepoType))
		}
		if filter.Search != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := 20
	offset := 0
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
		}
		offset = filter.Offset
	}

	var entities []entity.GitRepoEntity
	if err := query.Order("stars_count DESC, created_at DESC").Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	repos := make([]*git.Repository, len(entities))
	for i, e := range entities {
		repos[i] = entityToRepo(&e)
	}

	return repos, total, nil
}

// Update updates a repository.
func (r *GitRepositoryRepository) Update(ctx context.Context, repo *git.Repository) error {
	e := repoToEntity(repo)
	return r.db.WithContext(ctx).Model(e).Updates(e).Error
}

// Delete soft-deletes a repository.
func (r *GitRepositoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.GitRepoEntity{}).Where("id = ?", id).Update("deleted_at", now).Error
}

// UpdatePushedAt updates the pushed_at timestamp.
func (r *GitRepositoryRepository) UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.GitRepoEntity{}).Where("id = ?", id).Update("pushed_at", pushedAt).Error
}

// UpdateSize updates the size of a repository.
func (r *GitRepositoryRepository) UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error {
	return r.db.WithContext(ctx).Model(&entity.GitRepoEntity{}).Where("id = ?", id).Updates(map[string]interface{}{
		"size_bytes":     sizeBytes,
		"lfs_size_bytes": lfsSizeBytes,
	}).Error
}

// GetUserTotalStorage returns total storage used by a user.
func (r *GitRepositoryRepository) GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error) {
	var result struct {
		Total int64
	}
	err := r.db.WithContext(ctx).Model(&entity.GitRepoEntity{}).
		Select("COALESCE(SUM(size_bytes + lfs_size_bytes), 0) as total").
		Where("owner_id = ? AND deleted_at IS NULL", userID).
		Scan(&result).Error
	return result.Total, err
}

// Helper functions
func repoToEntity(r *git.Repository) *entity.GitRepoEntity {
	return &entity.GitRepoEntity{
		ID:            r.ID(),
		OwnerID:       r.OwnerID(),
		Name:          r.Name(),
		Slug:          r.Slug(),
		RepoType:      string(r.RepoType()),
		Visibility:    string(r.Visibility()),
		Description:   r.Description(),
		DefaultBranch: r.DefaultBranch(),
		SizeBytes:     r.SizeBytes(),
		LFSEnabled:    r.LFSEnabled(),
		LFSSizeBytes:  r.LFSSizeBytes(),
		StoragePath:   r.StoragePath(),
		StarsCount:    r.StarsCount(),
		ForksCount:    r.ForksCount(),
		ForkedFrom:    r.ForkedFrom(),
		CreatedAt:     r.CreatedAt(),
		UpdatedAt:     r.UpdatedAt(),
		PushedAt:      r.PushedAt(),
	}
}

func entityToRepo(e *entity.GitRepoEntity) *git.Repository {
	return git.ReconstructRepository(
		e.ID,
		e.OwnerID,
		e.Name,
		e.Slug,
		git.RepoType(e.RepoType),
		git.Visibility(e.Visibility),
		e.Description,
		e.DefaultBranch,
		e.SizeBytes,
		e.LFSEnabled,
		e.LFSSizeBytes,
		e.StoragePath,
		e.StarsCount,
		e.ForksCount,
		e.ForkedFrom,
		e.CreatedAt,
		e.UpdatedAt,
		e.PushedAt,
	)
}

// GitCollaboratorRepository implements git.CollaboratorRepository.
type GitCollaboratorRepository struct {
	db *gorm.DB
}

// NewGitCollaboratorRepository creates a new collaborator repository.
func NewGitCollaboratorRepository(db *gorm.DB) *GitCollaboratorRepository {
	return &GitCollaboratorRepository{db: db}
}

// Add adds a collaborator.
func (r *GitCollaboratorRepository) Add(ctx context.Context, collab *git.Collaborator) error {
	e := collabToEntity(collab)
	return r.db.WithContext(ctx).Create(e).Error
}

// Get retrieves a collaborator.
func (r *GitCollaboratorRepository) Get(ctx context.Context, repoID, userID uuid.UUID) (*git.Collaborator, error) {
	var e entity.GitCollaboratorEntity
	if err := r.db.WithContext(ctx).
		Where("repo_id = ? AND user_id = ?", repoID, userID).
		First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrNotCollaborator
		}
		return nil, err
	}
	return entityToCollab(&e), nil
}

// List lists collaborators for a repository.
func (r *GitCollaboratorRepository) List(ctx context.Context, repoID uuid.UUID) ([]*git.Collaborator, error) {
	var entities []entity.GitCollaboratorEntity
	if err := r.db.WithContext(ctx).Where("repo_id = ?", repoID).Find(&entities).Error; err != nil {
		return nil, err
	}

	collabs := make([]*git.Collaborator, len(entities))
	for i, e := range entities {
		collabs[i] = entityToCollab(&e)
	}

	return collabs, nil
}

// Update updates a collaborator's permission.
func (r *GitCollaboratorRepository) Update(ctx context.Context, collab *git.Collaborator) error {
	return r.db.WithContext(ctx).Model(&entity.GitCollaboratorEntity{}).
		Where("repo_id = ? AND user_id = ?", collab.RepoID(), collab.UserID()).
		Update("permission", string(collab.Permission())).Error
}

// Remove removes a collaborator.
func (r *GitCollaboratorRepository) Remove(ctx context.Context, repoID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("repo_id = ? AND user_id = ?", repoID, userID).
		Delete(&entity.GitCollaboratorEntity{}).Error
}

func collabToEntity(c *git.Collaborator) *entity.GitCollaboratorEntity {
	return &entity.GitCollaboratorEntity{
		RepoID:     c.RepoID(),
		UserID:     c.UserID(),
		Permission: string(c.Permission()),
		CreatedAt:  c.CreatedAt(),
	}
}

func entityToCollab(e *entity.GitCollaboratorEntity) *git.Collaborator {
	return git.ReconstructCollaborator(
		e.RepoID,
		e.UserID,
		git.Permission(e.Permission),
		e.CreatedAt,
	)
}

// GitPullRequestRepository implements git.PullRequestRepository.
type GitPullRequestRepository struct {
	db *gorm.DB
}

// NewGitPullRequestRepository creates a new pull request repository.
func NewGitPullRequestRepository(db *gorm.DB) *GitPullRequestRepository {
	return &GitPullRequestRepository{db: db}
}

// Create creates a new pull request.
func (r *GitPullRequestRepository) Create(ctx context.Context, pr *git.PullRequest) error {
	e := prToEntity(pr)
	return r.db.WithContext(ctx).Create(e).Error
}

// GetByID retrieves a pull request by ID.
func (r *GitPullRequestRepository) GetByID(ctx context.Context, id uuid.UUID) (*git.PullRequest, error) {
	var e entity.PullRequestEntity
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrPRNotFound
		}
		return nil, err
	}
	return entityToPR(&e), nil
}

// GetByNumber retrieves a pull request by number.
func (r *GitPullRequestRepository) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*git.PullRequest, error) {
	var e entity.PullRequestEntity
	if err := r.db.WithContext(ctx).
		Where("repo_id = ? AND number = ?", repoID, number).
		First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrPRNotFound
		}
		return nil, err
	}
	return entityToPR(&e), nil
}

// List lists pull requests for a repository.
func (r *GitPullRequestRepository) List(ctx context.Context, repoID uuid.UUID, status *git.PRStatus, limit, offset int) ([]*git.PullRequest, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.PullRequestEntity{}).Where("repo_id = ?", repoID)

	if status != nil {
		query = query.Where("status = ?", string(*status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	var entities []entity.PullRequestEntity
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	prs := make([]*git.PullRequest, len(entities))
	for i, e := range entities {
		prs[i] = entityToPR(&e)
	}

	return prs, total, nil
}

// Update updates a pull request.
func (r *GitPullRequestRepository) Update(ctx context.Context, pr *git.PullRequest) error {
	e := prToEntity(pr)
	return r.db.WithContext(ctx).Model(e).Updates(e).Error
}

// GetNextNumber gets the next PR number for a repository.
func (r *GitPullRequestRepository) GetNextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	var maxNumber int
	err := r.db.WithContext(ctx).Model(&entity.PullRequestEntity{}).
		Select("COALESCE(MAX(number), 0)").
		Where("repo_id = ?", repoID).
		Scan(&maxNumber).Error
	return maxNumber + 1, err
}

func prToEntity(p *git.PullRequest) *entity.PullRequestEntity {
	return &entity.PullRequestEntity{
		ID:           p.ID(),
		RepoID:       p.RepoID(),
		Number:       p.Number(),
		Title:        p.Title(),
		Description:  p.Description(),
		SourceBranch: p.SourceBranch(),
		TargetBranch: p.TargetBranch(),
		Status:       string(p.Status()),
		AuthorID:     p.AuthorID(),
		MergedBy:     p.MergedBy(),
		MergedAt:     p.MergedAt(),
		ClosedAt:     p.ClosedAt(),
		CreatedAt:    p.CreatedAt(),
		UpdatedAt:    p.UpdatedAt(),
	}
}

func entityToPR(e *entity.PullRequestEntity) *git.PullRequest {
	return git.ReconstructPullRequest(
		e.ID,
		e.RepoID,
		e.Number,
		e.Title,
		e.Description,
		e.SourceBranch,
		e.TargetBranch,
		git.PRStatus(e.Status),
		e.AuthorID,
		e.MergedBy,
		e.MergedAt,
		e.ClosedAt,
		e.CreatedAt,
		e.UpdatedAt,
	)
}

// GitLFSRepository implements git.LFSRepository.
type GitLFSRepository struct {
	db *gorm.DB
}

// NewGitLFSRepository creates a new LFS repository.
func NewGitLFSRepository(db *gorm.DB) *GitLFSRepository {
	return &GitLFSRepository{db: db}
}

// CreateObject creates a new LFS object.
func (r *GitLFSRepository) CreateObject(ctx context.Context, obj *git.LFSObject) error {
	e := &entity.LFSObjectEntity{
		OID:         obj.OID(),
		Size:        obj.Size(),
		StorageKey:  obj.StorageKey(),
		ContentType: obj.ContentType(),
		CreatedAt:   obj.CreatedAt(),
	}
	return r.db.WithContext(ctx).Create(e).Error
}

// GetObject retrieves an LFS object by OID.
func (r *GitLFSRepository) GetObject(ctx context.Context, oid string) (*git.LFSObject, error) {
	var e entity.LFSObjectEntity
	if err := r.db.WithContext(ctx).Where("oid = ?", oid).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrLFSObjectNotFound
		}
		return nil, err
	}
	return git.ReconstructLFSObject(e.OID, e.Size, e.StorageKey, e.ContentType, e.CreatedAt), nil
}

// LinkObjectToRepo links an LFS object to a repository.
func (r *GitLFSRepository) LinkObjectToRepo(ctx context.Context, link *git.LFSRepoObject) error {
	e := &entity.LFSRepoObjectEntity{
		RepoID:    link.RepoID(),
		OID:       link.OID(),
		CreatedAt: link.CreatedAt(),
	}
	return r.db.WithContext(ctx).Create(e).Error
}

// ListRepoObjects lists LFS objects for a repository.
func (r *GitLFSRepository) ListRepoObjects(ctx context.Context, repoID uuid.UUID) ([]*git.LFSRepoObject, error) {
	var entities []entity.LFSRepoObjectEntity
	if err := r.db.WithContext(ctx).Where("repo_id = ?", repoID).Find(&entities).Error; err != nil {
		return nil, err
	}

	objs := make([]*git.LFSRepoObject, len(entities))
	for i, e := range entities {
		objs[i] = git.ReconstructLFSRepoObject(e.RepoID, e.OID, e.CreatedAt)
	}

	return objs, nil
}

// CreateLock creates a new LFS lock.
func (r *GitLFSRepository) CreateLock(ctx context.Context, lock *git.LFSLock) error {
	e := &entity.LFSLockEntity{
		ID:       lock.ID(),
		RepoID:   lock.RepoID(),
		Path:     lock.Path(),
		OwnerID:  lock.OwnerID(),
		LockedAt: lock.LockedAt(),
	}
	return r.db.WithContext(ctx).Create(e).Error
}

// GetLock retrieves an LFS lock.
func (r *GitLFSRepository) GetLock(ctx context.Context, id uuid.UUID) (*git.LFSLock, error) {
	var e entity.LFSLockEntity
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrLockNotFound
		}
		return nil, err
	}
	return git.ReconstructLFSLock(e.ID, e.RepoID, e.Path, e.OwnerID, e.LockedAt), nil
}

// GetLockByPath retrieves an LFS lock by path.
func (r *GitLFSRepository) GetLockByPath(ctx context.Context, repoID uuid.UUID, path string) (*git.LFSLock, error) {
	var e entity.LFSLockEntity
	if err := r.db.WithContext(ctx).
		Where("repo_id = ? AND path = ?", repoID, path).
		First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, git.ErrLockNotFound
		}
		return nil, err
	}
	return git.ReconstructLFSLock(e.ID, e.RepoID, e.Path, e.OwnerID, e.LockedAt), nil
}

// ListLocks lists LFS locks for a repository.
func (r *GitLFSRepository) ListLocks(ctx context.Context, repoID uuid.UUID, path string, limit, offset int) ([]*git.LFSLock, error) {
	query := r.db.WithContext(ctx).Model(&entity.LFSLockEntity{}).Where("repo_id = ?", repoID)

	if path != "" {
		query = query.Where("path LIKE ?", path+"%")
	}

	if limit <= 0 {
		limit = 100
	}

	var entities []entity.LFSLockEntity
	if err := query.Limit(limit).Offset(offset).Find(&entities).Error; err != nil {
		return nil, err
	}

	locks := make([]*git.LFSLock, len(entities))
	for i, e := range entities {
		locks[i] = git.ReconstructLFSLock(e.ID, e.RepoID, e.Path, e.OwnerID, e.LockedAt)
	}

	return locks, nil
}

// DeleteLock deletes an LFS lock.
func (r *GitLFSRepository) DeleteLock(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.LFSLockEntity{}).Error
}
