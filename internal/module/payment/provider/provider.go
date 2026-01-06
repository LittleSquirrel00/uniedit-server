package provider

import (
	"context"
)

// PaymentScene represents the payment scenario.
type PaymentScene string

const (
	PaymentSceneWeb    PaymentScene = "web"    // Desktop web payment
	PaymentSceneH5     PaymentScene = "h5"     // Mobile web payment
	PaymentSceneApp    PaymentScene = "app"    // Native app payment
	PaymentSceneNative PaymentScene = "native" // QR code / scan payment
	PaymentSceneMini   PaymentScene = "mini"   // Mini program payment
)

// PaymentIntent represents a payment intent from the provider.
type PaymentIntent struct {
	ID           string
	ClientSecret string
	Amount       int64
	Currency     string
	Status       string
	Metadata     map[string]string
}

// NativePaymentOrder represents a native payment order (Alipay/WeChat).
// This is used for redirect-based or QR code payment flows.
type NativePaymentOrder struct {
	OrderID     string            // Our internal order ID
	TradeNo     string            // Provider's trade number
	PayURL      string            // Payment URL for redirect (web/h5)
	QRCode      string            // QR code content for scan payment
	AppPayData  string            // App SDK payment data (JSON string)
	MiniPayData map[string]string // Mini program payment params
	Amount      int64
	Currency    string
	ExpireTime  int64 // Unix timestamp when the order expires
}

// Charge represents a charge from the provider.
type Charge struct {
	ID             string
	Amount         int64
	Currency       string
	Status         string
	PaymentMethod  string
	FailureCode    string
	FailureMessage string
}

// Subscription represents a subscription from the provider.
type Subscription struct {
	ID                string
	CustomerID        string
	Status            string
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
	CancelAtPeriodEnd  bool
}

// Refund represents a refund from the provider.
type Refund struct {
	ID       string
	ChargeID string
	Amount   int64
	Status   string
	Reason   string
}

// Customer represents a customer from the provider.
type Customer struct {
	ID    string
	Email string
}

// PaymentMethodDetails represents payment method details.
type PaymentMethodDetails struct {
	ID        string
	Type      string
	CardBrand string
	CardLast4 string
	ExpMonth  int
	ExpYear   int
	IsDefault bool
}

// NotifyResult represents the result of a payment notification from provider.
type NotifyResult struct {
	TradeNo     string // Provider's trade number
	OutTradeNo  string // Our order/payment ID
	Amount      int64  // Amount in cents
	Status      string // Payment status
	PayerID     string // Payer's ID (openid for WeChat, buyer_id for Alipay)
	PayTime     int64  // Unix timestamp of payment
	RawData     string // Raw notification data for logging
	SuccessResp string // Response to send back to provider on success
}

// RefundResult represents the result of a refund.
type RefundResult struct {
	RefundNo     string // Provider's refund number
	OutRefundNo  string // Our refund ID
	Amount       int64  // Refund amount in cents
	Status       string // Refund status
	RefundTime   int64  // Unix timestamp of refund
}

// Provider defines the interface for payment providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Customer management
	CreateCustomer(ctx context.Context, email, name string) (*Customer, error)
	GetCustomer(ctx context.Context, customerID string) (*Customer, error)

	// Payment intents (Stripe-style)
	CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerID string, metadata map[string]string) (*PaymentIntent, error)
	GetPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntent, error)
	CancelPaymentIntent(ctx context.Context, paymentIntentID string) error

	// Subscriptions
	CreateSubscription(ctx context.Context, customerID, priceID string) (*Subscription, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, params map[string]interface{}) (*Subscription, error)
	CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error

	// Refunds
	CreateRefund(ctx context.Context, chargeID string, amount int64, reason string) (*Refund, error)

	// Payment methods
	AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
	ListPaymentMethods(ctx context.Context, customerID string) ([]*PaymentMethodDetails, error)
	SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error

	// Webhooks
	VerifyWebhookSignature(payload []byte, signature string) error
}

// NativePaymentProvider extends Provider for native payment flows (Alipay/WeChat).
// These providers use redirect/QR code flows instead of client-side SDKs.
type NativePaymentProvider interface {
	Provider

	// CreateNativePayment creates a payment order for native payment flow.
	// scene: payment scenario (web, h5, app, native, mini)
	// orderID: our internal order ID
	// amount: payment amount in cents
	// subject: payment subject/title
	// description: payment description
	// notifyURL: callback URL for async notification
	// returnURL: redirect URL after payment (for web/h5)
	// metadata: additional metadata
	CreateNativePayment(ctx context.Context, scene PaymentScene, orderID string, amount int64, subject, description, notifyURL, returnURL string, metadata map[string]string) (*NativePaymentOrder, error)

	// QueryPayment queries the payment status from provider.
	QueryPayment(ctx context.Context, orderID, tradeNo string) (*NotifyResult, error)

	// ClosePayment closes/cancels an unpaid order.
	ClosePayment(ctx context.Context, orderID, tradeNo string) error

	// RefundPayment creates a refund for a paid order.
	// Returns the refund result.
	RefundPayment(ctx context.Context, orderID, tradeNo, refundID string, refundAmount, totalAmount int64, reason string) (*RefundResult, error)

	// QueryRefund queries the refund status.
	QueryRefund(ctx context.Context, orderID, refundID string) (*RefundResult, error)

	// ParseNotify parses and verifies the async notification from provider.
	// Returns the parsed result and verification status.
	ParseNotify(ctx context.Context, body []byte, headers map[string]string) (*NotifyResult, error)
}
