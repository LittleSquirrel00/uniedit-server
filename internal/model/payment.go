package model

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the status of a payment.
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusCanceled   PaymentStatus = "canceled"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// IsTerminal returns true if the status is a terminal state.
func (s PaymentStatus) IsTerminal() bool {
	return s == PaymentStatusSucceeded || s == PaymentStatusFailed ||
		s == PaymentStatusCanceled || s == PaymentStatusRefunded
}

// IsSucceeded returns true if the status is succeeded.
func (s PaymentStatus) IsSucceeded() bool {
	return s == PaymentStatusSucceeded
}

// CanTransitionTo returns true if the status can transition to the target status.
func (s PaymentStatus) CanTransitionTo(target PaymentStatus) bool {
	switch s {
	case PaymentStatusPending:
		return target == PaymentStatusProcessing || target == PaymentStatusSucceeded ||
			target == PaymentStatusFailed || target == PaymentStatusCanceled
	case PaymentStatusProcessing:
		return target == PaymentStatusSucceeded || target == PaymentStatusFailed ||
			target == PaymentStatusCanceled
	case PaymentStatusSucceeded:
		return target == PaymentStatusRefunded
	case PaymentStatusFailed, PaymentStatusCanceled, PaymentStatusRefunded:
		return false
	default:
		return false
	}
}

// PaymentMethod represents a payment method type.
type PaymentMethod string

const (
	PaymentMethodCard   PaymentMethod = "card"
	PaymentMethodAlipay PaymentMethod = "alipay"
	PaymentMethodWechat PaymentMethod = "wechat"
)

// IsNative returns true if this is a native China payment method.
func (m PaymentMethod) IsNative() bool {
	return m == PaymentMethodAlipay || m == PaymentMethodWechat
}

// PaymentProvider represents a payment provider.
type PaymentProvider string

const (
	PaymentProviderStripe PaymentProvider = "stripe"
	PaymentProviderAlipay PaymentProvider = "alipay"
	PaymentProviderWechat PaymentProvider = "wechat"
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

// Payment represents a payment record.
type Payment struct {
	ID                    uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	OrderID               uuid.UUID      `json:"order_id" gorm:"type:uuid;not null;index"`
	UserID                uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Amount                int64          `json:"amount"`
	Currency              string         `json:"currency" gorm:"default:usd"`
	Method                PaymentMethod  `json:"method"`
	Status                PaymentStatus  `json:"status" gorm:"not null;default:pending"`
	Provider              string         `json:"provider" gorm:"default:stripe"`
	StripePaymentIntentID string         `json:"stripe_payment_intent_id,omitempty"`
	StripeChargeID        string         `json:"stripe_charge_id,omitempty"`
	TradeNo               string         `json:"trade_no,omitempty" gorm:"index"`
	PayerID               string         `json:"payer_id,omitempty"`
	FailureCode           *string        `json:"failure_code,omitempty"`
	FailureMessage        *string        `json:"failure_message,omitempty"`
	RefundedAmount        int64          `json:"refunded_amount" gorm:"default:0"`
	SucceededAt           *time.Time     `json:"succeeded_at,omitempty"`
	FailedAt              *time.Time     `json:"failed_at,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

// TableName returns the table name for GORM.
func (Payment) TableName() string {
	return "payments"
}

// WebhookEvent represents a stored webhook event for idempotency.
type WebhookEvent struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Provider    string     `json:"provider" gorm:"not null;index"`
	EventID     string     `json:"event_id" gorm:"uniqueIndex:idx_provider_event;not null"`
	EventType   string     `json:"event_type" gorm:"not null"`
	TradeNo     string     `json:"trade_no,omitempty" gorm:"index"`
	OutTradeNo  string     `json:"out_trade_no,omitempty" gorm:"index"`
	Data        string     `json:"data" gorm:"type:jsonb"`
	Processed   bool       `json:"processed" gorm:"default:false"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName returns the table name for GORM.
func (WebhookEvent) TableName() string {
	return "payment_webhook_events"
}

// PaymentFilter represents payment query filters.
type PaymentFilter struct {
	UserID   *uuid.UUID     `json:"user_id"`
	OrderID  *uuid.UUID     `json:"order_id"`
	Status   *PaymentStatus `json:"status"`
	Provider *string        `json:"provider"`
	PaginationRequest
}

// --- Request/Response DTOs ---

// CreatePaymentIntentRequest represents a request to create a payment intent.
type CreatePaymentIntentRequest struct {
	OrderID uuid.UUID `json:"order_id" binding:"required"`
}

// PaymentIntentResponse represents a payment intent response.
type PaymentIntentResponse struct {
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
}

// CreateNativePaymentRequest represents a request to create a native payment.
type CreateNativePaymentRequest struct {
	OrderID   uuid.UUID    `json:"order_id" binding:"required"`
	Method    string       `json:"method" binding:"required,oneof=alipay wechat"`
	Scene     PaymentScene `json:"scene" binding:"required"`
	ReturnURL string       `json:"return_url"`
	OpenID    string       `json:"openid"` // For WeChat mini program
}

// NativePaymentResponse represents a native payment response.
type NativePaymentResponse struct {
	PaymentID   uuid.UUID         `json:"payment_id"`
	OrderID     uuid.UUID         `json:"order_id"`
	Method      string            `json:"method"`
	PayURL      string            `json:"pay_url,omitempty"`
	QRCode      string            `json:"qr_code,omitempty"`
	AppPayData  string            `json:"app_pay_data,omitempty"`
	MiniPayData map[string]string `json:"mini_pay_data,omitempty"`
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	ExpireTime  int64             `json:"expire_time"`
}

// RefundRequest represents a refund request.
type RefundRequest struct {
	PaymentID uuid.UUID `json:"payment_id" binding:"required"`
	Amount    int64     `json:"amount"` // 0 means full refund
	Reason    string    `json:"reason"`
}

// PaymentMethodInfo represents payment method information.
type PaymentMethodInfo struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CardBrand string `json:"card_brand,omitempty"`
	CardLast4 string `json:"card_last4,omitempty"`
	ExpMonth  int    `json:"exp_month,omitempty"`
	ExpYear   int    `json:"exp_year,omitempty"`
	IsDefault bool   `json:"is_default"`
}

// --- Provider Types ---

// ProviderPaymentIntent represents a payment intent from the provider.
type ProviderPaymentIntent struct {
	ID           string
	ClientSecret string
	Amount       int64
	Currency     string
	Status       string
	Metadata     map[string]string
}

// ProviderNativeOrder represents a native payment order from the provider.
type ProviderNativeOrder struct {
	OrderID     string
	TradeNo     string
	PayURL      string
	QRCode      string
	AppPayData  string
	MiniPayData map[string]string
	Amount      int64
	Currency    string
	ExpireTime  int64
}

// ProviderRefund represents a refund from the provider.
type ProviderRefund struct {
	ID       string
	ChargeID string
	Amount   int64
	Status   string
	Reason   string
}

// ProviderNotifyResult represents the result of a payment notification.
type ProviderNotifyResult struct {
	TradeNo     string
	OutTradeNo  string
	Amount      int64
	Status      string
	PayerID     string
	PayTime     int64
	RawData     string
	SuccessResp string
}

// ProviderCustomer represents a customer from the provider.
type ProviderCustomer struct {
	ID    string
	Email string
}
