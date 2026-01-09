package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// GitPRNotFound indicates the pull request was not found.
var GitPRNotFound = errors.New("pull request not found")

// GitPullRequestDatabaseAdapter implements GitPullRequestDatabasePort using GORM.
type GitPullRequestDatabaseAdapter struct {
	db *gorm.DB
}

// NewGitPullRequestDatabaseAdapter creates a new pull request database adapter.
func NewGitPullRequestDatabaseAdapter(db *gorm.DB) *GitPullRequestDatabaseAdapter {
	return &GitPullRequestDatabaseAdapter{db: db}
}

// Create creates a new pull request.
func (a *GitPullRequestDatabaseAdapter) Create(ctx context.Context, pr *model.GitPullRequest) error {
	return a.db.WithContext(ctx).Create(pr).Error
}

// FindByID finds a pull request by ID.
func (a *GitPullRequestDatabaseAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.GitPullRequest, error) {
	var pr model.GitPullRequest
	err := a.db.WithContext(ctx).First(&pr, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pr, nil
}

// FindByNumber finds a pull request by repository and number.
func (a *GitPullRequestDatabaseAdapter) FindByNumber(ctx context.Context, repoID uuid.UUID, number int) (*model.GitPullRequest, error) {
	var pr model.GitPullRequest
	err := a.db.WithContext(ctx).
		Where("repo_id = ? AND number = ?", repoID, number).
		First(&pr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &pr, nil
}

// FindByRepo lists pull requests for a repository.
func (a *GitPullRequestDatabaseAdapter) FindByRepo(ctx context.Context, repoID uuid.UUID, status *model.GitPRStatus, limit, offset int) ([]*model.GitPullRequest, int64, error) {
	query := a.db.WithContext(ctx).Model(&model.GitPullRequest{}).Where("repo_id = ?", repoID)

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

	var prs []*model.GitPullRequest
	err := query.Order("created_at DESC").Find(&prs).Error
	return prs, total, err
}

// Update updates a pull request.
func (a *GitPullRequestDatabaseAdapter) Update(ctx context.Context, pr *model.GitPullRequest) error {
	return a.db.WithContext(ctx).Save(pr).Error
}

// GetNextNumber gets the next PR number for a repository.
func (a *GitPullRequestDatabaseAdapter) GetNextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	var maxNumber int
	err := a.db.WithContext(ctx).
		Model(&model.GitPullRequest{}).
		Select("COALESCE(MAX(number), 0)").
		Where("repo_id = ?", repoID).
		Scan(&maxNumber).Error
	if err != nil {
		return 0, err
	}
	return maxNumber + 1, nil
}

// Compile-time check
var _ outbound.GitPullRequestDatabasePort = (*GitPullRequestDatabaseAdapter)(nil)
