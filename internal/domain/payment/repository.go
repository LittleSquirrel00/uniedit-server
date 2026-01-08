package payment

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for payment data access.
// This interface is defined in the domain layer (Port) and implemented in infra layer (Adapter).
type Repository interface {
	// Payment operations
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPayment(ctx context.Context, id uuid.UUID) (*Payment, error)
	GetPaymentByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Payment, error)
	UpdatePayment(ctx context.Context, payment *Payment) error
	ListPaymentsByOrder(ctx context.Context, orderID uuid.UUID) ([]*Payment, error)

	// Stripe webhook event operations
	CreateWebhookEvent(ctx context.Context, event *StripeWebhookEvent) error
	WebhookEventExists(ctx context.Context, eventID string) (bool, error)
	MarkWebhookEventProcessed(ctx context.Context, eventID string, err error) error

	// Native payment webhook event operations (Alipay/WeChat)
	CreatePaymentWebhookEvent(ctx context.Context, event *PaymentWebhookEvent) error
	MarkPaymentWebhookEventProcessed(ctx context.Context, eventID uuid.UUID, err error) error
}
