package payment

import (
	"context"

	"github.com/google/uuid"
)

// Order type constants - mirrors order module constants.
const (
	OrderTypeTopup        = "topup"
	OrderTypeSubscription = "subscription"
	OrderTypeUpgrade      = "upgrade"
)

// OrderReader defines the interface for reading order information.
// This interface is defined in the payment module (consumer) following
// the Dependency Inversion Principle.
type OrderReader interface {
	// GetOrder returns order information by ID.
	GetOrder(ctx context.Context, id uuid.UUID) (*OrderInfo, error)

	// GetOrderByPaymentIntentID returns order information by Stripe PaymentIntent ID.
	GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*OrderInfo, error)

	// UpdateOrderStatus updates the order status.
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error

	// SetStripePaymentIntentID sets the Stripe PaymentIntent ID on an order.
	SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error
}

// OrderInfo is a slim view of order data needed by the payment module.
// It contains only the fields that payment processing requires.
type OrderInfo struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Type          string // "subscription", "topup", "upgrade"
	Status        string // "pending", "paid", "canceled", etc.
	Total         int64
	Currency      string
	CreditsAmount int64  // For topup orders
	PlanID        string // For subscription orders
}

// IsPending returns true if the order is pending payment.
func (o *OrderInfo) IsPending() bool {
	return o.Status == "pending"
}

// BillingReader defines the interface for reading billing information.
// Used by payment module to access subscription/customer data.
type BillingReader interface {
	// GetSubscription returns the user's subscription information.
	GetSubscription(ctx context.Context, userID uuid.UUID) (*SubscriptionInfo, error)

	// AddCredits adds credits to a user's balance.
	AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
}

// SubscriptionInfo is a slim view of subscription data.
type SubscriptionInfo struct {
	UserID           uuid.UUID
	PlanID           string
	Status           string
	StripeCustomerID string
}

// EventPublisher defines the interface for publishing domain events.
// This allows the payment module to publish events without depending on
// the concrete events package.
type EventPublisher interface {
	// Publish publishes a domain event to all registered handlers.
	Publish(event interface{})
}
