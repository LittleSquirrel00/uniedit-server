package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/domain/order"
	"github.com/uniedit/server/internal/infra/persistence/entity"
	"gorm.io/gorm"
)

// OrderRepository implements order.Repository interface.
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository.
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// --- Order Operations ---

func (r *OrderRepository) CreateOrder(ctx context.Context, o *order.Order) error {
	ent := entity.FromDomainOrder(o)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *OrderRepository) GetOrderWithItems(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order with items: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *OrderRepository) GetOrderByNo(ctx context.Context, orderNo string) (*order.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&ent, "order_no = ?", orderNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by no: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *OrderRepository) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*order.Order, error) {
	var ent entity.OrderEntity
	err := r.db.WithContext(ctx).
		First(&ent, "stripe_payment_intent_id = ?", paymentIntentID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by payment intent: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *OrderRepository) ListOrders(ctx context.Context, userID uuid.UUID, filter *order.OrderFilter, pagination *order.Pagination) ([]*order.Order, int64, error) {
	var entities []*entity.OrderEntity
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.OrderEntity{}).Where("user_id = ?", userID)

	// Apply filters
	if filter != nil {
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	// Apply pagination
	if pagination != nil {
		offset := (pagination.Page - 1) * pagination.PageSize
		query = query.Offset(offset).Limit(pagination.PageSize)
	}

	// Fetch results with items
	if err := query.Preload("Items").Order("created_at DESC").Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	// Convert to domain objects
	orders := make([]*order.Order, len(entities))
	for i, ent := range entities {
		orders[i] = ent.ToDomain()
	}

	return orders, total, nil
}

func (r *OrderRepository) UpdateOrder(ctx context.Context, o *order.Order) error {
	ent := entity.FromDomainOrder(o)
	if err := r.db.WithContext(ctx).Save(ent).Error; err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return nil
}

func (r *OrderRepository) ListPendingExpiredOrders(ctx context.Context) ([]*order.Order, error) {
	var entities []*entity.OrderEntity
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < NOW()", string(order.StatusPending)).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list pending expired orders: %w", err)
	}

	orders := make([]*order.Order, len(entities))
	for i, ent := range entities {
		orders[i] = ent.ToDomain()
	}
	return orders, nil
}

// --- Order Item Operations ---

func (r *OrderRepository) CreateOrderItems(ctx context.Context, items []*order.OrderItem) error {
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

func (r *OrderRepository) CreateInvoice(ctx context.Context, invoice *order.Invoice) error {
	ent := entity.FromDomainInvoice(invoice)
	if err := r.db.WithContext(ctx).Create(ent).Error; err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetInvoice(ctx context.Context, id uuid.UUID) (*order.Invoice, error) {
	var ent entity.InvoiceEntity
	err := r.db.WithContext(ctx).First(&ent, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	return ent.ToDomain(), nil
}

func (r *OrderRepository) ListInvoices(ctx context.Context, userID uuid.UUID) ([]*order.Invoice, error) {
	var entities []*entity.InvoiceEntity
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("issued_at DESC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	invoices := make([]*order.Invoice, len(entities))
	for i, ent := range entities {
		invoices[i] = ent.ToDomain()
	}
	return invoices, nil
}

func (r *OrderRepository) GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*order.Invoice, error) {
	var ent entity.InvoiceEntity
	err := r.db.WithContext(ctx).First(&ent, "order_id = ?", orderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by order: %w", err)
	}
	return ent.ToDomain(), nil
}

// Ensure OrderRepository implements order.Repository.
var _ order.Repository = (*OrderRepository)(nil)
