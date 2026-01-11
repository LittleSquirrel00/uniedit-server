package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// OrderDatabasePort defines the interface for order database operations.
type OrderDatabasePort interface {
	// Order operations
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	GetByIDWithItems(ctx context.Context, id uuid.UUID) (*model.Order, error)
	GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)
	GetByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Order, error)
	List(ctx context.Context, userID uuid.UUID, filter *model.OrderFilter, page, pageSize int) ([]*model.Order, int64, error)
	Update(ctx context.Context, order *model.Order) error
	ListPendingExpired(ctx context.Context) ([]*model.Order, error)
}

// OrderItemDatabasePort defines the interface for order item database operations.
type OrderItemDatabasePort interface {
	CreateBatch(ctx context.Context, items []*model.OrderItem) error
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.OrderItem, error)
}

// InvoiceDatabasePort defines the interface for invoice database operations.
type InvoiceDatabasePort interface {
	Create(ctx context.Context, invoice *model.Invoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Invoice, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*model.Invoice, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Invoice, error)
}
