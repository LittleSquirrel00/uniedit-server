package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrderItem represents a line item in an order.
// It is an entity within the Order aggregate.
type OrderItem struct {
	id          uuid.UUID
	orderID     uuid.UUID
	description string
	quantity    int
	unitPrice   Money
}

// NewOrderItem creates a new order item.
func NewOrderItem(description string, quantity int, unitPrice Money) (*OrderItem, error) {
	if description == "" {
		return nil, fmt.Errorf("description cannot be empty")
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	return &OrderItem{
		id:          uuid.New(),
		description: description,
		quantity:    quantity,
		unitPrice:   unitPrice,
	}, nil
}

// RestoreOrderItem recreates an OrderItem from persisted data.
// This bypasses validation for hydration from database.
func RestoreOrderItem(id, orderID uuid.UUID, description string, quantity int, unitPriceAmount int64, currency string) *OrderItem {
	return &OrderItem{
		id:          id,
		orderID:     orderID,
		description: description,
		quantity:    quantity,
		unitPrice:   NewMoney(unitPriceAmount, currency),
	}
}

// ID returns the item ID.
func (i *OrderItem) ID() uuid.UUID {
	return i.id
}

// OrderID returns the order ID this item belongs to.
func (i *OrderItem) OrderID() uuid.UUID {
	return i.orderID
}

// Description returns the item description.
func (i *OrderItem) Description() string {
	return i.description
}

// Quantity returns the quantity.
func (i *OrderItem) Quantity() int {
	return i.quantity
}

// UnitPrice returns the unit price.
func (i *OrderItem) UnitPrice() Money {
	return i.unitPrice
}

// Amount returns the total amount for this item (quantity * unit_price).
func (i *OrderItem) Amount() Money {
	return i.unitPrice.Multiply(i.quantity)
}

// setOrderID sets the order ID (called when adding to an order).
func (i *OrderItem) setOrderID(orderID uuid.UUID) {
	i.orderID = orderID
}

// Order is the aggregate root for purchase orders.
// It encapsulates all business rules related to orders.
type Order struct {
	id        uuid.UUID
	orderNo   string
	userID    uuid.UUID
	orderType OrderType
	status    OrderStatus

	subtotal Money
	discount Money
	tax      Money
	total    Money

	planID        *string
	creditsAmount int64

	stripePaymentIntentID string
	stripeInvoiceID       string

	paidAt     *time.Time
	canceledAt *time.Time
	refundedAt *time.Time
	expiresAt  *time.Time
	createdAt  time.Time
	updatedAt  time.Time

	items []*OrderItem
}

// NewOrder creates a new Order with the given parameters.
func NewOrder(userID uuid.UUID, orderNo string, orderType OrderType, currency string) (*Order, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	if orderNo == "" {
		return nil, fmt.Errorf("order number cannot be empty")
	}
	if !orderType.IsValid() {
		return nil, fmt.Errorf("invalid order type: %s", orderType)
	}
	if currency == "" {
		currency = "usd"
	}

	now := time.Now()
	return &Order{
		id:        uuid.New(),
		orderNo:   orderNo,
		userID:    userID,
		orderType: orderType,
		status:    StatusPending,
		subtotal:  NewMoney(0, currency),
		discount:  NewMoney(0, currency),
		tax:       NewMoney(0, currency),
		total:     NewMoney(0, currency),
		createdAt: now,
		updatedAt: now,
		items:     make([]*OrderItem, 0),
	}, nil
}

// RestoreOrder recreates an Order from persisted data.
// This bypasses validation for hydration from database.
func RestoreOrder(
	id uuid.UUID,
	orderNo string,
	userID uuid.UUID,
	orderType OrderType,
	status OrderStatus,
	subtotal, discount, tax, total int64,
	currency string,
	planID *string,
	creditsAmount int64,
	stripePaymentIntentID, stripeInvoiceID string,
	paidAt, canceledAt, refundedAt, expiresAt *time.Time,
	createdAt, updatedAt time.Time,
	items []*OrderItem,
) *Order {
	return &Order{
		id:                    id,
		orderNo:               orderNo,
		userID:                userID,
		orderType:             orderType,
		status:                status,
		subtotal:              NewMoney(subtotal, currency),
		discount:              NewMoney(discount, currency),
		tax:                   NewMoney(tax, currency),
		total:                 NewMoney(total, currency),
		planID:                planID,
		creditsAmount:         creditsAmount,
		stripePaymentIntentID: stripePaymentIntentID,
		stripeInvoiceID:       stripeInvoiceID,
		paidAt:                paidAt,
		canceledAt:            canceledAt,
		refundedAt:            refundedAt,
		expiresAt:             expiresAt,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
		items:                 items,
	}
}

// ID returns the order ID.
func (o *Order) ID() uuid.UUID {
	return o.id
}

// OrderNo returns the order number.
func (o *Order) OrderNo() string {
	return o.orderNo
}

// UserID returns the user ID.
func (o *Order) UserID() uuid.UUID {
	return o.userID
}

// Type returns the order type.
func (o *Order) Type() OrderType {
	return o.orderType
}

// Status returns the current order status.
func (o *Order) Status() OrderStatus {
	return o.status
}

// Subtotal returns the subtotal.
func (o *Order) Subtotal() Money {
	return o.subtotal
}

// Discount returns the discount.
func (o *Order) Discount() Money {
	return o.discount
}

// Tax returns the tax.
func (o *Order) Tax() Money {
	return o.tax
}

// Total returns the total.
func (o *Order) Total() Money {
	return o.total
}

// Currency returns the currency code.
func (o *Order) Currency() string {
	return o.total.Currency()
}

// PlanID returns the plan ID if this is a subscription order.
func (o *Order) PlanID() *string {
	return o.planID
}

// CreditsAmount returns the credits amount for topup orders.
func (o *Order) CreditsAmount() int64 {
	return o.creditsAmount
}

// StripePaymentIntentID returns the Stripe PaymentIntent ID.
func (o *Order) StripePaymentIntentID() string {
	return o.stripePaymentIntentID
}

// StripeInvoiceID returns the Stripe Invoice ID.
func (o *Order) StripeInvoiceID() string {
	return o.stripeInvoiceID
}

// PaidAt returns when the order was paid.
func (o *Order) PaidAt() *time.Time {
	return o.paidAt
}

// CanceledAt returns when the order was canceled.
func (o *Order) CanceledAt() *time.Time {
	return o.canceledAt
}

// RefundedAt returns when the order was refunded.
func (o *Order) RefundedAt() *time.Time {
	return o.refundedAt
}

// ExpiresAt returns when the order expires.
func (o *Order) ExpiresAt() *time.Time {
	return o.expiresAt
}

// CreatedAt returns when the order was created.
func (o *Order) CreatedAt() time.Time {
	return o.createdAt
}

// UpdatedAt returns when the order was last updated.
func (o *Order) UpdatedAt() time.Time {
	return o.updatedAt
}

// Items returns the order items.
func (o *Order) Items() []*OrderItem {
	result := make([]*OrderItem, len(o.items))
	copy(result, o.items)
	return result
}

// IsPending returns true if the order is pending payment.
func (o *Order) IsPending() bool {
	return o.status == StatusPending
}

// IsPaid returns true if the order has been paid.
func (o *Order) IsPaid() bool {
	return o.status == StatusPaid
}

// IsExpired returns true if the order has expired.
func (o *Order) IsExpired() bool {
	return o.expiresAt != nil && time.Now().After(*o.expiresAt)
}

// AddItem adds an item to the order.
func (o *Order) AddItem(item *OrderItem) error {
	if !o.IsPending() {
		return fmt.Errorf("cannot add items to non-pending order")
	}

	item.setOrderID(o.id)
	o.items = append(o.items, item)
	o.recalculateTotals()
	o.updatedAt = time.Now()
	return nil
}

// MarkAsPaid transitions the order to paid status.
func (o *Order) MarkAsPaid() error {
	if !o.status.CanTransitionTo(StatusPaid) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, o.status, StatusPaid)
	}
	o.status = StatusPaid
	now := time.Now()
	o.paidAt = &now
	o.updatedAt = now
	return nil
}

// Cancel transitions the order to canceled status.
func (o *Order) Cancel() error {
	if !o.status.CanTransitionTo(StatusCanceled) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, o.status, StatusCanceled)
	}
	o.status = StatusCanceled
	now := time.Now()
	o.canceledAt = &now
	o.updatedAt = now
	return nil
}

// MarkAsFailed transitions the order to failed status.
func (o *Order) MarkAsFailed() error {
	if !o.status.CanTransitionTo(StatusFailed) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, o.status, StatusFailed)
	}
	o.status = StatusFailed
	o.updatedAt = time.Now()
	return nil
}

// Refund transitions the order to refunded status.
func (o *Order) Refund() error {
	if !o.status.CanTransitionTo(StatusRefunded) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, o.status, StatusRefunded)
	}
	o.status = StatusRefunded
	now := time.Now()
	o.refundedAt = &now
	o.updatedAt = now
	return nil
}

// SetStripePaymentIntentID sets the Stripe PaymentIntent ID.
func (o *Order) SetStripePaymentIntentID(id string) {
	o.stripePaymentIntentID = id
	o.updatedAt = time.Now()
}

// SetStripeInvoiceID sets the Stripe Invoice ID.
func (o *Order) SetStripeInvoiceID(id string) {
	o.stripeInvoiceID = id
	o.updatedAt = time.Now()
}

// SetPlanID sets the plan ID for subscription orders.
func (o *Order) SetPlanID(planID string) {
	o.planID = &planID
	o.updatedAt = time.Now()
}

// SetCreditsAmount sets the credits amount for topup orders.
func (o *Order) SetCreditsAmount(amount int64) {
	o.creditsAmount = amount
	o.updatedAt = time.Now()
}

// SetExpiration sets the order expiration time.
func (o *Order) SetExpiration(expiresAt time.Time) {
	o.expiresAt = &expiresAt
	o.updatedAt = time.Now()
}

// SetTotals sets the order totals.
func (o *Order) SetTotals(subtotal, discount, tax, total int64) {
	currency := o.Currency()
	o.subtotal = NewMoney(subtotal, currency)
	o.discount = NewMoney(discount, currency)
	o.tax = NewMoney(tax, currency)
	o.total = NewMoney(total, currency)
	o.updatedAt = time.Now()
}

// recalculateTotals recalculates the order totals based on items.
func (o *Order) recalculateTotals() {
	var subtotal int64
	for _, item := range o.items {
		subtotal += item.Amount().Amount()
	}
	o.subtotal = NewMoney(subtotal, o.Currency())
	// Total = subtotal - discount + tax
	total := subtotal - o.discount.Amount() + o.tax.Amount()
	o.total = NewMoney(total, o.Currency())
}
