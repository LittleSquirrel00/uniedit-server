package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// GitRepoNotFound indicates the repository was not found.
var GitRepoNotFound = errors.New("repository not found")

// GitRepoDatabaseAdapter implements GitRepoDatabasePort using GORM.
type GitRepoDatabaseAdapter struct {
	db *gorm.DB
}

// NewGitRepoDatabaseAdapter creates a new Git repository database adapter.
func NewGitRepoDatabaseAdapter(db *gorm.DB) *GitRepoDatabaseAdapter {
	return &GitRepoDatabaseAdapter{db: db}
}

// Create creates a new repository.
func (a *GitRepoDatabaseAdapter) Create(ctx context.Context, repo *model.GitRepo) error {
	return a.db.WithContext(ctx).Create(repo).Error
}

// FindByID finds a repository by ID.
func (a *GitRepoDatabaseAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.GitRepo, error) {
	var repo model.GitRepo
	err := a.db.WithContext(ctx).First(&repo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

// FindByOwnerAndSlug finds a repository by owner and slug.
func (a *GitRepoDatabaseAdapter) FindByOwnerAndSlug(ctx context.Context, ownerID uuid.UUID, slug string) (*model.GitRepo, error) {
	var repo model.GitRepo
	err := a.db.WithContext(ctx).
		Where("owner_id = ? AND slug = ?", ownerID, slug).
		First(&repo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

// FindByOwner lists repositories owned by a user.
func (a *GitRepoDatabaseAdapter) FindByOwner(ctx context.Context, ownerID uuid.UUID, filter *outbound.GitRepoFilter) ([]*model.GitRepo, int64, error) {
	query := a.db.WithContext(ctx).Model(&model.GitRepo{}).Where("owner_id = ?", ownerID)

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

	var repos []*model.GitRepo
	err := query.Order("created_at DESC").Find(&repos).Error
	return repos, total, err
}

// FindPublic lists public repositories.
func (a *GitRepoDatabaseAdapter) FindPublic(ctx context.Context, filter *outbound.GitRepoFilter) ([]*model.GitRepo, int64, error) {
	query := a.db.WithContext(ctx).Model(&model.GitRepo{}).Where("visibility = ?", model.GitVisibilityPublic)

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

	var repos []*model.GitRepo
	err := query.Order("stars_count DESC, created_at DESC").Find(&repos).Error
	return repos, total, err
}

// Update updates a repository.
func (a *GitRepoDatabaseAdapter) Update(ctx context.Context, repo *model.GitRepo) error {
	return a.db.WithContext(ctx).Save(repo).Error
}

// Delete deletes a repository.
func (a *GitRepoDatabaseAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.GitRepo{}, "id = ?", id).Error
}

// UpdatePushedAt updates the last push timestamp.
func (a *GitRepoDatabaseAdapter) UpdatePushedAt(ctx context.Context, id uuid.UUID, pushedAt *time.Time) error {
	return a.db.WithContext(ctx).
		Model(&model.GitRepo{}).
		Where("id = ?", id).
		Update("pushed_at", pushedAt).Error
}

// UpdateSize updates repository size statistics.
func (a *GitRepoDatabaseAdapter) UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes, lfsSizeBytes int64) error {
	return a.db.WithContext(ctx).
		Model(&model.GitRepo{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"size_bytes":     sizeBytes,
			"lfs_size_bytes": lfsSizeBytes,
		}).Error
}

// IncrementStars increments the star count.
func (a *GitRepoDatabaseAdapter) IncrementStars(ctx context.Context, id uuid.UUID, delta int) error {
	return a.db.WithContext(ctx).
		Model(&model.GitRepo{}).
		Where("id = ?", id).
		Update("stars_count", gorm.Expr("stars_count + ?", delta)).Error
}

// IncrementForks increments the fork count.
func (a *GitRepoDatabaseAdapter) IncrementForks(ctx context.Context, id uuid.UUID, delta int) error {
	return a.db.WithContext(ctx).
		Model(&model.GitRepo{}).
		Where("id = ?", id).
		Update("forks_count", gorm.Expr("forks_count + ?", delta)).Error
}

// Compile-time check
var _ outbound.GitRepoDatabasePort = (*GitRepoDatabaseAdapter)(nil)

// ===== Collaborator Adapter =====

// GitCollaboratorNotFound indicates the collaborator was not found.
var GitCollaboratorNotFound = errors.New("collaborator not found")

// GitCollaboratorDatabaseAdapter implements GitCollaboratorDatabasePort using GORM.
type GitCollaboratorDatabaseAdapter struct {
	db *gorm.DB
}

// NewGitCollaboratorDatabaseAdapter creates a new collaborator database adapter.
func NewGitCollaboratorDatabaseAdapter(db *gorm.DB) *GitCollaboratorDatabaseAdapter {
	return &GitCollaboratorDatabaseAdapter{db: db}
}

// Add adds a collaborator to a repository.
func (a *GitCollaboratorDatabaseAdapter) Add(ctx context.Context, collab *model.GitRepoCollaborator) error {
	return a.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "repo_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"permission"}),
		}).
		Create(collab).Error
}

// FindByRepoAndUser finds a specific collaborator.
func (a *GitCollaboratorDatabaseAdapter) FindByRepoAndUser(ctx context.Context, repoID, userID uuid.UUID) (*model.GitRepoCollaborator, error) {
	var collab model.GitRepoCollaborator
	err := a.db.WithContext(ctx).
		Where("repo_id = ? AND user_id = ?", repoID, userID).
		First(&collab).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &collab, nil
}

// FindByRepo lists all collaborators for a repository.
func (a *GitCollaboratorDatabaseAdapter) FindByRepo(ctx context.Context, repoID uuid.UUID) ([]*model.GitRepoCollaborator, error) {
	var collabs []*model.GitRepoCollaborator
	err := a.db.WithContext(ctx).
		Where("repo_id = ?", repoID).
		Order("created_at ASC").
		Find(&collabs).Error
	return collabs, err
}

// FindByUser lists all repositories a user collaborates on.
func (a *GitCollaboratorDatabaseAdapter) FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.GitRepoCollaborator, error) {
	var collabs []*model.GitRepoCollaborator
	err := a.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&collabs).Error
	return collabs, err
}

// Update updates a collaborator's permission.
func (a *GitCollaboratorDatabaseAdapter) Update(ctx context.Context, collab *model.GitRepoCollaborator) error {
	return a.db.WithContext(ctx).
		Model(&model.GitRepoCollaborator{}).
		Where("repo_id = ? AND user_id = ?", collab.RepoID, collab.UserID).
		Update("permission", collab.Permission).Error
}

// Remove removes a collaborator.
func (a *GitCollaboratorDatabaseAdapter) Remove(ctx context.Context, repoID, userID uuid.UUID) error {
	result := a.db.WithContext(ctx).
		Delete(&model.GitRepoCollaborator{}, "repo_id = ? AND user_id = ?", repoID, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return GitCollaboratorNotFound
	}
	return nil
}

// Compile-time check
var _ outbound.GitCollaboratorDatabasePort = (*GitCollaboratorDatabaseAdapter)(nil)
