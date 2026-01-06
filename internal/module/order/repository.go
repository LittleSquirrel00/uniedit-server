package order

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines the interface for order data access.
type Repository interface {
	// Order operations
	CreateOrder(ctx context.Context, order *Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*Order, error)
	GetOrderWithItems(ctx context.Context, id uuid.UUID) (*Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*Order, error)
	GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Order, error)
	ListOrders(ctx context.Context, userID uuid.UUID, filter *OrderFilter, pagination *Pagination) ([]*Order, int64, error)
	UpdateOrder(ctx context.Context, order *Order) error
	ListPendingExpiredOrders(ctx context.Context) ([]*Order, error)

	// Order item operations
	CreateOrderItems(ctx context.Context, items []OrderItem) error

	// Invoice operations
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoice(ctx context.Context, id uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, userID uuid.UUID) ([]*Invoice, error)
	GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*Invoice, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new order repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Order Operations ---

func (r *repository) CreateOrder(ctx context.Context, order *Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *repository) GetOrder(ctx context.Context, id uuid.UUID) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *repository) GetOrderWithItems(ctx context.Context, id uuid.UUID) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *repository) GetOrderByNo(ctx context.Context, orderNo string) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, "order_no = ?", orderNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *repository) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).
		First(&order, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *repository) ListOrders(ctx context.Context, userID uuid.UUID, filter *OrderFilter, pagination *Pagination) ([]*Order, int64, error) {
	var orders []*Order
	var total int64

	query := r.db.WithContext(ctx).Model(&Order{}).Where("user_id = ?", userID)

	// Apply filters
	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.Type != nil {
			query = query.Where("type = ?", *filter.Type)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if pagination != nil {
		query = query.Offset(pagination.Offset()).Limit(pagination.PageSize)
	}

	// Fetch results with items
	if err := query.Preload("Items").Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *repository) UpdateOrder(ctx context.Context, order *Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *repository) ListPendingExpiredOrders(ctx context.Context) ([]*Order, error) {
	var orders []*Order
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < NOW()", OrderStatusPending).
		Find(&orders).Error
	return orders, err
}

// --- Order Item Operations ---

func (r *repository) CreateOrderItems(ctx context.Context, items []OrderItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// --- Invoice Operations ---

func (r *repository) CreateInvoice(ctx context.Context, invoice *Invoice) error {
	return r.db.WithContext(ctx).Create(invoice).Error
}

func (r *repository) GetInvoice(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	var invoice Invoice
	err := r.db.WithContext(ctx).First(&invoice, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return &invoice, nil
}

func (r *repository) ListInvoices(ctx context.Context, userID uuid.UUID) ([]*Invoice, error) {
	var invoices []*Invoice
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Find(&invoices).Error
	return invoices, err
}

func (r *repository) GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*Invoice, error) {
	var invoice Invoice
	err := r.db.WithContext(ctx).First(&invoice, "order_id = ?", orderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}
	return &invoice, nil
}
