package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// aiProviderAdapter implements outbound.AIProviderDatabasePort.
type aiProviderAdapter struct {
	db *gorm.DB
}

// NewAIProviderAdapter creates a new AI provider database adapter.
func NewAIProviderAdapter(db *gorm.DB) outbound.AIProviderDatabasePort {
	return &aiProviderAdapter{db: db}
}

func (a *aiProviderAdapter) Create(ctx context.Context, provider *model.AIProvider) error {
	return a.db.WithContext(ctx).Create(provider).Error
}

func (a *aiProviderAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.AIProvider, error) {
	var provider model.AIProvider
	err := a.db.WithContext(ctx).Preload("Models").First(&provider, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (a *aiProviderAdapter) FindAll(ctx context.Context) ([]*model.AIProvider, error) {
	var providers []*model.AIProvider
	err := a.db.WithContext(ctx).Preload("Models").Find(&providers).Error
	return providers, err
}

func (a *aiProviderAdapter) FindEnabled(ctx context.Context) ([]*model.AIProvider, error) {
	var providers []*model.AIProvider
	err := a.db.WithContext(ctx).
		Preload("Models", "enabled = ?", true).
		Where("enabled = ?", true).
		Find(&providers).Error
	return providers, err
}

func (a *aiProviderAdapter) FindByType(ctx context.Context, providerType model.AIProviderType) ([]*model.AIProvider, error) {
	var providers []*model.AIProvider
	err := a.db.WithContext(ctx).
		Preload("Models").
		Where("type = ?", providerType).
		Find(&providers).Error
	return providers, err
}

func (a *aiProviderAdapter) Update(ctx context.Context, provider *model.AIProvider) error {
	return a.db.WithContext(ctx).Save(provider).Error
}

func (a *aiProviderAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.AIProvider{}, "id = ?", id).Error
}

// Compile-time check
var _ outbound.AIProviderDatabasePort = (*aiProviderAdapter)(nil)
