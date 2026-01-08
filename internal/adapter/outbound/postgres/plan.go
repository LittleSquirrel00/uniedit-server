package postgres

import (
	"context"
	"errors"

	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// planAdapter implements outbound.PlanDatabasePort.
type planAdapter struct {
	db *gorm.DB
}

// NewPlanAdapter creates a new plan database adapter.
func NewPlanAdapter(db *gorm.DB) outbound.PlanDatabasePort {
	return &planAdapter{db: db}
}

func (a *planAdapter) ListActive(ctx context.Context) ([]*model.Plan, error) {
	var plans []*model.Plan
	err := a.db.WithContext(ctx).
		Where("active = ?", true).
		Order("display_order ASC").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

func (a *planAdapter) GetByID(ctx context.Context, id string) (*model.Plan, error) {
	var plan model.Plan
	err := a.db.WithContext(ctx).First(&plan, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func (a *planAdapter) Create(ctx context.Context, plan *model.Plan) error {
	return a.db.WithContext(ctx).Create(plan).Error
}

func (a *planAdapter) Update(ctx context.Context, plan *model.Plan) error {
	return a.db.WithContext(ctx).Save(plan).Error
}

// Compile-time check
var _ outbound.PlanDatabasePort = (*planAdapter)(nil)
