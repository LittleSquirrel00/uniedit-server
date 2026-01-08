package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/user"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// userAdapter implements outbound.UserDatabasePort.
type userAdapter struct {
	db *gorm.DB
}

// NewUserAdapter creates a new user database adapter.
func NewUserAdapter(db *gorm.DB) outbound.UserDatabasePort {
	return &userAdapter{db: db}
}

func (a *userAdapter) Create(ctx context.Context, u *model.User) error {
	return a.db.WithContext(ctx).Create(u).Error
}

func (a *userAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := a.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (a *userAdapter) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := a.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (a *userAdapter) FindByFilter(ctx context.Context, filter model.UserFilter) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := a.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")

	// Apply filters
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}
	if filter.Email != "" {
		query = query.Where("email ILIKE ?", "%"+filter.Email+"%")
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.IsAdmin != nil {
		query = query.Where("is_admin = ?", *filter.IsAdmin)
	}
	if filter.Search != "" {
		query = query.Where("name ILIKE ? OR email ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	filter.DefaultPagination()
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (a *userAdapter) Update(ctx context.Context, u *model.User) error {
	return a.db.WithContext(ctx).Save(u).Error
}

func (a *userAdapter) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return a.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     model.UserStatusDeleted,
			"deleted_at": now,
		}).Error
}

// Compile-time check
var _ outbound.UserDatabasePort = (*userAdapter)(nil)
