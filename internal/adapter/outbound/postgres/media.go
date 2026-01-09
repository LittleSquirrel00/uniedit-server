package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
)

// --- Provider Database Adapter ---

// MediaProviderDBAdapter implements MediaProviderDatabasePort.
type MediaProviderDBAdapter struct {
	db *gorm.DB
}

// NewMediaProviderDBAdapter creates a new media provider database adapter.
func NewMediaProviderDBAdapter(db *gorm.DB) *MediaProviderDBAdapter {
	return &MediaProviderDBAdapter{db: db}
}

func (a *MediaProviderDBAdapter) Create(ctx context.Context, provider *model.MediaProvider) error {
	return a.db.WithContext(ctx).Create(provider).Error
}

func (a *MediaProviderDBAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.MediaProvider, error) {
	var provider model.MediaProvider
	if err := a.db.WithContext(ctx).First(&provider, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func (a *MediaProviderDBAdapter) FindAll(ctx context.Context) ([]*model.MediaProvider, error) {
	var providers []*model.MediaProvider
	if err := a.db.WithContext(ctx).Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (a *MediaProviderDBAdapter) FindEnabled(ctx context.Context) ([]*model.MediaProvider, error) {
	var providers []*model.MediaProvider
	if err := a.db.WithContext(ctx).Where("enabled = ?", true).Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (a *MediaProviderDBAdapter) Update(ctx context.Context, provider *model.MediaProvider) error {
	return a.db.WithContext(ctx).Save(provider).Error
}

func (a *MediaProviderDBAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.MediaProvider{}, "id = ?", id).Error
}

var _ outbound.MediaProviderDatabasePort = (*MediaProviderDBAdapter)(nil)

// --- Model Database Adapter ---

// MediaModelDBAdapter implements MediaModelDatabasePort.
type MediaModelDBAdapter struct {
	db *gorm.DB
}

// NewMediaModelDBAdapter creates a new media model database adapter.
func NewMediaModelDBAdapter(db *gorm.DB) *MediaModelDBAdapter {
	return &MediaModelDBAdapter{db: db}
}

func (a *MediaModelDBAdapter) Create(ctx context.Context, m *model.MediaModel) error {
	return a.db.WithContext(ctx).Create(m).Error
}

func (a *MediaModelDBAdapter) FindByID(ctx context.Context, id string) (*model.MediaModel, error) {
	var m model.MediaModel
	if err := a.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (a *MediaModelDBAdapter) FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.MediaModel, error) {
	var models []*model.MediaModel
	if err := a.db.WithContext(ctx).Where("provider_id = ?", providerID).Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

func (a *MediaModelDBAdapter) FindByCapability(ctx context.Context, capability model.MediaCapability) ([]*model.MediaModel, error) {
	var models []*model.MediaModel
	// Use JSONB contains operator for PostgreSQL
	if err := a.db.WithContext(ctx).
		Where("capabilities @> ?", `["`+string(capability)+`"]`).
		Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

func (a *MediaModelDBAdapter) FindEnabled(ctx context.Context) ([]*model.MediaModel, error) {
	var models []*model.MediaModel
	if err := a.db.WithContext(ctx).Where("enabled = ?", true).Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

func (a *MediaModelDBAdapter) Update(ctx context.Context, m *model.MediaModel) error {
	return a.db.WithContext(ctx).Save(m).Error
}

func (a *MediaModelDBAdapter) Delete(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&model.MediaModel{}, "id = ?", id).Error
}

var _ outbound.MediaModelDatabasePort = (*MediaModelDBAdapter)(nil)

// --- Task Database Adapter ---

// MediaTaskDBAdapter implements MediaTaskDatabasePort.
type MediaTaskDBAdapter struct {
	db *gorm.DB
}

// NewMediaTaskDBAdapter creates a new media task database adapter.
func NewMediaTaskDBAdapter(db *gorm.DB) *MediaTaskDBAdapter {
	return &MediaTaskDBAdapter{db: db}
}

func (a *MediaTaskDBAdapter) Create(ctx context.Context, task *model.MediaTask) error {
	return a.db.WithContext(ctx).Create(task).Error
}

func (a *MediaTaskDBAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.MediaTask, error) {
	var task model.MediaTask
	if err := a.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

func (a *MediaTaskDBAdapter) FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*model.MediaTask, error) {
	var tasks []*model.MediaTask
	if err := a.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (a *MediaTaskDBAdapter) FindPending(ctx context.Context, limit int) ([]*model.MediaTask, error) {
	var tasks []*model.MediaTask
	if err := a.db.WithContext(ctx).
		Where("status = ?", model.MediaTaskStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (a *MediaTaskDBAdapter) Update(ctx context.Context, task *model.MediaTask) error {
	return a.db.WithContext(ctx).Save(task).Error
}

func (a *MediaTaskDBAdapter) UpdateStatus(ctx context.Context, id uuid.UUID, status model.MediaTaskStatus, progress int, output, errMsg string) error {
	updates := map[string]interface{}{
		"status":     status,
		"progress":   progress,
		"updated_at": gorm.Expr("NOW()"),
	}
	if output != "" {
		updates["output"] = output
	}
	if errMsg != "" {
		updates["error"] = errMsg
	}
	return a.db.WithContext(ctx).Model(&model.MediaTask{}).Where("id = ?", id).Updates(updates).Error
}

func (a *MediaTaskDBAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.MediaTask{}, "id = ?", id).Error
}

var _ outbound.MediaTaskDatabasePort = (*MediaTaskDBAdapter)(nil)
