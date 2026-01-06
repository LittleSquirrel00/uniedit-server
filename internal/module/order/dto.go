package order

import (
	"time"

	"github.com/google/uuid"
)

// CreateSubscriptionOrderRequest represents a request to create a subscription order.
type CreateSubscriptionOrderRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CreateTopupOrderRequest represents a request to create a top-up order.
type CreateTopupOrderRequest struct {
	Amount int64 `json:"amount" binding:"required,min=100"` // Minimum $1.00 (100 cents)
}

// OrderFilter represents filters for listing orders.
type OrderFilter struct {
	Status *OrderStatus `form:"status"`
	Type   *OrderType   `form:"type"`
}

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
}

// NewPagination creates pagination with defaults.
func NewPagination() *Pagination {
	return &Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// Offset returns the offset for database queries.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// OrderResponse represents an order in API responses.
type OrderResponse struct {
	ID            uuid.UUID         `json:"id"`
	OrderNo       string            `json:"order_no"`
	Type          OrderType         `json:"type"`
	Status        OrderStatus       `json:"status"`
	Subtotal      int64             `json:"subtotal"`
	Discount      int64             `json:"discount"`
	Tax           int64             `json:"tax"`
	Total         int64             `json:"total"`
	Currency      string            `json:"currency"`
	PlanID        *string           `json:"plan_id,omitempty"`
	CreditsAmount int64             `json:"credits_amount,omitempty"`
	PaidAt        *time.Time        `json:"paid_at,omitempty"`
	CanceledAt    *time.Time        `json:"canceled_at,omitempty"`
	RefundedAt    *time.Time        `json:"refunded_at,omitempty"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	Items         []OrderItemResponse `json:"items,omitempty"`
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
		Items:         make([]OrderItemResponse, len(o.Items)),
	}
	for i, item := range o.Items {
		resp.Items[i] = OrderItemResponse{
			ID:          item.ID,
			Description: item.Description,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Amount:      item.Amount,
		}
	}
	return resp
}

// OrderListResponse represents a paginated list of orders.
type OrderListResponse struct {
	Orders     []*OrderResponse `json:"orders"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
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

// PaymentIntentResponse represents a Stripe PaymentIntent for frontend.
type PaymentIntentResponse struct {
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
}
