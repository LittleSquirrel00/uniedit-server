package model

import (
	"fmt"
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

// String returns the string representation of the status.
func (s OrderStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid order status.
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusPaid, OrderStatusCanceled, OrderStatusRefunded, OrderStatusFailed:
		return true
	}
	return false
}

// IsTerminal returns true if the status is a terminal state.
func (s OrderStatus) IsTerminal() bool {
	return s == OrderStatusCanceled || s == OrderStatusRefunded
}

// CanTransitionTo checks if a transition from the current status to target is valid.
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	allowed := orderTransitions[s]
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

// orderTransitions defines valid state transitions.
var orderTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending:  {OrderStatusPaid, OrderStatusCanceled, OrderStatusFailed},
	OrderStatusPaid:     {OrderStatusRefunded},
	OrderStatusCanceled: {}, // Terminal state
	OrderStatusRefunded: {}, // Terminal state
	OrderStatusFailed:   {OrderStatusPending}, // Can retry
}

// OrderType represents the type of order.
type OrderType string

const (
	OrderTypeSubscription OrderType = "subscription"
	OrderTypeTopup        OrderType = "topup"
	OrderTypeUpgrade      OrderType = "upgrade"
)

// String returns the string representation of the order type.
func (t OrderType) String() string {
	return string(t)
}

// IsValid checks if the type is a valid order type.
func (t OrderType) IsValid() bool {
	switch t {
	case OrderTypeSubscription, OrderTypeTopup, OrderTypeUpgrade:
		return true
	}
	return false
}

// Money represents a monetary value with currency.
// It is an immutable value object.
type Money struct {
	amount   int64  // Amount in smallest currency unit (e.g., cents)
	currency string // ISO 4217 currency code (e.g., "usd")
}

// NewMoney creates a new Money value object.
func NewMoney(amount int64, currency string) Money {
	if currency == "" {
		currency = "usd"
	}
	return Money{amount: amount, currency: currency}
}

// Amount returns the amount in smallest currency unit.
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the ISO 4217 currency code.
func (m Money) Currency() string {
	return m.currency
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// Add returns a new Money with the sum of two Money values.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.currency, other.currency)
	}
	return NewMoney(m.amount+other.amount, m.currency), nil
}

// Subtract returns a new Money with the difference of two Money values.
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.currency, other.currency)
	}
	return NewMoney(m.amount-other.amount, m.currency), nil
}

// Multiply returns a new Money with the amount multiplied by the factor.
func (m Money) Multiply(factor int) Money {
	return NewMoney(m.amount*int64(factor), m.currency)
}

// Order represents a purchase order.
type Order struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderNo   string      `gorm:"uniqueIndex;not null"`
	UserID    uuid.UUID   `gorm:"type:uuid;not null;index"`
	Type      OrderType   `gorm:"not null"`
	Status    OrderStatus `gorm:"not null;default:pending"`
	Subtotal  int64       // In cents
	Discount  int64
	Tax       int64
	Total     int64
	Currency  string `gorm:"default:usd"`
	PlanID    *string
	CreditsAmount         int64
	StripePaymentIntentID string
	StripeInvoiceID       string
	PaidAt                *time.Time
	CanceledAt            *time.Time
	RefundedAt            *time.Time
	ExpiresAt             *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time

	// Relations
	Items []*OrderItem `gorm:"foreignKey:OrderID"`
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
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null"`
	Description string    `gorm:"not null"`
	Quantity    int       `gorm:"default:1"`
	UnitPrice   int64     // In cents
	Amount      int64     // quantity * unit_price
}

// TableName returns the database table name.
func (OrderItem) TableName() string {
	return "order_items"
}

// Invoice represents an invoice for a paid order.
type Invoice struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	InvoiceNo       string     `gorm:"uniqueIndex;not null"`
	OrderID         uuid.UUID  `gorm:"type:uuid;not null"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null"`
	Amount          int64
	Currency        string `gorm:"default:usd"`
	Status          string `gorm:"not null;default:draft"` // draft, finalized, paid, void
	PDFURL          string
	StripeInvoiceID string
	IssuedAt        time.Time
	DueAt           time.Time
	PaidAt          *time.Time
	CreatedAt       time.Time
}

// TableName returns the database table name.
func (Invoice) TableName() string {
	return "invoices"
}

// OrderResponse represents an order in API responses.
type OrderResponse struct {
	ID            uuid.UUID            `json:"id"`
	OrderNo       string               `json:"order_no"`
	Type          OrderType            `json:"type"`
	Status        OrderStatus          `json:"status"`
	Subtotal      int64                `json:"subtotal"`
	Discount      int64                `json:"discount"`
	Tax           int64                `json:"tax"`
	Total         int64                `json:"total"`
	Currency      string               `json:"currency"`
	PlanID        *string              `json:"plan_id,omitempty"`
	CreditsAmount int64                `json:"credits_amount,omitempty"`
	PaidAt        *time.Time           `json:"paid_at,omitempty"`
	CanceledAt    *time.Time           `json:"canceled_at,omitempty"`
	RefundedAt    *time.Time           `json:"refunded_at,omitempty"`
	ExpiresAt     *time.Time           `json:"expires_at,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	Items         []*OrderItemResponse `json:"items,omitempty"`
}

// OrderItemResponse represents an order item in API responses.
type OrderItemResponse struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	Amount      int64     `json:"amount"`
}

// ToResponse converts an Order to OrderResponse.
func (o *Order) ToResponse() *OrderResponse {
	resp := &OrderResponse{
		ID:            o.ID,
		OrderNo:       o.OrderNo,
		Type:          o.Type,
		Status:        o.Status,
		Subtotal:      o.Subtotal,
		Discount:      o.Discount,
		Tax:           o.Tax,
		Total:         o.Total,
		Currency:      o.Currency,
		PlanID:        o.PlanID,
		CreditsAmount: o.CreditsAmount,
		PaidAt:        o.PaidAt,
		CanceledAt:    o.CanceledAt,
		RefundedAt:    o.RefundedAt,
		ExpiresAt:     o.ExpiresAt,
		CreatedAt:     o.CreatedAt,
		Items:         make([]*OrderItemResponse, len(o.Items)),
	}
	for i, item := range o.Items {
		resp.Items[i] = item.ToResponse()
	}
	return resp
}

// ToResponse converts an OrderItem to OrderItemResponse.
func (i *OrderItem) ToResponse() *OrderItemResponse {
	return &OrderItemResponse{
		ID:          i.ID,
		Description: i.Description,
		Quantity:    i.Quantity,
		UnitPrice:   i.UnitPrice,
		Amount:      i.Amount,
	}
}

// InvoiceResponse represents an invoice in API responses.
type InvoiceResponse struct {
	ID        uuid.UUID  `json:"id"`
	InvoiceNo string     `json:"invoice_no"`
	OrderID   uuid.UUID  `json:"order_id"`
	Amount    int64      `json:"amount"`
	Currency  string     `json:"currency"`
	Status    string     `json:"status"`
	PDFURL    string     `json:"pdf_url,omitempty"`
	IssuedAt  time.Time  `json:"issued_at"`
	DueAt     time.Time  `json:"due_at"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
}

// ToResponse converts an Invoice to InvoiceResponse.
func (i *Invoice) ToResponse() *InvoiceResponse {
	return &InvoiceResponse{
		ID:        i.ID,
		InvoiceNo: i.InvoiceNo,
		OrderID:   i.OrderID,
		Amount:    i.Amount,
		Currency:  i.Currency,
		Status:    i.Status,
		PDFURL:    i.PDFURL,
		IssuedAt:  i.IssuedAt,
		DueAt:     i.DueAt,
		PaidAt:    i.PaidAt,
	}
}

// OrderFilter represents filters for listing orders.
type OrderFilter struct {
	Status *OrderStatus
	Type   *OrderType
}

// OrderListResponse represents a paginated list of orders.
type OrderListResponse struct {
	Orders     []*OrderResponse `json:"orders"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}
