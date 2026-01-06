package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the interface for user data access.
type Repository interface {
	// User operations
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter *UserFilter, pagination *Pagination) ([]*User, int64, error)

	// Email verification operations
	CreateVerification(ctx context.Context, verification *EmailVerification) error
	GetVerificationByToken(ctx context.Context, token string) (*EmailVerification, error)
	InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose VerificationPurpose) error
	MarkVerificationUsed(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new user repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- User Operations ---

func (r *repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     UserStatusDeleted,
			"deleted_at": now,
		}).Error
}

func (r *repository) List(ctx context.Context, filter *UserFilter, pagination *Pagination) ([]*User, int64, error) {
	var users []*User
	var total int64

	query := r.db.WithContext(ctx).Model(&User{}).Where("deleted_at IS NULL")

	// Apply filters
	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", *filter.Status)
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
	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// --- Email Verification Operations ---

func (r *repository) CreateVerification(ctx context.Context, verification *EmailVerification) error {
	return r.db.WithContext(ctx).Create(verification).Error
}

func (r *repository) GetVerificationByToken(ctx context.Context, token string) (*EmailVerification, error) {
	var verification EmailVerification
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return &verification, nil
}

func (r *repository) InvalidateUserVerifications(ctx context.Context, userID uuid.UUID, purpose VerificationPurpose) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&EmailVerification{}).
		Where("user_id = ? AND purpose = ? AND used_at IS NULL", userID, purpose).
		Update("used_at", now).Error
}

func (r *repository) MarkVerificationUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&EmailVerification{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}
