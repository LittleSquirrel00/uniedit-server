package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/domain/collaboration"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/infra/persistence/entity"
)

// UserRepository implements user.Repository using GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Ensure interface compliance
var _ user.Repository = (*UserRepository)(nil)

// --- User Operations ---

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	e := entity.FromDomainUser(u)
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var e entity.UserEntity
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return e.ToDomain(), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var e entity.UserEntity
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return e.ToDomain(), nil
}

func (r *UserRepository) GetByOAuth(ctx context.Context, provider, oauthID string) (*user.User, error) {
	var e entity.UserEntity
	err := r.db.WithContext(ctx).
		Where("oauth_provider = ? AND oauth_id = ? AND deleted_at IS NULL", provider, oauthID).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return e.ToDomain(), nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	e := entity.FromDomainUser(u)
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.UserEntity{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     string(user.StatusDeleted),
			"deleted_at": now,
			"updated_at": now,
		}).Error
}

func (r *UserRepository) List(ctx context.Context, filter *user.UserFilter, pagination *user.Pagination) ([]*user.User, int64, error) {
	var entities []entity.UserEntity
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.UserEntity{}).Where("deleted_at IS NULL")

	// Apply filters
	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", string(*filter.Status))
		}
		if filter.Email != nil {
			query = query.Where("email ILIKE ?", "%"+*filter.Email+"%")
		}
		if filter.IsAdmin != nil {
			query = query.Where("is_admin = ?", *filter.IsAdmin)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if pagination != nil {
		query = query.Offset(pagination.Offset()).Limit(pagination.PageSize)
	}

	// Fetch results
	if err := query.Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	// Convert to domain models
	users := make([]*user.User, len(entities))
	for i, e := range entities {
		users[i] = e.ToDomain()
	}

	return users, total, nil
}

// --- Email Verification Operations ---

func (r *UserRepository) CreateVerification(ctx context.Context, v *user.EmailVerification) error {
	e := entity.FromDomainVerification(v)
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *UserRepository) GetVerificationByToken(ctx context.Context, token string) (*user.EmailVerification, error) {
	var e entity.EmailVerificationEntity
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrInvalidToken
		}
		return nil, err
	}
	return e.ToDomain(), nil
}

func (r *UserRepository) InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose user.VerificationPurpose) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.EmailVerificationEntity{}).
		Where("user_id = ? AND purpose = ? AND used_at IS NULL", userID, string(purpose)).
		Update("used_at", now).Error
}

func (r *UserRepository) MarkVerificationUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.EmailVerificationEntity{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}

// --- UserLookup Implementation ---

// UserLookupAdapter adapts UserRepository to collaboration.UserLookup interface.
type UserLookupAdapter struct {
	repo *UserRepository
}

// NewUserLookupAdapter creates a new user lookup adapter.
func NewUserLookupAdapter(repo *UserRepository) *UserLookupAdapter {
	return &UserLookupAdapter{repo: repo}
}

// GetByEmail retrieves user info by email.
func (a *UserLookupAdapter) GetByEmail(ctx context.Context, email string) (*collaboration.UserInfo, error) {
	u, err := a.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &collaboration.UserInfo{
		ID:    u.ID(),
		Email: u.Email(),
		Name:  u.Name(),
	}, nil
}

// GetByID retrieves user info by ID.
func (a *UserLookupAdapter) GetByID(ctx context.Context, id uuid.UUID) (*collaboration.UserInfo, error) {
	u, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &collaboration.UserInfo{
		ID:    u.ID(),
		Email: u.Email(),
		Name:  u.Name(),
	}, nil
}
