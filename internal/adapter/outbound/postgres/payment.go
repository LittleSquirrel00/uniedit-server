package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
	"github.com/uniedit/server/internal/port/outbound"
	"gorm.io/gorm"
)

// paymentAdapter implements outbound.PaymentDatabasePort.
type paymentAdapter struct {
	db *gorm.DB
}

// NewPaymentAdapter creates a new payment database adapter.
func NewPaymentAdapter(db *gorm.DB) outbound.PaymentDatabasePort {
	return &paymentAdapter{db: db}
}

func (a *paymentAdapter) Create(ctx context.Context, payment *model.Payment) error {
	if err := a.db.WithContext(ctx).Create(payment).Error; err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (a *paymentAdapter) FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error) {
	var payment model.Payment
	err := a.db.WithContext(ctx).First(&payment, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find payment by id: %w", err)
	}
	return &payment, nil
}

func (a *paymentAdapter) FindByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Payment, error) {
	var payment model.Payment
	err := a.db.WithContext(ctx).First(&payment, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find payment by payment intent id: %w", err)
	}
	return &payment, nil
}

func (a *paymentAdapter) FindByTradeNo(ctx context.Context, tradeNo string) (*model.Payment, error) {
	var payment model.Payment
	err := a.db.WithContext(ctx).First(&payment, "trade_no = ?", tradeNo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find payment by trade no: %w", err)
	}
	return &payment, nil
}

func (a *paymentAdapter) FindByFilter(ctx context.Context, filter model.PaymentFilter) ([]*model.Payment, int64, error) {
	var payments []*model.Payment
	var total int64

	query := a.db.WithContext(ctx).Model(&model.Payment{})

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.OrderID != nil {
		query = query.Where("order_id = ?", *filter.OrderID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Provider != nil {
		query = query.Where("provider = ?", *filter.Provider)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count payments: %w", err)
	}

	filter.DefaultPagination()
	if err := query.Offset(filter.Offset()).Limit(filter.PageSize).Order("created_at DESC").Find(&payments).Error; err != nil {
		return nil, 0, fmt.Errorf("find payments: %w", err)
	}

	return payments, total, nil
}

func (a *paymentAdapter) FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.Payment, error) {
	var payments []*model.Payment
	err := a.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&payments).Error
	if err != nil {
		return nil, fmt.Errorf("find payments by order: %w", err)
	}
	return payments, nil
}

func (a *paymentAdapter) Update(ctx context.Context, payment *model.Payment) error {
	if err := a.db.WithContext(ctx).Save(payment).Error; err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	return nil
}

// Compile-time check
var _ outbound.PaymentDatabasePort = (*paymentAdapter)(nil)
