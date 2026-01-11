package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// aiModelAdapter implements outbound.AIModelDatabasePort.
type aiModelAdapter struct {
	db *gorm.DB
}

// NewAIModelAdapter creates a new AI model database adapter.
func NewAIModelAdapter(db *gorm.DB) outbound.AIModelDatabasePort {
	return &aiModelAdapter{db: db}
}

func (a *aiModelAdapter) Create(ctx context.Context, m *model.AIModel) error {
	return a.db.WithContext(ctx).Create(m).Error
}

func (a *aiModelAdapter) FindByID(ctx context.Context, id string) (*model.AIModel, error) {
	var m model.AIModel
	err := a.db.WithContext(ctx).Preload("Provider").First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *aiModelAdapter) FindByProvider(ctx context.Context, providerID uuid.UUID) ([]*model.AIModel, error) {
	var models []*model.AIModel
	err := a.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Find(&models).Error
	return models, err
}

func (a *aiModelAdapter) FindEnabled(ctx context.Context) ([]*model.AIModel, error) {
	var models []*model.AIModel
	err := a.db.WithContext(ctx).
		Preload("Provider").
		Where("enabled = ?", true).
		Find(&models).Error
	return models, err
}

func (a *aiModelAdapter) FindByCapability(ctx context.Context, capability model.AICapability) ([]*model.AIModel, error) {
	var models []*model.AIModel
	err := a.db.WithContext(ctx).
		Preload("Provider").
		Where("? = ANY(capabilities)", string(capability)).
		Where("enabled = ?", true).
		Find(&models).Error
	return models, err
}

func (a *aiModelAdapter) FindByCapabilities(ctx context.Context, capabilities []model.AICapability) ([]*model.AIModel, error) {
	if len(capabilities) == 0 {
		return a.FindEnabled(ctx)
	}

	query := a.db.WithContext(ctx).
		Preload("Provider").
		Where("enabled = ?", true)

	for _, cap := range capabilities {
		query = query.Where("? = ANY(capabilities)", string(cap))
	}

	var models []*model.AIModel
	err := query.Find(&models).Error
	return models, err
}

func (a *aiModelAdapter) Update(ctx context.Context, m *model.AIModel) error {
	return a.db.WithContext(ctx).Save(m).Error
}

func (a *aiModelAdapter) Delete(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&model.AIModel{}, "id = ?", id).Error
}

func (a *aiModelAdapter) DeleteByProvider(ctx context.Context, providerID uuid.UUID) error {
	return a.db.WithContext(ctx).Delete(&model.AIModel{}, "provider_id = ?", providerID).Error
}

// Compile-time check
var _ outbound.AIModelDatabasePort = (*aiModelAdapter)(nil)
