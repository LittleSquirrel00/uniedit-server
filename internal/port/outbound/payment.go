package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/model"
)

// PaymentDatabasePort defines payment persistence operations.
type PaymentDatabasePort interface {
	// Create creates a new payment record.
	Create(ctx context.Context, payment *model.Payment) error

	// FindByID finds a payment by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.Payment, error)

	// FindByPaymentIntentID finds a payment by Stripe PaymentIntent ID.
	FindByPaymentIntentID(ctx context.Context, paymentIntentID string) (*model.Payment, error)

	// FindByTradeNo finds a payment by provider trade number.
	FindByTradeNo(ctx context.Context, tradeNo string) (*model.Payment, error)

	// FindByFilter finds payments by filter.
	FindByFilter(ctx context.Context, filter model.PaymentFilter) ([]*model.Payment, int64, error)

	// FindByOrderID finds payments by order ID.
	FindByOrderID(ctx context.Context, orderID uuid.UUID) ([]*model.Payment, error)

	// Update updates a payment record.
	Update(ctx context.Context, payment *model.Payment) error
}

// WebhookEventDatabasePort defines webhook event persistence operations.
type WebhookEventDatabasePort interface {
	// Create creates a new webhook event record.
	Create(ctx context.Context, event *model.WebhookEvent) error

	// Exists checks if a webhook event exists by provider and event ID.
	Exists(ctx context.Context, provider, eventID string) (bool, error)

	// MarkProcessed marks a webhook event as processed.
	MarkProcessed(ctx context.Context, id uuid.UUID, processErr error) error
}

// PaymentProviderPort defines the interface for payment providers (Stripe-style).
type PaymentProviderPort interface {
	// Name returns the provider name.
	Name() string

	// CreateCustomer creates a customer in the provider.
	CreateCustomer(ctx context.Context, email, name string) (*model.ProviderCustomer, error)

	// GetCustomer gets a customer from the provider.
	GetCustomer(ctx context.Context, customerID string) (*model.ProviderCustomer, error)

	// CreatePaymentIntent creates a payment intent.
	CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID string, metadata map[string]string) (*model.ProviderPaymentIntent, error)

	// GetPaymentIntent gets a payment intent.
	GetPaymentIntent(ctx context.Context, paymentIntentID string) (*model.ProviderPaymentIntent, error)

	// CancelPaymentIntent cancels a payment intent.
	CancelPaymentIntent(ctx context.Context, paymentIntentID string) error

	// CreateRefund creates a refund.
	CreateRefund(ctx context.Context, chargeID string, amount int64, reason string) (*model.ProviderRefund, error)

	// ListPaymentMethods lists payment methods for a customer.
	ListPaymentMethods(ctx context.Context, customerID string) ([]*model.PaymentMethodInfo, error)

	// VerifyWebhookSignature verifies a webhook signature.
	VerifyWebhookSignature(payload []byte, signature string) error
}

// NativePaymentProviderPort extends PaymentProviderPort for native payment flows (Alipay/WeChat).
type NativePaymentProviderPort interface {
	PaymentProviderPort

	// CreateNativePayment creates a payment order for native payment flow.
	CreateNativePayment(ctx context.Context, scene model.PaymentScene, orderID string, amount int64, subject, description, notifyURL, returnURL string, metadata map[string]string) (*model.ProviderNativeOrder, error)

	// QueryPayment queries the payment status from provider.
	QueryPayment(ctx context.Context, orderID, tradeNo string) (*model.ProviderNotifyResult, error)

	// ClosePayment closes/cancels an unpaid order.
	ClosePayment(ctx context.Context, orderID, tradeNo string) error

	// RefundPayment creates a refund for a paid order.
	RefundPayment(ctx context.Context, orderID, tradeNo, refundID string, refundAmount, totalAmount int64, reason string) (*model.ProviderRefund, error)

	// ParseNotify parses and verifies the async notification from provider.
	ParseNotify(ctx context.Context, body []byte, headers map[string]string) (*model.ProviderNotifyResult, error)
}

// PaymentProviderRegistryPort defines the payment provider registry interface.
type PaymentProviderRegistryPort interface {
	// Get returns a payment provider by name.
	Get(name string) (PaymentProviderPort, error)

	// GetNative returns a native payment provider by name.
	GetNative(name string) (NativePaymentProviderPort, error)

	// GetNativeByMethod returns a native payment provider by payment method.
	GetNativeByMethod(method string) (NativePaymentProviderPort, error)

	// Register registers a payment provider.
	Register(provider PaymentProviderPort)

	// RegisterNative registers a native payment provider.
	RegisterNative(provider NativePaymentProviderPort)
}

// OrderReaderPort defines the interface for reading order information.
// This port allows the payment domain to access order data without
// depending on the order domain directly.
type OrderReaderPort interface {
	// GetOrder returns order information by ID.
	GetOrder(ctx context.Context, id uuid.UUID) (*PaymentOrderInfo, error)

	// GetOrderByPaymentIntentID returns order information by Stripe PaymentIntent ID.
	GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*PaymentOrderInfo, error)

	// UpdateOrderStatus updates the order status.
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error

	// SetStripePaymentIntentID sets the Stripe PaymentIntent ID on an order.
	SetStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, paymentIntentID string) error
}

// PaymentOrderInfo is a slim view of order data needed by the payment domain.
type PaymentOrderInfo struct {
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
func (o *PaymentOrderInfo) IsPending() bool {
	return o.Status == "pending"
}

// BillingReaderPort defines the interface for reading billing information.
// Used by payment domain to access subscription/customer data.
type BillingReaderPort interface {
	// GetSubscription returns the user's subscription information.
	GetSubscription(ctx context.Context, userID uuid.UUID) (*PaymentSubscriptionInfo, error)

	// AddCredits adds credits to a user's balance.
	AddCredits(ctx context.Context, userID uuid.UUID, amount int64, source string) error
}

// PaymentSubscriptionInfo is a slim view of subscription data.
type PaymentSubscriptionInfo struct {
	UserID           uuid.UUID
	PlanID           string
	Status           string
	StripeCustomerID string
}
