package git

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines the interface for git data access.
type Repository interface {
	// Repo operations
	Create(ctx context.Context, repo *GitRepo) error
	GetByID(ctx context.Context, id uuid.UUID) (*GitRepo, error)
	GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*GitRepo, error)
	List(ctx context.Context, ownerID uuid.UUID, filter *RepoFilter) ([]*GitRepo, int64, error)
	ListPublic(ctx context.Context, filter *RepoFilter) ([]*GitRepo, int64, error)
	Update(ctx context.Context, repo *GitRepo) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error
	UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error
	IncrementStars(ctx context.Context, id uuid.UUID, delta int) error
	IncrementForks(ctx context.Context, id uuid.UUID, delta int) error

	// Collaborator operations
	AddCollaborator(ctx context.Context, collab *RepoCollaborator) error
	GetCollaborator(ctx context.Context, repoID, userID uuid.UUID) (*RepoCollaborator, error)
	ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*RepoCollaborator, error)
	UpdateCollaborator(ctx context.Context, collab *RepoCollaborator) error
	RemoveCollaborator(ctx context.Context, repoID, userID uuid.UUID) error

	// LFS operations
	CreateLFSObject(ctx context.Context, obj *LFSObject) error
	GetLFSObject(ctx context.Context, oid string) (*LFSObject, error)
	LinkLFSObject(ctx context.Context, repoID uuid.UUID, oid string) error
	UnlinkLFSObject(ctx context.Context, repoID uuid.UUID, oid string) error
	GetRepoLFSSize(ctx context.Context, repoID uuid.UUID) (int64, error)
	GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error)
	ListRepoLFSObjects(ctx context.Context, repoID uuid.UUID) ([]*LFSObject, error)

	// Lock operations
	CreateLock(ctx context.Context, lock *LFSLock) error
	GetLock(ctx context.Context, id uuid.UUID) (*LFSLock, error)
	GetLockByPath(ctx context.Context, repoID uuid.UUID, path string) (*LFSLock, error)
	ListLocks(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*LFSLock, error)
	DeleteLock(ctx context.Context, id uuid.UUID) error

	// Pull Request operations
	CreatePR(ctx context.Context, pr *PullRequest) error
	GetPR(ctx context.Context, id uuid.UUID) (*PullRequest, error)
	GetPRByNumber(ctx context.Context, repoID uuid.UUID, number int) (*PullRequest, error)
	ListPRs(ctx context.Context, repoID uuid.UUID, status *PRStatus, limit, offset int) ([]*PullRequest, int64, error)
	UpdatePR(ctx context.Context, pr *PullRequest) error
	GetNextPRNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new git repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Repo Operations ---

func (r *repository) Create(ctx context.Context, repo *GitRepo) error {
	return r.db.WithContext(ctx).Create(repo).Error
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*GitRepo, error) {
	var repo GitRepo
	err := r.db.WithContext(ctx).First(&repo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return &repo, nil
}

func (r *repository) GetByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*GitRepo, error) {
	var repo GitRepo
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND slug = ?", ownerID, slug).
		First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRepoNotFound
		}
		return nil, err
	}
	return &repo, nil
}

func (r *repository) List(ctx context.Context, ownerID uuid.UUID, filter *RepoFilter) ([]*GitRepo, int64, error) {
	query := r.db.WithContext(ctx).Model(&GitRepo{}).Where("owner_id = ?", ownerID)

	if filter != nil {
		if filter.Type != nil {
			query = query.Where("repo_type = ?", *filter.Type)
		}
		if filter.Visibility != nil {
			query = query.Where("visibility = ?", *filter.Visibility)
		}
		if filter.Search != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?",
				"%"+filter.Search+"%", "%"+filter.Search+"%")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter != nil {
		if filter.PageSize > 0 {
			query = query.Limit(filter.PageSize)
		}
		if filter.Page > 0 && filter.PageSize > 0 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	var repos []*GitRepo
	err := query.Order("created_at DESC").Find(&repos).Error
	return repos, total, err
}

func (r *repository) ListPublic(ctx context.Context, filter *RepoFilter) ([]*GitRepo, int64, error) {
	query := r.db.WithContext(ctx).Model(&GitRepo{}).Where("visibility = ?", VisibilityPublic)

	if filter != nil {
		if filter.Type != nil {
			query = query.Where("repo_type = ?", *filter.Type)
		}
		if filter.Search != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?",
				"%"+filter.Search+"%", "%"+filter.Search+"%")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter != nil {
		if filter.PageSize > 0 {
			query = query.Limit(filter.PageSize)
		}
		if filter.Page > 0 && filter.PageSize > 0 {
			query = query.Offset((filter.Page - 1) * filter.PageSize)
		}
	}

	var repos []*GitRepo
	err := query.Order("stars_count DESC, created_at DESC").Find(&repos).Error
	return repos, total, err
}

func (r *repository) Update(ctx context.Context, repo *GitRepo) error {
	return r.db.WithContext(ctx).Save(repo).Error
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&GitRepo{}, "id = ?", id).Error
}

func (r *repository) UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&GitRepo{}).
		Where("id = ?", id).
		Update("pushed_at", pushedAt).Error
}

func (r *repository) UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error {
	return r.db.WithContext(ctx).
		Model(&GitRepo{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"size_bytes":     sizeBytes,
			"lfs_size_bytes": lfsSizeBytes,
		}).Error
}

func (r *repository) IncrementStars(ctx context.Context, id uuid.UUID, delta int) error {
	return r.db.WithContext(ctx).
		Model(&GitRepo{}).
		Where("id = ?", id).
		Update("stars_count", gorm.Expr("stars_count + ?", delta)).Error
}

func (r *repository) IncrementForks(ctx context.Context, id uuid.UUID, delta int) error {
	return r.db.WithContext(ctx).
		Model(&GitRepo{}).
		Where("id = ?", id).
		Update("forks_count", gorm.Expr("forks_count + ?", delta)).Error
}

// --- Collaborator Operations ---

func (r *repository) AddCollaborator(ctx context.Context, collab *RepoCollaborator) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repo_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"permission"}),
		}).
		Create(collab).Error
}

func (r *repository) GetCollaborator(ctx context.Context, repoID, userID uuid.UUID) (*RepoCollaborator, error) {
	var collab RepoCollaborator
	err := r.db.WithContext(ctx).
		Where("repo_id = ? AND user_id = ?", repoID, userID).
		First(&collab).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotCollaborator
		}
		return nil, err
	}
	return &collab, nil
}

func (r *repository) ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*RepoCollaborator, error) {
	var collabs []*RepoCollaborator
	err := r.db.WithContext(ctx).
		Where("repo_id = ?", repoID).
		Order("created_at ASC").
		Find(&collabs).Error
	return collabs, err
}

func (r *repository) UpdateCollaborator(ctx context.Context, collab *RepoCollaborator) error {
	return r.db.WithContext(ctx).
		Model(&RepoCollaborator{}).
		Where("repo_id = ? AND user_id = ?", collab.RepoID, collab.UserID).
		Update("permission", collab.Permission).Error
}

func (r *repository) RemoveCollaborator(ctx context.Context, repoID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Delete(&RepoCollaborator{}, "repo_id = ? AND user_id = ?", repoID, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotCollaborator
	}
	return nil
}

// --- LFS Operations ---

func (r *repository) CreateLFSObject(ctx context.Context, obj *LFSObject) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(obj).Error
}

func (r *repository) GetLFSObject(ctx context.Context, oid string) (*LFSObject, error) {
	var obj LFSObject
	err := r.db.WithContext(ctx).First(&obj, "oid = ?", oid).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLFSObjectNotFound
		}
		return nil, err
	}
	return &obj, nil
}

func (r *repository) LinkLFSObject(ctx context.Context, repoID uuid.UUID, oid string) error {
	link := LFSRepoObject{
		RepoID:    repoID,
		OID:       oid,
		CreatedAt: time.Now(),
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&link).Error
}

func (r *repository) UnlinkLFSObject(ctx context.Context, repoID uuid.UUID, oid string) error {
	return r.db.WithContext(ctx).
		Delete(&LFSRepoObject{}, "repo_id = ? AND oid = ?", repoID, oid).Error
}

func (r *repository) GetRepoLFSSize(ctx context.Context, repoID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&LFSRepoObject{}).
		Select("COALESCE(SUM(lfs_objects.size), 0)").
		Joins("JOIN lfs_objects ON lfs_objects.oid = lfs_repo_objects.oid").
		Where("lfs_repo_objects.repo_id = ?", repoID).
		Scan(&total).Error
	return total, err
}

func (r *repository) GetUserTotalStorage(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&GitRepo{}).
		Select("COALESCE(SUM(size_bytes + lfs_size_bytes), 0)").
		Where("owner_id = ?", userID).
		Scan(&total).Error
	return total, err
}

func (r *repository) ListRepoLFSObjects(ctx context.Context, repoID uuid.UUID) ([]*LFSObject, error) {
	var objects []*LFSObject
	err := r.db.WithContext(ctx).
		Model(&LFSObject{}).
		Joins("JOIN lfs_repo_objects ON lfs_repo_objects.oid = lfs_objects.oid").
		Where("lfs_repo_objects.repo_id = ?", repoID).
		Find(&objects).Error
	return objects, err
}

// --- Lock Operations ---

func (r *repository) CreateLock(ctx context.Context, lock *LFSLock) error {
	err := r.db.WithContext(ctx).Create(lock).Error
	if err != nil {
		// Check for unique constraint violation
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrLockAlreadyExists
		}
		return err
	}
	return nil
}

func (r *repository) GetLock(ctx context.Context, id uuid.UUID) (*LFSLock, error) {
	var lock LFSLock
	err := r.db.WithContext(ctx).First(&lock, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLockNotFound
		}
		return nil, err
	}
	return &lock, nil
}

func (r *repository) GetLockByPath(ctx context.Context, repoID uuid.UUID, path string) (*LFSLock, error) {
	var lock LFSLock
	err := r.db.WithContext(ctx).
		Where("repo_id = ? AND path = ?", repoID, path).
		First(&lock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLockNotFound
		}
		return nil, err
	}
	return &lock, nil
}

func (r *repository) ListLocks(ctx context.Context, repoID uuid.UUID, path string, limit int) ([]*LFSLock, error) {
	query := r.db.WithContext(ctx).Where("repo_id = ?", repoID)

	if path != "" {
		query = query.Where("path = ?", path)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var locks []*LFSLock
	err := query.Order("locked_at ASC").Find(&locks).Error
	return locks, err
}

func (r *repository) DeleteLock(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&LFSLock{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLockNotFound
	}
	return nil
}

// --- Pull Request Operations ---

func (r *repository) CreatePR(ctx context.Context, pr *PullRequest) error {
	return r.db.WithContext(ctx).Create(pr).Error
}

func (r *repository) GetPR(ctx context.Context, id uuid.UUID) (*PullRequest, error) {
	var pr PullRequest
	err := r.db.WithContext(ctx).First(&pr, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPRNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (r *repository) GetPRByNumber(ctx context.Context, repoID uuid.UUID, number int) (*PullRequest, error) {
	var pr PullRequest
	err := r.db.WithContext(ctx).
		Where("repo_id = ? AND number = ?", repoID, number).
		First(&pr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPRNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (r *repository) ListPRs(ctx context.Context, repoID uuid.UUID, status *PRStatus, limit, offset int) ([]*PullRequest, int64, error) {
	query := r.db.WithContext(ctx).Model(&PullRequest{}).Where("repo_id = ?", repoID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var prs []*PullRequest
	err := query.Order("created_at DESC").Find(&prs).Error
	return prs, total, err
}

func (r *repository) UpdatePR(ctx context.Context, pr *PullRequest) error {
	return r.db.WithContext(ctx).Save(pr).Error
}

func (r *repository) GetNextPRNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	var maxNumber int
	err := r.db.WithContext(ctx).
		Model(&PullRequest{}).
		Select("COALESCE(MAX(number), 0)").
		Where("repo_id = ?", repoID).
		Scan(&maxNumber).Error
	if err != nil {
		return 0, err
	}
	return maxNumber + 1, nil
}
