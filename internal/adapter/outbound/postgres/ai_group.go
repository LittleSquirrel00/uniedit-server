package postgres

import (
	"context"
	"errors"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// aiModelGroupAdapter implements outbound.AIModelGroupDatabasePort.
type aiModelGroupAdapter struct {
	db *gorm.DB
}

// NewAIModelGroupAdapter creates a new AI model group database adapter.
func NewAIModelGroupAdapter(db *gorm.DB) outbound.AIModelGroupDatabasePort {
	return &aiModelGroupAdapter{db: db}
}

func (a *aiModelGroupAdapter) Create(ctx context.Context, group *model.AIModelGroup) error {
	return a.db.WithContext(ctx).Create(group).Error
}

func (a *aiModelGroupAdapter) FindByID(ctx context.Context, id string) (*model.AIModelGroup, error) {
	var group model.AIModelGroup
	err := a.db.WithContext(ctx).First(&group, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (a *aiModelGroupAdapter) FindAll(ctx context.Context) ([]*model.AIModelGroup, error) {
	var groups []*model.AIModelGroup
	err := a.db.WithContext(ctx).Find(&groups).Error
	return groups, err
}

func (a *aiModelGroupAdapter) FindEnabled(ctx context.Context) ([]*model.AIModelGroup, error) {
	var groups []*model.AIModelGroup
	err := a.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&groups).Error
	return groups, err
}

func (a *aiModelGroupAdapter) FindByTaskType(ctx context.Context, taskType model.AITaskType) ([]*model.AIModelGroup, error) {
	var groups []*model.AIModelGroup
	err := a.db.WithContext(ctx).
		Where("task_type = ? AND enabled = ?", taskType, true).
		Find(&groups).Error
	return groups, err
}

func (a *aiModelGroupAdapter) Update(ctx context.Context, group *model.AIModelGroup) error {
	return a.db.WithContext(ctx).Save(group).Error
}

func (a *aiModelGroupAdapter) Delete(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&model.AIModelGroup{}, "id = ?", id).Error
}

// Compile-time check
var _ outbound.AIModelGroupDatabasePort = (*aiModelGroupAdapter)(nil)
