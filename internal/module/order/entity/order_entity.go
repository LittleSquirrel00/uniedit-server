package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/order/domain"
)

// OrderEntity is the persistence model for orders.
// It contains GORM tags for database mapping.
type OrderEntity struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderNo               string    `gorm:"uniqueIndex;not null"`
	UserID                uuid.UUID `gorm:"type:uuid;not null;index"`
	Type                  string    `gorm:"not null"`
	Status                string    `gorm:"not null;default:pending"`
	Subtotal              int64     // In cents
	Discount              int64
	Tax                   int64
	Total                 int64
	Currency              string     `gorm:"default:usd"`
	PlanID                *string
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
	Items []OrderItemEntity `gorm:"foreignKey:OrderID"`
}

// TableName returns the database table name.
func (OrderEntity) TableName() string {
	return "orders"
}

// ToDomain converts the entity to a domain Order.
func (e *OrderEntity) ToDomain() *domain.Order {
	items := make([]*domain.OrderItem, len(e.Items))
	for i, item := range e.Items {
		items[i] = item.ToDomain()
	}

	return domain.RestoreOrder(
		e.ID,
		e.OrderNo,
		e.UserID,
		domain.OrderType(e.Type),
		domain.OrderStatus(e.Status),
		e.Subtotal,
		e.Discount,
		e.Tax,
		e.Total,
		e.Currency,
		e.PlanID,
		e.CreditsAmount,
		e.StripePaymentIntentID,
		e.StripeInvoiceID,
		e.PaidAt,
		e.CanceledAt,
		e.RefundedAt,
		e.ExpiresAt,
		e.CreatedAt,
		e.UpdatedAt,
		items,
	)
}

// FromDomain creates an OrderEntity from a domain Order.
func FromDomain(o *domain.Order) *OrderEntity {
	items := make([]OrderItemEntity, len(o.Items()))
	for i, item := range o.Items() {
		items[i] = *OrderItemFromDomain(item)
	}

	return &OrderEntity{
		ID:                    o.ID(),
		OrderNo:               o.OrderNo(),
		UserID:                o.UserID(),
		Type:                  string(o.Type()),
		Status:                string(o.Status()),
		Subtotal:              o.Subtotal().Amount(),
		Discount:              o.Discount().Amount(),
		Tax:                   o.Tax().Amount(),
		Total:                 o.Total().Amount(),
		Currency:              o.Currency(),
		PlanID:                o.PlanID(),
		CreditsAmount:         o.CreditsAmount(),
		StripePaymentIntentID: o.StripePaymentIntentID(),
		StripeInvoiceID:       o.StripeInvoiceID(),
		PaidAt:                o.PaidAt(),
		CanceledAt:            o.CanceledAt(),
		RefundedAt:            o.RefundedAt(),
		ExpiresAt:             o.ExpiresAt(),
		CreatedAt:             o.CreatedAt(),
		UpdatedAt:             o.UpdatedAt(),
		Items:                 items,
	}
}

// OrderItemEntity is the persistence model for order items.
type OrderItemEntity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null"`
	Description string    `gorm:"not null"`
	Quantity    int       `gorm:"default:1"`
	UnitPrice   int64     // In cents
	Amount      int64     // quantity * unit_price
}

// TableName returns the database table name.
func (OrderItemEntity) TableName() string {
	return "order_items"
}

// ToDomain converts the entity to a domain OrderItem.
func (e *OrderItemEntity) ToDomain() *domain.OrderItem {
	return domain.RestoreOrderItem(
		e.ID,
		e.OrderID,
		e.Description,
		e.Quantity,
		e.UnitPrice,
		"usd", // Default currency, could be stored if needed
	)
}

// OrderItemFromDomain creates an OrderItemEntity from a domain OrderItem.
func OrderItemFromDomain(item *domain.OrderItem) *OrderItemEntity {
	return &OrderItemEntity{
		ID:          item.ID(),
		OrderID:     item.OrderID(),
		Description: item.Description(),
		Quantity:    item.Quantity(),
		UnitPrice:   item.UnitPrice().Amount(),
		Amount:      item.Amount().Amount(),
	}
}

// InvoiceEntity is the persistence model for invoices.
type InvoiceEntity struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	InvoiceNo       string     `gorm:"uniqueIndex;not null"`
	OrderID         uuid.UUID  `gorm:"type:uuid;not null"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null"`
	Amount          int64
	Currency        string     `gorm:"default:usd"`
	Status          string     `gorm:"not null;default:draft"` // draft, finalized, paid, void
	PDFURL          string
	StripeInvoiceID string
	IssuedAt        time.Time
	DueAt           time.Time
	PaidAt          *time.Time
	CreatedAt       time.Time
}

// TableName returns the database table name.
func (InvoiceEntity) TableName() string {
	return "invoices"
}
