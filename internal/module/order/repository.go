package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/order/domain"
	"github.com/uniedit/server/internal/module/order/entity"
	"gorm.io/gorm"
)

// Repository defines the interface for order data access.
type Repository interface {
	// Order operations
	CreateOrder(ctx context.Context, order *domain.Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetOrderWithItems(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*domain.Order, error)
	GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Order, error)
	ListOrders(ctx context.Context, userID uuid.UUID, filter *OrderFilter, pagination *Pagination) ([]*domain.Order, int64, error)
	UpdateOrder(ctx context.Context, order *domain.Order) error
	ListPendingExpiredOrders(ctx context.Context) ([]*domain.Order, error)

	// Order item operations
	CreateOrderItems(ctx context.Context, items []*domain.OrderItem) error

	// Invoice operations
	CreateInvoice(ctx context.Context, invoice *entity.InvoiceEntity) error
	GetInvoice(ctx context.Context, id uuid.UUID) (*entity.InvoiceEntity, error)
	ListInvoices(ctx context.Context, userID uuid.UUID) ([]*entity.InvoiceEntity, error)
	GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*entity.InvoiceEntity, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new order repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Order Operations ---

func (r *repository) CreateOrder(ctx context.Context, order *domain.Order) error {
	ent := entity.FromDomain(order)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *repository) GetOrder(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) GetOrderWithItems(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order with items: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) GetOrderByNo(ctx context.Context, orderNo string) (*domain.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&ent, "order_no = ?", orderNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by no: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).
		First(&ent, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by payment intent: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *repository) ListOrders(ctx context.Context, userID uuid.UUID, filter *OrderFilter, pagination *Pagination) ([]*domain.Order, int64, error) {
	var entities []*entity.OrderEntity
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.OrderEntity{}).Where("user_id = ?", userID)

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
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	// Apply pagination
	if pagination != nil {
		query = query.Offset(pagination.Offset()).Limit(pagination.PageSize)
	}

	// Fetch results with items
	if err := query.Preload("Items").Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	// Convert to domain objects
	orders := make([]*domain.Order, len(entities))
	for i, ent := range entities {
		orders[i] = ent.ToDomain()
	}

	return orders, total, nil
}

func (r *repository) UpdateOrder(ctx context.Context, order *domain.Order) error {
	ent := entity.FromDomain(order)
	if err := r.db.WithContext(ctx).Save(ent).Error; err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return nil
}

func (r *repository) ListPendingExpiredOrders(ctx context.Context) ([]*domain.Order, error) {
	var entities []*entity.OrderEntity
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < NOW()", string(domain.StatusPending)).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list pending expired orders: %w", err)
	}

	orders := make([]*domain.Order, len(entities))
	for i, ent := range entities {
		orders[i] = ent.ToDomain()
	}
	return orders, nil
}

// --- Order Item Operations ---

func (r *repository) CreateOrderItems(ctx context.Context, items []*domain.OrderItem) error {
	if len(items) == 0 {
		return nil
	}

	entities := make([]entity.OrderItemEntity, len(items))
	for i, item := range items {
		entities[i] = *entity.OrderItemFromDomain(item)
	}

	if err := r.db.WithContext(ctx).Create(&entities).Error; err != nil {
		return fmt.Errorf("create order items: %w", err)
	}
	return nil
}

// --- Invoice Operations ---

func (r *repository) CreateInvoice(ctx context.Context, invoice *entity.InvoiceEntity) error {
	if err := r.db.WithContext(ctx).Create(invoice).Error; err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}
	return nil
}

func (r *repository) GetInvoice(ctx context.Context, id uuid.UUID) (*entity.InvoiceEntity, error) {
	var invoice entity.InvoiceEntity
	err := r.db.WithContext(ctx).First(&invoice, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	return &invoice, nil
}

func (r *repository) ListInvoices(ctx context.Context, userID uuid.UUID) ([]*entity.InvoiceEntity, error) {
	var invoices []*entity.InvoiceEntity
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Find(&invoices).Error
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	return invoices, nil
}

func (r *repository) GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*entity.InvoiceEntity, error) {
	var invoice entity.InvoiceEntity
	err := r.db.WithContext(ctx).First(&invoice, "order_id = ?", orderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by order: %w", err)
	}
	return &invoice, nil
}
