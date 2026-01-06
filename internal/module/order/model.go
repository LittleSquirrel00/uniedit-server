package order

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the status of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusCanceled  OrderStatus = "canceled"
	OrderStatusRefunded  OrderStatus = "refunded"
	OrderStatusFailed    OrderStatus = "failed"
)

// OrderType represents the type of order.
type OrderType string

const (
	OrderTypeSubscription OrderType = "subscription"
	OrderTypeTopup        OrderType = "topup"
	OrderTypeUpgrade      OrderType = "upgrade"
)

// Order represents a purchase order.
type Order struct {
	ID                    uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderNo               string      `json:"order_no" gorm:"uniqueIndex;not null"`
	UserID                uuid.UUID   `json:"user_id" gorm:"type:uuid;not null;index"`
	Type                  OrderType   `json:"type" gorm:"not null"`
	Status                OrderStatus `json:"status" gorm:"not null;default:pending"`
	Subtotal              int64       `json:"subtotal"`          // In cents
	Discount              int64       `json:"discount"`
	Tax                   int64       `json:"tax"`
	Total                 int64       `json:"total"`
	Currency              string      `json:"currency" gorm:"default:usd"`
	PlanID                *string     `json:"plan_id,omitempty"`
	CreditsAmount         int64       `json:"credits_amount,omitempty"`
	StripePaymentIntentID string      `json:"-"`
	StripeInvoiceID       string      `json:"-"`
	PaidAt                *time.Time  `json:"paid_at,omitempty"`
	CanceledAt            *time.Time  `json:"canceled_at,omitempty"`
	RefundedAt            *time.Time  `json:"refunded_at,omitempty"`
	ExpiresAt             *time.Time  `json:"expires_at,omitempty"`
	CreatedAt             time.Time   `json:"created_at"`
	UpdatedAt             time.Time   `json:"updated_at"`

	// Relations
	Items []OrderItem `json:"items,omitempty" gorm:"foreignKey:OrderID"`
}

// TableName returns the database table name.
func (Order) TableName() string {
	return "orders"
}

// IsPending returns true if the order is pending payment.
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsPaid returns true if the order has been paid.
func (o *Order) IsPaid() bool {
	return o.Status == OrderStatusPaid
}

// IsExpired returns true if the order has expired.
func (o *Order) IsExpired() bool {
	return o.ExpiresAt != nil && time.Now().After(*o.ExpiresAt)
}

// OrderItem represents a line item in an order.
type OrderItem struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     uuid.UUID `json:"order_id" gorm:"type:uuid;not null"`
	Description string    `json:"description" gorm:"not null"`
	Quantity    int       `json:"quantity" gorm:"default:1"`
	UnitPrice   int64     `json:"unit_price"` // In cents
	Amount      int64     `json:"amount"`     // quantity * unit_price
}

// TableName returns the database table name.
func (OrderItem) TableName() string {
	return "order_items"
}

// Invoice represents an invoice for a paid order.
type Invoice struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	InvoiceNo       string     `json:"invoice_no" gorm:"uniqueIndex;not null"`
	OrderID         uuid.UUID  `json:"order_id" gorm:"type:uuid;not null"`
	UserID          uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	Amount          int64      `json:"amount"`
	Currency        string     `json:"currency" gorm:"default:usd"`
	Status          string     `json:"status" gorm:"not null;default:draft"` // draft, finalized, paid, void
	PDFURL          string     `json:"pdf_url,omitempty"`
	StripeInvoiceID string     `json:"-"`
	IssuedAt        time.Time  `json:"issued_at"`
	DueAt           time.Time  `json:"due_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// TableName returns the database table name.
func (Invoice) TableName() string {
	return "invoices"
}
