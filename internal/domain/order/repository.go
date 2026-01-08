package order

import (
	"context"

	"github.com/google/uuid"
)

// OrderFilter represents filters for listing orders.
type OrderFilter struct {
	Status   string
	Type     string
	DateFrom string
	DateTo   string
}

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int
	PageSize int
}

// Invoice represents an invoice for an order.
type Invoice struct {
	id        uuid.UUID
	orderID   uuid.UUID
	userID    uuid.UUID
	invoiceNo string
	amount    int64
	currency  string
	pdfURL    string
}

// NewInvoice creates a new Invoice.
func NewInvoice(orderID, userID uuid.UUID, invoiceNo string, amount int64, currency string) *Invoice {
	return &Invoice{
		id:        uuid.New(),
		orderID:   orderID,
		userID:    userID,
		invoiceNo: invoiceNo,
		amount:    amount,
		currency:  currency,
	}
}

// RestoreInvoice recreates an Invoice from persisted data.
func RestoreInvoice(id, orderID, userID uuid.UUID, invoiceNo string, amount int64, currency, pdfURL string) *Invoice {
	return &Invoice{
		id:        id,
		orderID:   orderID,
		userID:    userID,
		invoiceNo: invoiceNo,
		amount:    amount,
		currency:  currency,
		pdfURL:    pdfURL,
	}
}

// ID returns the invoice ID.
func (i *Invoice) ID() uuid.UUID { return i.id }

// OrderID returns the associated order ID.
func (i *Invoice) OrderID() uuid.UUID { return i.orderID }

// UserID returns the user ID.
func (i *Invoice) UserID() uuid.UUID { return i.userID }

// InvoiceNo returns the invoice number.
func (i *Invoice) InvoiceNo() string { return i.invoiceNo }

// Amount returns the invoice amount.
func (i *Invoice) Amount() int64 { return i.amount }

// Currency returns the currency code.
func (i *Invoice) Currency() string { return i.currency }

// PDFURL returns the PDF URL.
func (i *Invoice) PDFURL() string { return i.pdfURL }

// SetPDFURL sets the PDF URL.
func (i *Invoice) SetPDFURL(url string) { i.pdfURL = url }

// Repository defines the interface for order data access.
// This interface is defined in the domain layer (Port) and implemented in infra layer (Adapter).
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
	CreateOrderItems(ctx context.Context, items []*OrderItem) error

	// Invoice operations
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoice(ctx context.Context, id uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, userID uuid.UUID) ([]*Invoice, error)
	GetInvoiceByOrderID(ctx context.Context, orderID uuid.UUID) (*Invoice, error)
}
