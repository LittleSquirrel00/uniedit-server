package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrModelNotFound    = errors.New("model not found")
)

// Repository defines the interface for provider data access.
type Repository interface {
	// Provider operations
	CreateProvider(ctx context.Context, provider *Provider) error
	GetProvider(ctx context.Context, id uuid.UUID) (*Provider, error)
	GetProviderByName(ctx context.Context, name string) (*Provider, error)
	ListProviders(ctx context.Context, enabledOnly bool) ([]*Provider, error)
	UpdateProvider(ctx context.Context, provider *Provider) error
	DeleteProvider(ctx context.Context, id uuid.UUID) error

	// Model operations
	CreateModel(ctx context.Context, model *Model) error
	GetModel(ctx context.Context, id string) (*Model, error)
	ListModels(ctx context.Context, providerID *uuid.UUID, enabledOnly bool) ([]*Model, error)
	ListModelsByCapability(ctx context.Context, capability Capability) ([]*Model, error)
	UpdateModel(ctx context.Context, model *Model) error
	DeleteModel(ctx context.Context, id string) error

	// Combined operations
	GetProviderWithModels(ctx context.Context, id uuid.UUID) (*Provider, error)
	ListProvidersWithModels(ctx context.Context, enabledOnly bool) ([]*Provider, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new provider repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateProvider creates a new provider.
func (r *repository) CreateProvider(ctx context.Context, provider *Provider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

// GetProvider retrieves a provider by ID.
func (r *repository) GetProvider(ctx context.Context, id uuid.UUID) (*Provider, error) {
	var provider Provider
	err := r.db.WithContext(ctx).First(&provider, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, fmt.Errorf("get provider: %w", err)
	}
	return &provider, nil
}

// GetProviderByName retrieves a provider by name.
func (r *repository) GetProviderByName(ctx context.Context, name string) (*Provider, error) {
	var provider Provider
	err := r.db.WithContext(ctx).First(&provider, "name = ?", name).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, fmt.Errorf("get provider by name: %w", err)
	}
	return &provider, nil
}

// ListProviders lists all providers.
func (r *repository) ListProviders(ctx context.Context, enabledOnly bool) ([]*Provider, error) {
	var providers []*Provider
	query := r.db.WithContext(ctx)
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Order("priority DESC, name ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	return providers, nil
}

// UpdateProvider updates a provider.
func (r *repository) UpdateProvider(ctx context.Context, provider *Provider) error {
	result := r.db.WithContext(ctx).Save(provider)
	if result.Error != nil {
		return fmt.Errorf("update provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// DeleteProvider deletes a provider.
func (r *repository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&Provider{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// CreateModel creates a new model.
func (r *repository) CreateModel(ctx context.Context, model *Model) error {
	return r.db.WithContext(ctx).Create(model).Error
}

// GetModel retrieves a model by ID.
func (r *repository) GetModel(ctx context.Context, id string) (*Model, error) {
	var model Model
	err := r.db.WithContext(ctx).Preload("Provider").First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModelNotFound
		}
		return nil, fmt.Errorf("get model: %w", err)
	}
	return &model, nil
}

// ListModels lists all models.
func (r *repository) ListModels(ctx context.Context, providerID *uuid.UUID, enabledOnly bool) ([]*Model, error) {
	var models []*Model
	query := r.db.WithContext(ctx).Preload("Provider")
	if providerID != nil {
		query = query.Where("provider_id = ?", *providerID)
	}
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Order("name ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	return models, nil
}

// ListModelsByCapability lists models with a specific capability.
func (r *repository) ListModelsByCapability(ctx context.Context, capability Capability) ([]*Model, error) {
	var models []*Model
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Where("enabled = ? AND ? = ANY(capabilities)", true, string(capability)).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("list models by capability: %w", err)
	}
	return models, nil
}

// UpdateModel updates a model.
func (r *repository) UpdateModel(ctx context.Context, model *Model) error {
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("update model: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrModelNotFound
	}
	return nil
}

// DeleteModel deletes a model.
func (r *repository) DeleteModel(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Model{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete model: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrModelNotFound
	}
	return nil
}

// GetProviderWithModels retrieves a provider with its models.
func (r *repository) GetProviderWithModels(ctx context.Context, id uuid.UUID) (*Provider, error) {
	var provider Provider
	err := r.db.WithContext(ctx).Preload("Models").First(&provider, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, fmt.Errorf("get provider with models: %w", err)
	}
	return &provider, nil
}

// ListProvidersWithModels lists all providers with their models.
func (r *repository) ListProvidersWithModels(ctx context.Context, enabledOnly bool) ([]*Provider, error) {
	var providers []*Provider
	query := r.db.WithContext(ctx).Preload("Models")
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Order("priority DESC, name ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("list providers with models: %w", err)
	}
	return providers, nil
}
