package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// orderItemAdapter implements outbound.OrderItemDatabasePort.
type orderItemAdapter struct {
	db *gorm.DB
}

// NewOrderItemAdapter creates a new order item database adapter.
func NewOrderItemAdapter(db *gorm.DB) outbound.OrderItemDatabasePort {
	return &orderItemAdapter{db: db}
}

func (a *orderItemAdapter) CreateBatch(ctx context.Context, items []*model.OrderItem) error {
	if len(items) == 0 {
		return nil
	}
	return a.db.WithContext(ctx).Create(&items).Error
}

func (a *orderItemAdapter) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.OrderItem, error) {
	var items []*model.OrderItem
	err := a.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// Compile-time check
var _ outbound.OrderItemDatabasePort = (*orderItemAdapter)(nil)
