package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// orderAdapter implements outbound.OrderDatabasePort.
type orderAdapter struct {
	db *gorm.DB
}

// NewOrderAdapter creates a new order database adapter.
func NewOrderAdapter(db *gorm.DB) outbound.OrderDatabasePort {
	return &orderAdapter{db: db}
}

func (a *orderAdapter) Create(ctx context.Context, order *model.Order) error {
	return a.db.WithContext(ctx).Create(order).Error
}

func (a *orderAdapter) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := a.db.WithContext(ctx).First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (a *orderAdapter) GetByIDWithItems(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := a.db.WithContext(ctx).
		Preload("Items").
		First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (a *orderAdapter) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	var order model.Order
	err := a.db.WithContext(ctx).
		Preload("Items").
		First(&order, "order_no = ?", orderNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (a *orderAdapter) GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Order, error) {
	var order model.Order
	err := a.db.WithContext(ctx).
		First(&order, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (a *orderAdapter) List(ctx context.Context, userID uuid.UUID, filter *model.OrderFilter, page, pageSize int) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	query := a.db.WithContext(ctx).Model(&model.Order{}).Where("user_id = ?", userID)

	// Apply filters
	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", string(*filter.Status))
		}
		if filter.Type != nil {
			query = query.Where("type = ?", string(*filter.Type))
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	// Fetch results with items
	if err := query.Preload("Items").Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (a *orderAdapter) Update(ctx context.Context, order *model.Order) error {
	return a.db.WithContext(ctx).Save(order).Error
}

func (a *orderAdapter) ListPendingExpired(ctx context.Context) ([]*model.Order, error) {
	var orders []*model.Order
	err := a.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < NOW()", string(model.OrderStatusPending)).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// Compile-time check
var _ outbound.OrderDatabasePort = (*orderAdapter)(nil)
