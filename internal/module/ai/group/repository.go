package group

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	ErrGroupNotFound = errors.New("group not found")
)

// Repository defines the interface for group data access.
type Repository interface {
	Create(ctx context.Context, group *Group) error
	Get(ctx context.Context, id string) (*Group, error)
	GetByTaskType(ctx context.Context, taskType TaskType) ([]*Group, error)
	List(ctx context.Context, enabledOnly bool) ([]*Group, error)
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new group repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create creates a new group.
func (r *repository) Create(ctx context.Context, group *Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

// Get retrieves a group by ID.
func (r *repository) Get(ctx context.Context, id string) (*Group, error) {
	var group Group
	err := r.db.WithContext(ctx).First(&group, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group: %w", err)
	}
	return &group, nil
}

// GetByTaskType retrieves groups by task type.
func (r *repository) GetByTaskType(ctx context.Context, taskType TaskType) ([]*Group, error) {
	var groups []*Group
	err := r.db.WithContext(ctx).
		Where("task_type = ? AND enabled = ?", taskType, true).
		Order("name ASC").
		Find(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("get groups by task type: %w", err)
	}
	return groups, nil
}

// List lists all groups.
func (r *repository) List(ctx context.Context, enabledOnly bool) ([]*Group, error) {
	var groups []*Group
	query := r.db.WithContext(ctx)
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Order("task_type ASC, name ASC").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

// Update updates a group.
func (r *repository) Update(ctx context.Context, group *Group) error {
	result := r.db.WithContext(ctx).Save(group)
	if result.Error != nil {
		return fmt.Errorf("update group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// Delete deletes a group.
func (r *repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Group{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGroupNotFound
	}
	return nil
}
